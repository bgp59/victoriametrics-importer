// Credit based rate limit controller.
//
// The credit is a numerical quantity replenished periodically, at intervals T,
// with a constant number N. The replenished value may by capped to a max M>=N,
// or it may be unbound. The value R=N/T represents the target rate limit and
// M-N represents the burst limit.
//
// A user in need of n resources should request a credit ==/<= n before
// proceeding (the user may specify an interval nMin..n, nMin <= n). If credit
// is available the user receives a value c within the requested interval and it
// then should use no more than c.
//
// Use case: limit network utilization by choosing N/T = target bandwidth.

package vmi_internal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	CREDIT_NO_LIMIT    = 0
	CREDIT_EXACT_MATCH = 0
)

// Define an interface for testing:
type CreditController interface {
	GetCredit(desired, minAcceptable int) int
}

// The actual implementation:
type Credit struct {
	ctx            context.Context
	cancelFunc     context.CancelFunc
	wg             *sync.WaitGroup
	cond           *sync.Cond
	current        int
	max            int
	replenishValue int
	replenishInt   time.Duration
}

// Credit based reader, limiting the rate of data read from a byte buffer and
// implementing the io.ReadSeekCloser interface, so it can be used in
// http.Request.Body. This is used to control the rate of import into
// VictoriaMetrics.
type CreditReader struct {
	// Credit control:
	cc CreditController
	// Minimum acceptable credit:
	minC int
	// Bytes to return with the controlled rate:
	b []byte
	// Read pointer in b:
	r int
	// Total size of b:
	n int
	// Closed flag:
	closed bool
}

// Parse rate limit Mbps string. Supported formats: FLOAT or FLOAT:INTERVAL,
// where INTERVAL should be in the format supported by time.ParseDuration().
// FLOAT is equivalent w/ FLOAT:1s.
func ParseCreditRateSpec(spec string) (int, time.Duration, error) {
	mbps, interval := spec, "1s"
	i := strings.Index(spec, ":")
	if i >= 0 {
		mbps, interval = spec[:i], spec[i+1:]
	}
	mbpsf, err := strconv.ParseFloat(mbps, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("ParseCreditRateSpec(%q): %v", spec, err)
	}
	replenishInt, err := time.ParseDuration(interval)
	if err != nil {
		return 0, 0, fmt.Errorf("ParseCreditRateSpec(%q): %v", spec, err)
	}
	replenishValue := int(mbpsf * 1_000_000 / 8 * float64(replenishInt) / float64(1*time.Second))
	return replenishValue, replenishInt, nil
}

func NewCredit(replenishValue, max int, replenishInt time.Duration) *Credit {
	if max != CREDIT_NO_LIMIT && max < replenishValue {
		max = replenishValue
	}

	ctx, cancelFunc := context.WithCancel(context.Background())

	c := &Credit{
		ctx:            ctx,
		cancelFunc:     cancelFunc,
		wg:             &sync.WaitGroup{},
		cond:           sync.NewCond(&sync.Mutex{}),
		max:            max,
		replenishValue: replenishValue,
		replenishInt:   replenishInt,
	}
	c.startReplenish()
	return c
}

func NewCreditFromSpec(spec string) (*Credit, error) {
	replenishValue, replenishInt, err := ParseCreditRateSpec(spec)
	if err != nil {
		return nil, err
	}
	return NewCredit(replenishValue, 0, replenishInt), nil
}

func (c *Credit) startReplenish() {
	replenishInt := c.replenishInt
	nextReplenishTime := time.Now().Add(replenishInt)
	c.current = c.replenishValue
	ctx, wg := c.ctx, c.wg
	wg.Add(1)
	go func() {
		for {
			select {
			case <-ctx.Done():
				wg.Done()
				return
			default:
				pause := time.Until(nextReplenishTime)
				if pause > 0 {
					time.Sleep(pause)
				}
				nextReplenishTime = nextReplenishTime.Add(replenishInt)
				c.cond.L.Lock()
				c.current += c.replenishValue
				if c.max != CREDIT_NO_LIMIT && c.current > c.max {
					c.current = c.max
				}
				c.cond.Broadcast()
				c.cond.L.Unlock()
			}
		}
	}()
}

func (c *Credit) StopReplenish() {
	c.cancelFunc()
}

func (c *Credit) StopReplenishWait() {
	c.cancelFunc()
	c.wg.Wait()
}

func (c *Credit) GetCredit(desired, minAcceptable int) (got int) {
	if minAcceptable == CREDIT_EXACT_MATCH ||
		minAcceptable > desired {
		minAcceptable = desired
	}

	c.cond.L.Lock()
	defer c.cond.L.Unlock()

	for c.current < minAcceptable {
		c.cond.Wait()
	}

	if c.current >= desired {
		got = desired
	} else {
		got = c.current
	}
	c.current -= got
	return
}

func (c *Credit) String() string {
	if c == nil {
		return fmt.Sprintf("%v", nil)
	}
	return fmt.Sprintf(
		"%T{replenishValue=%d, replenishInt=%s, max=%d}",
		c, c.replenishValue, c.replenishInt, c.max,
	)
}

func NewCreditReader(cc CreditController, minAcceptable int, b []byte) *CreditReader {
	if minAcceptable < 0 {
		minAcceptable = 0
	}
	return &CreditReader{
		cc:   cc,
		minC: int(minAcceptable),
		b:    b,
		r:    0,
		n:    len(b),
	}
}

// Reuse w/ new data:
func (cr *CreditReader) Reuse(minAcceptable int, b []byte) {
	if minAcceptable >= 0 {
		cr.minC = minAcceptable
	}
	cr.b, cr.r, cr.n = b, 0, len(b)
}

// Reuse w/ the same data:
func (cr *CreditReader) Rewind() error {
	cr.r, cr.closed = 0, false
	return nil
}

// Implement the Read interface:
func (cr *CreditReader) Read(p []byte) (int, error) {
	if cr.closed {
		return 0, nil
	}
	available := cr.n - cr.r
	if available <= 0 {
		return 0, io.EOF
	}
	toRead := len(p)
	if toRead == 0 {
		return 0, nil
	}
	if available < toRead {
		toRead = available
	}
	toRead = int(cr.cc.GetCredit(toRead, cr.minC))
	if toRead == 0 {
		return 0, nil
	}
	s := cr.r
	cr.r += toRead
	copy(p, cr.b[s:cr.r])
	return toRead, nil
}

// Implement Seek interface:

// Modelled on io:
var errCreditReaderWhence = errors.New("Seek: invalid whence")
var errCreditReaderOffset = errors.New("Seek: invalid offset")

func (cr *CreditReader) Seek(offset int64, whence int) (int64, error) {
	var newR int64
	switch whence {
	case io.SeekCurrent:
		newR = int64(cr.r) + offset
	case io.SeekStart:
		newR = offset
	case io.SeekEnd:
		newR = int64(cr.n) - 1 + offset
	default:
		return 0, errCreditReaderWhence
	}
	if newR < 0 || newR >= int64(cr.n) {
		return 0, errCreditReaderOffset
	}
	if cr.closed {
		newR = int64(cr.r)
	} else {
		cr.r = int(newR)
	}
	return newR, nil
}

// Implement Close interface:
func (cr *CreditReader) Close() error {
	cr.closed = true
	return nil
}
