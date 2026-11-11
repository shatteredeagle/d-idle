package town_test

import (
	"context"
	"strings"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/gear/geartest"
)

// The player's identify commands against the town simulator (TASK-298). Like the other sim tests these SKIP without the
// game tables (docs/data.md, $D2_MPQ_DIR), so they did not run on the machine that wrote them (U); the money rules and
// the Cain menu lines are covered table-driven in playerid_test.go.

func unidRing(r *rig) d2.Unit {
	u := r.W.Add("rin", d2.ContInventory, 0, 0)
	u.Item.Quality = d2.QualityMagic
	u.Item.Flags &^= d2.FlagIdentified
	return u
}

func TestPlayerIdentifyUsesAScrollAndRefusesWithoutOne(t *testing.T) {
	for _, cls := range []d2.Class{d2.Sorceress, d2.Paladin} {
		geartest.Need(t)
		r := newRig(t)
		r.W.Player.Class = cls
		ring := unidRing(r)
		s, _ := r.W.Snapshot()
		if _, err := r.T.CheckPlayerIdentify(s, ring.ID); err == nil || !strings.Contains(err.Error(), "Identify all (Cain)") {
			t.Fatalf("class %v: without a scroll the check must name the other commands: %v", cls, err)
		}
		r.W.Add("isc", d2.ContInventory, 5, 0)
		s, _ = r.W.Snapshot()
		if _, err := r.T.CheckPlayerIdentify(s, ring.ID); err != nil {
			t.Fatalf("class %v: %v", cls, err)
		}
		if err := r.T.PlayerIdentify(context.Background(), ring.ID); err != nil {
			t.Fatalf("class %v: %v", cls, err)
		}
		s, _ = r.W.Snapshot()
		for _, u := range s.Inventory.Items {
			if u.ID == ring.ID && !u.Item.Identified() {
				t.Fatalf("class %v: the ring is still unidentified", cls)
			}
		}
		if _, err := r.T.CheckPlayerIdentify(s, ring.ID); err == nil {
			t.Fatalf("class %v: an identified item must be refused", cls)
		}
	}
}

func TestBuyIdentifyFillsTheTomeAndKeepsTheReserve(t *testing.T) {
	for _, cls := range []d2.Class{d2.Sorceress, d2.Paladin} {
		geartest.Need(t)
		r := newRig(t)
		r.W.Player.Class = cls
		r.W.Player.Gold = 5000
		tome := r.W.Add("ibk", d2.ContInventory, 9, 0)
		r.W.Stats[tome.Addr] = d2.Stats{70: 3}
		msg, err := r.T.BuyIdentify(context.Background(), 0)
		if err != nil || !strings.Contains(msg, "bought 17") {
			t.Fatalf("class %v: %q %v", cls, msg, err)
		}
		if q := r.W.Stats[tome.Addr][70]; q != 20 {
			t.Fatalf("class %v: tome holds %d, want 20", cls, q)
		}
		// poor: only part of the tome is bought and the purse never drops under the optional-buy reserve
		r.W.Stats[tome.Addr] = d2.Stats{70: 0}
		r.W.Player.Gold = 330
		msg, err = r.T.BuyIdentify(context.Background(), 0)
		if err == nil && !strings.Contains(msg, "short of gold") {
			t.Fatalf("class %v: a poor hero must report short of gold: %q", cls, msg)
		}
		if r.W.Player.Gold < r.Env.Tune.Town.MinGoldReserve {
			t.Fatalf("class %v: gold %d fell under the reserve", cls, r.W.Player.Gold)
		}
	}
}
