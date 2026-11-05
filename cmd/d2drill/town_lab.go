package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/drill"
	"d2lod-bot/internal/town"
)

// The D3 drills call the bot's own town code (town.Town: OpenForTest, BuyForTest, SellForTest, Chores,
// OpenStashForTest, DepositForTest, the same functions cmd/townlive and the chore run use); the lab only
// reads state and applies logged cheats.

func (l *lab) TView(ctx context.Context) (drill.TownView, error) {
	s, err := l.be.Env.Snapshot()
	if err != nil {
		return drill.TownView{}, err
	}
	if s.Player == nil || s.Inventory == nil {
		return drill.TownView{}, errors.New("no player or inventory in the snapshot")
	}
	v := drill.TownView{Area: s.Player.Area, InTown: s.Player.InTown(), Gold: s.Player.Gold, Stash: s.Player.StashGold,
		Life: s.Player.Life, Max: s.Player.MaxLife, Pack: map[string]int{}, DurCur: -1, Class: fmt.Sprint(s.Player.Class)}
	for _, u := range s.Inventory.Container(d2.ContInventory) {
		v.Pack[l.code(u.Class)]++
	}
	v.Belt = len(s.Inventory.Container(d2.ContBelt))
	if _, st, err := l.be.Client.Gold(ctx); err == nil {
		v.Stash = st
	}
	if c, m, err := l.be.Client.Durability(ctx); err == nil {
		v.DurCur, v.DurMax = c, m
	}
	v.Unid, v.UnidCodes = l.unid(ctx)
	v.Qty = l.quantities(ctx)
	if u, err := l.UI(ctx); err == nil {
		v.Panels = u.Panels
	}
	return v, nil
}

func (l *lab) SetGold(ctx context.Context, r *drill.Run, purse int) error {
	_, err := r.Cheat(ctx, fmt.Sprintf("debug stat 14 %d", purse), fmt.Sprintf("set the purse to %d gold: the drill's gold supply", purse))
	if err != nil {
		return err
	}
	return l.Tick(ctx, 5)
}

func (l *lab) SetStashGold(ctx context.Context, r *drill.Run, n int) error {
	_, err := r.Cheat(ctx, fmt.Sprintf("debug stat 15 %d", n), fmt.Sprintf("set the stash gold to %d so the deposit/withdraw sums start exact", n))
	if err != nil {
		return err
	}
	return l.Tick(ctx, 5)
}

func (l *lab) Wear(ctx context.Context, r *drill.Run) error {
	_, err := r.Cheat(ctx, "debug durability all 1", "set every worn item to durability 1 (about 5 percent, below the bot's repair knob of 30): the repair drill needs damaged gear")
	return err
}

func (l *lab) OpenNPC(ctx context.Context, class int) drill.OpenInfo {
	oi := drill.OpenInfo{Class: class}
	t0 := l.tick()
	_, err := l.tw.OpenForTest(ctx, class)
	oi.Ticks = l.tick() - t0
	if err != nil {
		oi.Err = err.Error()
	}
	if u, uerr := l.UI(ctx); uerr == nil {
		oi.Flag = u.Panels
	}
	return oi
}

func (l *lab) CloseAll(ctx context.Context) error {
	if err := l.be.Ctl.CloseAll(); err != nil {
		return err
	}
	_, err := l.be.Client.Step(ctx, 15)
	return err
}

// OpenVendor opens the NPC's dialog and its Trade line (the bot's OpenForTest and PickEntryForTest). It leaves the
// bot's cached vendor tab alone, so the drills never buy through it (town.Town.openShop resets that cache; the
// ForTest hooks do not); selling does not use tabs.
func (l *lab) OpenVendor(ctx context.Context, class int) error {
	npc, err := l.tw.OpenForTest(ctx, class)
	if err != nil {
		return fmt.Errorf("open vendor: %w", err)
	}
	if err := l.tw.PickEntryForTest(ctx, npc, town.EntryTrade, d2.UIShop); err != nil {
		return fmt.Errorf("pick trade: %w", err)
	}
	return nil
}

func (l *lab) SellPack(ctx context.Context, code string) (int, error) {
	s, err := l.be.Env.Snapshot()
	if err != nil || s.Inventory == nil {
		return 0, fmt.Errorf("no inventory: %v", err)
	}
	for _, u := range s.Inventory.Container(d2.ContInventory) {
		if l.code(u.Class) == code {
			return l.tw.SellForTest(ctx, u)
		}
	}
	return 0, fmt.Errorf("no %s in the pack", code)
}

func (l *lab) Chores(ctx context.Context, plan string) drill.ChoreInfo {
	var p town.ChorePlan
	for _, w := range strings.Fields(plan) {
		switch w {
		case "heal":
			p.Heal = true
		case "repair":
			p.Repair = true
		case "scrolls":
			p.Scrolls = true
		case "potions":
			p.Potions = true
		case "stash":
			p.Stash = true
		case "gold":
			p.Gold = true
		case "force":
			p.Force = true
		case "sell":
			p.Sell = true
		case "identify":
			p.Identify = true
		}
	}
	mark := l.rec.Mark()
	t0 := l.tick()
	rep, err := l.tw.Chores(ctx, p)
	ci := drill.ChoreInfo{Plan: plan, Bought: rep.Bought, Spent: rep.Spent, Earned: rep.Earned, GoldBefore: rep.GoldBefore, GoldAfter: rep.GoldAfter,
		Healed: rep.Healed, Repaired: rep.Repaired, Ticks: l.tick() - t0}
	if err != nil {
		ci.Err = err.Error()
	}
	for _, st := range rep.Steps {
		x := st.Name + " "
		if st.Ran {
			x += "ran"
		} else {
			x += "skipped"
		}
		if st.Err != "" {
			x += " err " + st.Err
		}
		ci.Steps = append(ci.Steps, x)
	}
	for _, ln := range l.rec.Since(mark) {
		if strings.Contains(ln, "town") || strings.Contains(ln, "repair") || strings.Contains(ln, "chore") {
			ci.Log = append(ci.Log, ln)
		}
	}
	return ci
}

func (l *lab) StashOpen(ctx context.Context) (int, error) {
	t0 := l.tick()
	err := l.tw.OpenStashForTest(ctx)
	return l.tick() - t0, err
}

func (l *lab) StashRoundTrip(ctx context.Context) error { return l.tw.StashRoundTripForTest(ctx) }

func (l *lab) StashDeposit(ctx context.Context, n int) error { return l.tw.DepositForTest(ctx, n) }

func (l *lab) StashWithdraw(ctx context.Context, n int) error {
	cctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	return l.be.Client.MoveGold(cctx, n, false)
}

func (l *lab) Unidentify(ctx context.Context, r *drill.Run) error {
	_, err := r.Cheat(ctx, "debug unid", "clear the identified flag of every pack item: the identify drill needs an unidentified item")
	return err
}

// unid reads "dump items": pack items whose identified flag is false.
func (l *lab) unid(ctx context.Context) (int, string) {
	r, err := l.be.Client.Do(ctx, "dump items")
	if err != nil {
		return 0, ""
	}
	line := strings.TrimSpace(r.Rest)
	for _, ln := range r.Lines {
		if strings.TrimSpace(ln) != "" {
			line = ln
			break
		}
	}
	var raw struct {
		Items []struct {
			Code  string `json:"code"`
			Loc   string `json:"loc"`
			Flags struct {
				Identified bool `json:"identified"`
			} `json:"flags"`
		} `json:"items"`
	}
	if json.Unmarshal([]byte(line), &raw) != nil {
		return 0, ""
	}
	n, codes := 0, []string{}
	for _, it := range raw.Items {
		if it.Loc == "inventory" && !it.Flags.Identified {
			n++
			codes = append(codes, it.Code)
		}
	}
	return n, strings.Join(codes, ",")
}

var _ drill.TownLab = (*lab)(nil)

func (l *lab) RepairKnob() int { return l.tune.Town.RepairBelowPct }

func (l *lab) StashDepositVerb(ctx context.Context, n int) error {
	cctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	return l.be.Client.MoveGold(cctx, n, true)
}

// quantities reads "dump items": the qty of every pack item with one (tomes and scrolls: the charges), "code=qty".
func (l *lab) quantities(ctx context.Context) string {
	r, err := l.be.Client.Do(ctx, "dump items")
	if err != nil {
		return ""
	}
	line := strings.TrimSpace(r.Rest)
	for _, ln := range r.Lines {
		if strings.TrimSpace(ln) != "" {
			line = ln
			break
		}
	}
	var raw struct {
		Items []struct {
			Code string `json:"code"`
			Loc  string `json:"loc"`
			Qty  int    `json:"qty"`
		} `json:"items"`
	}
	if json.Unmarshal([]byte(line), &raw) != nil {
		return ""
	}
	var q []string
	for _, it := range raw.Items {
		if it.Loc == "inventory" && it.Qty >= 0 {
			q = append(q, fmt.Sprintf("%s=%d", it.Code, it.Qty))
		}
	}
	return strings.Join(q, ",")
}

func (l *lab) RepairEstimate(ctx context.Context) (int, error) { return l.tw.RepairEstimateForTest() }

// SellExpect is the vendor price of a normal item of this code by the verified formula (Act 1, Normal).
func (l *lab) SellExpect(code string) int {
	b := l.db.ItemByCode(code)
	if b == nil {
		return 0
	}
	return b.SellPrice(l.db.PriceFactsOf(b), data.MaxBuy(1, 0))
}
