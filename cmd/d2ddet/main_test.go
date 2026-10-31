package main

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, name, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

const base = "# d2dctl journal v1 every=100\n>\t0\tstate\taaaa\n>\t50\tstep 50\ttick=50\n>\t50\tmonsters\tbbbb\n>\t100\tstep 50\ttick=100\nC\t100\tcccc\n>\t100\tcmd move 1 2\tdddd\n"

func TestCompareIdenticalAndPrefix(t *testing.T) {
	a := write(t, "a", base)
	b := write(t, "b", base+">\t100\tstate\teeee\n")
	if got := compare(a, a, 2, nil); got != 0 {
		t.Fatalf("identical: exit %d", got)
	}
	if got := compare(a, b, 2, nil); got != 0 {
		t.Fatalf("prefix: exit %d", got)
	}
}

func TestCompareFindsCommandDivergence(t *testing.T) {
	a := write(t, "a", base)
	b := write(t, "b", "# d2dctl journal v1 every=100\n>\t0\tstate\taaaa\n>\t50\tstep 50\ttick=50\n>\t50\tmonsters\tbbbb\n>\t100\tstep 50\ttick=100\nC\t100\tcccc\n>\t100\tcmd move 3 4\tdddd\n")
	if got := compare(a, b, 2, nil); got != 1 {
		t.Fatalf("exit %d want 1", got)
	}
	ra, _ := load(a, nil)
	rb, _ := load(b, nil)
	if c := classify(ra[5], rb[5]); c[:7] != "COMMAND" {
		t.Fatalf("classify: %s", c)
	}
	if c := classify(rec{kind: "C", val: "1"}, rec{kind: "C", val: "2"}); c[:10] != "STATE HASH" {
		t.Fatalf("classify: %s", c)
	}
	if c := classify(rec{kind: ">", cmd: "step 5", val: "tick=5"}, rec{kind: ">", cmd: "step 7", val: "tick=7"}); c[:10] != "STEP BATCH" {
		t.Fatalf("classify: %s", c)
	}
}

func TestIgnoreDropsVerb(t *testing.T) {
	a := write(t, "a", base)
	b := write(t, "b", base+">\t100\tdebug portals\tffff\n>\t100\tstate\teeee\n")
	if got := compare(a, b, 2, map[string]bool{"debug portals": true}); got != 0 {
		t.Fatalf("exit %d", got)
	}
}
