// Display metrics at stdout instead of sending them to import endpoints.

package vmi_internal

import (
	"bytes"
	"fmt"
	"os"
	"sync"

	"github.com/docker/go-units"
)

type StdoutMetricsQueue struct {
	// The buffer pool for queued metrics:
	bufPool *ReadFileBufPool
	// The metrics channel (queue):
	queue chan *bytes.Buffer
	// Fill with metrics up to the target size:
	batchTargetSize int
	// Wait goroutine on shutdown:
	wg *sync.WaitGroup
	// First time use flag, will print a specific header:
	firstUse bool
}

func NewStdoutMetricsQueue(poolCfg *CompressorPoolConfig) (*StdoutMetricsQueue, error) {
	if poolCfg == nil {
		poolCfg = DefaultCompressorPoolConfig()
	}

	batchTargetSize, err := units.RAMInBytes(poolCfg.BatchTargetSize)
	if err != nil {
		return nil, fmt.Errorf(
			"NewStdoutMetricsQueue: invalid batch_target_size %q: %v",
			poolCfg.BatchTargetSize, err,
		)
	}

	metricsQueue := &StdoutMetricsQueue{
		bufPool:         NewBufPool(poolCfg.BufferPoolMaxSize),
		queue:           make(chan *bytes.Buffer, poolCfg.MetricsQueueSize),
		batchTargetSize: int(batchTargetSize),
		wg:              &sync.WaitGroup{},
		firstUse:        true,
	}

	metricsQueue.wg.Add(1)
	go metricsQueue.loop()

	return metricsQueue, nil
}

func (mq *StdoutMetricsQueue) GetBuf() *bytes.Buffer {
	return mq.bufPool.GetBuf()
}

func (mq *StdoutMetricsQueue) ReturnBuf(buf *bytes.Buffer) {
	mq.bufPool.ReturnBuf(buf)
}

func (mq *StdoutMetricsQueue) QueueBuf(buf *bytes.Buffer) {
	mq.queue <- buf
}

func (mq *StdoutMetricsQueue) GetTargetSize() int {
	return mq.batchTargetSize
}

func (mq *StdoutMetricsQueue) loop() {
	defer mq.wg.Done()

	for {
		buf, isOpen := <-mq.queue
		if !isOpen {
			return
		}
		if mq.firstUse {
			os.Stdout.WriteString("\n# Metrics will be displayed to stdout\n\n")
			mq.firstUse = false
		}
		if buf.Len() > 0 {
			os.Stdout.Write(buf.Bytes())
			os.Stdout.WriteString("\n")
		}
		mq.bufPool.ReturnBuf(buf)
	}
}

func (mq *StdoutMetricsQueue) Shutdown() {
	close(mq.queue)
	mq.wg.Wait()
}
