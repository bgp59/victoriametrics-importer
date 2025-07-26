// internal metrics

package vmi_internal

import (
	"bytes"
	"fmt"
	"strconv"
	"time"
)

// Generate internal metrics:
const (
	INTERNAL_METRICS_CONFIG_INTERVAL_DEFAULT            = 5 * time.Second
	INTERNAL_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT = 12

	// This generator id:
	INTERNAL_METRICS_ID = "internal_metrics"
)

var internalMetricsLog = NewCompLogger(INTERNAL_METRICS_ID)

// The following OsInfo keys will be used as labels in OS info metrics:
var OSInfoLabelKeys = []string{
	"name",
	"release",
	"version",
	"machine",
}

// The following OSRelease keys will be used as labels in OS info metrics:
var OSReleaseLabelKeys = []string{
	"id",
	"name",
	"pretty_name",
	"version",
	"version_codename",
	"version_id",
}

type InternalMetricsConfig struct {
	Interval          time.Duration `yaml:"interval"`
	FullMetricsFactor int           `yaml:"full_metrics_factor"`
}

func DefaultInternalMetricsConfig() *InternalMetricsConfig {
	return &InternalMetricsConfig{
		Interval:          INTERNAL_METRICS_CONFIG_INTERVAL_DEFAULT,
		FullMetricsFactor: INTERNAL_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT,
	}
}

type internalMetricsGenFunc func(*bytes.Buffer, []byte) (int, int, *bytes.Buffer)

type InternalMetrics struct {
	GeneratorBase

	// Scheduler specific metrics:
	schedulerMetrics *SchedulerInternalMetrics

	// Compressor pool specific metrics:
	compressorPoolMetrics *CompressorPoolInternalMetrics

	// HTTP Endpoint Pool specific metrics:
	httpEndpointPoolMetrics *HttpEndpointPoolInternalMetrics

	// Go specific metrics:
	goMetrics *GoInternalMetrics

	// OS metrics related to this process:
	processMetrics *ProcessInternalMetrics

	// Generator specific metrics:
	generatorMetrics *GeneratorInternalMetrics

	// A cache for the actual generator function list, based on the above:
	mGenFuncList []internalMetricsGenFunc

	// Cache for additional metrics:
	vmiUptimeMetric    []byte
	vmiBuildinfoMetric []byte
	osInfoMetric       []byte
	osReleaseMetric    []byte
	osUptimeMetric     []byte

	// The following additional fields are needed for testing only. Left to
	// their default values, the usual objects will be used.
	version   string
	gitInfo   string
	bootTime  *time.Time
	startTs   *time.Time
	osInfo    map[string]string
	osRelease map[string]string
}

// Reference for importer uptime:
var startTs = time.Now()

func NewInternalMetrics(internalMetricsCfg *InternalMetricsConfig) (*InternalMetrics, error) {
	if internalMetricsCfg == nil {
		internalMetricsCfg = DefaultInternalMetricsConfig()
	}
	internalMetrics := &InternalMetrics{
		GeneratorBase: GeneratorBase{
			Id:                INTERNAL_METRICS_ID,
			Interval:          internalMetricsCfg.Interval,
			FullMetricsFactor: internalMetricsCfg.FullMetricsFactor,
		},
	}
	internalMetrics.schedulerMetrics = NewSchedulerInternalMetrics(internalMetrics)
	if compressorPool != nil {
		internalMetrics.compressorPoolMetrics = NewCompressorPoolInternalMetrics(internalMetrics)
	}
	if httpEndpointPool != nil {
		internalMetrics.httpEndpointPoolMetrics = NewHttpEndpointPoolInternalMetrics(internalMetrics)
	}
	internalMetrics.goMetrics = NewGoInternalMetrics(internalMetrics)
	internalMetrics.processMetrics = NewProcessInternalMetrics(internalMetrics)
	internalMetrics.generatorMetrics = NewGeneratorInternalMetrics(internalMetrics)
	internalMetricsLog.Infof(
		"id=%s, interval=%s, full_metrics_factor=%d",
		internalMetrics.Id, internalMetrics.Interval, internalMetrics.FullMetricsFactor,
	)
	return internalMetrics, nil
}

func (internalMetrics *InternalMetrics) initialize() {
	internalMetrics.GenBaseInit()
	internalMetrics.CycleNum = GetInitialCycleNum(internalMetrics.FullMetricsFactor)

	instance, hostname := internalMetrics.Instance, internalMetrics.Hostname

	internalMetrics.vmiUptimeMetric = []byte(fmt.Sprintf(
		`%s{%s="%s",%s="%s"} `, // N.B. whitespace before value!
		VMI_UPTIME_METRIC,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
	))

	version, gitInfo := Version, GitInfo
	if internalMetrics.version != "" {
		version = internalMetrics.version
	}
	if internalMetrics.gitInfo != "" {
		gitInfo = internalMetrics.gitInfo
	}
	internalMetrics.vmiBuildinfoMetric = []byte(fmt.Sprintf(
		`%s{%s="%s",%s="%s",%s="%s",%s="%s"} 1`, // value included
		VMI_BUILDINFO_METRIC,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
		VMI_VERSION_LABEL_NAME, version,
		VMI_GIT_INFO_LABEL_NAME, gitInfo,
	))

	osInfo, osRelease := OsInfo, OsRelease
	if internalMetrics.osInfo != nil {
		osInfo = internalMetrics.osInfo
	}
	if internalMetrics.osRelease != nil {
		osRelease = internalMetrics.osRelease
	}

	buf := &bytes.Buffer{}
	fmt.Fprintf(
		buf,
		`%s{%s="%s",%s="%s"`,
		OS_INFO_METRIC,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
	)
	for _, key := range OSInfoLabelKeys {
		fmt.Fprintf(buf, `,%s%s="%s"`, OS_INFO_LABEL_PREFIX, key, osInfo[key])
	}
	fmt.Fprintf(buf, `} 1`) // N.B. value included
	internalMetrics.osInfoMetric = bytes.Clone(buf.Bytes())

	buf.Reset()
	fmt.Fprintf(
		buf,
		`%s{%s="%s",%s="%s"`,
		OS_RELEASE_METRIC,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
	)
	for _, key := range OSReleaseLabelKeys {
		fmt.Fprintf(buf, `,%s%s="%s"`, OS_RELEASE_LABEL_PREFIX, key, osRelease[key])
	}
	fmt.Fprintf(buf, `} 1`) // N.B. value included
	internalMetrics.osReleaseMetric = bytes.Clone(buf.Bytes())

	internalMetrics.osUptimeMetric = []byte(fmt.Sprintf(
		`%s{%s="%s",%s="%s"} `, // N.B. space before value included
		OS_UPTIME_METRIC,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
	))

	if internalMetrics.bootTime == nil {
		internalMetrics.bootTime = &BootTime
	}

	if internalMetrics.startTs == nil {
		internalMetrics.startTs = &startTs
	}

	internalMetrics.Initialized = true
}

func (internalMetrics *InternalMetrics) TaskAction() bool {
	firstPass := !internalMetrics.Initialized
	if firstPass {
		internalMetrics.initialize()
	}

	schedulerMetrics := internalMetrics.schedulerMetrics
	compressorPoolMetrics := internalMetrics.compressorPoolMetrics
	httpEndpointPoolMetrics := internalMetrics.httpEndpointPoolMetrics
	goMetrics := internalMetrics.goMetrics
	processMetrics := internalMetrics.processMetrics
	generatorMetrics := internalMetrics.generatorMetrics

	if !internalMetrics.TestMode {
		// Collect stats from various sources:
		schedulerMetrics.stats[schedulerMetrics.currIndex] = scheduler.SnapStats(
			schedulerMetrics.stats[schedulerMetrics.currIndex],
		)
		if compressorPoolMetrics != nil {
			compressorPoolMetrics.stats[compressorPoolMetrics.currIndex] = compressorPool.SnapStats(
				compressorPoolMetrics.stats[compressorPoolMetrics.currIndex],
			)
		}
		if httpEndpointPoolMetrics != nil {
			httpEndpointPoolMetrics.stats[httpEndpointPoolMetrics.currIndex] = httpEndpointPool.SnapStats(
				httpEndpointPoolMetrics.stats[httpEndpointPoolMetrics.currIndex],
			)
		}
		goMetrics.SnapStats()
		processMetrics.SnapStats()
		generatorMetrics.SnapStats()
	}

	// Timestamp when all stats were collected:
	ts := internalMetrics.TimeNowFunc()

	// Metrics queue and buffer:
	metricsQueue := internalMetrics.MetricsQueue
	buf := metricsQueue.GetBuf()

	// Always start w/ the base metrics; this will also update the timestamp
	// suffix:
	metricsCount, _ := internalMetrics.GenBaseMetricsStart(buf, ts)
	tsSuffix := internalMetrics.TsSuffixBuf.Bytes()
	byteCount := 0

	if !internalMetrics.TestMode {
		var partialMetricsCount, partialByteCount int
		mGenFuncList := internalMetrics.mGenFuncList
		if mGenFuncList == nil {
			mGenFuncList = []internalMetricsGenFunc{
				schedulerMetrics.generateMetrics,
				goMetrics.generateMetrics,
				processMetrics.generateMetrics,
				generatorMetrics.generateMetrics,
			}
			if compressorPoolMetrics != nil {
				mGenFuncList = append(mGenFuncList, compressorPoolMetrics.generateMetrics)
			}
			if httpEndpointPoolMetrics != nil {
				mGenFuncList = append(mGenFuncList, httpEndpointPoolMetrics.generateMetrics)
			}
			internalMetrics.mGenFuncList = mGenFuncList
		}
		for _, mGenFunc := range mGenFuncList {
			partialMetricsCount, partialByteCount, buf = mGenFunc(buf, tsSuffix)
			metricsCount += partialMetricsCount
			byteCount += partialByteCount
		}
	}

	if buf == nil {
		buf = metricsQueue.GetBuf()
	}

	buf.Write(internalMetrics.vmiUptimeMetric)
	buf.WriteString(strconv.FormatFloat(ts.Sub(*internalMetrics.startTs).Seconds(), 'f', UPTIME_METRIC_PRECISION, 64))
	buf.Write(tsSuffix)
	metricsCount++

	buf.Write(internalMetrics.osUptimeMetric)
	buf.WriteString(strconv.FormatFloat(ts.Sub(*internalMetrics.bootTime).Seconds(), 'f', UPTIME_METRIC_PRECISION, 64))
	buf.Write(tsSuffix)
	metricsCount++

	if firstPass || internalMetrics.CycleNum == 0 {
		buf.Write(internalMetrics.vmiBuildinfoMetric)
		buf.Write(tsSuffix)
		metricsCount++

		buf.Write(internalMetrics.osInfoMetric)
		buf.Write(tsSuffix)
		metricsCount++

		buf.Write(internalMetrics.osReleaseMetric)
		buf.Write(tsSuffix)
		metricsCount++
	}

	// Add this generator's metrics by hand since it is the one that generates
	// such metrics so it cannot include itself in the general framework:
	imgMetrics := generatorMetrics.metricsCache[internalMetrics.Id]
	if imgMetrics == nil {
		generatorMetrics.updateMetricsCache(internalMetrics.Id)
		imgMetrics = generatorMetrics.metricsCache[internalMetrics.Id]
	}
	metricsCount += len(imgMetrics)

	buf.Write(imgMetrics[METRICS_GENERATOR_INVOCATION_COUNT])
	buf.WriteByte('1')
	buf.Write(tsSuffix)

	buf.Write(imgMetrics[METRICS_GENERATOR_METRICS_COUNT])
	buf.WriteString(strconv.FormatInt(int64(metricsCount), 10))
	buf.Write(tsSuffix)

	buf.Write(imgMetrics[METRICS_GENERATOR_BYTE_COUNT])

	// For the actual byte count, let m denote the count *without* the d bytes
	// needed for the representation of the count itself (the number of digits,
	// that is); d should satisfy: 10**(d-1) <= m + d < 10**d. The heuristic for
	// d is to start from d = 1 and to increment it until m + d < 10**d.
	byteCount, pow10 := byteCount+buf.Len()+len(tsSuffix)+1, 1
	for {
		pow10 *= 10
		if byteCount < pow10 {
			break
		}
		byteCount += 1
	}
	buf.WriteString(strconv.FormatInt(int64(byteCount), 10))
	buf.Write(tsSuffix)

	metricsQueue.QueueBuf(buf)

	if internalMetrics.CycleNum++; internalMetrics.CycleNum >= internalMetrics.FullMetricsFactor {
		internalMetrics.CycleNum = 0
	}

	return true
}

// Define and register the task builder:
func InternalMetricsTaskBuilder(vmiConfig *VmiConfig) (*Task, error) {
	if vmiConfig.InternalMetricsConfig.Interval <= 0 {
		internalMetricsLog.Infof(
			"interval=%s, metrics disabled", vmiConfig.InternalMetricsConfig.Interval,
		)
		return nil, nil
	}

	internalMetrics, err := NewInternalMetrics(vmiConfig.InternalMetricsConfig)
	if err != nil {
		return nil, err
	}
	return NewTask(internalMetrics.GetId(), internalMetrics.GetInterval(), internalMetrics.TaskAction), nil
}
