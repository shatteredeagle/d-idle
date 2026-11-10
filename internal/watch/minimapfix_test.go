package watch

import (
	"strings"
	"testing"
)

// The shell scheduler must not be able to wedge a panel (TASK-minimapfix): a stalled request is aborted, an error
// handler that throws still releases the slot, and one panel throwing does not stop the tick loop.
func TestShellSchedulerCannotWedge(t *testing.T) {
	b, err := webFS.ReadFile("web/shell.js")
	if err != nil {
		t.Fatal(err)
	}
	js := string(b)
	for _, want := range []string{"AbortController", "FETCH_TIMEOUT", "clearTimeout(timer)", "catch (e2)", "try { if (due(s, now)) run(s); }"} {
		if !strings.Contains(js, want) {
			t.Errorf("shell.js lost %q", want)
		}
	}
}
