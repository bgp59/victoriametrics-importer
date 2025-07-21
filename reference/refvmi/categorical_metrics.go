// Categorical metrics generator example

package refvmi

import (
	"bytes"
	"fmt"
	"time"

	"github.com/bgp59/victoriametrics-importer/refvmi/parser"

	"github.com/bgp59/victoriametrics-importer/vmi"
)

const (
	CATEGORICAL_METRICS_CONFIG_INTERVAL_DEFAULT            = 5 * time.Second
	CATEGORICAL_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT = 12

	// This Metrics Generator ID:
	CATEGORICAL_METRICS_ID = "categorical"
)

// The logger for this generator:
var categoricalMetricsLog = vmi.NewCompLogger(CATEGORICAL_METRICS_ID)

// The metrics generator context:
type CategoricalMetrics struct {
	vmi.GeneratorBase

	// The underlying parser:
	parser *parser.RandomCategoricalParser

	// The previous value:
	val []byte

	// Cache for various metrics:
	//  - the previous metric:
	categoricalMetric []byte
}

// The configuration for this generator. It should be loadable from a YAML file
// and it may include a parser related section:
type CategoricalMetricsConfig struct {
	// How often to generate the metrics in time.ParseDuration() format:
	Interval time.Duration `yaml:"interval"`

	// Normally metrics are generated only if there is a change in value from
	// the previous scan. However every N cycles the full set is generated. Use
	// 0 to generate full metrics every cycle.
	FullMetricsFactor int `yaml:"full_metrics_factor"`

	// Parser configuration:
	ParserConfig *parser.RandomCategoricalParserConfig `yaml:"parser_config"`
}

func DefaultCategoricalMetricsConfig() *CategoricalMetricsConfig {
	return &CategoricalMetricsConfig{
		Interval:          CATEGORICAL_METRICS_CONFIG_INTERVAL_DEFAULT,
		FullMetricsFactor: CATEGORICAL_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT,
		ParserConfig:      parser.DefaultRandomCategoricalParserConfig(),
	}
}

func NewCategoricalMetrics(cfg *CategoricalMetricsConfig) *CategoricalMetrics {
	if cfg == nil {
		cfg = DefaultCategoricalMetricsConfig()
	}
	return &CategoricalMetrics{
		GeneratorBase: vmi.GeneratorBase{
			Id:                CATEGORICAL_METRICS_ID,
			Interval:          cfg.Interval,
			CycleNum:          vmi.GetInitialCycleNum(cfg.FullMetricsFactor),
			FullMetricsFactor: cfg.FullMetricsFactor,
		},
		parser: parser.NewRandomCategoricalParser(cfg.ParserConfig),
	}
}

func (m *CategoricalMetrics) initialize() {
	m.GenBaseInit()
	m.Initialized = true
}

// The actual metrics generation, it will be registered as the wrapping task's activity:
func (m *CategoricalMetrics) TaskActivity() bool {
	if !m.Initialized {
		m.initialize()
	}

	// Parse new info and handle errors:
	if !m.TestMode {
		err := m.parser.Parse()
		if err != nil {
			categoricalMetricsLog.Errorf("Parse(): %v", err)
			return false // This will disable future invocations.
		}
	}

	// All new data retrieved:
	ts := m.TimeNowFunc()
	metricsQueue := m.MetricsQueue
	buf := metricsQueue.GetBuf()

	metricsCount, _ := m.GenBaseMetricsStart(buf, ts)
	tsSuffix := m.TsSuffixBuf.Bytes()

	// Update the value cache:
	currVal := m.parser.Val

	// The metrics proper:
	changed := !bytes.Equal(currVal, m.val)
	if changed {
		m.val = currVal
		if m.categoricalMetric != nil {
			// Mark the previous category as inactive:
			buf.Write(m.categoricalMetric)
			buf.WriteByte('0')
			buf.Write(tsSuffix)
			metricsCount += 1
		}
		// Rebuild the metric:
		m.categoricalMetric = []byte(fmt.Sprintf(
			`%s{%s="%s",%s="%s",%s="%s"} `, // N.B. space before value is included
			CATEGORICAL_METRIC,
			vmi.INSTANCE_LABEL_NAME, m.Instance,
			vmi.HOSTNAME_LABEL_NAME, m.Hostname,
			CATEGORY_LABEL, currVal,
		))
	}
	if m.CycleNum == 0 || changed {
		buf.Write(m.categoricalMetric)
		buf.WriteByte('1')
		buf.Write(tsSuffix)
		metricsCount += 1
	}

	vmi.UpdateMetricsGeneratorStats(m.Id, metricsCount, buf.Len())

	// Queue the buffer for publish:
	metricsQueue.QueueBuf(buf)

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
func CategoricalMetricsTaskBuilder(cfg any) ([]vmi.MetricsGeneratorTask, error) {
	if refvmiConfig, ok := cfg.(*RefvmiConfig); ok {
		if refvmiConfig == nil {
			refvmiConfig = DefaultRefvmiConfig()
		}
		categoricalMetricsConfig := refvmiConfig.CategoricalMetricsConfig
		if categoricalMetricsConfig == nil {
			categoricalMetricsConfig = DefaultCategoricalMetricsConfig()
		}
		tasks := make([]vmi.MetricsGeneratorTask, 0)
		if categoricalMetricsConfig.Interval <= 0 {
			categoricalMetricsLog.Infof("interval=%s, metrics disabled", categoricalMetricsConfig.Interval)
		} else {
			categoricalMetricsLog.Infof(
				"interval=%s, full_metrics_factor=%d, choice#: %d, repeat: 1 .. %d, seed: %d",
				categoricalMetricsConfig.Interval, categoricalMetricsConfig.FullMetricsFactor,
				len(categoricalMetricsConfig.ParserConfig.Choices),
				categoricalMetricsConfig.ParserConfig.MaxRepeat, categoricalMetricsConfig.ParserConfig.Seed,
			)
			tasks = append(tasks, NewCategoricalMetrics(categoricalMetricsConfig))
		}
		return tasks, nil
	} else {
		return nil, fmt.Errorf("cfg: CategoricalMetricsTaskBuilder passed wrong config type %T", cfg)
	}
}

func init() {
	vmi.RegisterTaskBuilder(CategoricalMetricsTaskBuilder)
}
