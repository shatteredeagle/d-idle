package scenario

import "testing"

func TestRunLengthReached(t *testing.T) {
	if !RunLengthReached([]byte(`{"msg":"virtual run length reached","virtual":3600000000000}`)) {
		t.Fatal("line present: reached")
	}
	if RunLengthReached([]byte(`{"msg":"zone entry"}`)) {
		t.Fatal("a log without the line is a cut run")
	}
}

func TestParseBuildInfo(t *testing.T) {
	out := "d2bot: go1.27.3\n\tbuild\tvcs.revision=5b12ce5e7\n\tbuild\tvcs.modified=true\n"
	c, d := ParseBuildInfo(out)
	if c != "5b12ce5e7" || !d {
		t.Fatalf("got %q %v", c, d)
	}
	c, d = ParseBuildInfo("build\t-ldflags=x\n")
	if c != "" || d {
		t.Fatalf("no stamp must give empty, got %q %v", c, d)
	}
}

func TestPaceOf(t *testing.T) {
	if PaceOf(10, 60) != 10 || PaceOf(10, 0) != 0 {
		t.Fatal("pace")
	}
}

func TestPeakParallel(t *testing.T) {
	log := SlotLogLine("start", "a", 10) + SlotLogLine("start", "b", 20) + SlotLogLine("end", "a", 30) +
		SlotLogLine("start", "c", 40) + SlotLogLine("start", "d", 50) + SlotLogLine("end", "x", 5) + "garbage\n"
	for _, tc := range []struct {
		id   string
		want int
	}{
		{"a", 2}, // b started while a ran; c and d started after a's end
		{"b", 3}, // c and d started while b still ran (a had ended)
		{"c", 3},
		{"d", 3},
		{"x", 0}, // no start line
		{"nope", 0},
	} {
		if got := PeakParallel([]byte(log), tc.id); got != tc.want {
			t.Errorf("%s: %d, want %d", tc.id, got, tc.want)
		}
	}
	if got := PeakParallel([]byte(SlotLogLine("start", "solo", 1)), "solo"); got != 1 {
		t.Fatalf("alone: %d, want 1", got)
	}
	// A run that ended before self started does not count.
	if got := PeakParallel([]byte(SlotLogLine("start", "p", 1)+SlotLogLine("end", "p", 2)+SlotLogLine("start", "q", 3)), "q"); got != 1 {
		t.Fatalf("after an ended run: %d, want 1", got)
	}
}
