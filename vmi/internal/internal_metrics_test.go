// Common definitions for internal metrics tests

package vmi_internal

import (
	"bytes"
	"fmt"
	"maps"
	"path"
	"testing"
	"time"

	vmi_testutils "github.com/bgp59/victoriametrics-importer/vmi/testutils"
)

type InternalMetricsTestCase struct {
	Name                string
	Description         string
	Instance            string
	Hostname            string
	CycleNum            int
	PromTs              int64
	PrevPromTs          *int64
	Version             string
	GitInfo             string
	BootTimeMsec        int64
	StartTimeMsec       int64
	OsInfo              map[string]string
	OsRelease           map[string]string
	WantMetrics         []string
	BatchTargetSizeList []int
	batchTargetSize     int
}

var internalMetricsTestCasesFile = path.Join(
	VmiTestCasesSubdir,
	"internal_metrics", "internal.json",
)

func newTestInternalMetrics(tc *InternalMetricsTestCase) (*InternalMetrics, error) {
	internalMetrics, err := NewInternalMetrics(nil)
	if err != nil {
		return nil, err
	}

	internalMetrics.Instance = tc.Instance
	internalMetrics.Hostname = tc.Hostname
	timeNowRetVal := time.UnixMilli(tc.PromTs)
	internalMetrics.TimeNowFunc = func() time.Time { return timeNowRetVal }
	internalMetrics.MetricsQueue = vmi_testutils.NewTestMetricsQueue(tc.batchTargetSize)
	internalMetrics.TestMode = true

	internalMetrics.version = tc.Version
	internalMetrics.gitInfo = tc.GitInfo
	bootTime := time.UnixMilli(tc.BootTimeMsec)
	internalMetrics.bootTime = &bootTime
	startTs := time.UnixMilli(tc.StartTimeMsec)
	internalMetrics.startTs = &startTs
	internalMetrics.osInfo = maps.Clone(tc.OsInfo)
	internalMetrics.osRelease = maps.Clone(tc.OsRelease)
	if tc.PrevPromTs != nil || tc.CycleNum > 0 {
		internalMetrics.initialize()
		internalMetrics.CycleNum = tc.CycleNum
		internalMetrics.GenBaseMetricsStart(nil, time.UnixMilli(*tc.PrevPromTs))
	}
	return internalMetrics, nil
}

func newTestInternalMetricsTsInit(tc *InternalMetricsTestCase) (*InternalMetrics, error) {
	internalMetrics, err := newTestInternalMetrics(tc)
	if err != nil {
		return nil, err
	}
	if !internalMetrics.Initialized {
		internalMetrics.initialize()
	}
	internalMetrics.GenBaseMetricsStart(nil, time.UnixMilli(tc.PromTs))
	return internalMetrics, nil
}

func testInternalMetrics(tc *InternalMetricsTestCase, t *testing.T) {
	tlc := vmi_testutils.NewTestCollectableLogger(t, RootLogger, nil)
	defer tlc.RestoreLog()

	t.Logf("Description: %s", tc.Description)

	internalMetrics, err := newTestInternalMetrics(tc)
	if err != nil {
		t.Fatal(err)
	}

	if !internalMetrics.TaskAction() {
		t.Fatal("TaskAction() returned false, expected true")
	}

	errBuf := &bytes.Buffer{}

	vmi_testutils.ValidateWantMetrics(
		tc.WantMetrics,
		METRICS_GENERATOR_METRICS_DELTA_METRIC,
		METRICS_GENERATOR_BYTE_DELTA_METRIC,
		errBuf,
	)

	testMetricsQueue := internalMetrics.MetricsQueue.(*vmi_testutils.TestMetricsQueue)
	testMetricsQueue.GenerateReport(tc.WantMetrics, true, errBuf)

	if errBuf.Len() > 0 {
		t.Fatal(errBuf)
	}
}

func TestInternalMetrics(t *testing.T) {
	t.Logf("Loading test cases from: %s", internalMetricsTestCasesFile)

	testCases := make([]*InternalMetricsTestCase, 0)
	err := vmi_testutils.LoadJsonFile(internalMetricsTestCasesFile, &testCases)
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
				func(t *testing.T) { testInternalMetrics(tc, t) },
			)
		}
	}
}
