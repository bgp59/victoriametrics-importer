// Collectable log, (*testing.T).Log style.

// If the test is not running in verbose mode, collect the app logger's output
// and display it JIT at Fatal[f] invocation:

package vmi_testutils

import (
	logrusx_testutil "github.com/bgp59/logrusx/testutils"
)

var NewTestCollectableLogger = logrusx_testutil.NewTestCollectableLogger
