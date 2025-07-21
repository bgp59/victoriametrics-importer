// vmi process proper metrics

package vmi_internal

import (
	"bytes"
	"fmt"
	"strconv"
	"time"
)

// Generate basic process metrics such as memory and CPU utilization for for
// this process:

type ProcessInternalMetrics struct {
	// Internal metrics, for common values:
	internalMetrics *InternalMetrics
	// Dual storage for snapping the stats, used as current, previous, toggled
	// after every metrics generation:
	cpuTime [2]float64
	// When the stats were collected:
	statsTs [2]time.Time
	// The current index:
	currIndex int
	// metrics, `name{label="val",...}`:
	pcpuMetric []byte
}

func NewProcessInternalMetrics(internalMetrics *InternalMetrics) *ProcessInternalMetrics {
	return &ProcessInternalMetrics{
		internalMetrics: internalMetrics,
		cpuTime:         [2]float64{-1, -1},
		statsTs:         [2]time.Time{},
		currIndex:       0,
	}
}

func (pim *ProcessInternalMetrics) SnapStats() {
	var err error
	pim.cpuTime[pim.currIndex], err = GetMyCpuTime()
	if err != nil {
		internalMetricsLog.Warnf("GetMyCpuTime(): %v", err)
		pim.cpuTime[pim.currIndex] = -1
	}
	pim.statsTs[pim.currIndex] = time.Now()
}

func (pim *ProcessInternalMetrics) updateMetricsCache() {
	instance, hostname := pim.internalMetrics.Instance, pim.internalMetrics.Hostname
	pim.pcpuMetric = []byte(fmt.Sprintf(
		`%s{%s="%s",%s="%s"} `, // N.B. include the whitespace separating the metric from value
		VMI_PROC_PCPU_METRIC,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
	))

}

func (pim *ProcessInternalMetrics) generateMetrics(buf *bytes.Buffer, tsSuffix []byte) (int, int, *bytes.Buffer) {
	const totalMetricsCount = 1

	// Update the metrics cache:
	if pim.pcpuMetric == nil {
		pim.updateMetricsCache()
	}

	mq := pim.internalMetrics.MetricsQueue
	metricsCount, partialByteCount, bufMaxSize := 0, 0, mq.GetTargetSize()

	if pim.cpuTime[1-pim.currIndex] >= 0 {
		if buf == nil {
			buf = mq.GetBuf()
		}
		// We have a previous CPU time, so we can calculate the delta:
		dTime := pim.statsTs[pim.currIndex].Sub(pim.statsTs[1-pim.currIndex]).Seconds()
		dTimeCpu := pim.cpuTime[pim.currIndex] - pim.cpuTime[1-pim.currIndex]
		buf.Write(pim.pcpuMetric)
		buf.WriteString(strconv.FormatFloat(dTimeCpu/dTime*100, 'f', 1, 64))
		buf.Write(tsSuffix)
		metricsCount++

		if n := buf.Len(); bufMaxSize > 0 && n >= bufMaxSize {
			partialByteCount += n
			mq.QueueBuf(buf)
			buf = nil
		}
	}

	// Flip the stats storage:
	pim.currIndex = 1 - pim.currIndex

	return metricsCount, partialByteCount, buf
}
