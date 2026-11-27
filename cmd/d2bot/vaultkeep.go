package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/bulkid"
	"d2lod-bot/internal/cheat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/ether"
	"d2lod-bot/internal/invdesk"
	"d2lod-bot/internal/merc"
	"d2lod-bot/internal/town"
	"d2lod-bot/internal/vault"
	"d2lod-bot/internal/vaultkeep"
	"d2lod-bot/internal/watch"
)

// The Keep queue of the Ethereal Vault in the bot (docs/vault.md, Keep; slice V5): the executor of internal/vaultkeep over the
// d2d client, the bot step (bot.VaultKeep), the page's routes (watch.VaultKeeper) and the Needs-you row.

// activeKeeper is the process's keeper for the Needs-you row (one hero per process).
var activeKeeper atomic.Pointer[vaultkeep.Keeper]

// keepOn is the kill switch: loot.vault and loot.vaultKeep on, and never in a measured run (D2BOT_RUN).
func keepOn(e *env.Env) func() bool {
	return func() bool {
		if e == nil || e.Tune == nil || os.Getenv("D2BOT_RUN") != "" {
			return false
		}
		return e.Tune.Loot.Vault != 0 && e.Tune.Loot.VaultKeep != 0
	}
}

// keepReady is the safe-moment test of the snapshot: alive, in town, no blocking panel, the cursor free. The bot step has
// checked fight and trip state already. Proximity to the stash chest is not required: `item create` is a memory verb of
// the fork (U: the fork's own proximity test exists only for the move verbs).
func keepReady(snap func() *d2.Snapshot) func() string {
	return func() string {
		s := snap()
		switch {
		case s == nil || s.Player == nil:
			return vaultkeep.ReasonNoGame
		case s.Player.MaxLife > 0 && s.Player.Life <= 0:
			return vaultkeep.ReasonDead
		case !s.Player.InTown():
			return "not in town"
		case s.UI.BlockingPanelOpen():
			return "a panel is open"
		case s.Inventory != nil && s.Inventory.Cursor != nil:
			return "an item is on the cursor"
		}
		return ""
	}
}

// wireVaultKeep builds the keeper for the hero and hangs it on the shared home and the runner ref.
func wireVaultKeep(ref *runnerRef, e *env.Env, c *d2dctl.Client, dataDir, currencyDir string, hero func() string, log *slog.Logger) {
	if ref == nil || c == nil || dataDir == "" || e == nil {
		return
	}
	home := vaultHomeFor(filepath.Join(dataDir, "vaults"))
	snap := func() *d2.Snapshot {
		if r := ref.p.Load(); r != nil {
			return r.LastSnapshot()
		}
		return nil
	}
	// Backup and Save are the fork's save verb: it writes through the character store, reads the bytes back and keeps its
	// .bak of the previous file (V design doc section 2: character_store.save). The bot has no save directory on the d2d
	// path and saves.Snapshot refuses in a game, so this is the existing mechanism; the vault record itself stays after a
	// Keep, so a bad create can never lose the item data.
	save := func(ctx context.Context) error { return c.SaveCharacter(ctx) }
	k := &vaultkeep.Keeper{
		Store:    func() *vault.Store { return home.Existing(hero()) },
		API:      c,
		Enabled:  keepOn(e),
		Ready:    keepReady(snap),
		Backup:   save,
		Save:     save,
		Notice:   func(text string) { log.Info("vault keep: " + text) },
		PerVisit: func() int { return e.Tune.Loot.VaultKeepPerVisit },
	}
	ha := heroActs{ref: ref, e: e, snap: snap, rescued: cainOpen(ref)}
	k.Tools, k.Vendor = ha, ha
	k.SellPerVisit = func() int { return e.Tune.Loot.VaultSellPerVisit }
	autoTrim := func() bool { return e.Tune.Loot.VaultAutoTrim != 0 }
	home.setAutoTrim(autoTrim)
	k.AutoSell = autoTrim
	k.Prompt = func() bool { return e.Tune.Loot.VaultSellPrompt != 0 }
	k.IdentifyPolicy = func() int { return bulkid.Effective(e.Tune.Identify, os.Getenv("D2BOT_RUN") != "").Policy }
	k.ShortBy = ha.short
	if dir := currencyDir; dir != "" { // identify.etherPerReveal (0: off): Ether pays only an explicit tap
		k.Ether = &bulkid.EtherPayer{W: lazyWallet{func() *ether.Wallet { return etherFor(dir, "", nil, nil).Wallet() }}, Price: func() int { return e.Tune.Identify.EtherPerReveal }}
	}
	k.Namer = ha.name
	pay := &d2dctl.Pay{C: c, Ledger: cheat.NewLedger(cheat.DefaultLedgerPath()), Actor: "bot:vault", Instance: os.Getenv("D2BOT_INSTANCE"),
		Character: hero(), Measured: measuredRun}
	k.Payer = goldPayer{heroActs: ha, pay: pay}
	// The instant sale: the engine's price (read), the stash credit (ledgered write), the stash read; loot.vaultSellInstant.
	k.Credit, k.Pricer, k.StashGold = pay, c, ha.Stash
	k.SellInstant = func() bool { return e.Tune.Loot.VaultSellInstant != 0 }
	k.SellConfirm = func() int { return e.Tune.Loot.VaultSellConfirm }
	k.SaleFile = func() string {
		h := hero()
		if h == "" || vault.ValidHero(h) != nil {
			return ""
		}
		return filepath.Join(vault.HeroDir(home.root, h), "sale.json")
	}
	k.Measured = measuredRun
	k.GoldOn = func() bool { return keepOn(e)() && e.Tune.Loot.VaultRevealGold != 0 }
	k.GoldConfirm = func() int { return e.Tune.Loot.VaultRevealGoldConfirm }
	k.Methods = func() []string { return revealMethods(e.Tune.Loot.VaultRevealMethods) }
	k.OrderFile = func() string {
		h := hero()
		if h == "" || vault.ValidHero(h) != nil {
			return ""
		}
		return filepath.Join(vault.HeroDir(home.root, h), "revealgold.json")
	}
	k.Hand = equipHand{ref: ref, e: e, snap: snap}
	k.EquipOn = equipOn(e)
	// Equip v2: the fork's direct verbs (ledgered, player-ordered only, never in a measured run). The knob also needs the Vault itself.
	k.Direct = &d2dctl.DirectEquip{C: c, Ledger: cheat.NewLedger(cheat.DefaultLedgerPath()), Actor: "bot:vault", Instance: os.Getenv("D2BOT_INSTANCE"),
		Character: hero(), Measured: measuredRun}
	k.DirectOn = func() bool { return e.Tune.Loot.Vault != 0 && e.Tune.Loot.VaultDirectEquip != 0 }
	k.Pinner = func(slot string, unit uint32, name string) {
		if d := ref.inv; d != nil {
			_ = d.PinSlot(slot, unit, name)
		}
	}
	k.EquipPath = filepath.Join(dataDir, "vaults", hero(), "vault", "equip.json")
	k.CarryPath = filepath.Join(dataDir, "vaults", hero(), "vault", "carried.json")
	if e.Data != nil {
		k.Chain = func(code string) []string {
			if b := e.Data.ItemByCode(code); b != nil {
				return merc.TypeChain(e.Data, b.Type)
			}
			return nil
		}
	}
	home.mu.Lock()
	home.keep = k
	home.mu.Unlock()
	ref.vk = k
	activeKeeper.Store(k)
}

// vaultKeepDep is bot.Deps.VaultKeep (nil interface when there is no keeper).
func vaultKeepDep(ref *runnerRef) bot.VaultKeep {
	if ref == nil || ref.vk == nil {
		return nil
	}
	return ref.vk
}

// keepFace adapts the keeper of a vaultHome to watch.VaultKeeper; late bound, so the page works before and without one.
type keepFace struct{ h *vaultHome }

func (f keepFace) k() *vaultkeep.Keeper {
	f.h.mu.Lock()
	defer f.h.mu.Unlock()
	return f.h.keep
}

func (f keepFace) Gate(ctx context.Context) (bool, string, string) {
	k := f.k()
	if k == nil {
		return false, "off", "this bot has no Keep queue"
	}
	ok, why := k.Gate(ctx)
	if ok {
		return true, "", ""
	}
	code := "stopped"
	switch why {
	case vaultkeep.ReasonOff:
		code = "off"
	case vaultkeep.ReasonDead:
		code = "dead"
	case vaultkeep.ReasonVerbs:
		code = "verbs"
	}
	return false, code, why
}

func (f keepFace) Enqueue(ctx context.Context, ids []string, target string, now bool) []watch.VaultKeepResult {
	k := f.k()
	if k == nil {
		return nil
	}
	var out []watch.VaultKeepResult
	for _, r := range k.Enqueue(ctx, ids, target, now) {
		out = append(out, watch.VaultKeepResult{ID: r.ID, OK: r.OK, Already: r.Already, Code: r.Code, Why: r.Why})
	}
	return out
}

func (f keepFace) Progress() watch.VaultKeepView {
	k := f.k()
	if k == nil {
		return watch.VaultKeepView{Why: "this bot has no Keep queue"}
	}
	in := k.Info()
	ok, why := k.Peek()
	return watch.VaultKeepView{Enabled: ok, Why: why, Queued: in.Queued, Creating: in.Creating, Done: in.Done, Full: in.Full, Line: k.ProgressLine()}
}

func init() {
	glanceExtra = append(glanceExtra, func(src *watch.GlanceSources, _ glanceDeps) {
		prev := src.Needs
		src.Needs = func() []watch.GlanceNeed {
			var out []watch.GlanceNeed
			if prev != nil {
				out = prev()
			}
			if k := activeKeeper.Load(); k != nil {
				if text := k.NeedsText(); text != "" {
					out = append(out, watch.GlanceNeed{ID: "vault:keep", Kind: "vault_keep", Sev: "warn", Text: text,
						Sub: "The items stay queued and arrive at the next town visit once there is room.", Actions: []watch.GlanceVerb{}})
				}
				if o := k.Offer(); o != nil {
					body, _ := json.Marshal(map[string]any{"ids": o.IDs, "confirm": true})
					out = append(out, watch.GlanceNeed{ID: "vault:sellprompt", Kind: "vault_sell", Sev: "info",
						Text: fmt.Sprintf("Short by %d gold for repairs and supplies: sell %d vault item%s (about %d gold)?", o.Short, len(o.IDs), plural(len(o.IDs)), o.Gold),
						Sub:  "Only unpinned items below rare are offered. Nothing is sold unless you say yes; the hero sells them at the vendor on the next town visit.",
						Actions: []watch.GlanceVerb{
							{Label: "Sell them", Post: watch.VaultSellRoute, Body: body},
							{Label: "Not now", Post: watch.VaultPromptRoute, Body: json.RawMessage(`{"dismiss":true}`)},
						}})
				}
			}
			return out
		}
	})
}

// heroActs is the game side of Reveal and Sell for one hero: the town's identify services and sell chore, the inventory desk's
// marks, and a fresh read of the purse. U live: the whole chain (the created item appearing in the snapshot, the mark, the
// vendor click path) is checked by the drill in docs/vault.md.
type heroActs struct {
	ref  *runnerRef
	e    *env.Env
	snap func() *d2.Snapshot
	// rescued is cainOpen's check: the Search for Cain is done (or he stands here).
	rescued func(*d2.Snapshot) bool
}

// Rescued says whether Cain is rescued for the hero (the vault's Reveal note while identify.policy counts on him).
func (h heroActs) Rescued() bool { return h.rescued != nil && h.rescued(h.snap()) }

func (h heroActs) town() *town.Town { return h.ref.tw.Load() }

// Charges counts the Tome of Identify's scrolls and the loose scrolls in the pack.
func (h heroActs) Charges(context.Context) (int, error) {
	tw := h.town()
	if tw == nil {
		return -1, errors.New("no town helper yet")
	}
	tome, _, scrolls := tw.IdentifyTools(h.snap())
	return max(tome, 0) + scrolls, nil
}

// Free is true while Deckard Cain stands in the town the hero is in: his Identify line costs nothing.
func (h heroActs) Free() bool {
	s := h.snap()
	if s == nil || s.Player == nil || !s.Player.InTown() {
		return false
	}
	for i := range s.Monsters {
		if m := &s.Monsters[i]; m.Alive() && town.IsCainClass(m.Class) {
			return true
		}
	}
	return false
}

// Identify reads the item with Cain (free: he identifies everything carried) or a scroll or tome charge (the server checks).
func (h heroActs) Identify(ctx context.Context, unit int, free bool) error {
	tw := h.town()
	if tw == nil {
		return errors.New("no town helper yet")
	}
	if free {
		_, err := tw.CainIdentify(ctx)
		return err
	}
	err := tw.PlayerIdentifyBatch(ctx, []uint32{uint32(unit)})[0]
	if errors.Is(err, town.ErrNoIDTool) {
		return vaultkeep.ErrNoTool
	}
	return err
}

// Gold is a fresh read of the purse (the last snapshot can be a tick old, which would fail the sale's read-back).
func (h heroActs) Gold(context.Context) (int, error) {
	s, err := h.e.Snapshot()
	if err != nil || s == nil || s.Player == nil {
		return 0, fmt.Errorf("purse unreadable: %v", err)
	}
	return s.Player.Gold, nil
}

// Sell marks the created pack item sell (the desk finds it once the snapshot lists it) and runs the town's own sell chore: the
// same marked-items sale the player's "sell now" uses, real vendor, real price. The chore sells every mark of the hero, the
// player's own included (they asked for those too). A fight or a busy panel postpones.
func (h heroActs) Sell(ctx context.Context, units []int) error {
	d, tw := h.ref.inv, h.town()
	if d == nil || tw == nil {
		return errors.New("selling is not available on this bot")
	}
	marked := units[:0:0]
	undo := func() {
		for _, u := range marked {
			_ = d.SetMark(uint32(u), "")
		}
	}
	for _, u := range units {
		var err error
		for try := 0; try < 8; try++ {
			if err = d.SetMark(uint32(u), invdesk.MarkSell); err == nil {
				break
			}
			var rf *invdesk.Refusal
			if !errors.As(err, &rf) || rf.Code != "item" {
				break
			}
			select { // the snapshot has not listed the created item yet
			case <-ctx.Done():
				undo()
				return ctx.Err()
			case <-time.After(400 * time.Millisecond):
			}
		}
		if err != nil {
			undo()
			return err
		}
		marked = append(marked, u)
	}
	_, err := townSeller{tw}.SellMarked(ctx)
	undo()
	if err != nil && (err.Error() == "town: a fight is on" || err.Error() == "leveler: a fight is on" || ctx.Err() != nil) {
		return fmt.Errorf("%v: %w", err, vaultkeep.ErrPostpone)
	}
	return err
}

// short is how much gold the next town visit lacks for the repair and the planned potion and scroll purchases (the merc
// revive price is not counted here: U, the budget keeps it inside the chores).
func (h heroActs) short() int {
	tw, s := h.town(), h.snap()
	if tw == nil || s == nil || s.Player == nil {
		return 0
	}
	return max(tw.RepairEstimate(s)+tw.PlannedSpend(s)-s.Player.Gold, 0)
}

// name is the display name of an identified record, from the item the pack now holds.
func (h heroActs) name(back d2dctl.VaultRecord, _ vault.Rec) string {
	tw := h.town()
	s, err := h.e.Snapshot()
	if tw == nil || err != nil || s == nil || s.Inventory == nil {
		return ""
	}
	for _, u := range s.Inventory.Items {
		if int(u.ID) == back.UnitID && u.Item != nil {
			return tw.ItemName(u)
		}
	}
	return ""
}

// actFace adapts the keeper to watch.VaultActor; late bound like keepFace.
type actFace struct{ h *vaultHome }

func (f actFace) k() *vaultkeep.Keeper {
	f.h.mu.Lock()
	defer f.h.mu.Unlock()
	return f.h.keep
}

func gateCode(why string) string {
	switch why {
	case vaultkeep.ReasonOff:
		return "off"
	case vaultkeep.ReasonDead:
		return "dead"
	case vaultkeep.ReasonVerbs:
		return "verbs"
	}
	return "stopped"
}

func (f actFace) gate(ok bool, why string) (bool, string, string) {
	if ok {
		return true, "", ""
	}
	return false, gateCode(why), why
}

func (f actFace) GateReveal(ctx context.Context) (bool, string, string) {
	k := f.k()
	if k == nil {
		return false, "off", "this bot has no Reveal"
	}
	ok, why := k.GateReveal(ctx)
	return f.gate(ok, why)
}

func (f actFace) GateSell(ctx context.Context) (bool, string, string) {
	k := f.k()
	if k == nil {
		return false, "off", "this bot has no Sell"
	}
	ok, why := k.GateSell(ctx)
	return f.gate(ok, why)
}

func results(in []vaultkeep.Result) []watch.VaultKeepResult {
	var out []watch.VaultKeepResult
	for _, r := range in {
		out = append(out, watch.VaultKeepResult{ID: r.ID, OK: r.OK, Already: r.Already, Code: r.Code, Why: r.Why})
	}
	return out
}

func (f actFace) EnqueueReveal(ctx context.Context, ids []string, now bool) []watch.VaultKeepResult {
	if k := f.k(); k != nil {
		return results(k.EnqueueReveal(ctx, ids, now))
	}
	return nil
}

func (f actFace) EnqueueRevealEther(ctx context.Context, ids []string, now bool) []watch.VaultKeepResult {
	if k := f.k(); k != nil {
		return results(k.EnqueueRevealEther(ctx, ids, now))
	}
	return nil
}

func saleOf(v *vaultkeep.SaleView) *watch.VaultSale {
	if v == nil {
		return nil
	}
	out := &watch.VaultSale{ID: v.ID, State: v.State, Items: v.Items, Done: v.Done, Gold: v.Gold, StashBefore: v.StashBefore, StashAfter: v.StashAfter, Note: v.Note, Line: v.Line, All: v.All}
	out.Skipped = reasonsOf(v.Skipped)
	return out
}

func reasonsOf(rs []vaultkeep.SaleReason) []watch.VaultSaleReason {
	var out []watch.VaultSaleReason
	for _, r := range rs {
		out = append(out, watch.VaultSaleReason{Code: r.Code, N: r.N, Why: r.Why})
	}
	return out
}

// SaleOn, SaleQuote and SellNow are the instant sale (watch.VaultSaleActor).
func (f actFace) SaleOn(ctx context.Context) (bool, string) {
	if k := f.k(); k != nil {
		return k.SaleOn(ctx)
	}
	return false, "this bot has no Sell"
}

func (f actFace) SaleQuote(ctx context.Context, ids []string) watch.VaultSaleQuote {
	k := f.k()
	if k == nil {
		return watch.VaultSaleQuote{Why: "this bot has no Sell"}
	}
	q := k.SaleQuote(ctx, ids)
	out := watch.VaultSaleQuote{Asked: q.Asked, Total: q.Total, Stash: q.Stash, StashAfter: q.StashAfter, Room: q.Room, ConfirmAt: q.ConfirmAt, NeedConfirm: q.NeedConfirm, Why: q.Why, Count: len(q.Lines)}
	for _, l := range q.Lines {
		out.Lines = append(out.Lines, watch.VaultSaleLine{ID: l.ID, Gold: l.Gold})
	}
	out.Reasons = reasonsOf(q.Reasons)
	return out
}

// SellAllQuote, SellAll and SellAllIDs are the whole-vault sale (watch.VaultSellAllActor).
func (f actFace) SellAllQuote(ctx context.Context) watch.VaultSaleQuote {
	k := f.k()
	if k == nil {
		return watch.VaultSaleQuote{Why: "this bot has no Sell"}
	}
	q := k.SellAllPlan(ctx).Quote
	out := watch.VaultSaleQuote{Asked: q.Asked, Total: q.Total, Stash: q.Stash, StashAfter: q.StashAfter, Room: q.Room, ConfirmAt: q.ConfirmAt, Why: q.Why, Count: len(q.Lines)}
	out.Reasons = reasonsOf(q.Reasons)
	return out
}

func (f actFace) SellAll(ctx context.Context, orderID string) (string, string, *watch.VaultSale) {
	k := f.k()
	if k == nil {
		return "off", "this bot has no Sell", nil
	}
	o := k.SellAll(ctx, orderID)
	return o.Code, o.Why, saleOf(o.View)
}

func (f actFace) SellAllIDs() []string {
	if k := f.k(); k != nil {
		return k.SellAllIDs()
	}
	return nil
}

func (f actFace) SellNow(ctx context.Context, ids []string, confirm bool) ([]watch.VaultKeepResult, *watch.VaultSale) {
	k := f.k()
	if k == nil {
		return nil, nil
	}
	res, v := k.SellNow(ctx, ids, confirm)
	return results(res), saleOf(v)
}

func (f actFace) EnqueueSell(ctx context.Context, ids []string, now bool) []watch.VaultKeepResult {
	if k := f.k(); k != nil {
		return results(k.EnqueueSell(ctx, ids, now))
	}
	return nil
}

func (f actFace) Cancel(ids []string) []watch.VaultKeepResult {
	if k := f.k(); k != nil {
		return results(k.Cancel(ids))
	}
	return nil
}

func (f actFace) DismissOffer() {
	if k := f.k(); k != nil {
		k.DismissOffer()
	}
}

func (f actFace) Act(ctx context.Context) watch.VaultActView {
	k := f.k()
	if k == nil {
		w := "this bot has no Reveal or Sell"
		return watch.VaultActView{Reveal: watch.VaultRevealView{Why: w, Code: "off"}, Sell: watch.VaultSellView{Why: w, Code: "off"}}
	}
	k.BackfillProps(ctx, 10) // revealed records without stat lines get the ones their bytes carry (a few per page load)
	in, p := k.Info(), k.RevealPrice(ctx)
	v := watch.VaultActView{
		Reveal: watch.VaultRevealView{Note: p.Note, Charges: p.Charges, Free: p.Free, PerItem: p.PerItem, EtherPer: p.EtherPer, Balance: p.Balance, Queued: in.RevQueued, Working: in.RevWorking, Done: in.Revealed, Line: k.RevealLine()},
		Sell:   watch.VaultSellView{Queued: in.SellQueued, Working: in.SellWorking, Sold: in.Sold, Gold: in.GoldEarned, Line: k.SellLine()},
	}
	v.Reveal.Methods = k.OfferedMethods()
	v.Reveal.Enabled, v.Reveal.Why = k.GateReveal(ctx)
	if !v.Reveal.Enabled {
		v.Reveal.Code = gateCode(v.Reveal.Why)
	}
	if v.Reveal.Enabled && !p.Free && p.Charges == 0 {
		v.Reveal.Enabled, v.Reveal.Code, v.Reveal.Why = false, "noscroll", vaultkeep.ReasonNoScroll
	}
	v.Sell.Enabled, v.Sell.Why = k.GateSell(ctx)
	if !v.Sell.Enabled {
		v.Sell.Code = gateCode(v.Sell.Why)
	}
	sf := k.SaleFacts(ctx)
	v.Sell.Instant, v.Sell.ConfirmAt, v.Sell.InstantWhy = sf.On, sf.ConfirmAt, sf.Why
	if sf.On { // an instant sale needs no vendor and no town: only the knob, the verbs and a vault
		v.Sell.Enabled, v.Sell.Why, v.Sell.Code = true, "", ""
	}
	v.Sell.Sale = saleOf(k.SaleView())
	n, left := k.SellAllCount()
	v.Sell.All = &watch.VaultSellAll{N: n, Left: reasonsOf(left)}
	if o := k.Offer(); o != nil {
		v.Prompt = &watch.VaultPrompt{Short: o.Short, IDs: o.IDs, Gold: o.Gold, Names: o.Names}
	}
	return v
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// Price is the vendor price of one Scroll of Identify from the item tables (the same figure the town's buy errand plans
// with); 0 when the tables lack it.
func (h heroActs) Price() int {
	if h.e == nil || h.e.Data == nil {
		return 0
	}
	if b := h.e.Data.ItemByCode("isc"); b != nil {
		return b.Cost
	}
	return 0
}

// Stash reads the stash gold fresh (the last snapshot can be a tick old): the gold a gold reveal pays from. The purse is
// never read for it and never touched.
func (h heroActs) Stash(context.Context) (int, error) {
	s := h.snap()
	if fresh, ferr := h.e.Snapshot(); ferr == nil && fresh != nil && fresh.Player != nil {
		s = fresh
	}
	if s == nil || s.Player == nil {
		return 0, errors.New("the stash gold is unreadable")
	}
	return s.Player.StashGold, nil
}

// goldPayer is the vault keeper's payer: the price and the stash gold of heroActs, the payment itself through the fork's
// `debug paygold` (d2dctl.Pay: ledgered with the reason "vault gold reveal order <id>", refused in a measured run).
type goldPayer struct {
	heroActs
	pay *d2dctl.Pay
}

func (p goldPayer) PayGold(ctx context.Context, amount int, order string) (d2dctl.PayResult, error) {
	return p.pay.PayGold(ctx, amount, order)
}

// measuredRun is true in a measured run (gate, soak, baseline, fidelity: D2BOT_RUN names it): a gold reveal never pays then.
func measuredRun() bool { return strings.TrimSpace(os.Getenv(cheat.RunEnv)) != "" }

func viewOf(v vaultkeep.GoldView) watch.VaultGoldOrder {
	return watch.VaultGoldOrder{ID: v.ID, State: v.State, Total: v.Total, Want: v.Want, Done: v.Done, Paid: v.Paid, StashAfter: v.StashAfter, Note: v.Note, Line: v.Line}
}

func (f actFace) GoldInfo(ctx context.Context) watch.VaultGoldInfo {
	k := f.k()
	if k == nil {
		return watch.VaultGoldInfo{}
	}
	g := k.GoldFacts(ctx)
	out := watch.VaultGoldInfo{On: g.On, ScrollGold: g.Price, Stash: g.Stash, ConfirmAt: g.Confirm, Why: g.Why, Decode: g.Decode}
	if v := k.GoldView(); v != nil {
		o := viewOf(*v)
		out.Order = &o
	}
	return out
}

func (f actFace) EnqueueRevealGold(ctx context.Context, ids []string, maxGold int, confirm bool) ([]watch.VaultKeepResult, *watch.VaultGoldOrder) {
	k := f.k()
	if k == nil {
		return nil, nil
	}
	res, v := k.EnqueueRevealGold(ctx, ids, maxGold, confirm)
	var o *watch.VaultGoldOrder
	if v != nil {
		x := viewOf(*v)
		o = &x
	}
	return results(res), o
}

func (f actFace) EnqueueRevealGoldAll(ctx context.Context, maxGold int, confirm bool) ([]watch.VaultKeepResult, *watch.VaultGoldOrder) {
	k := f.k()
	if k == nil {
		return nil, nil
	}
	res, v := k.EnqueueRevealGoldAll(ctx, maxGold, confirm)
	var o *watch.VaultGoldOrder
	if v != nil {
		x := viewOf(*v)
		o = &x
	}
	return results(res), o
}

func (f actFace) CancelGold() (watch.VaultGoldOrder, bool) {
	k := f.k()
	if k == nil {
		return watch.VaultGoldOrder{}, false
	}
	v, ok := k.CancelGold()
	return viewOf(v), ok
}

// revealMethods names the payments of the knob loot.vaultRevealMethods bit mask (1 free, 2 ether, 4 gold). 0 is read as the
// default (gold only): the page always offers at least one way.
func revealMethods(mask int) []string {
	if mask <= 0 {
		mask = 4
	}
	var out []string
	for i, name := range []string{"free", "ether", "gold"} {
		if mask&(1<<i) != 0 {
			out = append(out, name)
		}
	}
	return out
}
