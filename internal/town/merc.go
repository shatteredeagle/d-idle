package town

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/resource"
)

// Mercenary hire and revive (agent LEVELERCLASS). Tags: V = verified live (how stated), S = secondary, U = not verified.
//
// What is known:
//   - hireling.txt (V, real tables): Normal Act 2 "Desert Mercenary", Id 6/7/8 = Combat (Prayer), Defensive (Defiance),
//     Offensive (Blessed Aim), seller class 198 (Greiz), gold factor 350, rows by merc level 9/31/55 (and 9/43/75). Might and
//     Holy Freeze only appear from Nightmare on (HirelingsFor(2, Nightmare)).
//   - Greiz is class 198, stands next to the Sewers stairs of Lut Gholein at about (5028,5052) in the seed of Palli on i2
//     (V live, 2026-10-28), and a nav walk to him (GoToUnit) stepped onto the stairs and killed the character twice;
//     talking to him therefore clicks him from a distance and lets the game walk (Town.noWalk).
//
//   - V live (Palli, clvl 13, i2, 2026-10-28): Greiz's dialog is Talk / Hire / Cancel (screenshot; Hire = line 1 of 3).
//     The hire window lists mercenaries as rows "Name - Lvl: n  Life  Def  Cost: c" with the TYPE as text under the name
//     ("Defensive", "Combat", "Offensive" = Defiance, Prayer, Blessed Aim in Normal) and a Cancel button; the order is
//     per game and the type is not in memory (U: no reader), so the aura of a row is only known from the screen.
//     A single hover over a row's text at x 250 did not highlight or select it; two hover moves at x 400/402 (2 px apart)
//     highlighted the row (blue) and ONE click hired it at once, with no confirmation: gold fell 20000 -> 19650 (the row
//     said 360), the portrait "Pratham" appeared top left and a spear-armed Desert Mercenary stood beside the player.
//     Row text lines were at y 136 + 33 i (i = 0..6, seven visible, a scrollbar for more: U); only row 4 was clicked.
//
// What is NOT known (U): which row carries which aura without reading the screen (hence MercWant.Row or a RowPicker),
// rows beyond the seven visible, the revive line (the merc was never dead), the merc's inventory.

// MercWant says which mercenary to hire.
type MercWant struct {
	Act     int             // 1-based act of the hireling NPC (2 = Desert Mercenary)
	Diff    data.Difficulty // difficulty of the table rows (Normal for the first character)
	Aura    string          // skills.txt name of the wanted aura: "Blessed Aim", "Defiance", "Prayer", "Might" ...
	MaxGold int             // do not pay more than this (0 = whatever the gold allows)
	// Row is the hire-window row to click (0 = top, V seven visible). A negative Row asks Pick; without a picker the
	// hire stops with ErrMercRowUnknown because the aura of a row is only visible on screen (U: no memory reader).
	Row  int
	Pick RowPicker
}

// RowPicker chooses a hire-window row for a want: for example a screenshot reader that looks for MercTypeLabel(aura).
type RowPicker func(ctx context.Context, w MercWant) (row int, err error)

// MercTypeLabel is the type text the hire window prints for an aura in Normal (V: Defensive, Combat, Offensive screen
// text; the mapping to auras is hireling.txt V: Id 7 Defiance, Id 6 Prayer, Id 8 Blessed Aim).
func MercTypeLabel(aura string) string {
	switch strings.ToLower(aura) {
	case "defiance":
		return "Defensive"
	case "prayer":
		return "Combat"
	case "blessed aim":
		return "Offensive"
	}
	return ""
}

// MercChoice is one hireable row: which hireling.txt row, its skill and the row index on the hire window.
type MercChoice struct {
	Row   *data.Hireling
	Aura  string
	Index int // position among the candidates in file order (the hire window lists them in this order: U)
}

// MercCandidates lists the hireling types of the act and difficulty that carry the aura (one entry per hireling.txt
// Id, first row of the id: the hire window shows merc levels 8-10 at clvl 13, which row applies per level is U).
// Pure, tested against real tables on the worker.
func MercCandidates(db *data.DB, w MercWant) []MercChoice {
	best := map[int]*data.Hireling{}
	var ids []int
	for _, h := range db.HirelingsFor(w.Act, w.Diff) {
		if _, ok := best[h.ID]; !ok {
			ids = append(ids, h.ID)
			best[h.ID] = h
		}
	}
	sort.Ints(ids)
	var out []MercChoice
	for i, id := range ids {
		h := best[id]
		a := mercAura(h)
		if w.Aura == "" || strings.EqualFold(a, w.Aura) {
			out = append(out, MercChoice{Row: h, Aura: a, Index: i})
		}
	}
	return out
}

// mercAura is the aura skill of a hireling row: the first listed skill that skills.txt flags as an aura is not known
// here without the DB, so the Desert Mercenary rule is used: Skill1 is the melee attack (Jab), Skill2 the aura.
func mercAura(h *data.Hireling) string {
	if len(h.Skills) >= 2 {
		return h.Skills[1].Name
	}
	return ""
}

// HireReport is what a hire attempt did.
type HireReport struct {
	Hired   bool
	Choice  MercChoice
	Spent   int
	Steps   []string
	Skipped string
}

// ReviveReport is what a revive attempt did.
type ReviveReport struct {
	Revived bool
	Spent   int
	Steps   []string
	Skipped string
}

// Hirer is the mercenary contract for the bot: hire once, revive after a death. Implemented by *Town.
type Hirer interface {
	Hire(ctx context.Context, w MercWant) (HireReport, error)
	Revive(ctx context.Context) (ReviveReport, error)
}

var _ Hirer = (*Town)(nil)

var (
	// ErrMercUnverified: a step needs a click position that no live run has verified yet.
	ErrMercUnverified = errors.New("town: mercenary dialog step not verified live")
	ErrNoMercNPC      = errors.New("town: no mercenary NPC in this act")
	ErrNoCandidate    = errors.New("town: no hireling row matches")
	// ErrMercRowUnknown: no row to click was given and the aura of a row is only readable on screen.
	ErrMercRowUnknown = errors.New("town: hire row unknown (set MercWant.Row or Pick)")
)

// MercMenu is the dialog of the hireling NPC top to bottom: Greiz (class 198) Talk, Hire, Cancel, V live. EntryTrade
// stands for "Hire". Other acts' hireling NPCs (Kashya 150 has Talk, Cancel only in MenuOrder; Asheara 252, Qual-Kehk
// 515) are U.
var MercMenu = map[int][]Entry{198: {EntryTalk, EntryTrade, EntryCancel}}

// Hire window geometry at 800x600, V live on Greiz for row 4 (the others follow the 33 px pitch of the screenshot, U).
const (
	MercRowX     = 400 // hover x; x 250 on the same row did not highlight (V)
	MercRowY0    = 136 // text line of row 0
	MercRowPitch = 33
	MercRowCount = 7 // visible rows; more need the scrollbar (U)
)

// HasMerc reports a living mercenary anywhere in the unit table (d2.FindMerc, by class: the owner words read 0/0, V
// docs/friendly.md). The old 40-subtile proximity limit is gone: a merc left behind in another area is absent from
// the table, and one that lags far behind in the same area is still alive.
func HasMerc(s *d2.Snapshot) bool { return d2.FindMerc(s).Alive }

// isMercClass asks "is this a hireling", not "is it friendly" (golems are friendly, not mercs).
func isMercClass(c int) bool { return d2.IsMercClass(c) }

// HirePolicy decides what to hire in Act 2 given the merc state and gold: nothing while one lives (a second hire is
// refused or destroys the old gear, S/U), and by difficulty Normal Defiance, Nightmare Might (the only difficulty
// with Might, V hireling.tsv: Hell rows are Prayer/Defiance/Blessed Aim again), Hell Defiance. The cost is the factor
// of the table row times merc level, so only a floor is checked here; Hire verifies by the gold falling (V).
func HirePolicy(diff data.Difficulty, merc d2.Merc, gold int) (MercWant, bool) {
	if merc.Alive {
		return MercWant{}, false
	}
	w := MercWant{Act: 2, Diff: diff, Aura: "Defiance", Row: -1}
	floor := 400 // Normal: factor 350 read 360 at merc level 9 (V docs/town.md)
	switch diff {
	case data.Nightmare:
		w.Aura, floor = "Might", 8000 // U: scaled from factor 7900
	case data.Hell:
		floor = 15000 // U: scaled from factor 15000
	}
	return w, gold >= floor
}

// Hire talks to the act's mercenary NPC and clicks one row of the hire window. The click needs the row: w.Row or w.Pick
// (see MercWant); without either the dialog is never opened. Proof: the gold falls. The NPC is clicked from where the
// character stands (noWalk): Greiz stands beside the Sewers hatch and a nav walk to him killed Palli twice (V live).
func (t *Town) Hire(ctx context.Context, w MercWant) (HireReport, error) {
	var rep HireReport
	s, err := t.current()
	if err != nil {
		return rep, err
	}
	a := ActByNo(w.Act)
	if a == nil || len(a.Merc) == 0 {
		return rep, ErrNoMercNPC
	}
	if cur := ActOfLevel(s.Player.Area); cur == nil || cur.No != w.Act {
		return rep, fmt.Errorf("%w: not in act %d", ErrNotInTown, w.Act)
	}
	if _, ok := MercMenu[a.Merc[0]]; !ok {
		rep.Skipped = "dialog of this hireling NPC unverified"
		return rep, ErrMercUnverified
	}
	cands := MercCandidates(t.e.Data, w)
	if len(cands) == 0 {
		return rep, fmt.Errorf("%w: act %d aura %q", ErrNoCandidate, w.Act, w.Aura)
	}
	rep.Choice = cands[0]
	rep.Steps = append(rep.Steps, fmt.Sprintf("want %s %s (hireling id %d)", rep.Choice.Row.Name, rep.Choice.Aura, rep.Choice.Row.ID))
	row := w.Row
	if w.Pick != nil {
		if row, err = w.Pick(ctx, w); err != nil {
			return rep, err
		}
	} else if row < 0 {
		return rep, ErrMercRowUnknown
	}
	if row >= MercRowCount {
		return rep, fmt.Errorf("%w: row %d is below the seven visible rows", ErrMercUnverified, row)
	}
	t.noWalk = true
	defer func() { t.noWalk = false }()
	npc, err := t.openNPC(ctx, a.Merc, t.npcPreset(a.Merc...))
	if err != nil {
		return rep, err
	}
	rep.Steps = append(rep.Steps, "dialog open")
	before := s.Player.Gold
	// PickLine hovers before the click; the plain pickEntry click did not open the hire window in the live run (V).
	idx, n, _ := menuLine(npc.Class, EntryTrade)
	if err := t.PickLine(ctx, npc, idx, n); err != nil {
		return rep, err
	}
	t.e.Clock.Sleep(1500 * time.Millisecond) // the window has no flag we know (U)
	x, y := MercRowX, MercRowY0+MercRowPitch*row+3
	// Four hover moves (3 px away, on the row, 2 px aside, back) with long pauses, then one click. V live: this took
	// 10 s per step by hand (townlive talk h: w:10 h: w:10 k:) and hired; the pauses here are a guess (U) until a run
	// with shorter ones is verified.
	for _, p := range [][2]int{{x - 3, y}, {x, y}, {x + 2, y + 2}, {x + 2, y}} {
		if err := t.e.In.MoveMouse(p[0], p[1]); err != nil {
			return rep, err
		}
		t.e.Clock.Sleep(3 * time.Second)
	}
	if err := t.e.In.Click(input.Left, x+2, y); err != nil {
		return rep, err
	}
	s2, ok, err := t.h.Wait(ctx, 4*time.Second, func(s *d2.Snapshot) bool { return s.Player.Gold < before })
	if err != nil || !ok {
		_ = t.h.CloseAll(ctx)
		return rep, fmt.Errorf("town: hire not confirmed (gold unchanged): %v", err)
	}
	rep.Hired, rep.Spent = true, before-s2.Player.Gold
	rep.Steps = append(rep.Steps, fmt.Sprintf("hired row %d for %d gold", row, rep.Spent))
	return rep, t.h.CloseAll(ctx)
}

// ReviveMenu is the dialog of a hireling NPC while the merc is dead, top to bottom. U: never seen live (the merc was
// never dead); the line order and the very existence of a "Resurrect" line are guesses from Kolbot and the community
// pages (S docs/merc-farm.md 1.3). ReviveLine overrides the index once a live run found it.
var (
	ReviveMenu = []Entry{EntryTalk, EntryTrade, EntryResurrect, EntryCancel}
	ReviveLine = -1
)

// reviveNPCs lists the hireling NPC classes that revive any merc: Kashya 150, Greiz 198, Asheara 252, Qual-Kehk 515
// and Tyrael 367 in Act 4 (S: Hive Workshop and Maxroll; the Tyrael class is U, from quest/dialog.go and Kolbot).
var reviveNPCs = map[int][]int{1: {150}, 2: {198}, 3: {252}, 4: {367}, 5: {515}}

// ReviveReady is the pure pre-flight of a revive: the merc must be known and dead, the act must have a reviver, and
// the gold must cover the price. level 0 prices at 0 (unknown level).
func ReviveReady(act int, merc d2.Merc, known bool, gold, level int) error {
	if _, ok := reviveNPCs[act]; !ok {
		return ErrNoMercNPC
	}
	if merc.Alive {
		return ErrMercAlive
	}
	if !known && !merc.Present {
		return ErrNoMerc
	}
	if need := resource.MercReviveCost(level); gold < need {
		return fmt.Errorf("%w: have %d, need about %d", ErrMercGold, gold, need)
	}
	return nil
}

var (
	ErrMercAlive = errors.New("town: the mercenary is alive")
	ErrNoMerc    = errors.New("town: no mercenary was ever hired")
	ErrMercGold  = errors.New("town: not enough gold to revive")
)

// SetReviveUnverified lets Revive click the U dialog (live probe only). Off by default: the skeleton refuses with
// ErrMercUnverified so nothing clicks an unseen dialog.
func (t *Town) SetReviveUnverified(on bool) { t.reviveUnverified = on }

// SetMercKnown tells Revive that a merc was hired this run (a dead merc may vanish from the unit table, U).
func (t *Town) SetMercKnown(known bool) { t.mercKnown = known }

// Revive talks to the current act's hireling NPC and picks the resurrect line, then proves it by the gold falling and a
// living merc unit in the table. Needs SetReviveUnverified(true): every step after opening the dialog is U.
func (t *Town) Revive(ctx context.Context) (ReviveReport, error) {
	var rep ReviveReport
	s, err := t.current()
	if err != nil {
		return rep, err
	}
	a := ActOfLevel(s.Player.Area)
	if a == nil {
		return rep, ErrNotInTown
	}
	m := d2.FindMerc(s)
	if err := ReviveReady(a.No, m, t.mercKnown, s.Player.Gold, m.Level); err != nil {
		rep.Skipped = err.Error()
		return rep, err
	}
	if !t.reviveUnverified {
		rep.Skipped = "revive dialog unverified"
		return rep, ErrMercUnverified
	}
	classes := reviveNPCs[a.No]
	t.noWalk = true
	defer func() { t.noWalk = false }()
	npc, err := t.openNPC(ctx, classes, t.npcPreset(classes...))
	if err != nil {
		return rep, err
	}
	rep.Steps = append(rep.Steps, "dialog open")
	idx, n := ReviveLine, len(ReviveMenu)
	if idx < 0 {
		for i, e := range ReviveMenu {
			if e == EntryResurrect {
				idx = i
			}
		}
	}
	before := s.Player.Gold
	if err := t.PickLine(ctx, npc, idx, n); err != nil {
		return rep, err
	}
	s2, ok, err := t.h.Wait(ctx, 4*time.Second, func(s *d2.Snapshot) bool {
		return s.Player.Gold < before && d2.FindMerc(s).Alive
	})
	if err != nil || !ok {
		_ = t.h.CloseAll(ctx)
		return rep, fmt.Errorf("town: revive not confirmed (gold or merc unchanged): %v", err)
	}
	rep.Revived, rep.Spent = true, before-s2.Player.Gold
	rep.Steps = append(rep.Steps, fmt.Sprintf("revived for %d gold", rep.Spent))
	return rep, t.h.CloseAll(ctx)
}
