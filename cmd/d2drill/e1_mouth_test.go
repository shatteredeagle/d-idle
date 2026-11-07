package main

import (
	"strconv"
	"testing"
)

func TestE1MouthEval(t *testing.T) {
	st := func(x, y int) string {
		return `{"msg":"status","state":"clearing","area":6,"x":` + itoa(x) + `,"y":` + itoa(y) + `}`
	}
	gate := `{"msg":"nav mouth gate: pocket escape"}`
	restart := `{"msg":"d2d: Save and Exit requested"}`
	for _, c := range []struct {
		name  string
		lines []string
		want  string
	}{
		{"left with the gate", []string{st(87, 287), gate, st(120, 300)}, "PASS"},
		{"restart first", []string{st(87, 287), gate, restart, st(120, 300)}, "FAIL"},
		{"walked off, no gate", []string{st(87, 287), st(120, 300)}, "INCONCLUSIVE"},
		{"never left", []string{st(87, 287), gate, st(90, 289)}, "FAIL"},
	} {
		if v, why := e1MouthEval(c.lines, nil); v != c.want {
			t.Errorf("%s: %s (%s), want %s", c.name, v, why, c.want)
		}
	}
}

func itoa(n int) string { return strconv.Itoa(n) }
