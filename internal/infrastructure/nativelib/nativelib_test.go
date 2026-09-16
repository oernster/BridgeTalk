package nativelib

import "testing"

func TestNoAddressIsAnEmptyString(t *testing.T) {
	t.Parallel()

	if got := String(0); got != "" {
		t.Errorf("String(0) = %q, want empty", got)
	}
}
