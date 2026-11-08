package status

import (
	"strings"
	"testing"
)

func TestAreaNameAct1Complete(t *testing.T) {
	for id := 1; id <= 39; id++ {
		if n := AreaName(id); n == "" || strings.HasPrefix(n, "Unknown") {
			t.Errorf("act 1 area %d has no name: %q", id, n)
		}
	}
	for id, want := range map[int]string{17: "Burial Grounds", 37: "Catacombs Level 4", 38: "Tristram", 25: "Tower Cellar Level 5"} {
		if got := AreaName(id); got != want {
			t.Errorf("AreaName(%d) = %q, want %q", id, got, want)
		}
	}
	if got := AreaName(999); got != "Unknown area (id 999)" {
		t.Errorf("unknown id: %q", got)
	}
	if AreaName(0) != "" {
		t.Error("id 0 must stay empty")
	}
}
