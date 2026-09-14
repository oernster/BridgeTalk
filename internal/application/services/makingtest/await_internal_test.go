package makingtest

// Await's failure, reached through awaitWithin with a limit that passes at once.

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// failures stands in for a test, recording what Fatalf would have failed it with.
type failures struct {
	testing.TB
	messages []string
}

func (f *failures) Helper() {}

func (f *failures) Fatalf(format string, args ...any) {
	f.messages = append(f.messages, fmt.Sprintf(format, args...))
}

func TestAWaitPastItsLimitFailsNamingWhatNeverHappened(t *testing.T) {
	t.Parallel()
	test := &failures{}
	awaitWithin(test, make(chan struct{}), "making starting", time.Nanosecond)
	if len(test.messages) != 1 || !strings.Contains(test.messages[0], "making starting") {
		t.Errorf("failed with %q, want one failure naming making starting", test.messages)
	}
}
