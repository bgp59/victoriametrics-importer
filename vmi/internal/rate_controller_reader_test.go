package vmi_internal

import (
	"fmt"
	"io"
	"testing"
)

type CreditMock struct {
	// What to return in the next GetCredit:
	retVal int
}

func (cm *CreditMock) GetCredit(desired, minAcceptable int) int {
	return cm.retVal
}

type CreditReaderTestStep struct {
	getCreditRetVal int
	wantReadN       int
	wantReadErr     error
}

func (step *CreditReaderTestStep) String() string {
	return fmt.Sprintf(
		"{%d, %d, %v}",
		step.getCreditRetVal, step.wantReadN, step.wantReadErr,
	)
}

type CreditReaderTestCase struct {
	name        string
	readBufSize int
	crBufSize   int
	steps       []*CreditReaderTestStep
}

func testCreditReader(tc *CreditReaderTestCase, t *testing.T) {
	t.Logf(`
name=%q
readBufSize=%d
crBufSize=%d
steps=%v
`,
		tc.name, tc.readBufSize, tc.crBufSize, tc.steps,
	)

	cc := &CreditMock{}
	cr := NewCreditReader(cc, 0, make([]byte, tc.crBufSize))
	p, s := make([]byte, tc.readBufSize), 0
	for i, step := range tc.steps {
		cc.retVal = step.getCreditRetVal
		gotN, gotErr := cr.Read(p[s:])
		if step.wantReadN != gotN || step.wantReadErr != gotErr {
			t.Fatalf(
				"step[%d]: (n, err): want: (%d, %v), got: (%d, %v)",
				i,
				step.wantReadN, step.wantReadErr,
				gotN, gotErr,
			)
		}
		s += gotN
	}
}

func TestCreditReader(t *testing.T) {
	for _, tc := range []*CreditReaderTestCase{
		{
			name:        "read_match",
			readBufSize: 10,
			crBufSize:   10,
			steps: []*CreditReaderTestStep{
				{3, 3, nil},
				{4, 4, nil},
				{3, 3, nil},
				// At EOF, the credit will not be invoked, hence the unrealistic
				// value:
				{10000, 0, io.EOF},
				{10000, 0, io.EOF},
			},
		},
		{
			name:        "zero_len_read_buf",
			readBufSize: 0,
			crBufSize:   10,
			steps: []*CreditReaderTestStep{
				// The credit will not be invoked, hence the unrealistic value:
				{10000, 0, nil},
				{10000, 0, nil},
			},
		},
		{
			name:        "under_read",
			readBufSize: 10,
			crBufSize:   20,
			steps: []*CreditReaderTestStep{
				{3, 3, nil},
				{4, 4, nil},
				{3, 3, nil},
				// The credit will not be invoked, hence the unrealistic
				// value:
				{10000, 0, nil},
				{10000, 0, nil},
			},
		},
		{
			name:        "over_read",
			readBufSize: 20,
			crBufSize:   10,
			steps: []*CreditReaderTestStep{
				{3, 3, nil},
				{4, 4, nil},
				{3, 3, nil},
				// At EOF, the credit will not be invoked, hence the unrealistic
				// value:
				{10000, 0, io.EOF},
				{10000, 0, io.EOF},
			},
		},
	} {
		t.Run(
			tc.name,
			func(t *testing.T) { testCreditReader(tc, t) },
		)
	}
}
