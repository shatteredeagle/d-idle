package main

import (
	"context"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/cheat"
	"d2lod-bot/internal/cheat/d2s"
	"d2lod-bot/internal/mem"
)

// synthSave builds a minimal v96 save of the given class (no real save data).
func synthSave(name string, class byte) []byte {
	head := make([]byte, 0x2FD)
	binary.LittleEndian.PutUint32(head[0:], 0xAA55AA55)
	binary.LittleEndian.PutUint32(head[4:], 0x60)
	copy(head[0x14:], name)
	head[0x24] = 0x20
	head[0x28] = class
	head[0x2B] = 1
	head[0xA8] = 0x80
	copy(head[0x14F:], "Woo!")
	copy(head[0x279:], "WS")
	copy(head[0x2CA:], "w4")
	var bits []byte
	pos := uint(0)
	put := func(v uint32, n uint) {
		for i := uint(0); i < n; i++ {
			if pos%8 == 0 {
				bits = append(bits, 0)
			}
			if v>>i&1 == 1 {
				bits[len(bits)-1] |= 1 << (pos % 8)
			}
			pos++
		}
	}
	put(0, 9)
	put(10, 10)
	put(0x1FF, 9)
	out := append(head, 'g', 'f')
	out = append(out, bits...)
	out = append(out, 'i', 'f')
	out = append(out, make([]byte, 30)...)
	out = append(out, 'J', 'M', 0, 0, 'j', 'f', 0, 0, 'k', 'f', 0)
	binary.LittleEndian.PutUint32(out[8:], uint32(len(out)))
	var sum uint32
	for i, c := range out {
		if i >= 12 && i < 16 {
			c = 0
		}
		sum = sum<<1 | sum>>31
		sum += uint32(c)
	}
	binary.LittleEndian.PutUint32(out[12:], sum)
	return out
}

func fakeTables() skillTables {
	skills := map[string]struct {
		id   int
		code string
	}{"fire bolt": {36, "sor"}, "bash": {126, "bar"}}
	codes := []string{"ama", "sor", "nec", "pal", "bar", "dru", "ass"}
	return skillTables{
		skill: func(n string) (int, string, bool) {
			s, ok := skills[strings.ToLower(n)]
			return s.id, s.code, ok
		},
		classCode: func(c int) string {
			if c < 0 || c >= len(codes) {
				return ""
			}
			return codes[c]
		},
	}
}

func testEnv(t *testing.T) *cheat.Env {
	dir := t.TempDir()
	return &cheat.Env{
		Ledger:    cheat.NewLedger(filepath.Join(dir, "state", "cheats.jsonl")),
		Instance:  "i5",
		Now:       func() time.Time { return time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC) },
		SaveDir:   filepath.Join(dir, "saves"),
		BackupDir: filepath.Join(dir, "backups"),
		InGame:    func() (bool, error) { return false, nil },
		Open:      func() (mem.Writer, func(), error) { return nil, nil, errors.New("no game in tests") },
	}
}

func put(t *testing.T, env *cheat.Env, name string, class byte) {
	t.Helper()
	if err := os.MkdirAll(env.SaveDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(env.SaveDir, name+".d2s"), synthSave(name, class), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSkillLevelByNameWritesSaveAndLedger(t *testing.T) {
	env := testEnv(t)
	put(t, env, "Combi", 1)
	req, err := skillLevelRequest(fakeTables(), env, "Combi", "Fire Bolt", 13, "SK setup: skill level 13", "tester")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.Do(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(env.SaveDir, "Combi.d2s"))
	f, err := d2s.Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	if got := f.SkillLevel(36); got != 13 {
		t.Fatalf("skill level %d", got)
	}
	es, _ := env.Ledger.List(cheat.Filter{Kind: cheat.KindSaveEdit, OnlySuccess: true})
	if len(es) != 1 || es[0].Reason == "" || es[0].Params["skillname"] != "Fire Bolt" {
		t.Fatalf("ledger %+v", es)
	}
}

func TestSkillLevelRefusals(t *testing.T) {
	env := testEnv(t)
	put(t, env, "Combi", 1)
	put(t, env, "Stranger", 1)
	tb := fakeTables()
	for name, c := range map[string]struct{ file, skill string }{
		"unknown skill": {"Combi", "Fire Bolte"},
		"wrong class":   {"Combi", "Bash"},
	} {
		if _, err := skillLevelRequest(tb, env, c.file, c.skill, 5, "x", "t"); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	if _, err := skillLevelRequest(tb, env, "Combi", "Fire Bolt", 300, "x", "t"); err == nil {
		t.Error("level 300 accepted")
	}
	if _, err := skillLevelRequest(tb, env, "Nobody", "Fire Bolt", 5, "x", "t"); err == nil {
		t.Error("missing save accepted")
	}
	put(t, env, "Weird", 9)
	if _, err := skillLevelRequest(tb, env, "Weird", "Fire Bolt", 5, "x", "t"); err == nil || !strings.Contains(err.Error(), "class") {
		t.Errorf("unknown class: %v", err)
	}
	// non-throwaway: request builds, Do refuses and ledgers
	req, err := skillLevelRequest(tb, env, "Stranger", "Fire Bolt", 5, "x", "t")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.Do(context.Background(), req); !errors.Is(err, cheat.ErrNotAllowed) {
		t.Fatalf("non-throwaway: %v", err)
	}
	// no reason: refused, ledgered
	req.Reason = ""
	if _, err := env.Do(context.Background(), req); !errors.Is(err, cheat.ErrNoReason) {
		t.Fatalf("no reason: %v", err)
	}
	es, _ := env.Ledger.List(cheat.Filter{})
	if len(es) != 2 {
		t.Fatalf("want 2 ledger refusals, got %d", len(es))
	}
}
