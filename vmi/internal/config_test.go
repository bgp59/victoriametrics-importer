package vmi_internal

import (
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/huandu/go-clone"
)

type LoadConfigTestCase struct {
	Name          string
	Description   string
	GenConfig     any
	Data          string
	WantVmiConfig *VmiConfig
	WantGenConfig any
	WantErr       error
}

type Gen1ConfigTest struct {
	Id       string        `yaml:"id"`
	Interval time.Duration `yaml:"interval"`
	Exclude  []string      `yaml:"exclude"`
}

type Gen2ConfigTest struct {
	Id       string        `yaml:"id"`
	Interval time.Duration `yaml:"interval"`
	Timeout  time.Duration `yaml:"timeout"`
	Include  []string      `yaml:"include"`
}

type GenConfigTest struct {
	Gen1 *Gen1ConfigTest `yaml:"gen1"`
	Gen2 *Gen2ConfigTest `yaml:"gen2"`
}

func defaultGenConfig() *GenConfigTest {
	return &GenConfigTest{
		Gen1: &Gen1ConfigTest{Id: "gen1"},
		Gen2: &Gen2ConfigTest{Id: "gen2"},
	}
}

func testLoadConfig(t *testing.T, tc *LoadConfigTestCase) {
	if tc.Description == "" {
		t.Log(tc.Description)
	}
	genConfig := clone.Clone(tc.GenConfig)
	gotVmiConfig, err := LoadConfig("", genConfig, []byte(strings.ReplaceAll(tc.Data, "\t", "  ")))
	if tc.WantErr == nil && err != nil {
		t.Fatal(err)
	}
	if tc.WantErr != nil && err == nil {
		t.Fatalf("err: want %v, got %v", tc.WantErr, err)
	}

	if diff := cmp.Diff(tc.WantVmiConfig, gotVmiConfig); diff != "" {
		t.Fatalf("VmiConfig mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(tc.WantGenConfig, genConfig); diff != "" {
		t.Fatalf("GenConfig mismatch (-want +got):\n%s", diff)
	}
}

func TestLoadVmiConfig(t *testing.T) {
	generatorsData := `
		generators:
			- name: gen1
			  type: test
			  config:
				foo: bar
			- name: gen2
			  type: test
			  config:
				foo: baz
	`
	ignoredData := `
		ignore:
			- name: name1
			  type: test
			  config:
				foo: bar
			- name: name2
			  type: test
			  config:
				foo: baz
	`
	name1 := "vmi_config"
	data1 := `
		vmi_config:
			instance: inst1
			shutdown_max_wait: 7s
	`
	vmiCfg1 := DefaultVmiConfig()
	vmiCfg1.Instance = "inst1"
	vmiCfg1.ShutdownMaxWait = 7 * time.Second

	name2 := "scheduler_config"
	data2 := `
		vmi_config:
			scheduler_config:
				num_workers: 5
	`
	vmiCfg2 := DefaultVmiConfig()
	vmiCfg2.SchedulerConfig.NumWorkers = 5

	name3 := "compressor_pool_config"
	data3 := `
		vmi_config:
			compressor_pool_config:
				num_compressors: 13
	`
	vmiCfg3 := DefaultVmiConfig()
	vmiCfg3.CompressorPoolConfig.NumCompressors = 13

	name4 := "http_endpoint_pool_config"
	data4 := `
		vmi_config:
			http_endpoint_pool_config:
				endpoints:
					- url: http://host1:8081
					  mark_unhealthy_threshold: 11
					- url: http://host2:8082
					  mark_unhealthy_threshold: 22
	`
	vmiCfg4 := DefaultVmiConfig()
	vmiCfg4.HttpEndpointPoolConfig.Endpoints = []*HttpEndpointConfig{
		{
			URL:                    "http://host1:8081",
			MarkUnhealthyThreshold: 11,
		},
		{
			URL:                    "http://host2:8082",
			MarkUnhealthyThreshold: 22,
		},
	}

	name5 := "log_config"
	data5 := `
		vmi_config:
			log_config:
				level: debug
	`
	vmiCfg5 := DefaultVmiConfig()
	vmiCfg5.LoggerConfig.Level = "debug"

	name6 := "internal_metrics_config"
	data6 := `
		vmi_config:
			internal_metrics_config:
				interval: 13s
	`
	vmiCfg6 := DefaultVmiConfig()
	vmiCfg6.InternalMetricsConfig.Interval = 13 * time.Second

	for _, tc := range []*LoadConfigTestCase{
		{
			Name:          "default",
			WantVmiConfig: DefaultVmiConfig(),
		},
		{
			Name: "vmi_config_empty",
			Data: `
				vmi_config:
			`,
			WantVmiConfig: DefaultVmiConfig(),
		},
		{
			Name:          name1,
			Data:          data1,
			WantVmiConfig: vmiCfg1,
		},
		{
			Name:          name2,
			Data:          data2,
			WantVmiConfig: vmiCfg2,
		},
		{
			Name:          name3,
			Data:          data3,
			WantVmiConfig: vmiCfg3,
		},
		{
			Name:          name4,
			Data:          data4,
			WantVmiConfig: vmiCfg4,
		},
		{
			Name:          name5,
			Data:          data5,
			WantVmiConfig: vmiCfg5,
		},
		{
			Name:          name6,
			Data:          data6,
			WantVmiConfig: vmiCfg6,
		},
		{
			Name:          name1 + "_plus_generators",
			Data:          data1 + generatorsData,
			WantVmiConfig: vmiCfg1,
		},
		{
			Name:          "generators_plus_" + name1,
			Data:          generatorsData + data1,
			WantVmiConfig: vmiCfg1,
		},
		{
			Name:          name1 + "_plus_ignored",
			Data:          data1 + ignoredData,
			WantVmiConfig: vmiCfg1,
		},
	} {
		t.Run(
			tc.Name,
			func(t *testing.T) { testLoadConfig(t, tc) },
		)
	}
}

func TestLoadGenConfig(t *testing.T) {
	data := `
		generators:
			gen1:
				#id: gen1
				interval: 10s
				exclude: ["foo", "bar"]
			gen2:
				id: gentwo
				interval: 20s
				timeout: 30s
				include: ["baz", "qux"]
	`
	wantGenConfig := defaultGenConfig()
	wantGenConfig.Gen1.Id = "gen1"
	wantGenConfig.Gen1.Interval = 10 * time.Second
	wantGenConfig.Gen1.Exclude = []string{"foo", "bar"}
	wantGenConfig.Gen2.Id = "gentwo"
	wantGenConfig.Gen2.Interval = 20 * time.Second
	wantGenConfig.Gen2.Timeout = 30 * time.Second
	wantGenConfig.Gen2.Include = []string{"baz", "qux"}
	tc := &LoadConfigTestCase{
		Name:          "gen_config",
		Description:   "Test loading generator configuration",
		GenConfig:     defaultGenConfig(),
		Data:          data,
		WantVmiConfig: DefaultVmiConfig(),
		WantGenConfig: wantGenConfig,
		WantErr:       nil,
	}
	t.Run(
		tc.Name,
		func(t *testing.T) { testLoadConfig(t, tc) },
	)
}
