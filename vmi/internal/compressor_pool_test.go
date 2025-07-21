// Unit tests for compressor_pool.go

package vmi_internal

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	vmi_testutils "github.com/bgp59/victoriametrics-importer/vmi/testutils"
)

type CompressorPoolTestCase struct {
	// CompressorPoolConfig overrides, they are applied if non nil; they should
	// be supplied with the expected type for the fields:
	NumCompressors   any
	CompressLevel    any
	BatchTargetSize  any
	FlushInterval    any
	numQueuedBuffers int
	wantError        error
}

type SenderMock struct {
	bufs [][]byte
	mu   *sync.Mutex
}

var compressorUint64StatsNames = []string{
	"COMPRESSOR_STATS_READ_COUNT",
	"COMPRESSOR_STATS_READ_BYTE_COUNT",
	"COMPRESSOR_STATS_SEND_COUNT",
	"COMPRESSOR_STATS_SEND_BYTE_COUNT",
	"COMPRESSOR_STATS_TIMEOUT_FLUSH_COUNT",
	"COMPRESSOR_STATS_SEND_ERROR_COUNT",
	"COMPRESSOR_STATS_WRITE_ERROR_COUNT",
}

var compressorFloat64StatsNames = []string{
	"COMPRESSOR_STATS_COMPRESSION_FACTOR",
}

func NewSenderMock() *SenderMock {
	return &SenderMock{
		bufs: make([][]byte, 0),
		mu:   &sync.Mutex{},
	}

}

func (sender *SenderMock) SendBuffer(b []byte, timeout time.Duration, gzipped bool) error {
	var buf []byte
	if gzipped {
		r, err := gzip.NewReader(bytes.NewBuffer(b))
		if err != nil {
			return fmt.Errorf("SenderMock: SendBuffer((%d bytes), ...)): gzip.NewReader: %v", len(b), err)
		}
		buf, err = io.ReadAll(r)
		if err != nil {
			return fmt.Errorf("SenderMock: SendBuffer((%d bytes), ...)): ReadAll: %v", len(b), err)
		}
		r.Close()
	} else {
		buf = make([]byte, len(b))
		copy(buf, b)
	}
	sender.mu.Lock()
	sender.bufs = append(sender.bufs, buf)
	sender.mu.Unlock()
	return nil
}

func (sender *SenderMock) MapLines() map[string]int {
	sender.mu.Lock()
	defer sender.mu.Unlock()

	lineMap := make(map[string]int)

	for _, buf := range sender.bufs {
		s, n := 0, len(buf)
		for i := 0; i < n; i++ {
			if buf[i] == '\n' {
				lineMap[string(buf[s:i])] += 1
				s = i + 1
			}
		}
		if s < n {
			lineMap[string(buf[s:n])] += 1
		}
	}
	return lineMap
}

func makeTestCompressorPool(tc *CompressorPoolTestCase) (*CompressorPool, error) {
	poolCfg := DefaultCompressorPoolConfig()
	if numCompressors, ok := tc.NumCompressors.(int); ok {
		poolCfg.NumCompressors = numCompressors
	}
	if batchTargetSize, ok := tc.BatchTargetSize.(string); ok {
		poolCfg.BatchTargetSize = batchTargetSize
	}
	if compressLevel, ok := tc.CompressLevel.(int); ok {
		poolCfg.CompressionLevel = compressLevel
	}
	if flushInterval, ok := tc.FlushInterval.(time.Duration); ok {
		poolCfg.FlushInterval = flushInterval
	}
	return NewCompressorPool(poolCfg)
}

func testCompressorPoolCreate(tc *CompressorPoolTestCase, t *testing.T) {
	tlc := vmi_testutils.NewTestLogCollect(t, RootLogger, logrus.DebugLevel)
	defer tlc.RestoreLog()

	pool, err := makeTestCompressorPool(tc)
	if err != nil && tc.wantError == nil ||
		err == nil && tc.wantError != nil ||
		err != nil && tc.wantError != nil && err.Error() != tc.wantError.Error() {
		t.Fatalf("error:\n\twant: %v\n\t got: %v", tc.wantError, err)
	} else if err != nil {
		return
	}
	pool.Start(nil)
	pool.Shutdown()
}

func testCompressorPoolQueue(tc *CompressorPoolTestCase, t *testing.T) {
	tlc := vmi_testutils.NewTestLogCollect(t, RootLogger, logrus.DebugLevel)
	defer tlc.RestoreLog()

	pool, err := makeTestCompressorPool(tc)
	if err != nil {
		t.Fatal(err)
	}
	sender := NewSenderMock()
	pool.Start(sender)
	poolStopped := false

	numQueuedBuffers := tc.numQueuedBuffers
	if numQueuedBuffers < 0 {
		numQueuedBuffers = 1
	}

	wantLineMap := make(map[string]int)

	wantReadByteCount := 0
	for bufIndx := 0; bufIndx < numQueuedBuffers; bufIndx++ {
		buf := pool.GetBuf()
		lineIndx := 0
		for buf.Len() < pool.batchTargetSize {
			line := fmt.Sprintf("line# %06d-%015d", bufIndx, lineIndx)
			wantLineMap[line] += 1
			buf.WriteString(line)
			buf.WriteByte('\n')
			wantReadByteCount += len(line) + 1
			lineIndx++
		}
		pool.QueueBuf(buf)
	}

	if pool.flushInterval > 0 {
		pause := pool.flushInterval + 200*time.Millisecond
		t.Logf(
			"Pause %s (> flushInterval=%s) to ensure that the last buffer is written",
			pause, pool.flushInterval,
		)
		time.Sleep(pause)
	} else {
		pool.Shutdown()
		poolStopped = true
	}

	t.Log("Collect stats")
	gotLineMap := sender.MapLines()

	if !poolStopped {
		pool.Shutdown()
	}

	poolStats := pool.SnapStats(nil)
	statsBuf := &bytes.Buffer{}
	fmt.Fprintf(statsBuf, "Compressor stats:")
	gotReadCount, gotReadByteCount := 0, 0
	for compressorId, compressorStats := range poolStats {
		fmt.Fprintf(statsBuf, "\ncompressor %s:", compressorId)
		for i, val := range compressorStats.Uint64Stats {
			fmt.Fprintf(statsBuf, "\n\t%s: %d", compressorUint64StatsNames[i], val)
		}
		gotReadCount += int(compressorStats.Uint64Stats[COMPRESSOR_STATS_READ_COUNT])
		gotReadByteCount += int(compressorStats.Uint64Stats[COMPRESSOR_STATS_READ_BYTE_COUNT])
		for i, val := range compressorStats.Float64Stats {
			fmt.Fprintf(statsBuf, "\n\t%s: %f", compressorFloat64StatsNames[i], val)
		}
	}
	t.Log(statsBuf)

	errBuf := &bytes.Buffer{}

	if numQueuedBuffers != gotReadCount {
		fmt.Fprintf(errBuf, "\nstats read count total: want :%d, got: %d", numQueuedBuffers, gotReadCount)
	}
	if wantReadByteCount != gotReadByteCount {
		fmt.Fprintf(errBuf, "\nstats read byte count total: want :%d, got: %d", wantReadByteCount, gotReadCount)
	}
	if errBuf.Len() > 0 {
		t.Fatal(errBuf)
	}

	missingCount := 0
	for line, wantCount := range wantLineMap {
		gotCount, ok := gotLineMap[line]
		if !ok {
			fmt.Fprintf(errBuf, "\nmissing %q", line)
			missingCount++
			continue
		}
		if wantCount != gotCount {
			fmt.Fprintf(errBuf, "\n%q count: want: %q, got: %q", line, wantCount, gotCount)
		}
		delete(gotLineMap, line)
	}
	if missingCount > 0 {
		fmt.Fprintf(errBuf, "\nmissingCount: %d, total: %d", missingCount, len(wantLineMap))
	}

	for line := range gotLineMap {
		fmt.Fprintf(errBuf, "\nunexpected %q", line)
	}
	if len(gotLineMap) > 0 {
		fmt.Fprintf(errBuf, "\nunexpectedCount: %d, total: %d", len(gotLineMap), len(wantLineMap))
	}

	if errBuf.Len() > 0 {
		t.Fatal(errBuf)
	}
}

func TestCompressorPoolCreate(t *testing.T) {
	for _, tc := range []*CompressorPoolTestCase{
		{
			NumCompressors: 1,
		},
		{
			NumCompressors: COMPRESSOR_POOL_MAX_NUM_COMPRESSORS,
		},
		{
			NumCompressors: -1,
		},
		{
			BatchTargetSize: "16m",
		},
		{
			FlushInterval: 100 * time.Millisecond,
		},
		{
			BatchTargetSize: "13z",
			wantError:       fmt.Errorf(`NewCompressorPool: invalid batch_target_size "13z": invalid suffix: 'z'`),
		},
	} {
		t.Run(
			"",
			func(t *testing.T) { testCompressorPoolCreate(tc, t) },
		)
	}
}

func TestCompressorPoolQueue(t *testing.T) {
	for _, tc := range []*CompressorPoolTestCase{
		{
			NumCompressors:   1,
			FlushInterval:    0,
			numQueuedBuffers: 15,
		},
		{
			NumCompressors:   1,
			CompressLevel:    gzip.BestCompression,
			FlushInterval:    0,
			numQueuedBuffers: 15,
		},
		{
			NumCompressors:   COMPRESSOR_POOL_MAX_NUM_COMPRESSORS,
			FlushInterval:    0,
			numQueuedBuffers: 15 * COMPRESSOR_POOL_MAX_NUM_COMPRESSORS,
		},
		{
			NumCompressors:   1,
			FlushInterval:    500 * time.Millisecond,
			BatchTargetSize:  "1k",
			numQueuedBuffers: 15,
		},
		{
			NumCompressors:   COMPRESSOR_POOL_MAX_NUM_COMPRESSORS,
			FlushInterval:    500 * time.Millisecond,
			BatchTargetSize:  "1k",
			numQueuedBuffers: 15 * COMPRESSOR_POOL_MAX_NUM_COMPRESSORS,
		},
	} {
		t.Run(
			"",
			func(t *testing.T) { testCompressorPoolQueue(tc, t) },
		)
	}
}
