package goals

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPersistenceRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "state.json")
	m, _, c := newMgr(t, path)
	mustAdd(t, m, lvl(30))
	mustAdd(t, m, Goal{Type: Idle, Schedule: []Window{{Days: []int{1, 2}, Start: "22:00", End: "06:00"}}})
	m.Start()
	m.Handle(Event{Type: EvKill, Monster: "Fallen", XP: 40})
	m.Handle(Event{Type: EvDrop, Item: &Item{Name: "Tir Rune", Quality: "normal", Type: "rune"}})
	m.Handle(Event{Type: EvAreaChange, Area: "Blood Moor"})
	c.advance(5 * time.Second)
	m.Tick(c.now())
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}

	m2, f2, _ := newMgr(t, path)
	gs := m2.Goals()
	if len(gs) != 2 || gs[0].Title != "Reach level 30" || len(gs[1].Schedule) != 1 || gs[1].Schedule[0].Days[1] != 2 {
		t.Fatalf("goals: %+v", gs)
	}
	if gs[0].State != StateQueued || gs[0].Progress.Kills != 1 || gs[0].Progress.ActiveSeconds != 5 {
		t.Fatalf("active goal should come back queued with progress: %+v", gs[0])
	}
	st := m2.StatsSummary()
	if st.Totals.Kills != 1 || st.Totals.XP != 40 || len(st.Drops) != 1 || st.Qualities["normal"] != 1 || len(st.Areas) != 1 {
		t.Fatalf("stats: %+v", st)
	}
	if m2.Snapshot().Mode != ModeStopped || len(f2.started) != 0 {
		t.Fatal("must not start playing after a restart unless autoResume is set")
	}
	g := mustAdd(t, m2, lvl(40))
	if g.ID != "g3" {
		t.Fatalf("ids must continue: %s", g.ID)
	}
	if _, err := os.Stat(path + ".tmp"); err == nil {
		t.Fatal("tmp file left behind")
	}
}

func writeValidFile(t *testing.T, path string, titleLevel int) []byte {
	t.Helper()
	m, _, _ := newMgr(t, path)
	mustAdd(t, m, lvl(titleLevel))
	m.Close()
	os.Remove(path + ".bak")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestCrashAndCorruptionRecovery(t *testing.T) {
	good := func(t *testing.T, dir string) (string, []byte) {
		p := filepath.Join(dir, "state.json")
		return p, writeValidFile(t, p, 11)
	}
	tests := []struct {
		name     string
		prepare  func(t *testing.T, path string, valid []byte)
		wantGoal string // "" = empty queue
		warn     string
	}{
		{"truncated main, no backup", func(t *testing.T, p string, v []byte) {
			os.WriteFile(p, v[:len(v)/2], 0o600)
		}, "", "unreadable"},
		{"truncated main, good backup", func(t *testing.T, p string, v []byte) {
			os.WriteFile(p+".bak", v, 0o600)
			os.WriteFile(p, v[:len(v)/2], 0o600)
		}, "Reach level 11", "restored from backup"},
		{"garbage main, good backup", func(t *testing.T, p string, v []byte) {
			os.WriteFile(p+".bak", v, 0o600)
			os.WriteFile(p, []byte("\x00\x00\x00 not json"), 0o600)
		}, "Reach level 11", "restored from backup"},
		{"empty main", func(t *testing.T, p string, v []byte) {
			os.WriteFile(p, nil, 0o600)
		}, "", "unreadable"},
		{"crash between renames: only backup and tmp", func(t *testing.T, p string, v []byte) {
			os.Rename(p, p+".bak")
			os.WriteFile(p+".tmp", v[:10], 0o600)
		}, "Reach level 11", "restored from backup"},
		{"both corrupt", func(t *testing.T, p string, v []byte) {
			os.WriteFile(p, v[:5], 0o600)
			os.WriteFile(p+".bak", v[:7], 0o600)
		}, "", "unreadable"},
		{"valid json but no version", func(t *testing.T, p string, v []byte) {
			os.WriteFile(p, []byte(`{"goals":[]}`), 0o600)
		}, "", "missing version"},
		{"stale tmp is ignored", func(t *testing.T, p string, v []byte) {
			os.WriteFile(p+".tmp", []byte("{"), 0o600)
		}, "Reach level 11", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			p, valid := good(t, dir)
			tc.prepare(t, p, valid)
			m, _, _ := newMgr(t, p)
			gs := m.Goals()
			if tc.wantGoal == "" && len(gs) != 0 || tc.wantGoal != "" && (len(gs) != 1 || gs[0].Title != tc.wantGoal) {
				t.Fatalf("goals = %+v", gs)
			}
			w := m.Snapshot().LoadWarning
			if tc.warn == "" && w != "" || !strings.Contains(w, tc.warn) {
				t.Fatalf("warning = %q, want %q", w, tc.warn)
			}
			// the manager is usable and saves cleanly afterwards
			mustAdd(t, m, lvl(12))
			if err := m.Close(); err != nil {
				t.Fatal(err)
			}
			m2, _, _ := newMgr(t, p)
			if len(m2.Goals()) != len(gs)+1 || m2.Snapshot().LoadWarning != "" {
				t.Fatalf("after recovery: %d goals, warning %q", len(m2.Goals()), m2.Snapshot().LoadWarning)
			}
		})
	}
}

func TestCorruptFileIsKeptAside(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "state.json")
	os.WriteFile(p, []byte("{broken"), 0o600)
	newMgr(t, p)
	m, _ := filepath.Glob(p + ".corrupt-*")
	if len(m) != 1 {
		t.Fatalf("corrupt copies: %v", m)
	}
	if b, _ := os.ReadFile(m[0]); string(b) != "{broken" {
		t.Fatal("the corrupt file must be preserved for inspection")
	}
	for i := 0; i < 6; i++ {
		os.WriteFile(p, []byte("{broken"), 0o600)
		os.Rename(p, p+".corrupt-"+string(rune('a'+i)))
	}
	os.WriteFile(p, []byte("{broken"), 0o600)
	newMgr(t, p)
	if m, _ := filepath.Glob(p + ".corrupt-*"); len(m) != 3 {
		t.Fatalf("pruned to %d, want 3", len(m))
	}
}

func TestFutureVersionRefused(t *testing.T) {
	p := filepath.Join(t.TempDir(), "state.json")
	data := []byte(`{"version": 99, "goals": []}`)
	os.WriteFile(p, data, 0o600)
	_, err := New(Options{Runner: newFake(), Path: p})
	if !errors.Is(err, ErrFutureVersion) {
		t.Fatalf("err = %v", err)
	}
	if b, _ := os.ReadFile(p); string(b) != string(data) {
		t.Fatal("a newer file must be left untouched")
	}
	if m, _ := filepath.Glob(p + "*"); len(m) != 1 {
		t.Fatalf("nothing else may be created: %v", m)
	}
}

func TestMigration(t *testing.T) {
	// Pretend the current version is 2 with a migration that renames a field; here we only test the
	// machinery with a v0-free table by faking an old doc through the migrations hook.
	// Build a doc at "version 1" and run decodeState: no migration is needed.
	doc := map[string]any{"version": 1, "nextId": 3, "mode": "stopped", "goals": []any{map[string]any{
		"id": "g3", "type": "reach_level", "title": "Reach level 9", "params": map[string]any{"level": 9}, "state": "queued"}}}
	b, _ := json.Marshal(doc)
	st, err := decodeState(b)
	if err != nil || len(st.Goals) != 1 {
		t.Fatalf("%v %+v", err, st)
	}
	// A missing migration step is reported rather than silently accepted. Simulate by decoding as if the
	// build were v2.
	if _, err := decodeStateAt(b, 2, map[int]func(map[string]json.RawMessage) error{}); err == nil || !strings.Contains(err.Error(), "no migration") {
		t.Fatalf("err = %v", err)
	}
	// And a present one is applied.
	mig := map[int]func(map[string]json.RawMessage) error{1: func(raw map[string]json.RawMessage) error {
		raw["nextId"] = json.RawMessage("42")
		return nil
	}}
	st, err = decodeStateAt(b, 2, mig)
	if err != nil || st.NextID != 42 {
		t.Fatalf("%v %+v", err, st)
	}
}

func TestInvalidGoalsInFileAreDropped(t *testing.T) {
	p := filepath.Join(t.TempDir(), "state.json")
	doc := `{"version":1,"nextId":2,"mode":"stopped","goals":[
	 {"id":"g1","type":"reach_level","params":{"level":9},"state":"active"},
	 {"id":"g2","type":"reach_level","params":{"level":900},"state":"queued"},
	 null, {"id":"","type":"idle"}]}`
	os.WriteFile(p, []byte(doc), 0o600)
	m, _, _ := newMgr(t, p)
	gs := m.Goals()
	if len(gs) != 1 || gs[0].ID != "g1" || gs[0].State != StateQueued {
		t.Fatalf("%+v", gs)
	}
	if !strings.Contains(m.Snapshot().LoadWarning, "3 invalid") {
		t.Fatalf("warning %q", m.Snapshot().LoadWarning)
	}
}

func TestWriteAtomicKeepsBackup(t *testing.T) {
	p := filepath.Join(t.TempDir(), "f.json")
	if err := writeAtomic(p, []byte("one")); err != nil {
		t.Fatal(err)
	}
	if err := writeAtomic(p, []byte("two")); err != nil {
		t.Fatal(err)
	}
	a, _ := os.ReadFile(p)
	b, _ := os.ReadFile(p + ".bak")
	if string(a) != "two" || string(b) != "one" {
		t.Fatalf("%s %s", a, b)
	}
}

func TestProgressOptionalFieldsAreAdditive(t *testing.T) {
	b, _ := json.Marshal(Progress{Current: 1, Target: 2})
	for _, k := range []string{"steps", "phase", "etaSeconds", "source", "verified"} {
		if strings.Contains(string(b), `"`+k+`"`) {
			t.Fatalf("empty %s serialised: %s", k, b)
		}
	}
	in := Progress{Current: 3, Target: 5, Phase: "fight", ETASeconds: 90, Source: "memory", Verified: true,
		Steps: []Step{{ID: "s1", Title: "seal 1", State: StateDone, Current: 1, Target: 1}}}
	b, _ = json.Marshal(in)
	var out Progress
	if err := json.Unmarshal(b, &out); err != nil || out.Phase != "fight" || out.ETASeconds != 90 || !out.Verified || len(out.Steps) != 1 || out.Steps[0].State != StateDone {
		t.Fatalf("round trip %s %+v %v", b, out, err)
	}
	// a version 1 file without the new fields still decodes
	if err := json.Unmarshal([]byte(`{"current":1,"target":2,"percent":50,"kills":0,"bossKills":0,"drops":0,"deaths":0,"activeSeconds":3}`), &out); err != nil {
		t.Fatal(err)
	}
}
