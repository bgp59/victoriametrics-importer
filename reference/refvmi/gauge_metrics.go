// Gauge metrics generator example

package refvmi

import (
	"bytes"
	"fmt"
	"time"

	"github.com/bgp59/victoriametrics-importer/refvmi/parser"

	"github.com/bgp59/victoriametrics-importer/vmi"
)

const (
	GAUGE_METRICS_CONFIG_INTERVAL_DEFAULT            = 2 * time.Second
	GAUGE_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT = 10

	// This Metrics Generator ID:
	GAUGE_METRICS_ID = "gauge"
)

// The logger for this generator:
var gaugeMetricsLog = vmi.NewCompLogger(GAUGE_METRICS_ID)

// The metrics generator context:
type GaugeMetrics struct {
	vmi.GeneratorBase

	// The underlying parser:
	parser *parser.RandomGaugeParser

	// Dual buffer for current/previous value, needed for the delta approach:
	valCache [2][]byte

	// The index in the cache for the current value(s). A special value of -1
	// will indicate that the structure was not initialized. JIT initialization
	// is needed for testing, which relies on changes *after* the structure was
	// built (e.g. set .instance, .hostname).
	currentIndex int

	// Cache for various metrics:
	//  - metrics proper:
	gaugeMetric []byte
}

// The configuration for this generator. It should be loadable from a YAML file
// and it may include a parser related section:
type GaugeMetricsConfig struct {
	// How often to generate the metrics in time.ParseDuration() format:
	Interval time.Duration `yaml:"interval"`

	// Normally metrics are generated only if there is a change in value from
	// the previous scan. However every N cycles the full set is generated. Use
	// 0 to generate full metrics every cycle.
	FullMetricsFactor int `yaml:"full_metrics_factor"`

	// Parser configuration:
	ParserConfig *parser.RandomGaugeParserConfig `yaml:"parser_config"`
}

func DefaultGaugeMetricsConfig() *GaugeMetricsConfig {
	return &GaugeMetricsConfig{
		Interval:          GAUGE_METRICS_CONFIG_INTERVAL_DEFAULT,
		FullMetricsFactor: GAUGE_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT,
		ParserConfig:      parser.DefaultRandomGaugeParserConfig(),
	}
}

func NewGaugeMetrics(cfg *GaugeMetricsConfig) *GaugeMetrics {
	if cfg == nil {
		cfg = DefaultGaugeMetricsConfig()
	}
	return &GaugeMetrics{
		GeneratorBase: vmi.GeneratorBase{
			Id:                GAUGE_METRICS_ID,
			Interval:          cfg.Interval,
			CycleNum:          vmi.GetInitialCycleNum(cfg.FullMetricsFactor),
			FullMetricsFactor: cfg.FullMetricsFactor,
		},
		parser:       parser.NewRandomGaugeParser(cfg.ParserConfig),
		currentIndex: -1,
	}
}

// Update metrics cache, normally this is needed only at the 1st time generation:
func (m *GaugeMetrics) initialize() {
	m.GenBaseInit()

	instance, hostname := m.Instance, m.Hostname

	m.gaugeMetric = []byte(fmt.Sprintf(
		`%s{%s="%s",%s="%s"} `, // N.B. space before value is included
		GAUGE_METRIC,
		vmi.INSTANCE_LABEL_NAME, instance,
		vmi.HOSTNAME_LABEL_NAME, hostname,
	))

	m.Initialized = true
}

// The actual metrics generation, it will be registered as the wrapping task's activity:
func (m *GaugeMetrics) TaskActivity() bool {
	if !m.Initialized {
		m.initialize()
	}

	// Parse new info and handle errors:
	if !m.TestMode {
		err := m.parser.Parse()
		if err != nil {
			gaugeMetricsLog.Errorf("Parse(): %v", err)
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
	currVal := m.parser.ValBytes
	if len(m.valCache[currIndex]) < len(currVal) {
		m.valCache[currIndex] = make([]byte, len(currVal))
	}
	copy(m.valCache[currIndex], currVal)

	metricsQueue := m.MetricsQueue
	buf := metricsQueue.GetBuf()
	metricsCount, _ := m.GenBaseMetricsStart(buf, ts)
	tsSuffix := m.TsSuffixBuf.Bytes()

	prevVal := m.valCache[1-currIndex]
	if !hasPrev || m.CycleNum == 0 || !bytes.Equal(currVal, prevVal) {
		buf.Write(m.gaugeMetric)
		buf.Write(currVal)
		buf.Write(tsSuffix)
		metricsCount += 1
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
func GaugeMetricsTaskBuilder(cfg any) ([]vmi.MetricsGeneratorTask, error) {
	if refvmiConfig, ok := cfg.(*RefvmiConfig); ok {
		if refvmiConfig == nil {
			refvmiConfig = DefaultRefvmiConfig()
		}
		gaugeMetricsConfig := refvmiConfig.GaugeMetricsConfig
		if gaugeMetricsConfig == nil {
			gaugeMetricsConfig = DefaultGaugeMetricsConfig()
		}
		tasks := make([]vmi.MetricsGeneratorTask, 0)
		if gaugeMetricsConfig.Interval <= 0 {
			gaugeMetricsLog.Infof("interval=%s, metrics disabled", gaugeMetricsConfig.Interval)
		} else {
			gaugeMetricsLog.Infof(
				"interval=%s, full_metrics_factor=%d, range: %d .. %d, repeat: 1 .. %d, seed: %d",
				gaugeMetricsConfig.Interval, gaugeMetricsConfig.FullMetricsFactor,
				gaugeMetricsConfig.ParserConfig.Min, gaugeMetricsConfig.ParserConfig.Max,
				gaugeMetricsConfig.ParserConfig.MaxRepeat, gaugeMetricsConfig.ParserConfig.Seed,
			)
			tasks = append(tasks, NewGaugeMetrics(gaugeMetricsConfig))
		}
		return tasks, nil
	} else {
		return nil, fmt.Errorf("cfg: GaugeMetricsTaskBuilder passed wrong config type %T", cfg)
	}
}

func init() {
	vmi.RegisterTaskBuilder(GaugeMetricsTaskBuilder)
}
