// Importer configuration

// The configuration is loaded from a YAML file, with the following structure:
//
//  vmi_config:
//    instance: vmi
//    use_short_hostname: false
//    shutdown_max_wait: 5s
//    log_config:
//      ...
//    compressor_pool_config:
//	    ...
//    http_endpoint_pool_config:
//      ...
//    scheduler_config:
//      ...
//    internal_metrics_config:
//      ...
//  generators:
//     gen1:
//       ...
//     gen2:
//       ...
//
// The "vmi_config" section maps to the VmiConfig structure, which is
// defined in this package. The "generators" section is importer specific and
// is not defined here. It is expected to be a map of generator names to
// their specific configurations, which will be used by the importer to
// instantiate the generators.

package vmi_internal

import (
	"fmt"
	"io"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	VMI_CONFIG_SECTION_NAME = "vmi_config"
	GENERATORS_SECTION_NAME = "generators"

	VMI_CONFIG_USE_SHORT_HOSTNAME_DEFAULT = false
	VMI_CONFIG_SHUTDOWN_MAX_WAIT_DEFAULT  = 5 * time.Second
)

type VmiConfig struct {
	// The instance name, default "vmi". It may be overridden by --instance
	// command line arg.
	Instance string `yaml:"instance"`

	// Whether to use short hostname or not as the value for hostname label.
	// Typically the hostname is determined from the hostname system call and if
	// the flag below is in effect, it is stripped of domain part. However if
	// the hostname is overridden by --hostname command line arg, that value is
	// used as-is.
	UseShortHostname bool `yaml:"use_short_hostname"`

	// How long to wait for a graceful shutdown. A negative value signifies
	// indefinite wait and 0 stands for no wait at all (exit abruptly).
	ShutdownMaxWait time.Duration `yaml:"shutdown_max_wait"`

	// Specific components configuration.
	LoggerConfig           *LoggerConfig           `yaml:"log_config"`
	CompressorPoolConfig   *CompressorPoolConfig   `yaml:"compressor_pool_config"`
	HttpEndpointPoolConfig *HttpEndpointPoolConfig `yaml:"http_endpoint_pool_config"`
	SchedulerConfig        *SchedulerConfig        `yaml:"scheduler_config"`

	// Internal metrics configuration.
	InternalMetricsConfig *InternalMetricsConfig `yaml:"internal_metrics_config"`
}

func DefaultVmiConfig() *VmiConfig {
	return &VmiConfig{
		Instance:               Instance,
		UseShortHostname:       VMI_CONFIG_USE_SHORT_HOSTNAME_DEFAULT,
		ShutdownMaxWait:        VMI_CONFIG_SHUTDOWN_MAX_WAIT_DEFAULT,
		LoggerConfig:           DefaultLoggerConfig(),
		CompressorPoolConfig:   DefaultCompressorPoolConfig(),
		HttpEndpointPoolConfig: DefaultHttpEndpointPoolConfig(),
		SchedulerConfig:        DefaultSchedulerConfig(),
		InternalMetricsConfig:  DefaultInternalMetricsConfig(),
	}
}

// LoadConfig loads the configuration from the specified YAML file (or buffer,
// for testing) as follows:
//   - the vmi_config section is returned as a *VmiConfig structure
//   - the generators section is loaded into the provided genConfig structure,
//     which expected to have been primed with default values.
//
// Additionally an error is returned if the configuration could not be
// loaded or parsed.
func LoadConfig(cfgFile string, genConfig any, buf []byte) (*VmiConfig, error) {
	if buf == nil {
		// Normal case, buf is pre-populated only for testing.
		f, err := os.Open(cfgFile)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		buf, err = io.ReadAll(f)
		if err != nil {
			return nil, fmt.Errorf("file: %q: %v", cfgFile, err)
		}
	}

	docNode := yaml.Node{}
	err := yaml.Unmarshal(buf, &docNode)
	if err != nil {
		return nil, fmt.Errorf("file: %q: %v", cfgFile, err)
	}

	vmiConfig := DefaultVmiConfig()
	if docNode.Kind == yaml.DocumentNode && len(docNode.Content) > 0 {
		rootNode := docNode.Content[0]
		if rootNode.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("file: %q: invalid YAML root node %q", cfgFile, rootNode.Tag)
		}
		var toCfg any = nil
		for _, n := range rootNode.Content {
			if n.Kind == yaml.ScalarNode {
				switch n.Value {
				case VMI_CONFIG_SECTION_NAME:
					toCfg = vmiConfig
				case GENERATORS_SECTION_NAME:
					toCfg = genConfig
				}
				continue
			}
			if n.Kind == yaml.MappingNode && toCfg != nil {
				if err = n.Decode(toCfg); err != nil {
					return nil, fmt.Errorf("file: %q: %v", cfgFile, err)
				}
			}
			toCfg = nil
		}
	}

	return vmiConfig, nil
}
