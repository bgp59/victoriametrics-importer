// Tests for the credit mechanism

package vmi_internal

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"testing"
	"time"
)

// Test scenarios:
//
// 1/ Rate limitation for over-subscription:
//
// Run N credit requestors for a given duration D, each asking in a tight loop
// for a random credit between minDesired, maxDesired. Tally the received value.
// Verify that the actual rate, gotRate = sum(gotten credit)/runDuration, is ~=
// replenishValue/replenishInt by checking that abs(gotRate - targetRate) /
// targetRate <= relativeError threshold.
//
// 2/ Lack of starvation for under-subscription:
//
// Run N credit requestors for a given duration D, each asking in a tight loop
// for a random credit between minDesired, maxDesired until each receives a
// specific (per client, that is) targetCredit, or D expires. The duration is
// such that sum(targetCredit) <= replenishRate * D, so no client should be
// starved of resources. Tally the received credit on a per client basis and
// verify at the end that for each client max(gotCredit - targetCredit, 0) /
// targetCredit <= relativeError threshold.

const (
	TEST_CREDIT_MAX_RELATIVE_ERROR = 0.2
)

type stopFunc func()

type CreditTestCase struct {
	name                   string
	replenishValue         int
	replenishInt           time.Duration
	minDesired, maxDesired int
	numRequestors          int           // superseded by len(targetCredit), if > 0
	testDuration           time.Duration // computed based on sum(targetCredit), if len(targetCredit) > 0
	targetCredit           []int
}

type TestCreditContext struct {
	tc                  *CreditTestCase
	wg                  *sync.WaitGroup
	m                   *sync.Mutex
	c                   *Credit
	stopFnList          []stopFunc
	receivedCreditTotal int
	receivedCredit      []int
	creditRequestCount  int
}

type ParseCreditRateSpecTestCase struct {
	spec               string
	wantReplenishValue int
	wantReplenishInt   time.Duration
	wantError          error
}

func (tcCtx *TestCreditContext) start() {
	tcCtx.c = NewCredit(tcCtx.tc.replenishValue, tcCtx.tc.replenishValue, tcCtx.tc.replenishInt)
	for clientIndex := 0; clientIndex < len(tcCtx.stopFnList); clientIndex++ {
		tcCtx.stopFnList[clientIndex] = startTestCreditUser(tcCtx, clientIndex)
	}
}

func (tcCtx *TestCreditContext) stop() {
	for _, stopFn := range tcCtx.stopFnList {
		stopFn()
	}
	tcCtx.wg.Wait()
	tcCtx.c.StopReplenishWait()
}

func NewTestCreditContext(tc *CreditTestCase) *TestCreditContext {
	tcCtx := &TestCreditContext{
		tc: tc,
		wg: &sync.WaitGroup{},
		m:  &sync.Mutex{},
	}
	var n int
	if tc.targetCredit != nil {
		n = len(tc.targetCredit)
		tcCtx.receivedCredit = make([]int, n)
	} else {
		n = tc.numRequestors
	}
	tcCtx.stopFnList = make([]stopFunc, n)
	return tcCtx
}

func startTestCreditUser(tcCtx *TestCreditContext, clientIndex int) stopFunc {
	ctx, cancelFunc := context.WithCancel(context.Background())
	c, minDesired, maxDesired := tcCtx.c, tcCtx.tc.minDesired, tcCtx.tc.maxDesired
	receivedCredit, targetCredit := tcCtx.receivedCredit, tcCtx.tc.targetCredit
	go func() {
		done := false
		for !done {
			select {
			case <-ctx.Done():
				done = true
			default:
				desired := int(rand.Int63n(int64(maxDesired-minDesired))) + minDesired
				if targetCredit != nil {
					neededCredit := int(0)
					if targetCredit[clientIndex] > receivedCredit[clientIndex] {
						neededCredit = targetCredit[clientIndex] - receivedCredit[clientIndex]
					}
					if neededCredit < desired {
						desired = neededCredit
					}
				}
				minAcceptable := int(0)
				if desired > 1 {
					minAcceptable = int(rand.Int63n(int64(desired-1))) + 1
				}
				got := c.GetCredit(desired, minAcceptable)
				if targetCredit != nil {
					receivedCredit[clientIndex] += got
					if targetCredit[clientIndex] <= receivedCredit[clientIndex] {
						done = true
					}
				}
				tcCtx.m.Lock()
				tcCtx.receivedCreditTotal += got
				tcCtx.creditRequestCount += 1
				tcCtx.m.Unlock()
			}
		}
		tcCtx.wg.Done()
	}()
	tcCtx.wg.Add(1)

	return func() {
		cancelFunc()
	}
}

func testCredit(
	tc *CreditTestCase,
	t *testing.T,
) {
	t.Logf(`
name=%q
replenishValue/Interval=%d/%s
minDesired..maxDesired=%d..%d
numRequestors=%d (inferred if targetCredit is set)
testDuration=%s (inferred if targetCredit is set)
targetCredit=%v
`,
		tc.name,
		tc.replenishValue, tc.replenishInt,
		tc.minDesired, tc.maxDesired,
		tc.numRequestors,
		tc.testDuration,
		tc.targetCredit,
	)

	var testDuration time.Duration
	if tc.targetCredit != nil {
		totalTargetCredit := int(0)
		for _, credit := range tc.targetCredit {
			totalTargetCredit += credit
		}
		testDuration = time.Duration(
			float64(totalTargetCredit)/
				(float64(tc.replenishValue)/tc.replenishInt.Seconds())*
				float64(time.Second)) + tc.replenishInt
	} else {
		testDuration = tc.testDuration
	}

	tcCtx := NewTestCreditContext(tc)
	tcCtx.start()
	testStart := time.Now()
	time.Sleep(testDuration)
	tcCtx.stop()
	actualDuration := time.Since(testStart)
	targetRate := float64(tc.replenishValue) / tc.replenishInt.Seconds()
	actualRate := float64(tcCtx.receivedCreditTotal) / actualDuration.Seconds()
	relativeError := math.Abs(actualRate-targetRate) / targetRate
	msg := fmt.Sprintf(
		"\nDuration: %s/%s, Req#: %d, rate: want: %.03f/sec, got: %.03f/sec, relativeError: want: <=%.02f, got: %.02f",
		testDuration.String(),
		actualDuration.String(),
		tcCtx.creditRequestCount,
		targetRate, actualRate,
		TEST_CREDIT_MAX_RELATIVE_ERROR, relativeError,
	)
	if relativeError > TEST_CREDIT_MAX_RELATIVE_ERROR {
		t.Fatal(msg)
	} else {
		t.Log(msg)
	}

	if tc.targetCredit != nil {
		for i, wantCredit := range tc.targetCredit {
			gotCredit := tcCtx.receivedCredit[i]
			relativeDeficit := (float64(gotCredit) - float64(wantCredit)) / float64(wantCredit)
			if relativeDeficit < 0 {
				relativeDeficit = 0
			}
			msg := fmt.Sprintf(
				"\nClient[%d]: credit: want: %d, got: %d, relativeDeficit: want: <=%.02f, got: %.02f",
				i, wantCredit, gotCredit,
				TEST_CREDIT_MAX_RELATIVE_ERROR, relativeDeficit,
			)
			if relativeDeficit > TEST_CREDIT_MAX_RELATIVE_ERROR {
				t.Fatal(msg)
			} else {
				t.Log(msg)
			}
		}
	}
}

func TestCredit(t *testing.T) {
	for _, tc := range []*CreditTestCase{
		{
			name:           "over_subscription",
			replenishValue: 12_500,
			replenishInt:   100 * time.Millisecond,
			minDesired:     1000,
			maxDesired:     10_000,
			numRequestors:  2,
			testDuration:   2 * time.Second,
		},
		{
			name:           "over_subscription",
			replenishValue: 12_500,
			replenishInt:   100 * time.Millisecond,
			minDesired:     1000,
			maxDesired:     10_000,
			numRequestors:  4,
			testDuration:   2 * time.Second,
		},
		{
			name:           "over_subscription",
			replenishValue: 125_000,
			replenishInt:   100 * time.Millisecond,
			minDesired:     1,
			maxDesired:     50_000,
			numRequestors:  16,
			testDuration:   2 * time.Second,
		},
		{
			name:           "under_subscription",
			replenishValue: 125_000,
			replenishInt:   100 * time.Millisecond,
			minDesired:     1,
			maxDesired:     50_000,
			targetCredit:   []int{1_250_000, 2_000_000},
		},
		{
			name:           "under_subscription",
			replenishValue: 125_000,
			replenishInt:   100 * time.Millisecond,
			minDesired:     1,
			maxDesired:     5_000,
			targetCredit:   []int{250_000, 2_000_000, 999_999},
		},
	} {
		t.Run(
			tc.name,
			func(t *testing.T) { testCredit(tc, t) },
		)
	}
}

func TestCreditStop(t *testing.T) {
	replenishValue := 100
	wantCreditAfterStop := 100 * replenishValue
	timeout := 100 * time.Millisecond
	replenishInt := 10 * timeout

	creditVal := make(chan int)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	go func() {
		credit := NewCredit(replenishValue, replenishValue, replenishInt)
		credit.StopReplenishWait()
		creditVal <- credit.GetCredit(wantCreditAfterStop, CREDIT_EXACT_MATCH)
	}()

	select {
	case gotCreditAfterStop := <-creditVal:
		if wantCreditAfterStop != gotCreditAfterStop {
			t.Fatalf("Credit after stop: want: %d, got: %d", wantCreditAfterStop, gotCreditAfterStop)
		}
	case <-ctx.Done():
		t.Fatalf("Timeout after %s", timeout)
	}
}

func testParseCreditRateSpec(tc *ParseCreditRateSpecTestCase, t *testing.T) {
	gotReplenishValue, gotReplenishInt, gotErr := ParseCreditRateSpec(tc.spec)
	if gotErr != nil && tc.wantError == nil ||
		gotErr == nil && tc.wantError != nil ||
		gotErr != nil && tc.wantError != nil && gotErr.Error() != tc.wantError.Error() {
		t.Fatalf("spec: %q: err: want: %v, got: %v", tc.spec, tc.wantError, gotErr)
	}
	if gotErr == nil && tc.wantError == nil {
		if gotReplenishValue != tc.wantReplenishValue {
			t.Fatalf("spec: %q: replenishValue: want: %d, got: %d",
				tc.spec, tc.wantReplenishValue, gotReplenishValue,
			)
		}
		if gotReplenishInt != tc.wantReplenishInt {
			t.Fatalf("spec: %q: replenishInt: want: %d (%s), got: %d (%s)",
				tc.spec, tc.wantReplenishInt, tc.wantReplenishInt, gotReplenishInt, gotReplenishInt,
			)
		}
	}
}

func TestParseCreditRateSpec(t *testing.T) {
	for _, tc := range []*ParseCreditRateSpecTestCase{
		{"1", 125_000, 1 * time.Second, nil},
		{"1.", 125_000, 1 * time.Second, nil},
		{"1.:0.1s", 12_500, 100 * time.Millisecond, nil},
		{".5:0.1s", 6_250, 100 * time.Millisecond, nil},
		{".5:0.01s", 625, 10 * time.Millisecond, nil},
		{".2:0.1s", 2_500, 100 * time.Millisecond, nil},
		{".2:10ms", 250, 10 * time.Millisecond, nil},
		{"", 0, 0, fmt.Errorf(`ParseCreditRateSpec(""): strconv.ParseFloat: parsing "": invalid syntax`)},
		{"2x:1ms", 0, 0, fmt.Errorf(`ParseCreditRateSpec("2x:1ms"): strconv.ParseFloat: parsing "2x": invalid syntax`)},
		{"1:33", 0, 0, fmt.Errorf(`ParseCreditRateSpec("1:33"): time: missing unit in duration "33"`)},
	} {
		t.Run(
			fmt.Sprintf("spec=%s", tc.spec),
			func(t *testing.T) { testParseCreditRateSpec(tc, t) },
		)
	}
}
