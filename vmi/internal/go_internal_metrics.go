// Internal metrics for the vmi Go process

package vmi_internal

import (
	"bytes"
	"fmt"
	"runtime"
	"strconv"
)

const (
	// The order in the metrics cache:
	GO_NUM_GOROUTINE_METRIC_INDEX = iota
	GO_MEM_SYS_BYTES_METRIC_INDEX
	GO_MEM_HEAP_BYTES_METRIC_INDEX
	GO_MEM_HEAP_SYS_BYTES_METRIC_INDEX
	GO_MEM_MALLOCS_DELTA_METRIC_INDEX
	GO_MEM_FREE_DELTA_METRIC_INDEX
	GO_MEM_IN_USE_OBJECT_COUNT_METRIC_INDEX
	GO_MEM_NUM_GC_DELTA_METRIC_INDEX

	// Must be last:
	GO_INTERNAL_METRICS_NUM
)

var goInternalMetricsNameMap = map[int]string{
	GO_NUM_GOROUTINE_METRIC_INDEX:           GO_NUM_GOROUTINE_METRIC,
	GO_MEM_SYS_BYTES_METRIC_INDEX:           GO_MEM_SYS_BYTES_METRIC,
	GO_MEM_HEAP_BYTES_METRIC_INDEX:          GO_MEM_HEAP_BYTES_METRIC,
	GO_MEM_HEAP_SYS_BYTES_METRIC_INDEX:      GO_MEM_HEAP_SYS_BYTES_METRIC,
	GO_MEM_MALLOCS_DELTA_METRIC_INDEX:       GO_MEM_MALLOCS_DELTA_METRIC,
	GO_MEM_FREE_DELTA_METRIC_INDEX:          GO_MEM_FREE_DELTA_METRIC,
	GO_MEM_IN_USE_OBJECT_COUNT_METRIC_INDEX: GO_MEM_IN_USE_OBJECT_COUNT_METRIC,
	GO_MEM_NUM_GC_DELTA_METRIC_INDEX:        GO_MEM_NUM_GC_DELTA_METRIC,
}

type GoInternalMetrics struct {
	// Internal metrics, for common values:
	internalMetrics *InternalMetrics
	// Snap data:
	goVersion    string
	numGoRoutine int
	// Dual storage for snapping Go runtime data, used as current, previous,
	// toggled after every metrics generation:
	memStats [2]*runtime.MemStats
	// The current index:
	currIndex int
	// Cache for Go metrics, `name{label="val",...}`, indexed by the
	// stats index:
	metricsCache map[int][]byte
}

func NewGoInternalMetrics(internalMetrics *InternalMetrics) *GoInternalMetrics {
	gim := &GoInternalMetrics{
		goVersion:       runtime.Version(),
		internalMetrics: internalMetrics,
	}
	gim.memStats[0] = &runtime.MemStats{}
	gim.memStats[1] = &runtime.MemStats{}
	return gim
}

func (gim *GoInternalMetrics) SnapStats() {
	if gim.memStats[gim.currIndex] == nil {
		gim.memStats[gim.currIndex] = &runtime.MemStats{}
	}
	runtime.ReadMemStats(gim.memStats[gim.currIndex])
	gim.numGoRoutine = runtime.NumGoroutine()
}

func (gim *GoInternalMetrics) updateMetricsCache() {
	instance, hostname := gim.internalMetrics.Instance, gim.internalMetrics.Hostname

	gim.metricsCache = make(map[int][]byte)

	for index, name := range goInternalMetricsNameMap {
		gim.metricsCache[index] = []byte(fmt.Sprintf(
			`%s{%s="%s",%s="%s"} `, // N.B. include the whitespace separating the metric from value
			name,
			INSTANCE_LABEL_NAME, instance,
			HOSTNAME_LABEL_NAME, hostname,
		))
	}
}

func (gim *GoInternalMetrics) generateMetrics(buf *bytes.Buffer, tsSuffix []byte) (int, int, *bytes.Buffer) {
	metricsCache := gim.metricsCache
	if metricsCache == nil {
		gim.updateMetricsCache()
		metricsCache = gim.metricsCache
	}

	currMemStats, prevMemStats := gim.memStats[gim.currIndex], gim.memStats[1-gim.currIndex]

	mq := gim.internalMetrics.MetricsQueue
	metricsCount, partialByteCount, bufMaxSize := 0, 0, mq.GetTargetSize()

	if buf == nil {
		buf = mq.GetBuf()
	}

	buf.Write(metricsCache[GO_NUM_GOROUTINE_METRIC_INDEX])
	buf.WriteString(strconv.Itoa(gim.numGoRoutine))
	buf.Write(tsSuffix)
	metricsCount++

	buf.Write(metricsCache[GO_MEM_SYS_BYTES_METRIC_INDEX])
	buf.WriteString(strconv.FormatUint(currMemStats.Sys, 10))
	buf.Write(tsSuffix)
	metricsCount++

	buf.Write(metricsCache[GO_MEM_HEAP_BYTES_METRIC_INDEX])
	buf.WriteString(strconv.FormatUint(currMemStats.HeapAlloc, 10))
	buf.Write(tsSuffix)
	metricsCount++

	buf.Write(metricsCache[GO_MEM_HEAP_SYS_BYTES_METRIC_INDEX])
	buf.WriteString(strconv.FormatUint(currMemStats.HeapSys, 10))
	buf.Write(tsSuffix)
	metricsCount++

	buf.Write(metricsCache[GO_MEM_IN_USE_OBJECT_COUNT_METRIC_INDEX])
	buf.WriteString(strconv.FormatUint(currMemStats.HeapObjects, 10))
	buf.Write(tsSuffix)
	metricsCount++

	// Note that deltas below work even at the 1st pass because prevMemStats has
	// been primed w/ 0 when GoInternalMetrics was created:
	buf.Write(metricsCache[GO_MEM_MALLOCS_DELTA_METRIC_INDEX])
	buf.WriteString(strconv.FormatUint(currMemStats.Mallocs-prevMemStats.Mallocs, 10))
	buf.Write(tsSuffix)
	metricsCount++

	buf.Write(metricsCache[GO_MEM_FREE_DELTA_METRIC_INDEX])
	buf.WriteString(strconv.FormatUint(currMemStats.Frees-prevMemStats.Frees, 10))
	buf.Write(tsSuffix)
	metricsCount++

	buf.Write(metricsCache[GO_MEM_NUM_GC_DELTA_METRIC_INDEX])
	buf.WriteString(strconv.FormatUint(uint64(currMemStats.NumGC-prevMemStats.NumGC), 10))
	buf.Write(tsSuffix)
	metricsCount++

	if n := buf.Len(); bufMaxSize > 0 && n >= bufMaxSize {
		partialByteCount += n
		mq.QueueBuf(buf)
		buf = nil
	}

	// Flip the stats storage:
	gim.currIndex = 1 - gim.currIndex

	return metricsCount, partialByteCount, buf
}
