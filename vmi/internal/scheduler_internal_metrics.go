// Scheduler metrics:

package vmi_internal

import (
	"bytes"
	"fmt"
	"strconv"
)

// Map stats index to the metric name:
type taskStatsIndexMetricMap map[int][]byte

type SchedulerInternalMetrics struct {
	// Internal metrics, for common values:
	internalMetrics *InternalMetrics
	// Dual storage for snapping the stats, used as current, previous, toggled
	// after every metrics generation:
	stats [2]SchedulerStats
	// The current index:
	currIndex int
	// Cache the full metrics for each taskId and stats index:
	uint64DeltaMetricsCache map[string]taskStatsIndexMetricMap
}

// The following stats will be used to generate deltas:
var taskStatsUint64DeltaMetricsNameMap = map[int]string{
	TASK_STATS_SCHEDULED_COUNT:    TASK_STATS_SCHEDULED_DELTA_METRIC,
	TASK_STATS_DELAYED_COUNT:      TASK_STATS_DELAYED_DELTA_METRIC,
	TASK_STATS_OVERRUN_COUNT:      TASK_STATS_OVERRUN_DELTA_METRIC,
	TASK_STATS_EXECUTED_COUNT:     TASK_STATS_EXECUTED_DELTA_METRIC,
	TASK_STATS_NEXT_TS_HACK_COUNT: TASK_STATS_NEXT_TS_HACK_DELTA_METRIC,
	TASK_STATS_TOTAL_RUNTIME:      TASK_STATS_AVG_RUNTIME_METRIC,
}

func NewSchedulerInternalMetrics(internalMetrics *InternalMetrics) *SchedulerInternalMetrics {
	return &SchedulerInternalMetrics{
		internalMetrics:         internalMetrics,
		uint64DeltaMetricsCache: make(map[string]taskStatsIndexMetricMap),
	}
}

func (sim *SchedulerInternalMetrics) updateMetricsCache(taskId string) {
	instance, hostname := sim.internalMetrics.Instance, sim.internalMetrics.Hostname

	indexMetricMap := make(taskStatsIndexMetricMap)
	for index, name := range taskStatsUint64DeltaMetricsNameMap {
		metric := fmt.Sprintf(
			`%s{%s="%s",%s="%s",%s="%s"} `, // N.B. include the whitespace separating the metric from value
			name,
			INSTANCE_LABEL_NAME, instance,
			HOSTNAME_LABEL_NAME, hostname,
			TASK_STATS_TASK_ID_LABEL_NAME, taskId,
		)
		indexMetricMap[index] = []byte(metric)
	}
	sim.uint64DeltaMetricsCache[taskId] = indexMetricMap
}

func (sim *SchedulerInternalMetrics) generateMetrics(buf *bytes.Buffer, tsSuffix []byte) (int, int, *bytes.Buffer) {
	mq := sim.internalMetrics.MetricsQueue
	metricsCount, partialByteCount, bufMaxSize := 0, 0, mq.GetTargetSize()

	currStats, prevStats := sim.stats[sim.currIndex], sim.stats[1-sim.currIndex]
	var prevTaskStats *TaskStats
	for taskId, currTaskStats := range currStats {
		if buf == nil {
			buf = mq.GetBuf()
		}

		if prevStats != nil {
			prevTaskStats = prevStats[taskId]
		} else {
			prevTaskStats = nil
		}
		uint64IndexMetricMap := sim.uint64DeltaMetricsCache[taskId]
		if uint64IndexMetricMap == nil {
			sim.updateMetricsCache(taskId)
			uint64IndexMetricMap = sim.uint64DeltaMetricsCache[taskId]
		}
		executedCount, runtime, avgRuntimeMetric := uint64(0), uint64(0), []byte(nil)
		for index, metric := range uint64IndexMetricMap {
			val := currTaskStats.Uint64Stats[index]
			if prevTaskStats != nil {
				val -= prevTaskStats.Uint64Stats[index]
			}
			if index == TASK_STATS_TOTAL_RUNTIME {
				runtime, avgRuntimeMetric = val, metric
				// Postpone writing the avg runtime metric until we know how
				// many tasks were executed.
				continue
			}
			if index == TASK_STATS_EXECUTED_COUNT {
				executedCount = val
			}

			buf.Write(metric)
			buf.WriteString(strconv.FormatUint(val, 10))
			buf.Write(tsSuffix)
			metricsCount++
		}
		if executedCount > 0 {
			buf.Write(avgRuntimeMetric)
			buf.WriteString(strconv.FormatFloat(
				// N.B. The runtime is in microseconds, so we need to convert it to seconds.
				float64(runtime)/1_000_000.0/float64(executedCount),
				'f', TASK_STATS_AVG_RUNTIME_METRIC_PRECISION, 64,
			))
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
	sim.currIndex = 1 - sim.currIndex

	return metricsCount, partialByteCount, buf
}
