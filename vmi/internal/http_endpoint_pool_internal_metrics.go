// Internal metrics for HTTP Endpoint Pool

package vmi_internal

import (
	"bytes"
	"fmt"
	"strconv"
)

var httpEndpointStatsDeltaMetricsNameMap = map[int]string{
	HTTP_ENDPOINT_STATS_SEND_BUFFER_COUNT:        HTTP_ENDPOINT_STATS_SEND_BUFFER_DELTA_METRIC,
	HTTP_ENDPOINT_STATS_SEND_BUFFER_BYTE_COUNT:   HTTP_ENDPOINT_STATS_SEND_BUFFER_BYTE_DELTA_METRIC,
	HTTP_ENDPOINT_STATS_SEND_BUFFER_ERROR_COUNT:  HTTP_ENDPOINT_STATS_SEND_BUFFER_ERROR_DELTA_METRIC,
	HTTP_ENDPOINT_STATS_HEALTH_CHECK_COUNT:       HTTP_ENDPOINT_STATS_HEALTH_CHECK_DELTA_METRIC,
	HTTP_ENDPOINT_STATS_HEALTH_CHECK_ERROR_COUNT: HTTP_ENDPOINT_STATS_HEALTH_CHECK_ERROR_DELTA_METRIC,
}

var httpEndpointPoolStatsDeltaMetricsNameMap = map[int]string{
	HTTP_ENDPOINT_POOL_STATS_HEALTHY_ROTATE_COUNT:      HTTP_ENDPOINT_POOL_STATS_HEALTHY_ROTATE_DELTA_METRIC,
	HTTP_ENDPOINT_POOL_STATS_NO_HEALTHY_EP_ERROR_COUNT: HTTP_ENDPOINT_POOL_STATS_NO_HEALTHY_EP_ERROR_DELTA_METRIC,
}

type httpEndpointPoolStatsIndexMetricMap map[int][]byte

type HttpEndpointPoolInternalMetrics struct {
	// Internal metrics, for common values:
	internalMetrics *InternalMetrics
	// Dual storage for snapping the stats, used as current, previous, toggled
	// after every metrics generation:
	stats [2]*HttpEndpointPoolStats
	// The current index:
	currIndex int
	// Cache for the endpoint metrics, `name{label="val",...}`, indexed by the
	// URL and the stats index:
	endpointDeltaMetricsCache map[string]httpEndpointPoolStatsIndexMetricMap
	// Cache for the pool metrics, `name{label="val",...}`,  indexed by the
	// stats index:
	poolDeltaMetricsCache httpEndpointPoolStatsIndexMetricMap
}

func NewHttpEndpointPoolInternalMetrics(internalMetrics *InternalMetrics) *HttpEndpointPoolInternalMetrics {
	return &HttpEndpointPoolInternalMetrics{
		internalMetrics:           internalMetrics,
		endpointDeltaMetricsCache: make(map[string]httpEndpointPoolStatsIndexMetricMap),
	}
}

func (eppim *HttpEndpointPoolInternalMetrics) updatePoolMetricsCache() {
	instance, hostname := eppim.internalMetrics.Instance, eppim.internalMetrics.Hostname
	eppim.poolDeltaMetricsCache = make(httpEndpointPoolStatsIndexMetricMap)
	for index, name := range httpEndpointPoolStatsDeltaMetricsNameMap {
		eppim.poolDeltaMetricsCache[index] = []byte(fmt.Sprintf(
			`%s{%s="%s",%s="%s"} `, // N.B. include the whitespace separating the metric from value
			name,
			INSTANCE_LABEL_NAME, instance,
			HOSTNAME_LABEL_NAME, hostname,
		))
	}
}

func (eppim *HttpEndpointPoolInternalMetrics) updateEPMetricsCache(url string) {
	instance, hostname := eppim.internalMetrics.Instance, eppim.internalMetrics.Hostname

	indexMetricMap := make(httpEndpointPoolStatsIndexMetricMap)
	for index, name := range httpEndpointStatsDeltaMetricsNameMap {
		metric := fmt.Sprintf(
			`%s{%s="%s",%s="%s",%s="%s"} `, // N.B. include the whitespace separating the metric from value
			name,
			INSTANCE_LABEL_NAME, instance,
			HOSTNAME_LABEL_NAME, hostname,
			HTTP_ENDPOINT_URL_LABEL_NAME, url,
		)
		indexMetricMap[index] = []byte(metric)
	}
	eppim.endpointDeltaMetricsCache[url] = indexMetricMap
	indexMetricMap = make(httpEndpointPoolStatsIndexMetricMap)
}

func (eppim *HttpEndpointPoolInternalMetrics) generateMetrics(buf *bytes.Buffer, tsSuffix []byte) (int, int, *bytes.Buffer) {
	// Ensure that metrics cache is up-to-date:
	indexMetricMap := eppim.poolDeltaMetricsCache
	if indexMetricMap == nil {
		// N.B. This will update all the other metrics caches!
		eppim.updatePoolMetricsCache()
		indexMetricMap = eppim.poolDeltaMetricsCache
	}

	mq := eppim.internalMetrics.MetricsQueue
	metricsCount, partialByteCount, bufMaxSize := 0, 0, mq.GetTargetSize()

	currStats, prevStats := eppim.stats[eppim.currIndex], eppim.stats[1-eppim.currIndex]
	currPoolStats := currStats.PoolStats

	var prevPoolStats HttpPoolStats
	if prevStats != nil {
		prevPoolStats = prevStats.PoolStats
	}

	if buf == nil {
		buf = mq.GetBuf()
	}
	for index, metric := range indexMetricMap {
		val := currPoolStats[index]
		if prevPoolStats != nil {
			val -= prevPoolStats[index]
		}
		buf.Write(metric)
		buf.WriteString(strconv.FormatUint(val, 10))
		buf.Write(tsSuffix)
		metricsCount++
	}
	if n := buf.Len(); bufMaxSize > 0 && n >= bufMaxSize {
		partialByteCount += n
		mq.QueueBuf(buf)
		buf = nil
	}

	var prevEPStats HttpEndpointStats
	for url, currEPStats := range currStats.EndpointStats {
		if buf == nil {
			buf = mq.GetBuf()
		}

		if prevStats != nil {
			prevEPStats = prevStats.EndpointStats[url]
		} else {
			prevEPStats = nil
		}
		indexMetricMap := eppim.endpointDeltaMetricsCache[url]
		if indexMetricMap == nil {
			// N.B. This will update all the other metrics cache for this URL!
			eppim.updateEPMetricsCache(url)
			indexMetricMap = eppim.endpointDeltaMetricsCache[url]
		}
		for index, metric := range indexMetricMap {
			val := currEPStats[index]
			if prevEPStats != nil {
				val -= prevEPStats[index]
			}
			buf.Write(metric)
			buf.WriteString(strconv.FormatUint(val, 10))
			buf.Write(tsSuffix)
			metricsCount++
		}

		if n := buf.Len(); bufMaxSize > 0 && n >= bufMaxSize {
			partialByteCount += n
			mq.QueueBuf(buf)
			buf = nil
		}
	}

	// Flip the stats storage:
	eppim.currIndex = 1 - eppim.currIndex

	return metricsCount, partialByteCount, buf
}
