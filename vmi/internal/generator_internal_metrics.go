package vmi_internal

import (
	"bytes"
	"fmt"
	"sync"
)

// Each metrics generator will maintain the following common stats:
const (
	// Indexes into the per generator []int stats:
	METRICS_GENERATOR_INVOCATION_COUNT = iota
	METRICS_GENERATOR_METRICS_COUNT
	METRICS_GENERATOR_BYTE_COUNT
	// Must be last:
	METRICS_GENERATOR_NUM_STATS
)

type MetricsGeneratorStats map[string][]uint64

type MetricsGeneratorStatsContainer struct {
	// Stats proper:
	stats MetricsGeneratorStats
	// Lock:
	mu *sync.Mutex
}

var MetricsGeneratorStatsMetricsNameMap = map[int]string{
	METRICS_GENERATOR_INVOCATION_COUNT: METRICS_GENERATOR_INVOCATION_DELTA_METRIC,
	METRICS_GENERATOR_METRICS_COUNT:    METRICS_GENERATOR_METRICS_DELTA_METRIC,
	METRICS_GENERATOR_BYTE_COUNT:       METRICS_GENERATOR_BYTE_DELTA_METRIC,
}

func NewMetricsGeneratorStatsContainer() *MetricsGeneratorStatsContainer {
	return &MetricsGeneratorStatsContainer{
		stats: make(MetricsGeneratorStats),
		mu:    &sync.Mutex{},
	}
}

func (mgsc *MetricsGeneratorStatsContainer) Update(genId string, metricCount, byteCount uint64) {
	mgsc.mu.Lock()
	defer mgsc.mu.Unlock()

	genStats := mgsc.stats[genId]
	if genStats == nil {
		genStats = make([]uint64, METRICS_GENERATOR_NUM_STATS)
		mgsc.stats[genId] = genStats
	}
	genStats[METRICS_GENERATOR_INVOCATION_COUNT]++
	genStats[METRICS_GENERATOR_METRICS_COUNT] += metricCount
	genStats[METRICS_GENERATOR_BYTE_COUNT] += byteCount
}

func (mgsc *MetricsGeneratorStatsContainer) Clear() {
	mgsc.mu.Lock()
	defer mgsc.mu.Unlock()
	clear(mgsc.stats)
}

type GeneratorInternalMetrics struct {
	// Internal metrics, for common values:
	internalMetrics *InternalMetrics
	// Dual storage for snapping generator stats, used as current, previous,
	// toggled after every metrics generation:
	generatorStats [2]MetricsGeneratorStats
	// The current index:
	currIndex int
	// Cache for the metrics, `name{label="val",...}`, indexed by the generator
	// Id and stats index:
	metricsCache map[string][][]byte
}

func NewGeneratorInternalMetrics(internalMetrics *InternalMetrics) *GeneratorInternalMetrics {
	return &GeneratorInternalMetrics{
		internalMetrics: internalMetrics,
		metricsCache:    make(map[string][][]byte),
	}
}

func (gim *GeneratorInternalMetrics) SnapStats() {
	MetricsGenStats.mu.Lock()
	defer MetricsGenStats.mu.Unlock()

	toStats := gim.generatorStats[gim.currIndex]
	if toStats == nil {
		toStats = make(MetricsGeneratorStats)
		gim.generatorStats[gim.currIndex] = toStats
	}

	for genId, genStats := range MetricsGenStats.stats {
		toGenStats := toStats[genId]
		if toGenStats == nil {
			toGenStats = make([]uint64, METRICS_GENERATOR_NUM_STATS)
			toStats[genId] = toGenStats
		}
		copy(toGenStats, genStats)
	}
}

func (gim *GeneratorInternalMetrics) updateMetricsCache(genId string) {
	instance, hostname := gim.internalMetrics.Instance, gim.internalMetrics.Hostname

	indexMetricMap := make([][]byte, METRICS_GENERATOR_NUM_STATS)
	for index, name := range MetricsGeneratorStatsMetricsNameMap {
		indexMetricMap[index] = []byte(fmt.Sprintf(
			`%s{%s="%s",%s="%s",%s="%s"} `, // N.B. include the whitespace separating the metric from value
			name,
			INSTANCE_LABEL_NAME, instance,
			HOSTNAME_LABEL_NAME, hostname,
			METRICS_GENERATOR_ID_LABEL_NAME, genId,
		))
	}
	gim.metricsCache[genId] = indexMetricMap
}

func (gim *GeneratorInternalMetrics) generateMetrics(buf *bytes.Buffer, tsSuffix []byte) (int, int, *bytes.Buffer) {
	crtStats, prevStats := gim.generatorStats[gim.currIndex], gim.generatorStats[1-gim.currIndex]

	mq := gim.internalMetrics.MetricsQueue
	metricsCount, partialByteCount, bufMaxSize := 0, 0, mq.GetTargetSize()

	var prevGenStats []uint64
	for genId, crtGenStats := range crtStats {
		if buf == nil {
			buf = mq.GetBuf()
		}

		metrics := gim.metricsCache[genId]
		if metrics == nil {
			gim.updateMetricsCache(genId)
			metrics = gim.metricsCache[genId]
		}
		if prevStats != nil {
			prevGenStats = prevStats[genId]
		} else {
			prevGenStats = nil
		}
		for index, metric := range metrics {
			val := crtGenStats[index]
			if prevGenStats != nil {
				val -= prevGenStats[index]
			}
			if buf == nil {
				buf = mq.GetBuf()
			}
			buf.Write(metric)
			fmt.Fprintf(buf, "%d", val)
			buf.Write(tsSuffix)
			metricsCount++
		}

		if n := buf.Len(); bufMaxSize > 0 && n >= bufMaxSize {
			partialByteCount += n
			mq.QueueBuf(buf)
			buf = nil
		}

	}

	gim.currIndex = 1 - gim.currIndex

	return metricsCount, partialByteCount, buf
}
