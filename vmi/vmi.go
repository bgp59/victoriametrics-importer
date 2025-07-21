// The public face of the importer for the users of this package

package vmi

import (
	"flag"

	"github.com/sirupsen/logrus"

	vmi_internal "github.com/bgp59/victoriametrics-importer/vmi/internal"
)

const (
	INSTANCE_LABEL_NAME             = vmi_internal.INSTANCE_LABEL_NAME
	HOSTNAME_LABEL_NAME             = vmi_internal.HOSTNAME_LABEL_NAME
	METRICS_GENERATOR_ID_LABEL_NAME = vmi_internal.METRICS_GENERATOR_ID_LABEL_NAME
)

type BufferQueue = vmi_internal.BufferQueue
type MetricsGeneratorTask = vmi_internal.MetricsGeneratorTask
type GeneratorBase = vmi_internal.GeneratorBase

// The instance should be primed w/ the desired default *before* invoking
// the runner, typically from an init(). Its value may be modified via
// config and command line args.
func SetDefaultInstance(instance string) {
	vmi_internal.Instance = instance
}

// Set the config flag default value, typically to
// <default_instance>-config.yaml:
func SetDefaultConfigFile(filePath string) {
	if configFlag := flag.Lookup(vmi_internal.CONFIG_FLAG_NAME); configFlag != nil {
		if err := configFlag.Value.Set(filePath); err == nil {
			configFlag.DefValue = filePath
		}
	}
}

// Update build info: version (semver) and git info. This function should be
// called *before* the runner is invoked, typically from an init() function.
func UpdateBuildInfo(version, gitInfo string) {
	vmi_internal.Version = version
	vmi_internal.GitInfo = gitInfo
}

// Get the instance, which is typically set from the command line or config.
func GetInstance() string {
	return vmi_internal.Instance
}

// Get the hostname, based on OS, config and/or command line arg.
func GetHostname() string {
	return vmi_internal.Hostname
}

// The root logger. Needed only for tests where the logger is captured (see
// vmi/testutils/log_collector.go), its actual type is obscured. The only use
// case for call is during tests, as follows:
//
//	func TestSomethingWithLogger() {
//		tlc := vmi_testutils.NewTestLogCollect(t, vmi.GetRootLogger(), nil)
//		defer tlc.RestoreLog()
//		// Everything logged via the VMI logger will be captured by the tlc object
//		// and it will be displayed in the test output at the end, if the test fails
//		// or if it is run in verbose mode.
//	}
func GetRootLogger() any { return vmi_internal.RootLogger }

// Create new component logger w/ comp=compName field:
func NewCompLogger(comp string) *logrus.Entry {
	return vmi_internal.NewCompLogger(comp)
}

// When logging files, the log file name is derived from the file path
// typically relative to the module root dir. The logger maintains a list of
// prefixes to strip and the following function will add the caller's module
// path to it. The latter is inferred from the caller's file path, going up
// N dirs. Typically the call is made from main.init() so the parameter is 0
// (assuming that main.go is at the root dir of the module).
func AddCallerSrcPathPrefixToLogger(upNDirs int) {
	// skip = 1 below to base the caller's path on the caller of this function.
	vmi_internal.AddCallerSrcPathPrefixToLogger(upNDirs, 1)
}

// Utility function to build a basic auth header value from the username and
// password. The password is is first subject to env var interpolation. If the
// resulting string starts with "file://", then the rest of the string is
// interpreted as a path to a file containing the password:
func BuildHtmlBasicAuth(username, password string) (string, error) {
	return vmi_internal.BuildHtmlBasicAuth(username, password)
}

// The MetricsQueue will be initialized by the runner, depending upon config and
// command line args. It can be either a compressor queue sending data to an
// HTTP end-point pool (typical case), or, for test purposes it could be a
// print-to-stdout queue.
//
// The general flow of the TaskActivity implementation:
//
//	MetricsQueue <- GetMetricsQueue()
//	repeat until no more metrics
//	- buf <- MetricsQueue.GetBuf()
//	- fill buf with metrics until it reaches MetricsQueue.GetTargetSize() or
//	  there are no more metrics
//	- MetricsQueue.QueueBuf(buf)
func GetMetricsQueue() BufferQueue {
	return vmi_internal.MetricsQueue
}

// Each metrics generator has a set of standard stats, indexed by the generator
// ID. The stats are updated by the generator at the end of each run and they
// are used to create generator specific internal metrics.
func UpdateMetricsGeneratorStats[T interface{ int | int64 | uint64 }](genId string, metricCount, byteCount T) {
	vmi_internal.MetricsGenStats.Update(genId, uint64(metricCount), uint64(byteCount))
}

// Metrics generation may take the delta approach whereby a specific metric is
// generated only if its value has changed from the previous scan. However in
// order to avoid going back too far in the past for the last value, the
// generation will occur periodically, even if there was no change. To that end,
// each (group of) metric(s) will have a cycle# and full metrics factor (FMF).
// The cycle# is incremented modulo FMF for every scan and every time it reaches
// 0 it indicates a full metrics cycle (FMC). To (approximately) even out FMC
// occurrences, the cycle# is initialized differently every time a new metric is
// created.
func GetInitialCycleNum(fullMetricsFactor int) int {
	return vmi_internal.GetInitialCycleNum(fullMetricsFactor)
}

// All metrics generators have to register with the scheduler as a task or
// tasks. Each generator will have a task builder function, which given a
// generators config argument, will return a list of generator tasks and an
// error condition. The builders will be registered with runner from `init()'
// functions inside the generators. The argument is cast as `any' because the
// actual data structure is opaque and immaterial to the this framework.
func RegisterTaskBuilder(tb func(any) ([]MetricsGeneratorTask, error)) {
	vmi_internal.RegisterTaskBuilder(tb)
}

// The runner is the entry point for the generator loop. It takes as an argument
// the generators config primed with default values, it loads the config file
// thus altering some of the defaults and it invokes the registered task
// builders to create the list of tasks which will then proceed to add to the
// scheduler. Normally it returns only when the process is interrupted via a
// signal, or if the initialization failed. Its return value should be used as
// process exit status.
func Run(genConfig any) int { return vmi_internal.Run(genConfig) }
