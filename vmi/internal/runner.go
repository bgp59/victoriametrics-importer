package vmi_internal

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bgp59/logrusx"
)

// The runner is the main entry point for an instance VMI importer.
//
// It is responsible for loading the configuration, setting up the environment,
// and running the importer. The runner is also responsible for handling errors
// and logging messages.
//
// The runner will create a logger, a scheduler, a compressor pool, and an HTTP
// endpoint pool. It will then add tasks to the scheduler, each task handling a
// set of metrics.
//
// Tasks are created at runtime based on the configuration. Each metrics
// generator is responsible for creating the appropriate task or tasks. It does
// so by registering a task builder function with the generator via init()
// functions; a builder takes in a configuration as an argument and it returns a
// a list (technically a slice) of tasks.
//
// Some of the configuration parameters may be overridden via command line
// arguments. The latter must be parsed by the main function *before* calling
// the runner.
//
// The runner will also handle the shutdown of the importer. It will wait for
// all tasks to finish before exiting. The shutdown will be triggered by a
// signal (SIGINT or SIGTERM) and it will have a grace period. If the tasks do
// not finish within the grace period, the runner will forcefully terminate the
// importer.

const (
	CONFIG_FLAG_NAME = "config"
	INSTANCE_DEFAULT = "vmi"
)

// The generated metrics are written into *bytes.Buffer's which are then queued
// into the metrics queue for transmission.
type BufferQueue interface {
	GetBuf() *bytes.Buffer
	ReturnBuf(b *bytes.Buffer)
	QueueBuf(b *bytes.Buffer)
	GetTargetSize() int
}

// The metrics generator interface which allows it to be scheduled as a Task:
type MetricsGeneratorTask interface {
	GetId() string
	GetInterval() time.Duration
	TaskActivity() bool
}

var (
	// The hostname, based on OS, config or command line arg.
	Hostname string

	// The instance should be primed w/ the desired default *before* invoking
	// the runner, most likely from an init() (e.g. set to "lsvmi" for Linux
	// Stats VictoriaMetrics importer) Its value may be modified via config and
	// command line args.
	Instance string = INSTANCE_DEFAULT

	// Build info, normally set via init() by the user of this package.
	Version string
	GitInfo string

	// Components:
	compressorPool   *CompressorPool
	httpEndpointPool *HttpEndpointPool
	MetricsGenStats  = NewMetricsGeneratorStatsContainer()
	MetricsQueue     BufferQueue
	scheduler        *Scheduler
	// The task builders are registered by the metrics generators via init()
	// functions. Each builder takes a configuration as an argument and returns
	// a list of MetricsGeneratorTask that perform the actual metrics generation.
	taskBuilders = struct {
		builders []func(config any) ([]MetricsGeneratorTask, error)
		mu       *sync.Mutex
	}{make([]func(config any) ([]MetricsGeneratorTask, error), 0), &sync.Mutex{}}

	// Metrics generation may take the delta approach whereby a specific metric
	// is generated only if its value has changed from the previous scan.
	// However in order to avoid going back too far in the past for the last
	// value, the generation will occur periodically, even if there was no
	// change. To that end, each (group of) metric(s) will have a cycle# and
	// full metrics factor (FMF). The cycle# is incremented modulo FMF for every
	// scan and every time it reaches 0 it indicates a full metrics cycle (FMC).
	// To even out (approx) FMC occurrences, the cycle# is initialized
	// differently every time a new metric is created. The next structure
	// provides the initial value.
	initialCycleNum = struct {
		cycleNum int
		mu       *sync.Mutex
	}{0, &sync.Mutex{}}
)

func RegisterTaskBuilder(tb func(config any) ([]MetricsGeneratorTask, error)) {
	taskBuilders.mu.Lock()
	taskBuilders.builders = append(taskBuilders.builders, tb)
	taskBuilders.mu.Unlock()
}

func GetInitialCycleNum(fullMetricsFactor int) int {
	if fullMetricsFactor <= 1 {
		return 0
	}
	initialCycleNum.mu.Lock()
	defer initialCycleNum.mu.Unlock()
	cycleNum := initialCycleNum.cycleNum
	if initialCycleNum.cycleNum++; initialCycleNum.cycleNum < 0 {
		// Max int rollover, cycle back to 0:
		initialCycleNum.cycleNum = 0
	}
	return cycleNum % fullMetricsFactor
}

// Command line args; they should be defined at package scope since the flags are
// parsed in main.
var (
	versionArg = flag.Bool(
		"version",
		false,
		FormatFlagUsage(
			`Print the version and exit`,
		),
	)

	configFileArg = flag.String(
		CONFIG_FLAG_NAME,
		fmt.Sprintf("%s-config.yaml", INSTANCE_DEFAULT),
		`Config file to load`,
	)

	hostnameArg = flag.String(
		"hostname",
		"",
		FormatFlagUsage(
			`Override the the value returned by hostname syscall`,
		),
	)

	instanceArg = flag.String(
		"instance",
		"",
		FormatFlagUsage(
			`Override the "vmi_config.instance" config setting`,
		),
	)

	useStdoutMetricsQueueArg = flag.Bool(
		"use-stdout-metrics-queue",
		false,
		FormatFlagUsage(
			`Print metrics to stdout instead of sending to import endpoints`,
		),
	)

	httpPoolEndpointsArg = flag.String(
		"http-pool-endpoints",
		"",
		FormatFlagUsage(
			`Override the "vmi_config.http_endpoint_pool_config.endpoints" config setting`,
		),
	)
)

func init() {
	logrusx.EnableLoggerArgs()
}

// The runner is the main entry point for an actual VMI importer instance. It
// should be called with the default generators configuration as its argument.
// The return value is the exit code of the executable.

var runnerLog = NewCompLogger("runner")

func Run(genConfig any) int {
	var (
		err           error
		shutdownTimer *time.Timer
		vmiConfig     *VmiConfig
	)

	if !flag.Parsed() {
		flag.Parse()
	}

	if *versionArg {
		fmt.Fprintf(os.Stderr, "Version: %s, GitInfo: %s\n", Version, GitInfo)
		return 0
	}

	configFile := *configFileArg
	vmiConfig, err = LoadConfig(configFile, genConfig, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config file: %v\n", err)
		return 1
	}

	// Override the config with command line args:
	if *instanceArg != "" {
		vmiConfig.Instance = *instanceArg
	}
	if *httpPoolEndpointsArg != "" {
		vmiConfig.HttpEndpointPoolConfig.OverrideEndpoints(*httpPoolEndpointsArg)
	}
	logrusx.ApplySetLoggerArgs(vmiConfig.LoggerConfig)

	// Set the logger level and file:
	err = SetLogger(vmiConfig.LoggerConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error setting the logger: %v\n", err)
		return 1
	}

	// Set the globals:
	Instance = vmiConfig.Instance
	if *hostnameArg != "" {
		Hostname = *hostnameArg
	} else {
		Hostname, err = os.Hostname()
		if err != nil {
			runnerLog.Errorf("Error getting hostname: %v", err)
			return 1
		}
		if vmiConfig.UseShortHostname {
			i := strings.Index(Hostname, ".")
			if i > 0 {
				Hostname = Hostname[:i]
			}
		}
	}

	// Create a stopped timer to provide timeout support at shutdown. The
	// shutdown of various components (scheduler, compressor, HTTP endpoint
	// pool) is performed via `defer` functions. Since they are executed in LIFO
	// order, the timeoutTimer's stop should be registered 1st.
	if vmiConfig.ShutdownMaxWait > 0 {
		// Create a stopped timer:
		shutdownTimer = time.NewTimer(1 * time.Hour)
		shutdownTimer.Stop()
		// The timer will be activated after a signal was received. All the
		// importer components will be shutdown and if they terminate before the
		// timer expires, the deferred stop below will get invoked.
		defer shutdownTimer.Stop()
	}

	// Set the metrics queue:
	if !*useStdoutMetricsQueueArg {
		// Real queue w/ compressed metrics sent to import endpoints:
		httpEndpointPool, err = NewHttpEndpointPool(vmiConfig.HttpEndpointPoolConfig)
		if err != nil {
			runnerLog.Fatal(err)
		}

		compressorPool, err = NewCompressorPool(vmiConfig.CompressorPoolConfig)
		if err != nil {
			runnerLog.Fatal(err)
		}
		MetricsQueue = compressorPool

		compressorPool.Start(httpEndpointPool)
		defer httpEndpointPool.Shutdown() // may timeout if all endpoints are down
		defer compressorPool.Shutdown()
	} else {
		// Simulated queue w/ metrics displayed to stdout:
		MetricsQueue, err = NewStdoutMetricsQueue(vmiConfig.CompressorPoolConfig)
		if err != nil {
			runnerLog.Fatal(err)
		}
		defer MetricsQueue.(*StdoutMetricsQueue).Shutdown()
	}

	// Scheduler:
	scheduler, err = NewScheduler(vmiConfig.SchedulerConfig)
	if err != nil {
		runnerLog.Fatal(err)
	}
	scheduler.Start()
	defer scheduler.Shutdown()

	// Initialize metrics generators:
	taskList := make([]*Task, 0)
	taskBuilders.mu.Lock()
	for _, tb := range taskBuilders.builders {
		genTasks, err := tb(genConfig)
		if err != nil {
			runnerLog.Fatal(err)
		}
		for _, genTask := range genTasks {
			taskList = append(taskList, NewTask(genTask.GetId(), genTask.GetInterval(), genTask.TaskActivity))
		}
	}
	taskBuilders.mu.Unlock()
	// Initialize internal metrics:
	task, err := InternalMetricsTaskBuilder(vmiConfig)
	if err != nil {
		runnerLog.Fatal(err)
	}
	taskList = append(taskList, task)

	// Add all tasks to the scheduler:
	for _, task := range taskList {
		scheduler.AddNewTask(task)
	}

	// Log instance and hostname, useful for dashboard variable selection:
	runnerLog.Infof("Instance: %s, Hostname: %s", Instance, Hostname)

	// Block until a signal is received:
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	sig := <-sigChan
	if vmiConfig.ShutdownMaxWait == 0 {
		runnerLog.Fatalf("%s signal received, force exit", sig)
	} else {
		runnerLog.Warnf("%s signal received, shutting down", sig)
	}

	if shutdownTimer != nil {
		// Trigger timeout watchdog: if it fires, it will forcibly exit the program.
		go func() {
			shutdownTimer.Reset(vmiConfig.ShutdownMaxWait)
			<-shutdownTimer.C
			runnerLog.Fatalf("shutdown timed out after %s, force exit", vmiConfig.ShutdownMaxWait)
		}()
	}

	return 0
}
