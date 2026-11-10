package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func load(t *testing.T) []line {
	t.Helper()
	f, err := os.Open("testdata/hour.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	l, err := parse(f)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func TestComputeFixture(t *testing.T) {
	for _, class := range []string{"paladin", "sorceress"} { // class is a label only; nothing is class specific
		r, err := Compute(load(t), class, time.Time{}, time.Hour, 90*time.Second)
		if err != nil {
			t.Fatal(err)
		}
		want := "| 2026-11-10 | v0.9.1-abc123 | " + class + " | 12:00Z | 8 | 180 | 1 | 109 | 2 | 5 | 2 | 13 | 1 |\n"
		if got := r.Row(); got != want {
			t.Errorf("row\n got %s want %s", got, want)
		}
	}
}

func TestFailsLoudly(t *testing.T) {
	l := load(t)
	cases := map[string]func() error{
		"short": func() error { _, e := Compute(l[:40], "paladin", time.Time{}, time.Hour, time.Minute); return e },
		"empty": func() error { _, e := Compute(nil, "paladin", time.Time{}, time.Hour, time.Minute); return e },
		"stale hero": func() error {
			_, e := Compute(l, "paladin", l[0].at.Add(50*time.Minute), 5*time.Minute, time.Minute)
			return e
		},
	}
	for n, f := range cases {
		if f() == nil {
			t.Errorf("%s: want an error", n)
		}
	}
}

func TestDryRunAndAppend(t *testing.T) {
	var out, errw bytes.Buffer
	if rc := run([]string{"-log", "testdata/hour.txt", "-class", "paladin", "-dry-run"}, nil, &out, &errw); rc != 0 {
		t.Fatalf("rc %d %s", rc, errw.String())
	}
	tab := filepath.Join(t.TempDir(), "p", "firsthour.md")
	out.Reset()
	for i := 0; i < 2; i++ {
		if rc := run([]string{"-log", "testdata/hour.txt", "-class", "paladin", "-table", tab}, nil, &out, &errw); rc != 0 {
			t.Fatalf("rc %d %s", rc, errw.String())
		}
	}
	b, _ := os.ReadFile(tab)
	if n := strings.Count(string(b), "| 2026-11-10 |"); n != 2 || !strings.HasPrefix(string(b), "# First hour") {
		t.Errorf("table:\n%s", b)
	}
	if rc := run([]string{"-log", "testdata/hour.txt"}, nil, &out, &errw); rc != 2 {
		t.Errorf("missing -class: rc %d", rc)
	}
}

func TestBuildSwapVisible(t *testing.T) {
	l := load(t)
	for i := range l {
		if l[i].kv["msg"] == "d2bot start" && i > 0 {
			l[i].kv["version"] = "v0.9.2-def"
		}
	}
	r, err := Compute(l, "paladin", time.Time{}, time.Hour, 90*time.Second)
	if err != nil || len(r.Versions) != 2 {
		t.Fatalf("%v %v", r.Versions, err)
	}
}
