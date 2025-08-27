package vmi_internal

import (
	"bytes"
	"fmt"
	"path"
	"testing"
	"time"

	vmi_testutils "github.com/bgp59/victoriametrics-importer/vmi/testutils"
)

type ProcessInternalMetricsTestCase struct {
	CurrCpuTime, PrevCpuTime float64
	InternalMetricsTestCase
}

var processInternalMetricsTestCasesFile = path.Join(
	VmiTestCasesSubdir,
	"internal_metrics", "process.json",
)

func newTestProcessInternalMetrics(tc *ProcessInternalMetricsTestCase) (*InternalMetrics, error) {
	internalMetrics, err := newTestInternalMetricsTsInit(&tc.InternalMetricsTestCase)
	if err != nil {
		return nil, err
	}
	pim := NewProcessInternalMetrics(internalMetrics)
	pim.cpuTime[pim.currIndex] = tc.CurrCpuTime
	pim.cpuTime[1-pim.currIndex] = tc.PrevCpuTime
	pim.statsTs[pim.currIndex] = time.UnixMilli(tc.PromTs)
	if tc.PrevPromTs != nil {
		pim.statsTs[1-pim.currIndex] = time.UnixMilli(*tc.PrevPromTs)
	}
	internalMetrics.processMetrics = pim
	return internalMetrics, nil
}

func testProcessInternalMetrics(tc *ProcessInternalMetricsTestCase, t *testing.T) {
	tlc := vmi_testutils.NewTestCollectableLogger(t, RootLogger, nil)
	defer tlc.RestoreLog()

	t.Logf("Description: %s", tc.Description)

	internalMetrics, err := newTestProcessInternalMetrics(tc)
	if err != nil {
		t.Fatal(err)
	}

	testMetricsQueue := internalMetrics.MetricsQueue.(*vmi_testutils.TestMetricsQueue)
	buf := testMetricsQueue.GetBuf()

	pim := internalMetrics.processMetrics
	wantCurrIndex := 1 - pim.currIndex

	gotMetricsCount, _, buf := pim.generateMetrics(buf, internalMetrics.TsSuffixBuf.Bytes())
	if buf != nil {
		testMetricsQueue.QueueBuf(buf)
	}

	errBuf := &bytes.Buffer{}

	gotCurrIndex := pim.currIndex
	if gotCurrIndex != wantCurrIndex {
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

func TestProcessInternalMetrics(t *testing.T) {
	t.Logf("Loading test cases from %q ...", processInternalMetricsTestCasesFile)
	testCases := make([]*ProcessInternalMetricsTestCase, 0)
	err := vmi_testutils.LoadJsonFile(processInternalMetricsTestCasesFile, &testCases)
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
				func(t *testing.T) { testProcessInternalMetrics(tc, t) },
			)
		}
	}
}
