// Tests for scheduler.go

package vmi_internal

import (
	"bytes"
	"fmt"
	"strconv"
	"testing"
	"time"

	vmi_testutils "github.com/bgp59/victoriametrics-importer/vmi/testutils"
)

type SchedulerExecuteTestCase struct {
	numWorkers int
	// The unit for intervals, execDurations and runTime:
	timeUnitSec float64
	// Set one task for each interval:
	intervals []float64
	// The execution times for each:
	execDurations [][]float64
	// How long to run the scheduler for:
	runTime float64
	// The scheduled intervals will be checked against the desired one and they
	// should be in the range of:
	//  (1 - scheduleIntervalPct/100)*interval .. (1 + scheduleIntervalPct/100)*interval
	// Use -1 to disable.
	scheduleIntervalPct float64
	// The maximum allowed number of irregular scheduling intervals, as
	// determined by the above:
	wantIrregularIntervalMaxCount []int
}

type TestTask struct {
	task          *Task
	execDurations []time.Duration // how long the task will take at invocation# i
	invokeIndx    int             // invocation# mod len(execDurations)
	invokeTss     []time.Time     // invocation timestamps
}

func (tt *TestTask) taskAction() bool {
	tt.invokeTss = append(tt.invokeTss, time.Now())
	schedulerLog.Infof(
		"Execute task %s: interval=%s, nextTs=%s",
		tt.task.id, tt.task.interval, tt.task.nextTs.Format(time.RFC3339Nano),
	)
	if n := len(tt.execDurations); n > 0 {
		time.Sleep(tt.execDurations[tt.invokeIndx])
		tt.invokeIndx++
		if tt.invokeIndx >= n {
			tt.invokeIndx = 0
		}
	}
	return true
}

func testSchedulerDurationFromSec(sec float64) time.Duration {
	return time.Duration(
		sec * float64(time.Second),
	)
}

func testSchedulerBuildTestTaskList(tc *SchedulerExecuteTestCase) []*TestTask {
	testTasks := make([]*TestTask, len(tc.intervals))
	for i, interval := range tc.intervals {
		tt := &TestTask{}
		if tc.execDurations != nil {
			execDurations := tc.execDurations[i]
			if execDurations != nil {
				tt.execDurations = make([]time.Duration, len(execDurations))
				for k, execDuration := range execDurations {
					tt.execDurations[k] = testSchedulerDurationFromSec(execDuration * tc.timeUnitSec)
				}
			}
		}
		tt.task = NewTask(strconv.Itoa(i), testSchedulerDurationFromSec(interval*tc.timeUnitSec), tt.taskAction)
		testTasks[i] = tt
	}
	return testTasks
}

func testSchedulerExecute(tc *SchedulerExecuteTestCase, t *testing.T) {
	tlc := vmi_testutils.NewTestLogCollect(t, RootLogger, nil)
	defer tlc.RestoreLog()

	numWorkers := tc.numWorkers
	if numWorkers <= 0 {
		numWorkers = 1
	}
	scheduler, err := NewScheduler(&SchedulerConfig{NumWorkers: tc.numWorkers})
	if err != nil {
		t.Fatal(err)
	}
	scheduler.Start()
	testTasks := testSchedulerBuildTestTaskList(tc)
	for _, testTask := range testTasks {
		scheduler.AddNewTask(testTask.task)
	}
	time.Sleep(testSchedulerDurationFromSec(tc.runTime * tc.timeUnitSec))
	scheduler.Shutdown()

	// Verify that each task was scheduled roughly at the expected intervals and
	// that it wasn't skipped:

	errBuf := &bytes.Buffer{}

	stats := scheduler.SnapStats(nil)

	type IrregularInterval struct {
		k        int
		interval float64
	}
	for i, testTask := range testTasks {
		task := testTask.task
		taskStats := stats[task.id]
		if taskStats == nil {
			fmt.Fprintf(errBuf, "\n task %s: missing stats", task.id)
			continue
		}
		pct := tc.scheduleIntervalPct / 100.
		intervalSec := task.interval.Seconds()
		minIntervalSec := (1 - pct) * intervalSec
		maxIntervalSec := (1 + pct) * intervalSec

		invokeTss := testTask.invokeTss
		// timestamp#0 -> #1 may be irregular, but everything #(k-1) -> #k, k >=
		// 2, should be checked:
		irregularIntervals := make([]*IrregularInterval, 0)
		for k := 2; k < len(invokeTss); k++ {
			gotIntervalSec := invokeTss[k].Sub(invokeTss[k-1]).Seconds()
			if gotIntervalSec < minIntervalSec || maxIntervalSec < gotIntervalSec {
				irregularIntervals = append(
					irregularIntervals,
					&IrregularInterval{k, gotIntervalSec},
				)
			}
		}
		wantIrregularIntervalMaxCount := 0
		if tc.wantIrregularIntervalMaxCount != nil {
			wantIrregularIntervalMaxCount = tc.wantIrregularIntervalMaxCount[i]
		}
		if len(irregularIntervals) > wantIrregularIntervalMaxCount {
			for _, irregularInterval := range irregularIntervals {
				fmt.Fprintf(
					errBuf,
					"\ntask %s execute# %d: want: %.06f..%.06f, got: %.06f sec from previous execution",
					task.id, irregularInterval.k,
					minIntervalSec, maxIntervalSec, irregularInterval.interval,
				)
			}
		}
		if taskStats.Uint64Stats[TASK_STATS_OVERRUN_COUNT] > uint64(wantIrregularIntervalMaxCount) {
			fmt.Fprintf(
				errBuf,
				"\ntask %s TASK_STATS_OVERRUN_COUNT: want max: %d, got: %d",
				task.id, wantIrregularIntervalMaxCount,
				taskStats.Uint64Stats[TASK_STATS_OVERRUN_COUNT],
			)
		}

	}

	if errBuf.Len() > 0 {
		t.Fatal(errBuf)
	}

}

func TestSchedulerExecute(t *testing.T) {
	scheduleIntervalPct := 20.

	for _, tc := range []*SchedulerExecuteTestCase{
		{
			numWorkers:  1,
			timeUnitSec: .1,
			intervals: []float64{
				1,
			},
			runTime:             40,
			scheduleIntervalPct: scheduleIntervalPct,
		},
		{
			numWorkers:  1,
			timeUnitSec: .1,
			intervals: []float64{
				4, 7, 3, 5, 1,
			},
			runTime:             43,
			scheduleIntervalPct: scheduleIntervalPct,
		},
		{
			numWorkers:  5,
			timeUnitSec: .1,
			intervals: []float64{
				4, 7, 3, 5, 1,
			},
			runTime:             43,
			scheduleIntervalPct: scheduleIntervalPct,
		},
		{
			numWorkers:  5,
			timeUnitSec: .1,
			intervals: []float64{
				4,
				7,
				3,
				5,
				1,
			},
			execDurations: [][]float64{
				{3},
				{6},
				{2},
				{4},
				nil,
			},
			runTime:             43,
			scheduleIntervalPct: scheduleIntervalPct,
		},
		{
			numWorkers:  5,
			timeUnitSec: .1,
			intervals: []float64{
				4,
				7,
				3,
				5,
				1,
			},
			execDurations: [][]float64{
				{3, 5, 3, 3, 3, 3},
				{6},
				{2},
				{4},
				nil,
			},
			runTime:             43,
			scheduleIntervalPct: scheduleIntervalPct,
			wantIrregularIntervalMaxCount: []int{
				2,
				0,
				0,
				0,
				0,
			},
		},
	} {
		t.Run(
			"",
			func(t *testing.T) {
				testSchedulerExecute(tc, t)
			},
		)
	}
}
