// Collectable log, (*testing.T).Log style.

// If the test is not running in verbose mode, collect the app logger's output
// and display it JIT at Fatal[f] invocation:

package vmi_testutils

import (
	"io"
	"testing"
)

// The interface expected from a collectable log:
type CollectableLog interface {
	GetLevel() any
	SetLevel(level any)
	GetOutput() io.Writer
	SetOutput(out io.Writer)
}

type TestLogCollect struct {
	log        CollectableLog
	savedOut   io.Writer
	savedLevel any
	t          *testing.T
}

func NewTestLogCollect(t *testing.T, log any, level any) *TestLogCollect {
	tlc := &TestLogCollect{
		t: t,
	}
	if log, ok := log.(CollectableLog); ok && log != nil {
		if !testing.Verbose() {
			tlc.log = log
			tlc.savedOut = log.GetOutput()
			log.SetOutput(tlc)
		}
		if level != nil {
			tlc.savedLevel = log.GetLevel()
			log.SetLevel(level)
		}
	}
	return tlc
}

func (tlc *TestLogCollect) Write(buf []byte) (int, error) {
	n := len(buf)
	if n > 0 && buf[n-1] == '\n' {
		buf = buf[:n-1]
	}
	tlc.t.Log(string(buf))
	return n, nil
}

func (tlc *TestLogCollect) RestoreLog() {
	if tlc.log != nil {
		if tlc.savedOut != nil {
			tlc.log.SetOutput(tlc.savedOut)
		}
		if tlc.savedLevel != nil {
			tlc.log.SetLevel(tlc.savedLevel)
		}
	}
}
