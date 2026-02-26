package vmi_internal

import (
	"bytes"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	vmi_testutils "github.com/bgp59/victoriametrics-importer/vmi/testutils"
)

type HttpEndpointPoolTestSendBuf struct {
	// The buffer to send:
	buf []byte
	// The indexes in the playbook where it is expected to appear, as a result of
	// the playback; ignored if wantErr is not nil:
	expectIndexes []int
	// The expected errorL
	wantError error
}

type HttpClientDoerPlaybackResult struct {
	results []*vmi_testutils.HttpClientDoerPlaybackRequest
	err     error
}

type HttpEndpointPoolTestCase struct {
	epCfgs   []*HttpEndpointConfig
	playbook []*vmi_testutils.HttpClientDoerPlaybackEntry
	sendBufs []*HttpEndpointPoolTestSendBuf
}

func buildTestHttpEndpointPool(tc *HttpEndpointPoolTestCase) (*HttpEndpointPool, error) {
	epPoolCfg := DefaultHttpEndpointPoolConfig()
	epPoolCfg.Endpoints = tc.epCfgs
	epPool, err := NewHttpEndpointPool(epPoolCfg)
	return epPool, err
}

func testHttpEndpointPoolCreate(tc *HttpEndpointPoolTestCase, t *testing.T) {
	tlc := vmi_testutils.NewTestCollectableLogger(t, RootLogger, nil)
	defer tlc.RestoreLog()

	epPool, err := buildTestHttpEndpointPool(tc)
	if err != nil {
		t.Fatal(err)
	}
	epPool.healthyRotateInterval = -1 // Ensure it is disabled
	defer epPool.Shutdown()

	i := 0
	for ep := epPool.healthy.head; ep != nil && i < len(tc.epCfgs); ep = ep.next {
		wantUrl := tc.epCfgs[i].URL
		if wantUrl != ep.url {
			t.Fatalf("ep#%d url: want: %q, got: %q", i, wantUrl, ep.url)
		}
		i++
	}
	if len(tc.epCfgs) != i {
		t.Fatalf("len(healthy): want: %d, got: %d", len(tc.epCfgs), i)
	}
}

func testHttpEndpointPoolRotate(tc *HttpEndpointPoolTestCase, t *testing.T) {
	tlc := vmi_testutils.NewTestCollectableLogger(t, RootLogger, logrus.DebugLevel)
	defer tlc.RestoreLog()

	epPool, err := buildTestHttpEndpointPool(tc)
	if err != nil {
		t.Fatal(err)
	}
	epPool.healthyRotateInterval = 0 // Ensure rotate w/ every call
	defer epPool.Shutdown()

	for i := 0; i < len(tc.epCfgs)*4/3; i++ {
		wantUrl := tc.epCfgs[i%len(tc.epCfgs)].URL
		ep := epPool.GetCurrentHealthy(0)
		if ep == nil {
			t.Fatalf("GetCurrentHealthy: want: %s, got: %v", wantUrl, nil)
		} else if wantUrl != ep.url {
			t.Fatalf("GetCurrentHealthy: want: %s, got: %s", wantUrl, ep.url)
		}
	}
}

func testHttpEndpointPoolReportError(tc *HttpEndpointPoolTestCase, t *testing.T) {
	testTimeout := 5 * time.Second

	tlc := vmi_testutils.NewTestCollectableLogger(t, RootLogger, logrus.DebugLevel)
	defer tlc.RestoreLog()

	epPool, err := buildTestHttpEndpointPool(tc)
	if err != nil {
		t.Fatal(err)

	}
	defer epPool.Shutdown()
	// Ensure rotate w/ every call
	epPool.healthyRotateInterval = 0
	// Ensure that the health check will proceed right away, since it is paced
	// by the ClientDoer mock:
	epPool.healthCheckInterval = 1 * time.Nanosecond // time.Ticker requires > 0

	mock := vmi_testutils.NewHttpClientDoerMock(testTimeout)
	defer mock.Cancel()
	epPool.client = mock

	// Run until each endpoint has been found N times error free at healthy
	// head:
	rotateN := 2
	pendingEP := make(map[*HttpEndpoint]int)
	for ep := epPool.healthy.head; ep != nil; ep = ep.next {
		pendingEP[ep] = rotateN
	}

	for len(pendingEP) > 0 {
		ep := epPool.GetCurrentHealthy(testTimeout)
		if ep == nil {
			t.Fatal(ErrHttpEndpointPoolNoHealthyEP)
		}
		if ep.numErrors == 0 {
			if _, ok := pendingEP[ep]; ok {
				pendingEP[ep] -= 1
				if pendingEP[ep] <= 0 {
					delete(pendingEP, ep)
				}
			}
		}
		epPool.ReportError(ep)
		if !ep.healthy {
			_, err = mock.GetRequest(ep.url)
			if err != nil {
				t.Fatal(err)
			}
			err = mock.SendResponse(ep.url, &http.Response{StatusCode: http.StatusOK}, nil)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}

func testHttpEndpointPoolSendBuf(tc *HttpEndpointPoolTestCase, t *testing.T) {
	testTimeout := 5 * time.Second

	tlc := vmi_testutils.NewTestCollectableLogger(t, RootLogger, logrus.DebugLevel)
	defer tlc.RestoreLog()

	epPool, err := buildTestHttpEndpointPool(tc)
	if err != nil {
		t.Fatal(err)

	}
	defer epPool.Shutdown()
	// Disable rotate:
	epPool.healthyRotateInterval = -1
	// Ensure that the health check will proceed right away, since it is paced
	// by the ClientDoer mock:
	epPool.healthCheckInterval = 1 * time.Nanosecond // time.Ticker requires > 0

	mock := vmi_testutils.NewHttpClientDoerMock(testTimeout)
	defer mock.Cancel()
	epPool.client = mock

	// Start the playback in the background and retrieve the exit code from a
	// channel:
	pbRetChan := make(chan *HttpClientDoerPlaybackResult, 1)
	go func() {
		results, err := mock.Play(tc.playbook)
		pbRetChan <- &HttpClientDoerPlaybackResult{results, err}
	}()

	// Send the buffers and collect the error status:
	gotErrors := make([]error, len(tc.sendBufs))
	for i, sendBuf := range tc.sendBufs {
		gotErrors[i] = epPool.SendBuffer(sendBuf.buf, testTimeout, false)
	}

	// Collect and verify the playback exit status:
	pbResultsErr := <-pbRetChan
	if pbResultsErr.err != nil {
		t.Fatal(pbResultsErr.err)
	}

	// Verify the status of the sent data:
	results := pbResultsErr.results
	for i, sendBuf := range tc.sendBufs {
		wantError, gotError := sendBuf.wantError, gotErrors[i]
		if !errors.Is(gotError, wantError) {
			t.Fatalf(
				"sendBuf[%d] error: want: %v, got: %v",
				i, wantError, gotError,
			)
		}
		if wantError == nil {
			wantBuf := sendBuf.buf
			for _, j := range sendBuf.expectIndexes {
				gotBuf := results[j].Body
				if !bytes.Equal(wantBuf, gotBuf) {
					t.Fatalf(
						"sendBuf[%d]:\n\twant:\n\t\t%q\n\t\t%v\n\tgot[pb#%d]:\n\t\t%q\n\t\t%v",
						i, wantBuf, wantBuf,
						j, gotBuf, gotBuf,
					)
				}
			}
		}
	}
}

func TestHttpEndpointPoolCreate(t *testing.T) {
	for _, tc := range []*HttpEndpointPoolTestCase{
		{
			epCfgs: []*HttpEndpointConfig{
				{"http://host1", 1},
			},
		},
		{
			epCfgs: []*HttpEndpointConfig{
				{"http://host1", 1},
				{"http://host2", 1},
			},
		},
	} {
		t.Run(
			"",
			func(t *testing.T) { testHttpEndpointPoolCreate(tc, t) },
		)
	}
}

func TestHttpEndpointPoolRotate(t *testing.T) {
	for _, tc := range []*HttpEndpointPoolTestCase{
		{
			epCfgs: []*HttpEndpointConfig{
				{"http://host1", 1},
			},
		},
		{
			epCfgs: []*HttpEndpointConfig{
				{"http://host1", 1},
				{"http://host2", 1},
				{"http://host3", 1},
				{"http://host4", 1},
			},
		},
	} {
		t.Run(
			"",
			func(t *testing.T) { testHttpEndpointPoolRotate(tc, t) },
		)
	}
}

func TestHttpEndpointPoolReportError(t *testing.T) {
	for _, tc := range []*HttpEndpointPoolTestCase{
		{
			epCfgs: []*HttpEndpointConfig{
				{"http://host1", 1},
			},
		},
		{
			epCfgs: []*HttpEndpointConfig{
				{"http://host1", 1},
				{"http://host2", 2},
				{"http://host3", 3},
				{"http://host4", 4},
			},
		},
	} {
		t.Run(
			"",
			func(t *testing.T) { testHttpEndpointPoolReportError(tc, t) },
		)
	}
}

func TestHttpEndpointPoolSendBuf(t *testing.T) {
	for _, tc := range []*HttpEndpointPoolTestCase{
		/////////////////////////////////////////////////////////////////////////////////////////
		{
			epCfgs: []*HttpEndpointConfig{
				{"http://host1", 1},
			},
			playbook: []*vmi_testutils.HttpClientDoerPlaybackEntry{
				{
					Url:      "http://host1",
					Response: &http.Response{StatusCode: http.StatusOK},
				},
			},
			sendBufs: []*HttpEndpointPoolTestSendBuf{
				{
					buf:           []byte("0-000000"),
					expectIndexes: []int{0},
				},
			},
		},
		/////////////////////////////////////////////////////////////////////////////////////////
		{
			epCfgs: []*HttpEndpointConfig{
				{"http://host1", 1},
				{"http://host2", 1},
			},
			playbook: []*vmi_testutils.HttpClientDoerPlaybackEntry{
				{
					Url:   "http://host1",
					Error: vmi_testutils.ErrHttpClientDoerMockGeneric,
				},
				{
					Url:      "http://host2",
					Response: &http.Response{StatusCode: http.StatusOK},
				},
				{
					Url:      "http://host1",
					Response: &http.Response{StatusCode: http.StatusOK},
				},
			},
			sendBufs: []*HttpEndpointPoolTestSendBuf{
				{
					buf:           []byte("1-000000"),
					expectIndexes: []int{0, 1},
				},
			},
		},
		/////////////////////////////////////////////////////////////////////////////////////////
		{
			epCfgs: []*HttpEndpointConfig{
				{"http://host1", 2},
				{"http://host2", 1},
			},
			playbook: []*vmi_testutils.HttpClientDoerPlaybackEntry{
				{
					Url:   "http://host1",
					Error: vmi_testutils.ErrHttpClientDoerMockGeneric,
				},
				{
					Url:   "http://host2",
					Error: vmi_testutils.ErrHttpClientDoerMockGeneric,
				},
				{
					Url:      "http://host2",
					Response: &http.Response{StatusCode: http.StatusOK},
				},
				{
					Url:      "http://host1",
					Response: &http.Response{StatusCode: http.StatusOK},
				},
			},
			sendBufs: []*HttpEndpointPoolTestSendBuf{
				{
					buf:           []byte("2-000000"),
					expectIndexes: []int{0, 1, 3},
				},
			},
		},
		/////////////////////////////////////////////////////////////////////////////////////////
		{
			epCfgs: []*HttpEndpointConfig{
				{"http://host1", 2},
				{"http://host2", 1},
			},
			playbook: []*vmi_testutils.HttpClientDoerPlaybackEntry{
				{
					Url:   "http://host1",
					Error: vmi_testutils.ErrHttpClientDoerMockGeneric,
				},
				{
					Url:   "http://host2",
					Error: vmi_testutils.ErrHttpClientDoerMockGeneric,
				},
				{
					Url:      "http://host2",
					Response: &http.Response{StatusCode: http.StatusOK},
				},
				{
					Url:      "http://host1",
					Response: &http.Response{StatusCode: http.StatusOK},
				},
				{
					Url:      "http://host1",
					Response: &http.Response{StatusCode: http.StatusOK},
				},
			},
			sendBufs: []*HttpEndpointPoolTestSendBuf{
				{
					buf:           []byte("3-000000"),
					expectIndexes: []int{0, 1, 3},
				},
				{
					buf:           []byte("3-000001"),
					expectIndexes: []int{4},
				},
			},
		},
		/////////////////////////////////////////////////////////////////////////////////////////
	} {
		t.Run(
			"",
			func(t *testing.T) { testHttpEndpointPoolSendBuf(tc, t) },
		)
	}
}
