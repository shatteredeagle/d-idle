package d2dctl_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dctl/fake"
)

// The journal records every command with its tick and a reply digest, skips the health verbs, and takes a checkpoint
// on the same connection when a step crosses the interval, so two identical runs give byte-identical journals.
func TestJournalRecordsAndCheckpoints(t *testing.T) {
	old := d2dctl.CheckpointVerbs
	d2dctl.CheckpointVerbs = []string{"monsters", "ground"} // the fake has no dump verb
	t.Cleanup(func() { d2dctl.CheckpointVerbs = old })

	run := func(name string) string {
		srv, c := start(t, fake.Options{})
		path := filepath.Join(t.TempDir(), name)
		if err := c.OpenJournal(path, 100); err != nil {
			t.Fatal(err)
		}
		_ = c.Ping(ctx())
		if _, err := c.Monsters(ctx()); err != nil {
			t.Fatal(err)
		}
		if _, err := c.Step(ctx(), 60); err != nil {
			t.Fatal(err)
		}
		if err := c.Prefetch(ctx(), "state", "npcs"); err != nil {
			t.Fatal(err)
		}
		_, _, _ = c.State(ctx())
		c.DropPrefetch()
		if _, err := c.Step(ctx(), 60); err != nil { // tick 120 crosses 100: checkpoint
			t.Fatal(err)
		}
		if _, err := c.Do(ctx(), "debug nosuch"); err == nil {
			t.Fatal("want err reply")
		}
		if err := c.CloseJournal(); err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		// the checkpoint verbs went over the wire, between the step and the next command
		lines := srv.Lines()
		want := []string{"ping", "monsters", "step 60", "state", "npcs", "step 60", "monsters", "ground", "debug nosuch"}
		if strings.Join(lines, ",") != strings.Join(want, ",") {
			t.Fatalf("wire: %v", lines)
		}
		return string(b)
	}
	a := run("a.jnl")
	b := run("b.jnl")
	if a != b {
		t.Fatalf("two identical runs differ:\n%s\n---\n%s", a, b)
	}
	want := []string{
		"# d2dctl journal v1 every=100 verbs=monsters;ground",
		">\t0\tmonsters\t",
		">\t60\tstep 60\ttick=60",
		">\t60\tstate\t",
		">\t60\tnpcs\t",
		">\t120\tstep 60\ttick=120",
		"C\t120\t",
		">\t120\tdebug nosuch\terr:unknown verb",
	}
	got := strings.Split(strings.TrimRight(a, "\n"), "\n")
	if len(got) != len(want) {
		t.Fatalf("lines %d want %d:\n%s", len(got), len(want), a)
	}
	for i := range want {
		if !strings.HasPrefix(got[i], want[i]) {
			t.Errorf("line %d: %q want prefix %q", i, got[i], want[i])
		}
	}
	if strings.Contains(a, "ping") {
		t.Error("ping recorded")
	}
}

func TestReplyHashIgnoresDumpFrame(t *testing.T) {
	x := &d2dctl.Reply{Lines: []string{`{"v":1,"time":{"frame":120,"game_ms":80,"tick":2},"player":{"x":1}}`}}
	y := &d2dctl.Reply{Lines: []string{`{"v":1,"time":{"frame":7731,"game_ms":80,"tick":2},"player":{"x":1}}`}}
	z := &d2dctl.Reply{Lines: []string{`{"v":1,"time":{"frame":120,"game_ms":80,"tick":3},"player":{"x":1}}`}}
	if d2dctl.ReplyHash(x) != d2dctl.ReplyHash(y) {
		t.Error("frame counted")
	}
	if d2dctl.ReplyHash(x) == d2dctl.ReplyHash(z) {
		t.Error("tick not counted")
	}
	s1 := &d2dctl.Reply{Lines: []string{"screen=ingame cam=30.70,13.70 walking=0 gold=0"}}
	s2 := &d2dctl.Reply{Lines: []string{"screen=ingame cam=30.91,13.82 walking=0 gold=0"}}
	s3 := &d2dctl.Reply{Lines: []string{"screen=ingame cam=30.70,13.70 walking=1 gold=0"}}
	if d2dctl.ReplyHash(s1) != d2dctl.ReplyHash(s2) || d2dctl.ReplyHash(s1) == d2dctl.ReplyHash(s3) {
		t.Error("cam masking")
	}
}
