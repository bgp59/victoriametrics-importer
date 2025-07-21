// ClientDoer interface for testing.

package vmi_testutils

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

var ErrHttpClientDoerMockCancelled = errors.New("HttpClientDoerMock cancelled")
var ErrHttpClientDoerMockPlayback = errors.New("HttpClientDoerMock playback error")
var ErrHttpClientDoerMockGeneric = errors.New("HttpClientDoerMock generic error")

// The request <-> response mapping is keyed by URL and it consists of a pair of
// channels of length 1.
type HttpClientDoerMockRespErr struct {
	Response *http.Response
	Error    error
}

type HttpClientDoerPlaybackEntry struct {
	Url      string
	Response *http.Response
	Error    error
}

type HttpClientDoerMockChannels struct {
	req     chan *http.Request
	respErr chan *HttpClientDoerMockRespErr
}

type HttpClientDoerMock struct {
	channels map[string]*HttpClientDoerMockChannels
	ctx      context.Context
	cancelFn context.CancelFunc
	mu       *sync.Mutex
	wg       *sync.WaitGroup
}

type HttpClientDoerPlaybackRequest struct {
	Url string
	// The received request, with the body set to nil; the body is stored
	// separately:
	Request *http.Request
	// The body of the request above:
	Body []byte
}

func NewHttpClientDoerMock(timeout time.Duration) *HttpClientDoerMock {
	mock := &HttpClientDoerMock{
		channels: make(map[string]*HttpClientDoerMockChannels, 0),
		mu:       &sync.Mutex{},
		wg:       &sync.WaitGroup{},
	}
	if timeout > 0 {
		mock.ctx, mock.cancelFn = context.WithTimeout(context.Background(), timeout)
	} else {
		mock.ctx, mock.cancelFn = context.WithCancel(context.Background())
	}
	return mock
}

func (mock *HttpClientDoerMock) Cancel() {
	mock.cancelFn()
	mock.wg.Wait()
}

func (mock *HttpClientDoerMock) getChannels(url string) *HttpClientDoerMockChannels {
	mock.mu.Lock()
	defer mock.mu.Unlock()
	channels := mock.channels[url]
	if channels == nil {
		channels = &HttpClientDoerMockChannels{
			req:     make(chan *http.Request, 1),
			respErr: make(chan *HttpClientDoerMockRespErr, 1),
		}
		mock.channels[url] = channels
	}
	return channels
}

func httpClientDoerMockAddReqToRes(req *http.Request, resp *http.Response) *http.Response {
	if resp == nil {
		return nil
	}
	newResp := *resp
	if newResp.Status == "" {
		newResp.Status = fmt.Sprintf(
			"%d %s", newResp.StatusCode, http.StatusText(newResp.StatusCode),
		)
	}
	newResp.Request = req.Clone(context.Background())
	newResp.Request.Body = nil
	return &newResp
}

func (mock *HttpClientDoerMock) Do(req *http.Request) (*http.Response, error) {
	mock.wg.Add(1)
	defer mock.wg.Done()
	url := req.URL.String()
	channels := mock.getChannels(url)
	cancelErr := fmt.Errorf("%s %q: %w", req.Method, url, ErrHttpClientDoerMockCancelled)
	select {
	case <-mock.ctx.Done():
		return nil, cancelErr
	case channels.req <- req:
	}

	select {
	case <-mock.ctx.Done():
		return nil, cancelErr
	case respErr := <-channels.respErr:
		return httpClientDoerMockAddReqToRes(req, respErr.Response), respErr.Error
	}
}

func (mock *HttpClientDoerMock) GetRequest(url string) (*http.Request, error) {
	mock.wg.Add(1)
	defer mock.wg.Done()
	channels := mock.getChannels(url)
	select {
	case <-mock.ctx.Done():
		return nil, fmt.Errorf("get req for %q: %w", url, ErrHttpClientDoerMockCancelled)
	case req := <-channels.req:
		return req, nil
	}
}

func (mock *HttpClientDoerMock) SendResponse(url string, resp *http.Response, err error) error {
	mock.wg.Add(1)
	defer mock.wg.Done()
	channels := mock.getChannels(url)
	select {
	case <-mock.ctx.Done():
		return fmt.Errorf("send resp to %q: %w", url, ErrHttpClientDoerMockCancelled)
	case channels.respErr <- &HttpClientDoerMockRespErr{resp, err}:
		return nil
	}
}

func (mock *HttpClientDoerMock) CloseIdleConnections() {}

func (mock *HttpClientDoerMock) Play(playbook []*HttpClientDoerPlaybackEntry) ([]*HttpClientDoerPlaybackRequest, error) {
	var (
		err  error
		req  *http.Request
		body []byte
	)

	requests := make([]*HttpClientDoerPlaybackRequest, len(playbook))
	for i, entry := range playbook {
		url := entry.Url
		req, err = mock.GetRequest(url)
		if err != nil {
			break
		}
		err = mock.SendResponse(url, entry.Response, entry.Error)
		if err != nil {
			break
		}
		if req.Body != nil {
			body, err = io.ReadAll(req.Body)
			if err != nil {
				break
			}
			req.Body = nil
		} else {
			body = nil
		}
		requests[i] = &HttpClientDoerPlaybackRequest{
			Url:     url,
			Request: req.Clone(context.Background()),
			Body:    body,
		}
	}

	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrHttpClientDoerMockPlayback, err)
	}
	return requests, nil
}
