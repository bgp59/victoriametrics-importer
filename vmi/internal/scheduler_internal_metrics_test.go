// Tests for scheduler internal metrics

package vmi_internal

import (
	"bytes"
	"fmt"
	"path"
	"testing"

	vmi_testutils "github.com/bgp59/victoriametrics-importer/vmi/testutils"
)

type SchedulerInternalMetricsTestCase struct {
	InternalMetricsTestCase
	CurrStats, PrevStats SchedulerStats
}

var schedulerInternalMetricsTestCasesFile = path.Join(
	VmiTestCasesSubdir,
	"internal_metrics", "scheduler.json",
)

func newTestSchedulerInternalMetrics(tc *SchedulerInternalMetricsTestCase) (*InternalMetrics, error) {
	internalMetrics, err := newTestInternalMetricsTsInit(&tc.InternalMetricsTestCase)
	if err != nil {
		return nil, err
	}
	schedulerInternalMetrics := NewSchedulerInternalMetrics(internalMetrics)
	schedulerInternalMetrics.stats[schedulerInternalMetrics.currIndex] = tc.CurrStats
	schedulerInternalMetrics.stats[1-schedulerInternalMetrics.currIndex] = tc.PrevStats
	internalMetrics.schedulerMetrics = schedulerInternalMetrics
	return internalMetrics, nil
}

func testSchedulerInternalMetrics(tc *SchedulerInternalMetricsTestCase, t *testing.T) {
	tlc := vmi_testutils.NewTestCollectableLogger(t, RootLogger, nil)
	defer tlc.RestoreLog()

	t.Logf("Description: %s", tc.Description)

	internalMetrics, err := newTestSchedulerInternalMetrics(tc)
	if err != nil {
		t.Fatal(err)
	}

	testMetricsQueue := internalMetrics.MetricsQueue.(*vmi_testutils.TestMetricsQueue)
	buf := testMetricsQueue.GetBuf()

	schedulerInternalMetrics := internalMetrics.schedulerMetrics
	wantCurrIndex := 1 - schedulerInternalMetrics.currIndex

	gotMetricsCount, _, buf := schedulerInternalMetrics.generateMetrics(buf, internalMetrics.TsSuffixBuf.Bytes())
	if buf != nil {
		testMetricsQueue.QueueBuf(buf)
	}

	errBuf := &bytes.Buffer{}

	gotCurrIndex := schedulerInternalMetrics.currIndex
	if wantCurrIndex != gotCurrIndex {
		fmt.Fprintf(
			errBuf,
			"\ncurrIndex: want: %d, got: %d",
			wantCurrIndex, gotCurrIndex,
		)
	}

	wantMetricsCount := len(tc.WantMetrics)
	if wantMetricsCount != gotMetricsCount {
		fmt.Fprintf(
			errBuf,
			"\nmetricsCount: want: %d, got: %d",
			wantMetricsCount, gotMetricsCount,
		)
	}

	testMetricsQueue.GenerateReport(tc.WantMetrics, true, errBuf)

	if errBuf.Len() > 0 {
		t.Fatal(errBuf)
	}
}

func TestSchedulerInternalMetrics(t *testing.T) {
	t.Logf("Loading test cases from %q ...", schedulerInternalMetricsTestCasesFile)
	testCases := make([]*SchedulerInternalMetricsTestCase, 0)
	err := vmi_testutils.LoadJsonFile(schedulerInternalMetricsTestCasesFile, &testCases)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range testCases {
		batchTargetSizeList := tc.BatchTargetSizeList
		if batchTargetSizeList == nil {
			batchTargetSizeList = []int{0}
		}
		for _, batchTargetSize := range batchTargetSizeList {
			tc.batchTargetSize = batchTargetSize
			t.Run(
				fmt.Sprintf("%s/bsz:%d", tc.Name, tc.batchTargetSize),
				func(t *testing.T) { testSchedulerInternalMetrics(tc, t) },
			)
		}
	}
}
