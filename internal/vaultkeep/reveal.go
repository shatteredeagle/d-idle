package vaultkeep

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"d2lod-bot/internal/bulkid"
	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/vault"
)

// Reveal (slice V6, docs/vault.md section Reveal): identify a vault record through the real identify path. The item is
// created into the pack (unidentified, exactly as dropped), identified by a scroll, a tome charge or Cain, read back, the
// vault record is superseded by the identified bytes (same id, new hash) and the temporary copy leaves the hero again. A
// record that is also queued for Keep is identified in place after its create and then kept: one create, not two.
//
//	reveal want -> rb (logged) -> item create -> rc (logged) -> identify -> read back -> reveal event (rt) -> item vault -> idle
//
// Every stage is a log line written BEFORE the game step it names, so a restart looks at the hero (locate) and resumes.

// Identifier is the game side of Reveal (cmd/d2bot adapts the town's identify services; tests use a fake).
type Identifier interface {
	// Charges is how many identify scrolls and tome charges the pack holds.
	Charges(ctx context.Context) (int, error)
	// Free reports that Cain stands in this town: identifying costs nothing there.
	Free() bool
	// Identify identifies the carried item unit; free selects Cain, else a scroll or a tome charge (the server checks).
	Identify(ctx context.Context, unit int, free bool) error
}

// ErrNoTool is what an Identifier returns when no scroll or tome charge exists.
var ErrNoTool = errors.New("no identify scroll or tome charge")

// Reasons of the Reveal and Sell errands.
const (
	ReasonNoScroll   = "no scroll; buy on request"
	ReasonRevDiffers = "the game identified a different item"
	ReasonRevStill   = "the item is still unidentified"
	ReasonRevDup     = "an identical identified item is already in the vault"
	ReasonNoSealed   = "the vault holds no affixes for this item and the game could not decode them just now: try again in game"
	revPrefix        = "reveal: "
	revKey           = "vr-"
)

func revStored(why string) string { return revPrefix + why }

// RevealReason strips the log prefix from a stored reveal reason ("" when it is not one).
func RevealReason(stored string) string {
	if !strings.HasPrefix(stored, revPrefix) {
		return ""
	}
	return strings.TrimPrefix(stored, revPrefix)
}

// charges reads the tool count; an unreadable pack counts as unknown (-1: the server decides).
func (k *Keeper) charges(ctx context.Context) int {
	if k.Tools == nil {
		return 0
	}
	n, err := k.Tools.Charges(ctx)
	if err != nil {
		return -1
	}
	return n
}

func (k *Keeper) free() bool { return k.Tools != nil && k.Tools.Free() }

// RevealInfo is what the page shows before the click: the price per item and what the hero holds.
type RevealInfo struct {
	Charges  int  `json:"charges"`            // -1 unknown
	Free     bool `json:"free"`               // Cain is here: no charge is used
	PerItem  int  `json:"perItem"`            // charges one reveal costs (1)
	EtherPer int  `json:"etherPer,omitempty"` // Ether one tapped reveal costs; 0: Ether is not a payer
	Balance  int  `json:"balance"`            // Ether in the wallet, -1 when there is none
	// Note is the viewer sentence while identify.policy wants Cain and he is not rescued ("" otherwise).
	Note string `json:"note,omitempty"`
}

// CainRescuer is an optional part of an Identifier: whether the Search for Cain is done for the hero.
type CainRescuer interface{ Rescued() bool }

// RevealNote says that identify is manual because Cain is not rescued, while identify.policy (1 or 2) counts on him.
func (k *Keeper) RevealNote() string {
	if k.IdentifyPolicy == nil {
		return ""
	}
	if p := k.IdentifyPolicy(); p != env.IdentifyCainFree && p != env.IdentifyScrolls {
		return ""
	}
	if r, ok := k.Tools.(CainRescuer); ok && !r.Rescued() {
		return bulkid.WhyNoCain
	}
	return ""
}

// RevealPrice reads the current price view.
func (k *Keeper) RevealPrice(ctx context.Context) RevealInfo {
	return RevealInfo{Charges: k.charges(ctx), Free: k.free(), PerItem: 1, EtherPer: k.Ether.Quote(1), Balance: k.Ether.Balance(), Note: k.RevealNote()}
}

// GateReveal says whether a Reveal press can be queued now (the same refusals as Keep, plus: the bot must be able to
// identify at all).
func (k *Keeper) GateReveal(ctx context.Context) (ok bool, why string) {
	if k.Tools == nil {
		return false, "this bot cannot identify items"
	}
	return k.Gate(ctx)
}

// EnqueueReveal queues Reveal for records. Idempotent; an identified record is a refusal ("state"); with no scroll, tome
// charge or Cain the refusal is "noscroll" (ReasonNoScroll): the page then offers the manual Buy action. The bot never buys
// scrolls for this.
func (k *Keeper) EnqueueReveal(ctx context.Context, ids []string, wantNow bool) []Result {
	st := k.store()
	out := make([]Result, 0, len(ids))
	free := k.free()
	avail := k.charges(ctx)
	queued := 0
	if st != nil {
		for _, r := range st.RevealQueued() {
			if r.RevQ {
				queued++
			}
		}
	}
	for _, id := range ids {
		r := Result{ID: id}
		switch {
		case st == nil:
			r.Code, r.Why = "missing", "there is no vault for this hero yet"
		default:
			rec, found := st.Get(id)
			switch {
			case !found:
				r.Code, r.Why = "missing", "no such item"
			case rec.RevQ:
				r.OK, r.Already = true, true
			case rec.Identified:
				r.Code, r.Why = "state", "it is identified already"
			case rec.State != vault.Recorded && rec.State != vault.KeepQ:
				r.Code, r.Why = "state", "only an item in the vault can be revealed (it is "+string(rec.State)+")"
			case !free && avail >= 0 && queued >= avail:
				r.Code, r.Why = "noscroll", ReasonNoScroll
			default:
				if err := st.RevealWant(id); err != nil {
					r.Code, r.Why = codeOf(err), err.Error()
				} else {
					r.OK = true
					queued++
				}
			}
		}
		out = append(out, r)
	}
	if wantNow {
		k.wantTrip(out)
	}
	return out
}

func (k *Keeper) wantTrip(out []Result) {
	for _, r := range out {
		if r.OK && !r.Already {
			k.mu.Lock()
			k.tripWant = true
			k.mu.Unlock()
			return
		}
	}
}

// Cancel takes queued Keep, Reveal and Sell requests back (idempotent for ids that are not queued: reported as "state").
func (k *Keeper) Cancel(ids []string) []Result {
	st := k.store()
	var out []Result
	for _, id := range ids {
		r := Result{ID: id}
		rec, found := vault.Rec{}, false
		if st != nil {
			rec, found = st.Get(id)
		}
		var err error
		switch {
		case !found:
			r.Code, r.Why = "missing", "no such item"
		case rec.State == vault.SellQ:
			err = st.CancelSell(id)
		case rec.State == vault.KeepQ && !rec.InFlight:
			if err = st.CancelKeep(id); err == nil && rec.RevQ {
				_ = st.RevealCancel(id)
			}
		case rec.RevQ && rec.Stage == "":
			err = st.RevealCancel(id)
		default:
			r.Code, r.Why = "state", "nothing is queued for it, or it is being worked on"
		}
		if err != nil {
			r.Code, r.Why = codeOf(err), "it is being worked on right now"
		}
		r.OK = r.Code == ""
		out = append(out, r)
	}
	return out
}

// presentUnit is the unit id of the extra copy of r's bytes the hero carries beyond the kept records the vault already
// counts, 0 when none. With several the newest (highest unit id) is the one this errand created.
func (k *Keeper) presentUnit(st *vault.Store, raw string, have []d2dctl.VaultRecord) int {
	n, top := 0, 0
	for _, h := range have {
		if strings.EqualFold(h.Norm, raw) {
			n++
			top = max(top, h.UnitID)
		}
	}
	kept := 0
	for _, o := range st.Dump() {
		if o.State == vault.Kept && strings.EqualFold(o.Raw, raw) {
			kept++
		}
	}
	if n > kept {
		return top
	}
	return 0
}

// adoptable finds the one carried item that is the identified form of r: same code, quality, level, sockets and
// ethereal bit, identified, and not a copy of any record the vault knows. More than one candidate is ambiguous (0).
func (k *Keeper) adoptable(st *vault.Store, r vault.Rec, have []d2dctl.VaultRecord) *d2dctl.VaultRecord {
	known := map[string]bool{}
	for _, o := range st.Dump() {
		known[strings.ToLower(o.Raw)] = true
	}
	var hit *d2dctl.VaultRecord
	for i := range have {
		h := &have[i]
		if !h.Identified || known[strings.ToLower(h.Norm)] || h.Code != r.Code || h.Quality != r.Quality || h.ILvl != r.Ilvl ||
			h.Sockets != r.Sockets || h.Ethereal != r.Eth {
			continue
		}
		if hit != nil {
			return nil
		}
		hit = h
	}
	return hit
}

// identifyUnit identifies one carried unit and reads it back. why is "" on success. unknown means the outcome cannot be
// told (the identify errored and the hero could not be read): the caller leaves the step in flight.
func (k *Keeper) identifyUnit(ctx context.Context, r vault.Rec, unit int, free bool) (back d2dctl.VaultRecord, why string, unknown bool) {
	if !free {
		if n := k.charges(ctx); n == 0 {
			return back, ReasonNoScroll, false
		}
	}
	if err := k.Tools.Identify(ctx, unit, free); err != nil {
		if errors.Is(err, ErrNoTool) {
			return back, ReasonNoScroll, false
		}
		// the reply may be lost while the game identified it: the hero decides
		b, w, readFailed := k.readIdentified(ctx, r, unit)
		switch {
		case w == "":
			return b, "", false
		case readFailed:
			return back, "the outcome is unknown: " + err.Error(), true
		}
		return back, "the game did not identify it: " + err.Error(), false
	}
	b, w, readFailed := k.readIdentified(ctx, r, unit)
	return b, w, readFailed
}

func (k *Keeper) readIdentified(ctx context.Context, r vault.Rec, unit int) (back d2dctl.VaultRecord, why string, readFailed bool) {
	recs, err := k.API.ItemFull(ctx, unit)
	if err != nil || len(recs) == 0 {
		return back, "the identified item could not be read back", true
	}
	back = recs[0]
	switch {
	case !back.Identified:
		return back, ReasonRevStill, false
	case back.Code != r.Code || back.Quality != r.Quality || back.ILvl != r.Ilvl:
		return back, ReasonRevDiffers, false
	}
	return back, "", false
}

// store writes the identified bytes over the record (same id, new hash). It returns a reason when the vault refuses.
func (k *Keeper) recordReveal(st *vault.Store, r vault.Rec, back d2dctl.VaultRecord) string {
	it := r.Item
	it.Raw, it.Identified, it.Value, it.Props = back.Norm, true, 0, back.Props
	it.Pic, it.Tint, it.FileIndex = back.Pic, back.Tint, back.FileIndex()
	if k.Namer != nil {
		if n := k.Namer(back, r); n != "" {
			it.Name = n
		}
	}
	if err := st.Reveal(r.ID, it); err != nil {
		if errors.Is(err, vault.ErrDuplicate) {
			return ReasonRevDup
		}
		return err.Error()
	}
	return ""
}

// revealInPlace identifies a record that is also being kept: the unit was just created for the keep. The item stays where
// it is; a failure leaves it kept unidentified (the reason is stored on the record).
func (k *Keeper) revealInPlace(ctx context.Context, st *vault.Store, r vault.Rec, unit int) (nr vault.Rec, why string, unknown bool) {
	_ = st.SetStage(r.ID, vault.StageRevMade, 0)
	back, why, unknown := k.identifyUnit(ctx, r, unit, k.free())
	if unknown {
		return r, why, true
	}
	if why == "" {
		why = k.recordReveal(st, r, back)
	}
	if why != "" {
		_ = st.RevealFail(r.ID, revStored(why))
		return r, why, false
	}
	nr, _ = st.Get(r.ID)
	return nr, "", false
}

// runReveal creates, identifies and removes the temporary copies of the pure reveal requests. Resumes what a restart left.
func (k *Keeper) runReveal(ctx context.Context) []string {
	st := k.store()
	if st == nil || k.Tools == nil {
		return nil
	}
	var list []vault.Rec
	for _, r := range st.RevealQueued() {
		if r.State == vault.Recorded {
			list = append(list, r)
		}
	}
	if len(list) == 0 {
		return nil
	}
	if !k.runGate(ctx) {
		return nil
	}
	var lines []string
	var have []d2dctl.VaultRecord
	haveOK := false
	look := func() ([]d2dctl.VaultRecord, error) {
		if !haveOK {
			h, err := k.API.ItemFull(ctx, -1)
			if err != nil {
				return nil, err
			}
			have, haveOK = h, true
		}
		return have, nil
	}
	budget := k.now().Add(orDur(k.Budget, defaultBudget))
	made := 0
	for _, r := range list {
		if ctx.Err() != nil {
			break
		}
		if r.Stage != "" {
			h, err := look()
			if err != nil {
				return append(lines, "reveal: cannot read the hero, in-flight reveals stay as they are: "+err.Error())
			}
			haveOK = false // the resume changes the hero
			if line := k.resumeReveal(ctx, st, r, h); line != "" {
				lines = append(lines, line)
			}
			continue
		}
		if made >= k.perVisit() || !k.now().Before(budget) {
			break
		}
		made++
		line, halt := k.revealFresh(ctx, st, r)
		if line != "" {
			lines = append(lines, line)
		}
		if halt {
			break
		}
	}
	return lines
}

// resumeReveal looks at the hero for the copy a stage left and finishes or undoes the step.
func (k *Keeper) resumeReveal(ctx context.Context, st *vault.Store, r vault.Rec, have []d2dctl.VaultRecord) string {
	switch r.Stage {
	case vault.StageRevCleanup: // the identified bytes are stored: the copy of them has to leave
		if u := k.presentUnit(st, r.Raw, have); u != 0 {
			if _, err := k.API.ItemVault(ctx, u); err != nil {
				return "reveal: " + r.Name + " identified, the temporary copy could not be removed yet: " + err.Error()
			}
		}
		_ = st.SetStage(r.ID, "", 0)
		return "reveal: " + r.Name + " revealed, temporary copy removed"
	case vault.StageRevBegin, vault.StageRevMade:
		if u := k.presentUnit(st, r.Raw, have); u != 0 { // the unidentified copy exists: identify it
			line, _ := k.finishReveal(ctx, st, r, u)
			return line
		}
		if h := k.adoptable(st, r, have); h != nil { // identified in the game, the vault never heard of it
			if why := k.recordReveal(st, r, *h); why != "" {
				_ = st.RevealFail(r.ID, revStored(why))
				return "reveal: " + r.Name + " not revealed: " + why
			}
			nr, _ := st.Get(r.ID)
			if _, err := k.API.ItemVault(ctx, h.UnitID); err != nil {
				return "reveal: " + r.Name + " revealed, the temporary copy could not be removed yet: " + err.Error()
			}
			_ = st.SetStage(nr.ID, "", 0)
			return "reveal: " + r.Name + " revealed after a restart"
		}
		_ = st.SetStage(r.ID, "", 0) // no copy: the create was never seen; the request stays queued
		return "reveal: " + r.Name + " was not created before the restart, queued again"
	}
	return ""
}

// revealFresh creates the temporary copy and runs the rest. halt stops the batch (a hero-wide problem).
func (k *Keeper) revealFresh(ctx context.Context, st *vault.Store, r vault.Rec) (string, bool) {
	free := k.free()
	if !free && k.charges(ctx) == 0 {
		_ = st.RevealFail(r.ID, revStored(ReasonNoScroll))
		k.note("noscroll", "Reveal: "+ReasonNoScroll)
		return "reveal: " + r.Name + " not revealed: " + ReasonNoScroll, true
	}
	k.clearNote("noscroll")
	var lines []string
	if !k.backupOnce(ctx, &lines) {
		return strings.Join(lines, "; "), true
	}
	if err := d2dctl.ValidHex(r.Raw); err != nil {
		_ = st.RevealFail(r.ID, revStored(ReasonBadData))
		return "reveal: " + r.Name + " not revealed: " + ReasonBadData, false
	}
	if err := st.SetStage(r.ID, vault.StageRevBegin, 0); err != nil {
		return "reveal: " + r.ID + " begin: " + err.Error(), true
	}
	res, err := k.API.ItemCreate(ctx, r.Raw, d2dctl.CreateOpts{Pack: true, Key: fmt.Sprintf("%s%s-%d", revKey, r.ID, r.Attempts+1)})
	if err != nil {
		return k.revealCreateErr(ctx, st, r, err)
	}
	back, err := k.API.ItemFull(ctx, res.ID)
	if err != nil || len(back) == 0 {
		return fmt.Sprintf("reveal: %s created (unit %d) but the read-back failed, left in flight", r.Name, res.ID), true
	}
	if !strings.EqualFold(back[0].Norm, r.Raw) {
		why := ReasonReadback
		if _, uerr := k.API.ItemVault(ctx, res.ID); uerr != nil {
			why += "; it could not be removed again, check the backpack"
		} else {
			why += "; the new copy was removed again"
		}
		k.trip("read-back mismatch")
		_ = st.RevealFail(r.ID, revStored(why))
		return "reveal: " + r.Name + " not revealed: " + why, true
	}
	if err := st.SetStage(r.ID, vault.StageRevMade, 0); err != nil {
		return "reveal: " + r.ID + " stage: " + err.Error(), true
	}
	return k.finishReveal(ctx, st, r, res.ID)
}

// trip switches creating off for the process after a mismatch (shared with Keep).
func (k *Keeper) trip(why string) {
	k.mu.Lock()
	k.tripped = why
	k.mu.Unlock()
	k.note("tripped", "Keep, Reveal and Sell stopped for this run: a created item did not match the recorded one. Nothing else is created until the bot restarts.")
}

func (k *Keeper) revealCreateErr(ctx context.Context, st *vault.Store, r vault.Rec, err error) (string, bool) {
	wait := func(why string) (string, bool) {
		_ = st.SetStage(r.ID, "", 0)
		k.setWaiting(why)
		return "reveal: " + r.Name + " waits: " + why, true
	}
	fail := func(why string) (string, bool) {
		_ = st.RevealFail(r.ID, revStored(why))
		return "reveal: " + r.Name + " not revealed: " + why, false
	}
	switch {
	case errors.Is(err, d2dctl.ErrStashFull):
		k.note("packfull", "Reveal: "+ReasonPackFull+". The request stays queued.")
		return wait(ReasonPackFull)
	case errors.Is(err, d2dctl.ErrVaultState):
		return wait(deadOrNoGame(err))
	case errors.Is(err, d2dctl.ErrVaultOff), errors.Is(err, d2dctl.ErrVaultUnsupported):
		return wait(ReasonVerbs)
	case errors.Is(err, d2dctl.ErrBusy):
		return wait(ReasonBusy)
	case errors.Is(err, d2dctl.ErrBadHex):
		return fail(ReasonBadData)
	case errors.Is(err, d2dctl.ErrVaultVersion):
		return fail(ReasonVersion)
	case errors.Is(err, d2dctl.ErrVaultDup):
		// the key was used in this game: the copy exists or was identified and removed; the next run looks at the hero
		return "reveal: " + r.Name + " already created in this game, resolved at the next look", true
	}
	var vr *d2dctl.VaultRefused
	if errors.As(err, &vr) {
		return fail(vr.Why)
	}
	return "reveal: " + r.Name + " create unanswered, left in flight: " + err.Error(), true
}

// finishReveal identifies the created copy (unit), stores the identified bytes and removes the copy. A failed identify
// removes the copy and ends the request with the reason; the record is unchanged.
func (k *Keeper) finishReveal(ctx context.Context, st *vault.Store, r vault.Rec, unit int) (string, bool) {
	free := k.free()
	back, why, unknown := k.identifyUnit(ctx, r, unit, free)
	if unknown {
		return "reveal: " + r.Name + " " + why + "; left in flight", true
	}
	if why != "" {
		if _, err := k.API.ItemVault(ctx, unit); err != nil {
			_ = st.SetStage(r.ID, vault.StageRevMade, 0)
			return "reveal: " + r.Name + " not revealed: " + why + "; the temporary copy could not be removed, check the backpack", true
		}
		_ = st.RevealFail(r.ID, revStored(why))
		return "reveal: " + r.Name + " not revealed: " + why, why == ReasonNoScroll
	}
	if why = k.recordReveal(st, r, back); why != "" {
		_, _ = k.API.ItemVault(ctx, unit)
		_ = st.RevealFail(r.ID, revStored(why))
		return "reveal: " + r.Name + " not revealed: " + why, false
	}
	k.mu.Lock()
	k.info.Revealed++
	k.mu.Unlock()
	if _, err := k.API.ItemVault(ctx, unit); err != nil {
		return "reveal: " + r.Name + " revealed, the temporary copy could not be removed yet: " + err.Error(), true
	}
	_ = st.SetStage(r.ID, "", 0)
	where := "1 charge"
	if free {
		where = "free at Cain"
	}
	return "reveal: " + r.Name + " revealed (" + where + ")", false
}

// EnqueueRevealEther is the explicit tap "Reveal with Ether". The payment order of a reveal (docs/design/bulk-identify-2026-11-18.md
// section 12): (1) Cain stands here: the normal free reveal, no Ether; (2) Ether, picked by the player: the record is
// unlocked from the affixes the vault already holds (no scroll, no tome charge, no game step) and the Ether is charged ONCE
// after the unlock succeeded, ledger key reveal:<id>; a failed unlock or a failed charge leaves the record sealed and the
// wallet as it was. A short wallet or price 0 refuses the whole press before anything changes. No automatic path calls this.
func (k *Keeper) EnqueueRevealEther(ctx context.Context, ids []string, wantNow bool) []Result {
	if k.free() {
		return k.EnqueueReveal(ctx, ids, wantNow)
	}
	out := make([]Result, len(ids))
	st := k.store()
	k.FillSealed(ctx, ids)
	eligible := 0
	for i, id := range ids {
		out[i] = Result{ID: id}
		rec, found := vault.Rec{}, false
		if st != nil {
			rec, found = st.Get(id)
		}
		switch {
		case st == nil:
			out[i].Code, out[i].Why = "missing", "there is no vault for this hero yet"
		case !found:
			out[i].Code, out[i].Why = "missing", "no such item"
		case rec.Unlocked: // a repeated tap after a paid reveal: nothing changes, nothing is charged
			out[i].OK, out[i].Already = true, true
		case rec.Identified:
			out[i].Code, out[i].Why = "state", "it is identified already"
		case rec.State != vault.Recorded && rec.State != vault.KeepQ:
			out[i].Code, out[i].Why = "state", "only an item in the vault can be revealed (it is "+string(rec.State)+")"
		case rec.Sealed == "":
			out[i].Code, out[i].Why = "nosealed", ReasonNoSealed
		default:
			eligible++
		}
	}
	if eligible == 0 {
		return out
	}
	if err := k.Ether.Check(eligible); err != nil {
		code := "ether"
		if errors.Is(err, bulkid.ErrEtherOff) {
			code = "etheroff"
		}
		for i := range out {
			if out[i].Code == "" && !out[i].OK {
				out[i].Code, out[i].Why = code, err.Error()
			}
		}
		return out
	}
	for i := range out {
		if out[i].Code != "" || out[i].OK {
			continue
		}
		id := out[i].ID
		if err := st.Unseal(id); err != nil {
			out[i].Code, out[i].Why = codeOf(err), err.Error()
			continue
		}
		if err := k.Ether.Charge(id); err != nil {
			_ = st.Reseal(id)
			out[i].Code, out[i].Why = "ether", err.Error()
			continue
		}
		out[i].OK = true
		k.mu.Lock()
		k.info.Revealed++
		k.mu.Unlock()
	}
	return out
}

// FillSealed gives every unidentified record in ids that holds no affixes the ones its own bytes carry: the fork's read-only
// `item decode <raw>` answers the identified view of the dropped bytes, and the record keeps its property list as Sealed
// (Store.Seal), exactly as the recorder does for a drop whose peek carried them. Nothing is created in the game and no
// scroll, tome charge or town visit is needed. A fork without the verb, no game right now or a decode that finds no
// properties leaves the record as it was (the caller then refuses it as nosealed). It returns how many records were filled.
func (k *Keeper) FillSealed(ctx context.Context, ids []string) int {
	st := k.store()
	dec, ok := k.API.(d2dctl.Decoder)
	if st == nil || !ok {
		return 0
	}
	n := 0
	for _, id := range ids {
		rec, found := st.Get(id)
		if !found || rec.Identified || rec.Sealed != "" || rec.Raw == "" || (rec.State != vault.Recorded && rec.State != vault.KeepQ) {
			continue
		}
		got, err := dec.ItemDecode(ctx, rec.Raw)
		if err != nil || got.Props == "" || got.Code != rec.Code {
			continue
		}
		if st.Seal(id, got.Props) == nil {
			n++
		}
	}
	return n
}

// BackfillProps gives up to max identified records that carry no stat lines (marked revealed, no properties) the ones their
// own bytes carry, decoded by the fork (read-only in the game). It returns how many it filled.
func (k *Keeper) BackfillProps(ctx context.Context, max int) int {
	st := k.store()
	dec, ok := k.API.(d2dctl.Decoder)
	if st == nil || !ok || max <= 0 {
		return 0
	}
	// Pick the candidates under the store lock (the common answer is none, and no copy of the vault is made), then decode.
	var todo []vault.Rec
	st.Scan(func(r *vault.Rec) bool {
		if !r.Identified || r.Props != "" || r.Raw == "" || r.State != vault.Recorded && r.State != vault.KeepQ && r.State != vault.SellQ {
			return true
		}
		if _, done := k.nop.Load(r.ID); done {
			return true
		}
		todo = append(todo, *r)
		return len(todo) < max
	})
	n := 0
	for _, r := range todo {
		if ctx.Err() != nil {
			break
		}
		got, err := dec.ItemDecode(ctx, r.Raw)
		if err != nil {
			if errors.Is(err, d2dctl.ErrVaultOff) || errors.Is(err, d2dctl.ErrVaultUnsupported) || errors.Is(err, d2dctl.ErrVaultState) {
				break
			}
			continue
		}
		if got.Props != "" && got.Code == r.Code && st.SetProps(r.ID, got.Props) == nil {
			n++
		} else {
			k.nop.Store(r.ID, true)
		}
	}
	return n
}
