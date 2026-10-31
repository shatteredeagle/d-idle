package main

import (
	"bytes"
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"d2lod-bot/internal/fleet"
)

type nopRT struct{}

func (nopRT) Start(context.Context, fleet.Alloc, fleet.Spec) error { return nil }
func (nopRT) Login(context.Context, fleet.Alloc, fleet.Spec) error { return nil }
func (nopRT) Alive(context.Context, fleet.Alloc) (bool, error)     { return true, nil }
func (nopRT) LeaveGame(context.Context, fleet.Alloc) error         { return nil }
func (nopRT) Stop(context.Context, fleet.Alloc) error              { return nil }

func TestCLIAgainstHandler(t *testing.T) {
	f, err := fleet.New(fleet.Config{Runtime: nopRT{}, DataDir: t.TempDir(), Limits: fleet.Limits{MaxLoadFactor: -1}})
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close(context.Background())
	srv := httptest.NewServer(fleet.Handler(f, ""))
	defer srv.Close()
	call := func(args ...string) (int, string, string) {
		var o, e bytes.Buffer
		c := run(append(args, "-url", srv.URL), &o, &e)
		return c, o.String(), e.String()
	}
	if c, o, _ := call("add", "Navi", "-class", "sorceress", "-role", "leader"); c != 0 || !strings.Contains(o, `"id":"i1"`) {
		t.Fatal(c, o)
	}
	if c, _, e := call("add", "Navi"); c != 1 || !strings.Contains(e, "409") {
		t.Fatal(c, e)
	}
	if c, o, _ := call("list"); c != 0 || !strings.Contains(o, "Navi") {
		t.Fatal(c, o)
	}
	if c, o, _ := call("status"); c != 0 || !strings.Contains(o, "capacity") {
		t.Fatal(c, o)
	}
	if c, _, _ := call("remove", "i1"); c != 0 {
		t.Fatal(c)
	}
	if c, _, _ := run2(); c != 2 {
		t.Fatal(c)
	}
}

func run2() (int, string, string) {
	var o, e bytes.Buffer
	return run(nil, &o, &e), o.String(), e.String()
}

func TestSeedCharacter(t *testing.T) {
	src, dst := t.TempDir(), filepath.Join(t.TempDir(), "x", "y")
	for _, n := range []string{"Bob.d2s", "Bob.key", "Other.d2s"} {
		os.WriteFile(filepath.Join(src, n), []byte(n), 0o644)
	}
	if err := seedCharacter(src, dst, "Bob"); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"Bob.d2s", "Bob.key"} {
		if _, err := os.Stat(filepath.Join(dst, n)); err != nil {
			t.Errorf("%s not copied", n)
		}
	}
	if _, err := os.Stat(filepath.Join(dst, "Other.d2s")); err == nil {
		t.Error("other character copied")
	}
	os.WriteFile(filepath.Join(dst, "Bob.d2s"), []byte("newer"), 0o644)
	seedCharacter(src, dst, "Bob")
	if b, _ := os.ReadFile(filepath.Join(dst, "Bob.d2s")); string(b) != "newer" {
		t.Error("existing character overwritten")
	}
}
