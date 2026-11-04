package bot

import (
	"log/slog"
	"strings"
	"testing"

	"d2lod-bot/internal/d2"
)

func TestMinuteLineLifeFields(t *testing.T) {
	g := newRig(t)
	buf := &lockedBuf{}
	g.r.e.Log = slog.New(slog.NewTextHandler(buf, nil))
	s := beltSnap(3, 2)
	s.Player.Life, s.Player.MaxLife, s.Player.Mana, s.Player.MaxMana = 61, 120, 30, 45
	g.r.obMu.Lock()
	g.r.minuteLine(s, s.Player, g.r.e.Clock.Now())
	g.r.obMu.Unlock()
	l := lastMinute(buf)
	for _, w := range []string{" life=61", " maxLife=120", " mp=30", " maxMana=45", " mana=0", " potions=3", " hpMinPct="} {
		if !strings.Contains(l, w) {
			t.Fatalf("%q missing: %s", w, l)
		}
	}
}

func TestZoneEntryAndSafetyLines(t *testing.T) {
	g := newRig(t)
	buf := &lockedBuf{}
	g.r.e.Log = slog.New(slog.NewTextHandler(buf, nil))
	for _, class := range []d2.Class{d2.Sorceress, d2.Barbarian} {
		s := beltSnap(2, 5)
		s.Player.Class, s.Player.Life, s.Player.MaxLife, s.Player.Mana, s.Player.MaxMana = class, 40, 90, 7, 11
		g.r.zoneEntry(s, s.Player)
		if got := buf.String(); !strings.Contains(got, `msg="zone entry" area=5 clvl=9 life=40/90 heal=2 mana=7/11 tome=0 merc=none`) {
			t.Fatalf("class %v: %s", class, got)
		}
	}
	s := beltSnap(1, 5)
	if kv := g.r.vitalsKV(s, s.Player); len(kv) != 12 || kv[0] != "life" || kv[3] != 100 {
		t.Fatalf("vitals %v", kv)
	}
}
