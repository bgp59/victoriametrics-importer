// Tests for HTTP endpoint pool internal metrics
package vmi_internal

import (
	"bytes"
	"fmt"
	"path"
	"testing"

	vmi_testutils "github.com/bgp59/victoriametrics-importer/vmi/testutils"
)

type HttpEndpointPoolInternalMetricsTestCase struct {
	InternalMetricsTestCase
	CurrStats, PrevStats *HttpEndpointPoolStats
}

var httpEndpointPoolInternalMetricsTestCasesFile = path.Join(
	VmiTestCasesSubdir,
	"internal_metrics", "http_endpoint_pool.json",
)

func newTestHttpEndpointPoolInternalMetrics(tc *HttpEndpointPoolInternalMetricsTestCase) (*InternalMetrics, error) {
	internalMetrics, err := newTestInternalMetricsTsInit(&tc.InternalMetricsTestCase)
	if err != nil {
		return nil, err
	}
	httpEndpointPoolInternalMetrics := NewHttpEndpointPoolInternalMetrics(internalMetrics)
	httpEndpointPoolInternalMetrics.stats[httpEndpointPoolInternalMetrics.currIndex] = tc.CurrStats
	httpEndpointPoolInternalMetrics.stats[1-httpEndpointPoolInternalMetrics.currIndex] = tc.PrevStats
	internalMetrics.httpEndpointPoolMetrics = httpEndpointPoolInternalMetrics
	return internalMetrics, nil
}

func testHttpEndpointPoolInternalMetrics(tc *HttpEndpointPoolInternalMetricsTestCase, t *testing.T) {
	tlc := vmi_testutils.NewTestLogCollect(t, RootLogger, nil)
	defer tlc.RestoreLog()

	t.Logf("Description: %s", tc.Description)

	internalMetrics, err := newTestHttpEndpointPoolInternalMetrics(tc)
	if err != nil {
		t.Fatal(err)
	}

	testMetricsQueue := internalMetrics.MetricsQueue.(*vmi_testutils.TestMetricsQueue)
	buf := testMetricsQueue.GetBuf()

	httpEndpointPoolInternalMetrics := internalMetrics.httpEndpointPoolMetrics
	wantCurrIndex := 1 - httpEndpointPoolInternalMetrics.currIndex

	gotMetricsCount, _, buf := httpEndpointPoolInternalMetrics.generateMetrics(buf, internalMetrics.TsSuffixBuf.Bytes())
	if buf != nil {
		testMetricsQueue.QueueBuf(buf)
	}

	errBuf := &bytes.Buffer{}

	gotCurrIndex := httpEndpointPoolInternalMetrics.currIndex
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

func TestHttpEndpointPoolInternalMetrics(t *testing.T) {
	t.Logf("Loading test cases from %q ...", httpEndpointPoolInternalMetricsTestCasesFile)
	testCases := make([]*HttpEndpointPoolInternalMetricsTestCase, 0)
	err := vmi_testutils.LoadJsonFile(httpEndpointPoolInternalMetricsTestCasesFile, &testCases)
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
				func(t *testing.T) { testHttpEndpointPoolInternalMetrics(tc, t) },
			)
		}
	}
}
