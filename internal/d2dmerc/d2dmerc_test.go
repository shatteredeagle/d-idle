package d2dmerc

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dctl/fake"
	"d2lod-bot/internal/d2dsrc"
	"d2lod-bot/internal/merc"
)

type noTables struct{}

func (noTables) ItemRow(string) (int, bool)      { return 0, false }
func (noTables) MonsterClass(string) (int, bool) { return 0, false }

func fixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "d2dsrc", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := json.Compact(&out, b); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

// source serves the recorded d2dsrc fixtures through a unix socket; head replaces the dump head line.
func source(t *testing.T, head string) *d2dsrc.Source {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "d.sock")
	ln, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	f := fake.Sample()
	pl := func(ls []string) string { return strings.Join(ls, "\n") + "\n" }
	reply := map[string]string{
		"info": "w=800 h=600 frame=100\nok\n", "state": f.State + "\nok\n",
		"monsters": pl(f.Monsters) + "ok\n", "npcs": pl(f.NPCs) + "ok\n", "ground": pl(f.Ground) + "ok\n",
		"menu": pl(f.Menu) + "ok\n", "items": pl(f.Items) + "ok\n",
		"debug players": pl(f.Players) + "ok\n", "debug portals": pl(f.Portals) + "ok\n",
		"debug corpses": pl(f.Corpses) + "ok\n", "debug warps": pl(f.Warps) + "ok\n", "debug level": "ok " + f.Level + "\n",
		"dump player,stats,skills,ui,merc,time": head + "\nok\n",
		"dump items":                            fixture(t, "dump_items_late.json") + "\nok\n",
		"dump quests":                           fixture(t, "dump_quests.json") + "\nok\n",
	}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer c.Close()
				rd := bufio.NewReader(c)
				for {
					line, err := rd.ReadString('\n')
					if err != nil {
						return
					}
					line = strings.TrimSpace(line)
					r, ok := reply[line]
					if !ok {
						r = "err unknown verb '" + strings.Fields(line)[0] + "'\n"
					}
					c.Write([]byte(r))
				}
			}()
		}
	}()
	cl := d2dctl.New(path, d2dctl.Options{})
	t.Cleanup(func() { cl.Close() })
	return d2dsrc.New(cl, d2dsrc.Options{Tables: noTables{}, Logf: func(string, ...any) {}})
}

func state(t *testing.T, head string, est func(*d2.Snapshot) int) merc.MercState {
	t.Helper()
	src := source(t, head)
	r := &Reader{Read: src.Read, LevelEstimate: est}
	st, err := r.State()
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func TestRecordedDumps(t *testing.T) {
	late := fixture(t, "dump_head_late.json") // merc RG alive 63/80
	st := state(t, late, nil)
	if !st.Known || !st.Owned || !st.Alive || st.Class != merc.ClassRogue || st.LifePct != 78 || st.Level != 0 || st.StatsKnown || st.FreeCells != -1 {
		t.Errorf("alive: %+v", st)
	}
	dead := strings.Replace(late, `"dead":false`, `"dead":true`, 1)
	st = state(t, dead, nil)
	if !st.Known || !st.Owned || st.Alive || st.LifePct != 0 || st.Class != merc.ClassRogue {
		t.Errorf("dead: %+v", st)
	}
	none := state(t, fixture(t, "dump_head_necro.json"), nil) // merc present=false
	if !none.Known || none.Owned {
		t.Errorf("none: %+v", none)
	}
}

func TestUnknownCodeKeepsOwned(t *testing.T) {
	odd := strings.Replace(fixture(t, "dump_head_late.json"), `"code":"RG"`, `"code":"A2"`, 1)
	st := state(t, odd, nil)
	if !st.Owned || st.Class != 0 {
		t.Errorf("%+v", st) // owned and alive, class unknown (only RG is V)
	}
}

func TestLevelEstimateFeedsRevive(t *testing.T) {
	dead := strings.Replace(fixture(t, "dump_head_late.json"), `"dead":false`, `"dead":true`, 1)
	k := merc.DefaultKnobs()
	ctx := merc.Ctx{Gold: 10000, Kit: 0, HaveTP: true, HavePotions: true, Knobs: k}
	st := state(t, dead, nil)
	if d, _ := merc.ShouldRevive(st, ctx, 0); d.Do || d.Reason != merc.ReasonNoLevel {
		t.Errorf("no estimate must refuse: %+v", d)
	}
	st = state(t, dead, func(s *d2.Snapshot) int { return s.Player.Level })
	d, price := merc.ShouldRevive(st, ctx, 0)
	if !d.Do || price != merc.ReviveCost(24) {
		t.Errorf("estimate: %+v price %d", d, price)
	}
}

// Stock d2d has no dump: the merc section is unknown, so State is Known=false and the policy fails closed.
func TestStockD2DIsUnknown(t *testing.T) {
	dir := t.TempDir()
	srv, err := fake.New(filepath.Join(dir, "f.sock"), fake.Sample(), fake.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	cl := d2dctl.New(srv.Path(), d2dctl.Options{})
	defer cl.Close()
	src := d2dsrc.New(cl, d2dsrc.Options{Tables: noTables{}, Logf: func(string, ...any) {}})
	st, err := (&Reader{Read: src.Read}).State()
	if err != nil {
		t.Fatal(err)
	}
	if st.Known {
		t.Errorf("stock d2d must be unknown: %+v", st)
	}
	if _, err := (&Reader{Read: src.Read}).Offers(); err != ErrNoOffers {
		t.Error("offers")
	}
}

func TestActsAgainstFake(t *testing.T) {
	srv, err := fake.New(filepath.Join(t.TempDir(), "f.sock"), fake.Sample(), fake.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	cl := d2dctl.New(srv.Path(), d2dctl.Options{})
	defer cl.Close()
	ctx := context.Background()
	// the fake's Kashya menu has talk and cancel only: no resurrect line, nothing is clicked but cancel
	if _, err := (&Reviver{C: cl, Timeout: 3 * time.Second}).Revive(ctx); !errors.Is(err, ErrNoReviveLine) {
		t.Errorf("revive: %v", err)
	}
	// the fake never grants a merc (state merc=-): the claim reports it, with the burial states
	if g, err := (Claimer{C: cl}).ClaimFreeRogue(ctx); g || err == nil {
		t.Errorf("claim: %v %v", g, err)
	}
}
