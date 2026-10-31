package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"d2lod-bot/internal/boss"
)

func TestResolveName(t *testing.T) {
	for in, want := range map[string]string{"smith": boss.NameSmith, "Countess": boss.NameCountess, "The Smith": boss.NameSmith, "ANDARIEL": boss.NameAndariel, "bloodraven": boss.NameBloodRaven} {
		if got, ok := resolveName(in); !ok || got != want {
			t.Errorf("%q -> %q %v", in, got, ok)
		}
	}
	if _, ok := resolveName("diablo"); ok {
		t.Error("unknown boss accepted")
	}
}

func TestAliasesHaveScripts(t *testing.T) {
	for a, n := range aliases {
		if _, ok := boss.Act1()[n]; !ok {
			t.Errorf("alias %s: no script %q", a, n)
		}
	}
}

func TestExitCode(t *testing.T) {
	cases := []struct {
		res  boss.Result
		err  error
		want int
	}{
		{boss.Result{Killed: true}, nil, exitKilled},
		{boss.Result{}, nil, exitFailed},
		{boss.Result{}, fmt.Errorf("x: %w", boss.ErrTimeout), exitTimeout},
		{boss.Result{}, context.DeadlineExceeded, exitTimeout},
		{boss.Result{}, boss.ErrDead, exitDied},
		{boss.Result{}, boss.ErrEscaped, exitFailed},
		{boss.Result{Killed: true}, boss.ErrChicken, exitFailed},
	}
	for i, c := range cases {
		if got := exitCode(c.res, c.err); got != c.want {
			t.Errorf("case %d: %d want %d", i, got, c.want)
		}
	}
}

type fakeKiller struct {
	res boss.Result
	err error
}

func (f *fakeKiller) KillBoss(context.Context, string) error { return f.err }

func TestRunKillPrintsResult(t *testing.T) {
	f := &fakeKiller{res: boss.Result{Killed: true, Loot: 2, Phases: []string{"locate", "attack", "loot"}, Notes: []string{"n1"}}}
	var b bytes.Buffer
	if code := runKill(context.Background(), &b, f, func() boss.Result { return f.res }, "The Smith"); code != exitKilled {
		t.Fatalf("code %d", code)
	}
	for _, w := range []string{`boss="The Smith"`, "killed=true", "loot=2", "locate,attack,loot", "note: n1", "exit: 0"} {
		if !strings.Contains(b.String(), w) {
			t.Errorf("missing %q in %q", w, b.String())
		}
	}
	f.err = errors.New("boom")
	b.Reset()
	if code := runKill(context.Background(), &b, f, func() boss.Result { return boss.Result{} }, "x"); code != exitFailed || !strings.Contains(b.String(), "err: boom") {
		t.Fatalf("code %d %q", code, b.String())
	}
}
