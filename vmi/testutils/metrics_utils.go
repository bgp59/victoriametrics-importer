// Utils for metrics testing:

package vmi_testutils

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// A test metrics queue which collects and indexes metrics:
type TestMetricsQueue struct {
	metrics         map[string]int
	batchTargetSize int
}

func NewTestMetricsQueue(batchTargetSize int) *TestMetricsQueue {
	return &TestMetricsQueue{
		metrics:         make(map[string]int, 0),
		batchTargetSize: batchTargetSize,
	}
}

// The BufferQueue interface:
func (mq *TestMetricsQueue) GetBuf() *bytes.Buffer {
	return &bytes.Buffer{}
}

func (mq *TestMetricsQueue) ReturnBuf(buf *bytes.Buffer) {
}

func (mq *TestMetricsQueue) QueueBuf(buf *bytes.Buffer) {
	if buf == nil || buf.Len() == 0 {
		return
	}
	for _, metric := range strings.Split(buf.String(), "\n") {
		metric = strings.TrimSpace(metric)
		if metric != "" {
			mq.metrics[metric] += 1
		}
	}
}

func (mq *TestMetricsQueue) GetTargetSize() int {
	return mq.batchTargetSize
}

func (mq *TestMetricsQueue) GenerateReport(wantMetrics []string, reportExtra bool, errBuf *bytes.Buffer) *bytes.Buffer {
	if errBuf == nil {
		errBuf = &bytes.Buffer{}
	}

	foundMetrics := make(map[string]bool)
	for _, wantMetric := range wantMetrics {
		wantMetric = strings.TrimSpace(wantMetric)
		if mq.metrics[wantMetric] == 0 {
			fmt.Fprintf(errBuf, "\nmissing metric: %s", wantMetric)
		} else {
			foundMetrics[wantMetric] = true
		}
	}

	if reportExtra {
		for gotMetric, count := range mq.metrics {
			if !foundMetrics[gotMetric] {
				fmt.Fprintf(errBuf, "\nunexpected metric: %s", gotMetric)
			}
			if count > 1 {
				fmt.Fprintf(errBuf, "\nmetric: %s: count: %d > 1", gotMetric, count)
			}
		}
	}
	return errBuf
}

func extractCount(metric string) (int, error) {
	fields := strings.Fields(metric)
	if len(fields) < 3 {
		return -1, fmt.Errorf("invalid metric format: %s", metric)
	}
	countStr := fields[len(fields)-2]
	count, err := strconv.Atoi(countStr)
	if err != nil {
		return -1, fmt.Errorf("failed to parse count from metric: %s, error: %v", metric, err)
	}
	return count, nil
}

func ValidateWantMetrics(wantMetrics []string, metricCountMetric, byteCountMetric string, errBuf *bytes.Buffer) *bytes.Buffer {
	if errBuf == nil {
		errBuf = &bytes.Buffer{}
	}

	wantMetricCount, gotMetricCount := len(wantMetrics), -1
	wantByteCount, gotByteCount := 0, -1
	for _, metric := range wantMetrics {
		var err error
		wantByteCount += len(metric) + 1 // +1 for the newline character
		if metricCountMetric != "" && strings.HasPrefix(metric, metricCountMetric+"{") {
			gotMetricCount, err = extractCount(metric)
			if err != nil {
				fmt.Fprintf(errBuf, "\n%s", err)
			}
			continue
		}
		if byteCountMetric != "" && strings.HasPrefix(metric, byteCountMetric+"{") {
			gotByteCount, err = extractCount(metric)
			if err != nil {
				fmt.Fprintf(errBuf, "\n%s", err)
			}
			continue
		}
	}

	if gotMetricCount >= 0 && gotMetricCount != wantMetricCount {
		fmt.Fprintf(errBuf, "\nmetric count mismatch: want %d, got %d", wantMetricCount, gotMetricCount)
	}
	if gotByteCount >= 0 && gotByteCount != wantByteCount {
		fmt.Fprintf(errBuf, "\nbyte count mismatch: want %d, got %d", wantByteCount, gotByteCount)
	}
	return errBuf
}
