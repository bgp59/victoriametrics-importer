// Counter metrics generator

package refvmi

import (
	"fmt"
	"strconv"
	"time"

	"github.com/bgp59/victoriametrics-importer/refvmi/parser"

	"github.com/bgp59/victoriametrics-importer/vmi"
)

const (
	COUNTER_METRICS_CONFIG_INTERVAL_DEFAULT            = 2 * time.Second
	COUNTER_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT = 10

	// This Metrics Generator ID:
	COUNTER_METRICS_ID = "counter"
)

// The logger for this generator:
var counterMetricsLog = vmi.NewCompLogger(COUNTER_METRICS_ID)

// The metrics generator context:
type CounterMetrics struct {
	vmi.GeneratorBase

	// The underlying parser:
	parser *parser.RandomCounterParser

	// Dual buffer for current/previous value, needed for the delta approach:
	valCache [2]uint32

	// Whether the previous delta was 0:
	zeroDelta bool

	// The index in the cache for the current value(s). A special value of -1
	// will indicate that the structure was not initialized. JIT initialization
	// is needed for testing, which relies on changes *after* the structure was
	// built (e.g. set .instance, .hostname).
	currentIndex int

	// Cache for various metrics:
	//  - metrics proper:
	counterDeltaMetric []byte
	counterRateMetric  []byte
}

// The configuration for this generator. It should be loadable from a YAML file
// and it may include a parser related section:
type CounterMetricsConfig struct {
	// How often to generate the metrics in time.ParseDuration() format:
	Interval time.Duration `yaml:"interval"`

	// Normally metrics are generated only if there is a change in value from
	// the previous scan. However every N cycles the full set is generated. Use
	// 0 to generate full metrics every cycle.
	FullMetricsFactor int `yaml:"full_metrics_factor"`

	// Parser configuration:
	ParserConfig *parser.RandomCounterParserConfig `yaml:"parser_config"`
}

func DefaultCounterMetricsConfig() *CounterMetricsConfig {
	return &CounterMetricsConfig{
		Interval:          COUNTER_METRICS_CONFIG_INTERVAL_DEFAULT,
		FullMetricsFactor: COUNTER_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT,
		ParserConfig:      parser.DefaultRandomCounterParserConfig(),
	}
}

func NewCounterMetrics(cfg *CounterMetricsConfig) *CounterMetrics {
	if cfg == nil {
		cfg = DefaultCounterMetricsConfig()
	}
	return &CounterMetrics{
		GeneratorBase: vmi.GeneratorBase{
			Id:                COUNTER_METRICS_ID,
			Interval:          cfg.Interval,
			CycleNum:          vmi.GetInitialCycleNum(cfg.FullMetricsFactor),
			FullMetricsFactor: cfg.FullMetricsFactor,
		},
		parser:       parser.NewRandomCounterParser(cfg.ParserConfig),
		currentIndex: -1,
	}
}

// Update metrics cache, normally this is needed only at the 1st time generation:
func (m *CounterMetrics) initialize() {
	m.GenBaseInit()

	instance, hostname := m.Instance, m.Hostname

	m.counterDeltaMetric = []byte(fmt.Sprintf(
		`%s{%s="%s",%s="%s"} `, // N.B. space before value is included
		COUNTER_DELTA_METRIC,
		vmi.INSTANCE_LABEL_NAME, instance,
		vmi.HOSTNAME_LABEL_NAME, hostname,
	))

	m.counterRateMetric = []byte(fmt.Sprintf(
		`%s{%s="%s",%s="%s"} `, // N.B. space before value is included
		COUNTER_RATE_METRIC,
		vmi.INSTANCE_LABEL_NAME, instance,
		vmi.HOSTNAME_LABEL_NAME, hostname,
	))

	m.Initialized = true
}

// The actual metrics generation, it will be registered as the wrapping task's activity:
func (m *CounterMetrics) TaskActivity() bool {
	if !m.Initialized {
		m.initialize()
	}

	// Parse new info and handle errors:
	if !m.TestMode {
		err := m.parser.Parse()
		if err != nil {
			counterMetricsLog.Errorf("Parse(): %v", err)
			return false // This will disable future invocations.
		}
	}
	// All new data retrieved:
	ts := m.TimeNowFunc()

	// Update the value cache:
	currIndex := m.currentIndex
	hasPrev := currIndex >= 0
	if !hasPrev {
		currIndex = 0
	}
	currVal := m.parser.Val
	m.valCache[currIndex] = currVal

	metricsQueue := m.MetricsQueue
	buf := metricsQueue.GetBuf()
	metricsCount, lastTs := m.GenBaseMetricsStart(buf, ts)

	// All metrics depend upon having a previous value:
	if hasPrev {
		tsSuffix := m.TsSuffixBuf.Bytes()

		delta := currVal - m.valCache[1-currIndex]
		deltaSec := ts.Sub(lastTs).Seconds()
		zeroDelta := delta == 0
		if !zeroDelta || m.CycleNum == 0 || !m.zeroDelta {
			buf.Write(m.counterDeltaMetric)
			buf.WriteString(strconv.FormatUint(uint64(delta), 10))
			buf.Write(tsSuffix)
			metricsCount += 1

			if deltaSec > 0 {
				buf.Write(m.counterRateMetric)
				buf.WriteString(strconv.FormatFloat(float64(delta)/deltaSec, 'f', 3, 64))
				buf.Write(tsSuffix)
				metricsCount += 1
			}
		}
		m.zeroDelta = zeroDelta
	}

	vmi.UpdateMetricsGeneratorStats(m.Id, metricsCount, buf.Len())

	// Queue the buffer for publish:
	metricsQueue.QueueBuf(buf)

	// Toggle dual cache index:
	m.currentIndex = 1 - currIndex

	// Update cycle#:
	if m.CycleNum += 1; m.CycleNum >= m.FullMetricsFactor {
		m.CycleNum = 0
	}

	// All OK:
	return true
}

// The task builder function will generate the list of tasks wrapping the
// metrics generators. It will be registered with the VMI framework and it will
// be invoked by vmi.Run().
func CounterMetricsTaskBuilder(cfg any) ([]vmi.MetricsGeneratorTask, error) {
	if refvmiConfig, ok := cfg.(*RefvmiConfig); ok {
		if refvmiConfig == nil {
			refvmiConfig = DefaultRefvmiConfig()
		}
		counterMetricsConfig := refvmiConfig.CounterMetricsConfig
		if counterMetricsConfig == nil {
			counterMetricsConfig = DefaultCounterMetricsConfig()
		}
		tasks := make([]vmi.MetricsGeneratorTask, 0)
		if counterMetricsConfig.Interval <= 0 {
			counterMetricsLog.Infof("interval=%s, metrics disabled", counterMetricsConfig.Interval)
		} else {
			counterMetricsLog.Infof(
				"interval=%s, full_metrics_factor=%d, initial: %d, increment: %d .. %d, repeat: 1 .. %d, seed: %d",
				counterMetricsConfig.Interval, counterMetricsConfig.FullMetricsFactor,
				counterMetricsConfig.ParserConfig.Init, counterMetricsConfig.ParserConfig.MinInc, counterMetricsConfig.ParserConfig.MaxInc,
				counterMetricsConfig.ParserConfig.MaxRepeat, counterMetricsConfig.ParserConfig.Seed,
			)
			tasks = append(tasks, NewCounterMetrics(counterMetricsConfig))
		}
		return tasks, nil
	} else {
		return nil, fmt.Errorf("cfg: CounterMetricsTaskBuilder passed wrong config type %T", cfg)
	}
}

func init() {
	vmi.RegisterTaskBuilder(CounterMetricsTaskBuilder)
}
