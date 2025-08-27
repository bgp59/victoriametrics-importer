// Tests for compressor pool internal metrics

package vmi_internal

import (
	"bytes"
	"fmt"
	"path"
	"testing"

	vmi_testutils "github.com/bgp59/victoriametrics-importer/vmi/testutils"
)

type CompressorPoolInternalMetricsTestCase struct {
	InternalMetricsTestCase
	CurrStats, PrevStats CompressorPoolStats
}

var compressorPoolInternalMetricsTestCasesFile = path.Join(
	VmiTestCasesSubdir,
	"internal_metrics", "compressor_pool.json",
)

func newTestCompressorPoolInternalMetrics(tc *CompressorPoolInternalMetricsTestCase) (*InternalMetrics, error) {
	internalMetrics, err := newTestInternalMetricsTsInit(&tc.InternalMetricsTestCase)
	if err != nil {
		return nil, err
	}
	compressorPoolInternalMetrics := NewCompressorPoolInternalMetrics(internalMetrics)
	compressorPoolInternalMetrics.stats[compressorPoolInternalMetrics.currIndex] = tc.CurrStats
	compressorPoolInternalMetrics.stats[1-compressorPoolInternalMetrics.currIndex] = tc.PrevStats
	internalMetrics.compressorPoolMetrics = compressorPoolInternalMetrics
	return internalMetrics, nil
}

func testCompressorPoolInternalMetrics(tc *CompressorPoolInternalMetricsTestCase, t *testing.T) {
	tlc := vmi_testutils.NewTestCollectableLogger(t, RootLogger, nil)
	defer tlc.RestoreLog()

	t.Logf("Description: %s", tc.Description)

	internalMetrics, err := newTestCompressorPoolInternalMetrics(tc)
	if err != nil {
		t.Fatal(err)
	}

	testMetricsQueue := internalMetrics.MetricsQueue.(*vmi_testutils.TestMetricsQueue)
	buf := testMetricsQueue.GetBuf()

	compressorPoolInternalMetrics := internalMetrics.compressorPoolMetrics
	wantCurrIndex := 1 - compressorPoolInternalMetrics.currIndex

	gotMetricsCount, _, buf := compressorPoolInternalMetrics.generateMetrics(buf, internalMetrics.TsSuffixBuf.Bytes())
	if buf != nil {
		testMetricsQueue.QueueBuf(buf)
	}

	errBuf := &bytes.Buffer{}

	gotCurrIndex := compressorPoolInternalMetrics.currIndex
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

func TestCompressorPoolInternalMetrics(t *testing.T) {
	t.Logf("Loading test cases from %q ...", compressorPoolInternalMetricsTestCasesFile)
	testCases := make([]*CompressorPoolInternalMetricsTestCase, 0)
	err := vmi_testutils.LoadJsonFile(compressorPoolInternalMetricsTestCasesFile, &testCases)
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
				func(t *testing.T) { testCompressorPoolInternalMetrics(tc, t) },
			)
		}
	}
}
