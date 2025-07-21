// Base struct to embed in generators.

package vmi_internal

import (
	"bytes"
	"fmt"
	"strconv"
	"time"
)

const (
	GENERATOR_RUNTIME_UNAVAILABLE = -1.
)

type GeneratorBase struct {
	// Unique generator ID:
	Id string
	// Scheduling interval:
	Interval time.Duration
	// Full metrics factor (see "Partial V. Full Metrics" in README.md):
	FullMetricsFactor int
	// The current cycle# used in conjunction with the FullMetricsFactor:
	CycleNum int
	// The timestamp of the last metrics generation:
	LastTs time.Time
	// Cache for generator metrics:
	DtimeMetric []byte
	// Cache for timestamp suffix, common to all/a group of metrics:
	TsSuffixBuf *bytes.Buffer
	// Whether the structure was initialized (caches, etc) or not:
	Initialized bool
	// The following fields, if left to their default values (type's nil) will
	// be set during initialization with the usual values. They may be
	// pre-populated during tests after the generator was created and before
	// initialization.
	Instance     string
	Hostname     string
	TimeNowFunc  func() time.Time
	MetricsQueue BufferQueue
	TestMode     bool
}

func (gb *GeneratorBase) GenBaseInit() {
	instance := gb.Instance
	if instance == "" {
		instance = Instance
		gb.Instance = instance
	}

	hostname := gb.Hostname
	if hostname == "" {
		hostname = Hostname
		gb.Hostname = hostname
	}

	if gb.TimeNowFunc == nil {
		gb.TimeNowFunc = time.Now
	}

	if gb.MetricsQueue == nil {
		gb.MetricsQueue = MetricsQueue
	}

	gb.DtimeMetric = []byte(fmt.Sprintf(
		`%s{%s="%s",%s="%s",%s="%s"} `, // N.B. space before value is included
		METRICS_GENERATOR_DTIME_METRIC,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
		METRICS_GENERATOR_ID_LABEL_NAME, gb.Id,
	))

	if gb.TsSuffixBuf == nil {
		gb.TsSuffixBuf = &bytes.Buffer{}
	}
}

// Start metrics generation; this should the 1st call in a metrics generation
// since it establishes the timestamp suffix. Call with the buffer for the
// metrics and the timestamp of the collection. Return the metric count and the
// last timestamp of the previous run. If the buffer is nil, then no metrics are
// generated, but the timestamp suffix is still updated.
func (gb *GeneratorBase) GenBaseMetricsStart(buf *bytes.Buffer, ts time.Time) (int, time.Time) {
	metricsCount := 0
	// If there is content in TsSuffixBuf then this is an indication of a
	// subsequent run. Publish runtime and interval.
	tsSuffixBuf := gb.TsSuffixBuf
	validPrev := tsSuffixBuf.Len() > 0
	tsSuffixBuf.Reset()
	// N.B. The space after the value and the ending `\n' are included.
	fmt.Fprintf(tsSuffixBuf, " %d\n", ts.UnixMilli())
	if validPrev && buf != nil {
		// Publish the actual interval since the prev run:
		buf.Write(gb.DtimeMetric)
		buf.WriteString(strconv.FormatFloat(ts.Sub(gb.LastTs).Seconds(), 'f', METRICS_GENERATOR_DTIME_METRIC_PRECISION, 64))
		buf.Write(tsSuffixBuf.Bytes())
		metricsCount++
	}
	lastTs := gb.LastTs
	gb.LastTs = ts
	return metricsCount, lastTs
}

// Satisfy GeneratorTask I/F:
func (gb *GeneratorBase) GetId() string              { return gb.Id }
func (gb *GeneratorBase) GetInterval() time.Duration { return gb.Interval }
