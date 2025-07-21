package vmi_internal

import (
	"bytes"
	"fmt"
	"maps"
	"path"
	"testing"

	vmi_testutils "github.com/bgp59/victoriametrics-importer/vmi/testutils"
)

type GeneratorInternalMetricsTestCase struct {
	CurrGeneratorStats, PrevGeneratorStats MetricsGeneratorStats
	InternalMetricsTestCase
}

var generatorInternalMetricsTestCasesFile = path.Join(
	VmiTestCasesSubdir,
	"internal_metrics", "generator.json",
)

func newGeneratorTestInternalMetrics(tc *GeneratorInternalMetricsTestCase) (*InternalMetrics, error) {
	internalMetrics, err := newTestInternalMetricsTsInit(&tc.InternalMetricsTestCase)
	if err != nil {
		return nil, err
	}
	gim := NewGeneratorInternalMetrics(internalMetrics)
	gim.generatorStats[gim.currIndex] = maps.Clone(tc.CurrGeneratorStats)
	gim.generatorStats[1-gim.currIndex] = maps.Clone(tc.PrevGeneratorStats)
	internalMetrics.generatorMetrics = gim
	return internalMetrics, nil
}

func testGeneratorInternalMetrics(tc *GeneratorInternalMetricsTestCase, t *testing.T) {
	tlc := vmi_testutils.NewTestLogCollect(t, RootLogger, nil)
	defer tlc.RestoreLog()

	t.Logf("Description: %s", tc.Description)

	internalMetrics, err := newGeneratorTestInternalMetrics(tc)
	if err != nil {
		t.Fatal(err)
	}

	testMetricsQueue := internalMetrics.MetricsQueue.(*vmi_testutils.TestMetricsQueue)
	buf := testMetricsQueue.GetBuf()

	gim := internalMetrics.generatorMetrics
	wantCurrIndex := 1 - gim.currIndex

	gotMetricsCount, _, buf := gim.generateMetrics(buf, internalMetrics.TsSuffixBuf.Bytes())
	if buf != nil {
		testMetricsQueue.QueueBuf(buf)
	}

	errBuf := &bytes.Buffer{}

	gotCurrIndex := gim.currIndex
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

func TestGeneratorInternalMetrics(t *testing.T) {
	t.Logf("Loading test cases from %q ...", generatorInternalMetricsTestCasesFile)
	testCases := make([]*GeneratorInternalMetricsTestCase, 0)
	err := vmi_testutils.LoadJsonFile(generatorInternalMetricsTestCasesFile, &testCases)
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
				func(t *testing.T) { testGeneratorInternalMetrics(tc, t) },
			)
		}
	}
}
