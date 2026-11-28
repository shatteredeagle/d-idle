package d2denv

import (
	"fmt"
	"testing"
)

func TestLogLeaveLevelByReason(t *testing.T) {
	for why, warn := range map[string]bool{"": true, "chicken": true, "leave": true, "savepoint": false, "newgame": false, "recycle": false} {
		var w, i int
		p := &Proc{Logf: func(string, ...any) { w++ }, InfoLogf: func(string, ...any) { i++ }}
		if why != "" {
			p.LeaveWhy = func() string { return why }
		}
		p.logLeave()
		if (w == 1) != warn || (i == 1) == warn {
			t.Errorf("%s: warn=%d info=%d, want warn=%v", fmt.Sprint(why), w, i, warn)
		}
	}
}
