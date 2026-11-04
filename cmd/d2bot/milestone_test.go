package main

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// The milestone lines come once per word, on the rising edge, and the view the reader returns is the same with and
// without a logger (logging changes no decision).
func TestMilestoneLogsOncePerWordAndChangesNothing(t *testing.T) {
	var logs []string
	clock := time.Unix(1000, 0)
	mk := func(logOn bool, words map[string]uint16) *saveQuestReader {
		r := newSaveQuestReader(func() string { return "" }, func() string { return "" })
		r.mem = fakeMem{words: words}
		r.now = func() time.Time { return clock }
		if logOn {
			r.log = func(msg string, args ...any) { logs = append(logs, fmt.Sprint(msg, args)) }
		}
		return r
	}
	w := map[string]uint16{"den_of_evil": 0x0001}
	r := mk(true, w)
	r.Read(0)
	clock = clock.Add(90 * time.Second)
	r.Read(0)
	n := 0
	for _, l := range logs {
		if strings.HasPrefix(l, "milestone[name den ") {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("den milestone logged %d times: %v", n, logs)
	}
	logs = nil
	r.mem = fakeMem{words: map[string]uint16{"den_of_evil": 1, "sisters_to_the_slaughter": 0x0001}}
	clock = clock.Add(60 * time.Second)
	r.Read(0)
	r.Read(0)
	j := strings.Join(logs, "\n")
	if !strings.Contains(j, "milestone[name andariel word 0x0001 clvl -1 virtual 2.5]") {
		t.Errorf("andariel milestone missing:\n%s", j)
	}
	if strings.Count(j, "act1: andariel killed words 6=0x0001 7=0x0000") != 1 {
		t.Errorf("andariel words line:\n%s", j)
	}
	if strings.Contains(j, "name den") {
		t.Errorf("den logged twice:\n%s", j)
	}
	a, _ := mk(false, map[string]uint16{"den_of_evil": 1}).Read(0)
	b, _ := mk(true, map[string]uint16{"den_of_evil": 1}).Read(0)
	if fmt.Sprint(a) != fmt.Sprint(b) {
		t.Errorf("logging changed the view: %v vs %v", a, b)
	}
}
