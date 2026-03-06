package cmd

import (
	"testing"

	runjobs "github.com/po1o/prompto/src/runtime/jobs"
)

func TestCurrentGID(t *testing.T) {
	if gid := runjobs.CurrentGID(); gid == 0 {
		t.Fatalf("CurrentGID returned 0")
	}
}
