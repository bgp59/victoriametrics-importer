// http client pool for lsvmi

package vmi_internal

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// A VMI is configured with a list of URL endpoints for import.
//
// The usable endpoints are placed into the healthy sub-list and its head is the
// current one in use for requests. If a transport error occurs, the endpoint is
// moved to the back of the list. When the number of transport errors exceeds a
// certain threshold, the endpoint is removed from the healthy list and it will
// be checked periodically via a test HTTP request. When the latter succeeds,
// the endpoint is returned to  the tail of the healthy list.
//
// To ensure a balanced use of all the endpoints, the healthy list is rotated
// periodically such that each endpoint will eventually be at the head. This
// also gives a chance for closing idle connections to endpoints not currently
// at the head. If the list has just one element then the idle connections are
// closed explicitly.

var epPoolLog = NewCompLogger("http_endpoint_pool")

const (
	// Endpoint default values:
	HTTP_ENDPOINT_URL_DEFAULT                      = "http://localhost:8428/api/v1/import/prometheus"
	HTTP_ENDPOINT_MARK_UNHEALTHY_THRESHOLD_DEFAULT = 1

	// Endpoint config pool default values:
	HTTP_ENDPOINT_POOL_CONFIG_SHUFFLE_DEFAULT                 = false
	HTTP_ENDPOINT_POOL_CONFIG_HEALTHY_ROTATE_INTERVAL_DEFAULT = 5 * time.Minute
	HTTP_ENDPOINT_POOL_CONFIG_ERROR_RESET_INTERVAL_DEFAULT    = 1 * time.Minute
	HTTP_ENDPOINT_POOL_CONFIG_HEALTH_CHECK_INTERVAL_DEFAULT   = 5 * time.Second
	HTTP_ENDPOINT_POOL_CONFIG_HEALTHY_MAX_WAIT_DEFAULT        = 10 * time.Second
	HTTP_ENDPOINT_POOL_CONFIG_SEND_BUFFER_TIMEOUT_DEFAULT     = 20 * time.Second
	HTTP_ENDPOINT_POOL_CONFIG_RATE_LIMIT_MBPS_DEFAULT         = ""
	// Endpoint config definitions, later they may be configurable:
	HTTP_ENDPOINT_POOL_HEALTHY_CHECK_MIN_INTERVAL    = 1 * time.Second
	HTTP_ENDPOINT_POOL_HEALTHY_POLL_INTERVAL         = 500 * time.Millisecond
	HTTP_ENDPOINT_POOL_HEALTH_CHECK_ERR_LOG_INTERVAL = 10 * time.Second

	// http.Transport config default values:
	//   Dialer config default values:
	HTTP_ENDPOINT_POOL_CONFIG_TCP_CONN_TIMEOUT_DEFAULT        = 2 * time.Second
	HTTP_ENDPOINT_POOL_CONFIG_TCP_KEEP_ALIVE_DEFAULT          = 15 * time.Second
	HTTP_ENDPOINT_POOL_CONFIG_MAX_IDLE_CONNS_DEFAULT          = 0 // No limit
	HTTP_ENDPOINT_POOL_CONFIG_MAX_IDLE_CONNS_PER_HOST_DEFAULT = 1
	HTTP_ENDPOINT_POOL_CONFIG_MAX_CONNS_PER_HOST_DEFAULT      = 0 // No limit
	HTTP_ENDPOINT_POOL_CONFIG_IDLE_CONN_TIMEOUT_DEFAULT       = 1 * time.Minute
	// http.Client config default values:
	HTTP_ENDPOINT_POOL_CONFIG_RESPONSE_TIMEOUT_DEFAULT = 5 * time.Second

	// Prefixes for the password field:
	HTTP_ENDPOINT_POOL_CONFIG_PASSWORD_FILE_PREFIX = "file:"
	HTTP_ENDPOINT_POOL_CONFIG_PASSWORD_ENV_PREFIX  = "env:"
	HTTP_ENDPOINT_POOL_CONFIG_PASSWORD_PASS_PREFIX = "pass:"
)

// The HTTP endpoint pool interface as seen by the compressor:
type Sender interface {
	SendBuffer(b []byte, timeout time.Duration, gzipped bool) error
}

// Endpoint stats:
const (
	HTTP_ENDPOINT_STATS_SEND_BUFFER_COUNT = iota
	HTTP_ENDPOINT_STATS_SEND_BUFFER_BYTE_COUNT
	HTTP_ENDPOINT_STATS_SEND_BUFFER_ERROR_COUNT
	HTTP_ENDPOINT_STATS_HEALTH_CHECK_COUNT
	HTTP_ENDPOINT_STATS_HEALTH_CHECK_ERROR_COUNT
	// Must be last:
	HTTP_ENDPOINT_STATS_LEN
)

// Endpoint pool stats:
const (
	HTTP_ENDPOINT_POOL_STATS_HEALTHY_ROTATE_COUNT = iota
	HTTP_ENDPOINT_POOL_STATS_NO_HEALTHY_EP_ERROR_COUNT
	// Must be last:
	HTTP_ENDPOINT_POOL_STATS_LEN
)

type HttpEndpointStats []uint64

type HttpPoolStats []uint64

type HttpEndpointPoolStats struct {
	PoolStats HttpPoolStats
	// Endpoint stats are indexed by URL:
	EndpointStats map[string]HttpEndpointStats
}

func NewHttpEndpointPoolStats() *HttpEndpointPoolStats {
	return &HttpEndpointPoolStats{
		PoolStats:     make(HttpPoolStats, HTTP_ENDPOINT_POOL_STATS_LEN),
		EndpointStats: make(map[string]HttpEndpointStats),
	}
}

func (pool *HttpEndpointPool) SnapStats(to *HttpEndpointPoolStats) *HttpEndpointPoolStats {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	stats := pool.stats
	if stats == nil {
		return nil
	}
	if to == nil {
		to = NewHttpEndpointPoolStats()
	}

	copy(to.PoolStats, stats.PoolStats)

	for url, epStats := range stats.EndpointStats {
		toEpStats := to.EndpointStats[url]
		if toEpStats == nil {
			toEpStats = make(HttpEndpointStats, HTTP_ENDPOINT_STATS_LEN)
			to.EndpointStats[url] = toEpStats
		}
		copy(toEpStats, epStats)
	}

	return to
}

// Define a mockable interface to substitute http.Client.Do() for testing purposes:
type HttpClientDoer interface {
	Do(req *http.Request) (*http.Response, error)
	CloseIdleConnections()
}

// Interface for a http.Request body w/ retries:
type ReadSeekRewindCloser interface {
	io.ReadSeekCloser
	Rewind() error
}

// Convert bytes.Reader into ReadSeekRewindCloser such that it can be used
// as body for http.Request w/ retries:
type BytesReadSeekCloser struct {
	rs        io.ReadSeeker
	closed    bool
	closedPos int64
}

func (brsc *BytesReadSeekCloser) Read(p []byte) (int, error) {
	if brsc.closed {
		return 0, nil
	}
	return brsc.rs.Read(p)
}

func (brsc *BytesReadSeekCloser) Seek(offset int64, whence int) (int64, error) {
	if brsc.closed {
		return brsc.closedPos, nil
	}
	return brsc.rs.Seek(offset, whence)
}

func (brsc *BytesReadSeekCloser) Close() error {
	brsc.closed = true
	return nil
}

// Reuse, for HTTP retries:
func (brsc *BytesReadSeekCloser) Rewind() error {
	brsc.closed = false
	_, err := brsc.Seek(0, io.SeekStart)
	return err
}

func NewBytesReadSeekCloser(b []byte) *BytesReadSeekCloser {
	return &BytesReadSeekCloser{
		rs:        bytes.NewReader(b),
		closed:    len(b) == 0,
		closedPos: int64(len(b)),
	}
}

type HttpEndpoint struct {
	// The URL that accepts PUT w/ Prometheus exposition format data:
	url string
	// The parsed format for above, to be used for http calls:
	URL *url.URL
	// The threshold for failed accesses count, used for declaring the endpoint
	// unhealthy; this may be > 1 for cases where the host name part of the URL
	// is some kind of a DNS pool which is resolved to a list of addresses, in
	// which case it should be set to the number of pool members. Just because
	// one member is unhealthy, it doesn't mean that others cannot be used. The
	// net/http Transport connection cache will remove the failed connection and
	// the name to address resolution mechanism should no longer resolve to this
	// failed IP.
	markUnhealthyThreshold int
	// State:
	healthy bool
	// The number of errors so far that is compared against the threshold above:
	numErrors int
	// The timestamp of the most recent error:
	errorTs time.Time
	// Doubly linked list:
	prev, next *HttpEndpoint
}

type HttpEndpointConfig struct {
	URL                    string
	MarkUnhealthyThreshold int `yaml:"mark_unhealthy_threshold"`
}

// The list of HTTP codes that denote success:
var HttpEndpointPoolSuccessCodes = map[int]bool{
	http.StatusOK:        true,
	http.StatusNoContent: true,
}

// The list of HTTP codes that should be retried:
var HttpEndpointPoolRetryCodes = map[int]bool{}

// Error codes:
var ErrHttpEndpointPoolNoHealthyEP = errors.New("no healthy HTTP endpoint available")

func DefaultHttpEndpointConfig() *HttpEndpointConfig {
	return &HttpEndpointConfig{
		URL:                    HTTP_ENDPOINT_URL_DEFAULT,
		MarkUnhealthyThreshold: 0, // i.e. fallback over pool definition or default
	}
}

func NewHttpEndpoint(cfg *HttpEndpointConfig) (*HttpEndpoint, error) {
	var err error
	if cfg == nil {
		cfg = DefaultHttpEndpointConfig()
	}
	ep := &HttpEndpoint{
		url:                    cfg.URL,
		markUnhealthyThreshold: cfg.MarkUnhealthyThreshold,
	}
	if ep.URL, err = url.Parse(ep.url); err != nil {
		err = fmt.Errorf("NewHttpEndpoint(%s): %v", ep.url, err)
		ep = nil
	}
	return ep, err
}

type HttpEndpointDoublyLinkedList struct {
	head, tail *HttpEndpoint
}

func (epDblLnkList *HttpEndpointDoublyLinkedList) Insert(ep, after *HttpEndpoint) {
	ep.prev = after
	if after != nil {
		ep.next = after.next
		after.next = ep
	} else {
		// Add to head:
		ep.next = epDblLnkList.head
		epDblLnkList.head = ep
	}
	if ep.next == nil {
		// Added to tail:
		epDblLnkList.tail = ep
	}
}

func (epDblLnkList *HttpEndpointDoublyLinkedList) Remove(ep *HttpEndpoint) {
	if ep.prev != nil {
		ep.prev.next = ep.next
	} else {
		epDblLnkList.head = ep.next
	}
	if ep.next != nil {
		ep.next.prev = ep.prev
	} else {
		epDblLnkList.tail = ep.prev
	}
	ep.prev = nil
	ep.next = nil
}

func (epDblLnkList *HttpEndpointDoublyLinkedList) AddToHead(ep *HttpEndpoint) {
	epDblLnkList.Insert(ep, nil)
}

func (epDblLnkList *HttpEndpointDoublyLinkedList) AddToTail(ep *HttpEndpoint) {
	epDblLnkList.Insert(ep, epDblLnkList.tail)
}

type HttpEndpointPool struct {
	// The healthy list:
	healthy *HttpEndpointDoublyLinkedList
	// Authorization header, if any:
	authorization string
	// How often to rotate the healthy list. Set to 0 to rotate after every use
	// or to -1 to disable the rotation:
	healthyRotateInterval time.Duration
	// The time stamp when the last change to the head of the healthy list
	// occurred (most likely due to rotation):
	healthyHeadChangeTs time.Time
	// The rotation, which occurs *before* the endpoint is selected, should be
	// disabled for its 1st use; for instance the endpoint has been just
	// promoted to the head because the previous one had an error.
	firstUse bool
	// A failed endpoint is moved to the back of the usable list, as long as the
	// cumulative error count is less than the threshold. If enough time passes
	// before it makes it back to the head of the list, then the error count
	// used to declare it unhealthy is no longer relevant and it should be
	// reset. The following defines the interval after which older errors may be
	// ignored; use 0 to disable:
	errorResetInterval time.Duration
	// How often to check if an unhealthy endpoint has become healthy:
	healthCheckInterval time.Duration
	// How long to wait for a healthy endpoint, in case healthy list is empty;
	// normally this should be > HealthCheckInterval.
	healthyMaxWait time.Duration
	// How often to poll for a healthy endpoint; this is not configurable for now:
	healthyPollInterval time.Duration
	// How often to log health check errors, if repeated:
	healthCheckErrLogInterval time.Duration
	// How long to wait for a SendBuffer call to succeed; normally this should
	// be longer than healthyMaxWait or other HTTP timeouts:
	sendBufferTimeout time.Duration
	// Rate limiting credit mechanism, if not nil:
	credit CreditController
	// The http client as a mockable interface:
	client HttpClientDoer
	// Access lock:
	mu *sync.Mutex
	// Context and wait group for health checking goroutines:
	ctx         context.Context
	ctxCancelFn context.CancelFunc
	wg          *sync.WaitGroup
	// Whether the pool was shutdown or not:
	shutdown bool
	// Endpoint and pool stats:
	stats *HttpEndpointPoolStats
}

type HttpEndpointPoolConfig struct {
	Endpoints              []*HttpEndpointConfig `yaml:"endpoints"`
	Username               string                `yaml:"username"`
	Password               string                `yaml:"password"`
	MarkUnhealthyThreshold int                   `yaml:"mark_unhealthy_threshold"`
	Shuffle                bool                  `yaml:"shuffle"`
	HealthyRotateInterval  time.Duration         `yaml:"healthy_rotate_interval"`
	ErrorResetInterval     time.Duration         `yaml:"error_reset_interval"`
	HealthCheckInterval    time.Duration         `yaml:"health_check_interval"`
	HealthyMaxWait         time.Duration         `yaml:"healthy_max_wait"`
	SendBufferTimeout      time.Duration         `yaml:"send_buffer_timeout"`
	RateLimitMbps          string                `yaml:"rate_limit_mbps"`
	IgnoreTLSVerify        bool                  `yaml:"ignore_tls_verify"`
	TcpConnTimeout         time.Duration         `yaml:"tcp_conn_timeout"`
	TcpKeepAlive           time.Duration         `yaml:"tcp_keep_alive"`
	MaxIdleConns           int                   `yaml:"max_idle_conns"`
	MaxIdleConnsPerHost    int                   `yaml:"max_idle_conns_per_host"`
	MaxConnsPerHost        int                   `yaml:"max_conns_per_host"`
	IdleConnTimeout        time.Duration         `yaml:"idle_conn_timeout"`
	ResponseTimeout        time.Duration         `yaml:"response_timeout"`
}

func DefaultHttpEndpointPoolConfig() *HttpEndpointPoolConfig {
	return &HttpEndpointPoolConfig{
		Shuffle:                HTTP_ENDPOINT_POOL_CONFIG_SHUFFLE_DEFAULT,
		MarkUnhealthyThreshold: 0, // i.e. fallback over default
		HealthyRotateInterval:  HTTP_ENDPOINT_POOL_CONFIG_HEALTHY_ROTATE_INTERVAL_DEFAULT,
		ErrorResetInterval:     HTTP_ENDPOINT_POOL_CONFIG_ERROR_RESET_INTERVAL_DEFAULT,
		HealthCheckInterval:    HTTP_ENDPOINT_POOL_CONFIG_HEALTH_CHECK_INTERVAL_DEFAULT,
		HealthyMaxWait:         HTTP_ENDPOINT_POOL_CONFIG_HEALTHY_MAX_WAIT_DEFAULT,
		SendBufferTimeout:      HTTP_ENDPOINT_POOL_CONFIG_SEND_BUFFER_TIMEOUT_DEFAULT,
		RateLimitMbps:          HTTP_ENDPOINT_POOL_CONFIG_RATE_LIMIT_MBPS_DEFAULT,
		TcpConnTimeout:         HTTP_ENDPOINT_POOL_CONFIG_TCP_CONN_TIMEOUT_DEFAULT,
		TcpKeepAlive:           HTTP_ENDPOINT_POOL_CONFIG_TCP_KEEP_ALIVE_DEFAULT,
		MaxIdleConns:           HTTP_ENDPOINT_POOL_CONFIG_MAX_IDLE_CONNS_DEFAULT,
		MaxIdleConnsPerHost:    HTTP_ENDPOINT_POOL_CONFIG_MAX_IDLE_CONNS_PER_HOST_DEFAULT,
		MaxConnsPerHost:        HTTP_ENDPOINT_POOL_CONFIG_MAX_CONNS_PER_HOST_DEFAULT,
		IdleConnTimeout:        HTTP_ENDPOINT_POOL_CONFIG_IDLE_CONN_TIMEOUT_DEFAULT,
		ResponseTimeout:        HTTP_ENDPOINT_POOL_CONFIG_RESPONSE_TIMEOUT_DEFAULT,
	}
}

// Used for command line argument override:
func (poolCfg *HttpEndpointPoolConfig) OverrideEndpoints(urlList string) {
	urls := strings.Split(urlList, ",")
	poolCfg.Endpoints = make([]*HttpEndpointConfig, len(urls))
	for i, url := range urls {
		poolCfg.Endpoints[i] = &HttpEndpointConfig{
			URL:                    url,
			MarkUnhealthyThreshold: 0, // i.e. use pool config fallback or default
		}
	}
}

func LoadPasswordSpec(password string) (string, error) {
	if strings.HasPrefix(password, HTTP_ENDPOINT_POOL_CONFIG_PASSWORD_FILE_PREFIX) {
		passwordFile := os.ExpandEnv(password[len(HTTP_ENDPOINT_POOL_CONFIG_PASSWORD_FILE_PREFIX):])
		if content, err := os.ReadFile(passwordFile); err != nil {
			return "", fmt.Errorf("LoadPasswordSpec: password file: %s: %v", passwordFile, err)
		} else {
			password = strings.TrimSpace(string(content))
		}
	} else if strings.HasPrefix(password, HTTP_ENDPOINT_POOL_CONFIG_PASSWORD_ENV_PREFIX) {
		password = os.Getenv(password[len(HTTP_ENDPOINT_POOL_CONFIG_PASSWORD_ENV_PREFIX):])
	} else if strings.HasPrefix(password, HTTP_ENDPOINT_POOL_CONFIG_PASSWORD_PASS_PREFIX) {
		password = password[len(HTTP_ENDPOINT_POOL_CONFIG_PASSWORD_PASS_PREFIX):]
	}
	return password, nil
}

func BuildHtmlBasicAuth(username, password string) (string, error) {
	authorization := ""
	if username != "" {
		if password, err := LoadPasswordSpec(password); err == nil {
			authorization = "Basic " + base64.StdEncoding.EncodeToString(
				[]byte(username+":"+password),
			)
		} else {
			return "", err
		}
	}
	return authorization, nil
}

func NewHttpEndpointPool(poolCfg *HttpEndpointPoolConfig) (*HttpEndpointPool, error) {
	var err error

	if poolCfg == nil {
		poolCfg = DefaultHttpEndpointPoolConfig()
	}

	authorization, err := BuildHtmlBasicAuth(poolCfg.Username, poolCfg.Password)
	if err != nil {
		return nil, fmt.Errorf("NewHttpEndpointPool: %v", err)
	}

	dialer := &net.Dialer{
		Timeout:   poolCfg.TcpConnTimeout,
		KeepAlive: poolCfg.TcpKeepAlive,
	}
	transport := &http.Transport{
		DialContext:         dialer.DialContext,
		DisableKeepAlives:   false,
		IdleConnTimeout:     poolCfg.IdleConnTimeout,
		MaxIdleConns:        poolCfg.MaxIdleConns,
		MaxIdleConnsPerHost: poolCfg.MaxIdleConnsPerHost,
		MaxConnsPerHost:     poolCfg.MaxConnsPerHost,
	}
	if poolCfg.IgnoreTLSVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	client := &http.Client{
		Timeout:   poolCfg.ResponseTimeout,
		Transport: transport,
	}

	healthCheckInterval := poolCfg.HealthCheckInterval
	if healthCheckInterval < HTTP_ENDPOINT_POOL_HEALTHY_CHECK_MIN_INTERVAL {
		epPoolLog.Warnf(
			"healthy_check_interval %s too small, it will be adjusted to %s",
			healthCheckInterval, HTTP_ENDPOINT_POOL_HEALTHY_CHECK_MIN_INTERVAL,
		)
		healthCheckInterval = HTTP_ENDPOINT_POOL_HEALTHY_CHECK_MIN_INTERVAL
	}
	epPool := &HttpEndpointPool{
		healthy:                   &HttpEndpointDoublyLinkedList{},
		authorization:             authorization,
		healthyPollInterval:       HTTP_ENDPOINT_POOL_HEALTHY_POLL_INTERVAL,
		healthCheckErrLogInterval: HTTP_ENDPOINT_POOL_HEALTH_CHECK_ERR_LOG_INTERVAL,
		healthyRotateInterval:     poolCfg.HealthyRotateInterval,
		errorResetInterval:        poolCfg.ErrorResetInterval,
		healthCheckInterval:       healthCheckInterval,
		sendBufferTimeout:         poolCfg.SendBufferTimeout,
		healthyMaxWait:            poolCfg.HealthyMaxWait,
		firstUse:                  true,
		client:                    client,
		mu:                        &sync.Mutex{},
		wg:                        &sync.WaitGroup{},
		stats:                     NewHttpEndpointPoolStats(),
	}
	epPool.ctx, epPool.ctxCancelFn = context.WithCancel(context.Background())
	if poolCfg.RateLimitMbps != "" {
		if epPool.credit, err = NewCreditFromSpec(poolCfg.RateLimitMbps); err != nil {
			return nil, fmt.Errorf("NewHttpEndpointPool: rate_limit_mbps: %v", err)
		}
	}

	epPoolLog.Infof("healthy_rotate_interval=%s", epPool.healthyRotateInterval)
	epPoolLog.Infof("error_reset_interval=%s", epPool.errorResetInterval)
	epPoolLog.Infof("health_check_interval=%s", epPool.healthCheckInterval)
	epPoolLog.Infof("healthy_max_wait=%s", epPool.healthyMaxWait)
	epPoolLog.Infof("healthy_poll_interval=%s", epPool.healthyPollInterval)
	epPoolLog.Infof("max_idle_conns=%d", transport.MaxIdleConns)
	epPoolLog.Infof("send_buffer_timeout=%s", epPool.sendBufferTimeout)
	epPoolLog.Infof("rate_limit_mbps=%v", epPool.credit)
	epPoolLog.Infof("tcp_conn_timeout=%s", dialer.Timeout)
	epPoolLog.Infof("tcp_keep_alive=%s", dialer.KeepAlive)
	epPoolLog.Infof("max_idle_conns_per_host=%d", transport.MaxIdleConnsPerHost)
	epPoolLog.Infof("max_conns_per_host=%d", transport.MaxConnsPerHost)
	epPoolLog.Infof("idle_conn_timeout=%s", transport.IdleConnTimeout)
	epPoolLog.Infof("response_timeout=%s", client.Timeout)

	endpoints := poolCfg.Endpoints
	if len(endpoints) == 0 {
		endpoints = []*HttpEndpointConfig{DefaultHttpEndpointConfig()}
	}
	if poolCfg.Shuffle && len(endpoints) > 1 {
		epPoolLog.Info("shuffle the endpoint list")
		rand.Shuffle(len(endpoints), func(i, j int) { endpoints[i], endpoints[j] = endpoints[j], endpoints[i] })
	}
	for _, epCfg := range endpoints {
		cfg := *epCfg
		if cfg.URL == "" {
			cfg.URL = HTTP_ENDPOINT_URL_DEFAULT
		}
		if cfg.MarkUnhealthyThreshold <= 0 {
			cfg.MarkUnhealthyThreshold = poolCfg.MarkUnhealthyThreshold
		}
		if cfg.MarkUnhealthyThreshold <= 0 {
			cfg.MarkUnhealthyThreshold = HTTP_ENDPOINT_MARK_UNHEALTHY_THRESHOLD_DEFAULT
		}
		if ep, err := NewHttpEndpoint(&cfg); err != nil {
			return nil, err
		} else {
			epPool.stats.EndpointStats[ep.url] = make(HttpEndpointStats, HTTP_ENDPOINT_STATS_LEN)
			epPool.MoveToHealthy(ep)
		}
	}
	if epPool.healthy.head == nil {
		epPoolLog.Warn(ErrHttpEndpointPoolNoHealthyEP)
	}

	return epPool, nil
}

func (epPool *HttpEndpointPool) HealthCheck(ep *HttpEndpoint) {
	defer epPool.wg.Done()

	var (
		prevErr        error
		prevStatusCode int       = -1
		errorLogTs     time.Time = time.Now()
	)

	sameErr := func(err1, err2 error) bool {
		return err1 == nil && err2 == nil ||
			err1 != nil && err2 != nil && err1.Error() == err2.Error()
	}

	sameStatus := func(prevStatusCode int, resp *http.Response) bool {
		return resp == nil && prevStatusCode == -1 ||
			resp != nil && prevStatusCode == resp.StatusCode
	}

	epPoolLog.Warnf("start health check for %s", ep.url)

	stats, mu, url := epPool.stats, epPool.mu, ep.url
	req, err := http.NewRequestWithContext(
		epPool.ctx,
		http.MethodPut,
		ep.url,
		nil,
	)
	if err != nil {
		epPoolLog.Warnf("health check req for %s: %v (disabled permanently)", ep.url, err)
		return
	}
	req.Header.Add("Content-Type", "text/html")
	if epPool.authorization != "" {
		req.Header.Add("Authorization", epPool.authorization)
	}

	ticker := time.NewTicker(epPool.healthCheckInterval)
	defer ticker.Stop()

	for repeatCount, healthy := 0, false; !healthy; {
		select {
		case <-epPool.ctx.Done():
			epPoolLog.Warnf("cancel health check for %s", ep.url)
			return
		case <-ticker.C:
			res, err := epPool.client.Do(req)
			if res != nil && res.Body != nil {
				res.Body.Close()
			}
			healthy = err == nil && res != nil && HttpEndpointPoolSuccessCodes[res.StatusCode]
			if healthy {
				epPoolLog.Infof("%s %q: %s", req.Method, req.URL, res.Status)
				epPool.MoveToHealthy(ep)
			} else {
				if !sameErr(err, prevErr) || !sameStatus(prevStatusCode, res) {
					repeatCount = 1
				} else {
					repeatCount += 1
				}
				if RootLogger.IsEnabledForDebug || repeatCount == 1 ||
					time.Since(errorLogTs) >= epPool.healthCheckErrLogInterval {
					repeatCountMsg := ""
					if repeatCount > 1 {
						repeatCountMsg = fmt.Sprintf(" (%d times)", repeatCount)
					}
					errorLogTs = time.Now()
					if err != nil {
						epPoolLog.Warnf("%v%s", err, repeatCountMsg)
					} else {
						epPoolLog.Warnf("%s %q: %s%s", req.Method, req.URL, res.Status, repeatCountMsg)
					}
				}
				prevErr = err
				if res != nil {
					prevStatusCode = res.StatusCode
				} else {
					prevStatusCode = -1
				}
			}
			mu.Lock()
			stats.EndpointStats[url][HTTP_ENDPOINT_STATS_HEALTH_CHECK_COUNT] += 1
			if !healthy {
				stats.EndpointStats[url][HTTP_ENDPOINT_STATS_HEALTH_CHECK_ERROR_COUNT] += 1
			}
			mu.Unlock()
		}
	}
}

func (epPool *HttpEndpointPool) ReportError(ep *HttpEndpoint) {
	epPool.mu.Lock()
	defer epPool.mu.Unlock()
	ep.numErrors += 1
	ep.errorTs = time.Now()
	epPoolLog.Warnf(
		"%s: error#: %d, threshold: %d",
		ep.url, ep.numErrors, ep.markUnhealthyThreshold,
	)
	if !ep.healthy {
		// Already in the unhealthy state:
		return
	}
	if ep.numErrors < ep.markUnhealthyThreshold {
		if epPool.healthy.head != epPool.healthy.tail {
			// Re-add at tail:
			epPool.healthy.Remove(ep)
			epPool.healthy.AddToTail(ep)
			epPool.firstUse = true
			if RootLogger.IsEnabledForDebug {
				epPoolLog.Debugf(
					"%s: error#: %d, threshold: %d rotated to healthy list tail",
					ep.url, ep.numErrors, ep.markUnhealthyThreshold,
				)
			}
		}
	} else {
		// Initiate health check:
		epPool.healthy.Remove(ep)
		ep.healthy = false
		if !epPool.shutdown {
			epPoolLog.Warnf("%s moved to health check", ep.url)
			epPool.wg.Add(1)
			go epPool.HealthCheck(ep)
		}
	}

	head := epPool.healthy.head
	if head == nil {
		epPoolLog.Warn(ErrHttpEndpointPoolNoHealthyEP)
	} else {
		if RootLogger.IsEnabledForDebug {
			epPoolLog.Debugf(
				"%s: error#: %d, threshold: %d is at the head of the healthy list",
				head.url, head.numErrors, head.markUnhealthyThreshold,
			)
		}
	}

}

func (epPool *HttpEndpointPool) MoveToHealthy(ep *HttpEndpoint) {
	epPool.mu.Lock()
	defer epPool.mu.Unlock()
	if ep.healthy {
		// Already in the healthy state:
		return
	}
	ep.healthy = true
	ep.numErrors = 0
	epPool.healthy.AddToTail(ep)
	if epPool.healthy.head == ep {
		epPoolLog.Infof("%s is at the head of the healthy list", ep.url)
	} else {
		epPoolLog.Infof("%s appended to the healthy list", ep.url)
	}
}

// Get the current healthy endpoint or nil if none available after max wait; if
// maxWait < 0 then the pool healthyMaxWait is used:
func (epPool *HttpEndpointPool) GetCurrentHealthy(maxWait time.Duration) *HttpEndpoint {
	if maxWait < 0 {
		maxWait = epPool.healthyMaxWait
	}

	epPool.mu.Lock()
	defer epPool.mu.Unlock()

	// There is no sync.Condition Wait with timeout, so poll until deadline or
	// shutdown, waiting for a healthy endpoint. It shouldn't impact the overall
	// efficiency since this is not the normal operating condition.
	deadline := time.Now().Add(maxWait)
	for epPool.healthy.head == nil && !epPool.shutdown {
		timeLeft := time.Until(deadline)
		if timeLeft <= 0 {
			return nil
		}
		epPool.mu.Unlock()
		time.Sleep(min(epPool.healthyPollInterval, timeLeft))
		epPool.mu.Lock()
	}
	ep := epPool.healthy.head
	if ep != nil {
		// Rotate as needed:
		if epPool.firstUse {
			epPool.healthyHeadChangeTs = time.Now()
			epPool.firstUse = false
		} else if epPool.healthyRotateInterval == 0 ||
			epPool.healthyRotateInterval > 0 &&
				time.Since(epPool.healthyHeadChangeTs) >= epPool.healthyRotateInterval {
			if epPool.healthy.head != epPool.healthy.tail {
				epPool.healthy.Remove(ep)
				epPool.healthy.AddToTail(ep)
				if RootLogger.IsEnabledForDebug {
					epPoolLog.Debugf(
						"%s: error#: %d, threshold: %d rotated to healthy list tail",
						ep.url, ep.numErrors, ep.markUnhealthyThreshold,
					)
				}
				ep = epPool.healthy.head
				epPool.healthyHeadChangeTs = time.Now()
				if RootLogger.IsEnabledForDebug {
					epPoolLog.Debugf(
						"%s: error#: %d, threshold: %d rotated to healthy list head",
						ep.url, ep.numErrors, ep.markUnhealthyThreshold,
					)
				}
				epPool.stats.PoolStats[HTTP_ENDPOINT_POOL_STATS_HEALTHY_ROTATE_COUNT] += 1
			}
		}
		// Apply error reset as needed:
		if ep.numErrors > 0 &&
			epPool.errorResetInterval > 0 &&
			time.Since(ep.errorTs) >= epPool.errorResetInterval {
			epPoolLog.Infof("%s: error#: %d->0)", ep.url, ep.numErrors)
			ep.numErrors = 0
		}
	}
	return ep
}

// SendBuffer: the main reason for the pool is to send buffers w/ load balancing
// and retries. If timeout is < 0 then the pool's sendBufferTimeout is used:
func (epPool *HttpEndpointPool) SendBuffer(b []byte, timeout time.Duration, gzipped bool) error {
	var body ReadSeekRewindCloser

	stats, mu := epPool.stats, epPool.mu

	header := http.Header{
		"Content-Type": {"text/html"},
	}
	if gzipped {
		header.Add("Content-Encoding", "gzip")
	}
	if epPool.authorization != "" {
		header.Add("Authorization", epPool.authorization)
	}

	mu.Lock()
	if epPool.credit != nil {
		body = NewCreditReader(epPool.credit, 128, b)
	} else {
		body = NewBytesReadSeekCloser(b)
	}
	mu.Unlock()

	if timeout < 0 {
		timeout = epPool.sendBufferTimeout
	}
	deadline := time.Now().Add(timeout)
	for attempt := 1; ; attempt++ {
		maxWait := time.Until(deadline)
		if maxWait < 0 {
			maxWait = 0
		}
		ep := epPool.GetCurrentHealthy(maxWait)
		if ep == nil {
			mu.Lock()
			stats.PoolStats[HTTP_ENDPOINT_POOL_STATS_NO_HEALTHY_EP_ERROR_COUNT] += 1
			mu.Unlock()
			return fmt.Errorf(
				"SendBuffer attempt# %d: %w", attempt, ErrHttpEndpointPoolNoHealthyEP,
			)
		}
		if attempt > 1 {
			body.Rewind()
		}
		req := &http.Request{
			Method: http.MethodPut,
			Header: header.Clone(),
			URL:    ep.URL,
			//ContentLength: int64(len(b)),
			Body: body,
		}
		res, err := epPool.client.Do(req)
		sent := err == nil && res != nil
		success := sent && HttpEndpointPoolSuccessCodes[res.StatusCode]
		nonRetryable := sent && !HttpEndpointPoolRetryCodes[res.StatusCode]

		url := ep.url
		epStats := stats.EndpointStats[url]
		mu.Lock()
		epStats[HTTP_ENDPOINT_STATS_SEND_BUFFER_COUNT] += 1
		if sent {
			epStats[HTTP_ENDPOINT_STATS_SEND_BUFFER_BYTE_COUNT] += uint64(len(b))
		}
		if !success {
			epStats[HTTP_ENDPOINT_STATS_SEND_BUFFER_ERROR_COUNT] += 1
		}
		mu.Unlock()

		if success {
			return nil
		}
		if nonRetryable {
			return fmt.Errorf(
				"SendBuffer attempt# %d: %s %s: %s", attempt, req.Method, ep.url, res.Status,
			)
		}
		// Report the failure:
		if err != nil {
			epPoolLog.Warnf("SendBuffer attempt# %d: %v", attempt, err)
		} else if res != nil {
			epPoolLog.Warnf("SendBuffer attempt# %d: %s %s: %s", attempt, req.Method, ep.url, res.Status)
		} else {
			epPoolLog.Warnf("SendBuffer attempt# %d: %s %s: no response", attempt, req.Method, ep.url)
		}
		// There is something wrong w/ the endpoint:
		epPool.ReportError(ep)
	}
}

// Needed for testing or clean exit in general:
func (epPool *HttpEndpointPool) Shutdown() {
	epPool.mu.Lock()
	toShutdown := !epPool.shutdown
	if toShutdown {
		epPool.shutdown = true
	}
	epPool.mu.Unlock()

	if !toShutdown {
		epPoolLog.Warn("pool already shutdown")
		return
	}

	epPoolLog.Info("initiate pool shutdown")
	epPoolLog.Info("stop health check goroutines")
	epPool.ctxCancelFn()
	epPool.wg.Wait()
	epPoolLog.Info("all health check goroutines completed")
	if credit, ok := epPool.credit.(*Credit); ok {
		epPool.mu.Lock()
		credit.StopReplenishWait()
		epPool.credit = nil
		epPool.mu.Unlock()
	}
	epPoolLog.Info("pool shutdown complete")
}
