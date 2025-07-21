// Generator configuration

package refvmi

// The following structure will be populated from the `generators' section of
// the YAML config file.
type RefvmiConfig struct {
	GaugeMetricsConfig       *GaugeMetricsConfig       `yaml:"gauge_metrics"`
	CounterMetricsConfig     *CounterMetricsConfig     `yaml:"counter_metrics"`
	CategoricalMetricsConfig *CategoricalMetricsConfig `yaml:"categorical_metrics"`
}

func DefaultRefvmiConfig() *RefvmiConfig {
	return &RefvmiConfig{
		GaugeMetricsConfig:       DefaultGaugeMetricsConfig(),
		CounterMetricsConfig:     DefaultCounterMetricsConfig(),
		CategoricalMetricsConfig: DefaultCategoricalMetricsConfig(),
	}
}
