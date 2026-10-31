package quest

import (
	"context"
	"errors"
	"fmt"
)

// NPC class ids (monstats hcIdx). V live: Gheed 147, Akara 148, Kashya 150, Charsi 154, Warriv 155 (docs/map.md).
// Everything else is U (game data and Kolbot constants; check with `townlive units` in the act's town).
const (
	Gheed    = 147
	Akara    = 148
	Kashya   = 150
	Charsi   = 154
	Warriv   = 155
	CainTown = 245 // U: Deckard Cain in the Rogue Encampment after the rescue (acts.go lists 146, 244, 245, ...)
	WarrivA2 = 175 // U
	Jerhyn   = 201 // U
	Meshif   = 210 // U (class of the A2 harbour Meshif; the A3 one may differ)
	Hratli   = 253 // U
	Tyrael   = 367 // U (A4 Pandemonium Fortress)
)

// Object classes (U unless noted).
const (
	ObjTreeInifuss  = 30 // U: Tree of Inifuss, Dark Wood
	ObjCairnStone   = 17 // U: Stony Field cairn stones (class of the first stone)
	ObjCainGibbet   = 26 // U: Cain's cage in Tristram
	ObjRedPortal    = 60 // generic "Portal": V live at the Arcane/Canyon portal (docs/act2.md), S Kolbot for Anya's portal in Harrogath; NOT the Act 3 to 4 portal, see ObjRedPortalA3
	ObjTownPortal   = 59 // V-ish: town.ObjPortal (UNVERIFIED live there too)
	ObjWaypointAct1 = 119
	ObjWaypointAct2 = 156
	ObjWaypointAct3 = 237
	ObjWaypointAct4 = 398
	ObjWaypointAct5 = 429
)

// Menu levels of an NPC dialog.
const (
	LevelMain = "main" // Talk / Trade / ... / Cancel
	LevelTalk = "talk" // the Talk submenu (Introduction / Gossip / quest rows / Cancel)
)

// Row is one line of a dialog, top to bottom. Cancel is always last. Next names the submenu a click opens
// ("" = a leaf that acts, closes, or opens a window). When, if set, makes the row present only while the quest
// has at least that status and not more than Until (U: rows appear and disappear with quest state).
type Row struct {
	Label string
	Tag   string // V or U
	Next  string
	When  *Cond
	Until Status // exclusive upper bound when When is set; 0 means no bound
}

// Menu is the rows of one level of one NPC.
type Menu struct {
	NPC   int
	Level string
	Rows  []Row
}

// Menus is the dialog table as data. Only Akara's rows are live-verified (docs/progression/results.md V-04,
// 2026-10-28); everything else is U and must be confirmed with the checks in docs/quest.md before a handler
// relies on a row index. Handlers pick rows by Label through RowIndex, so a corrected table fixes them all.
var Menus = []Menu{
	// V-04: Talk menu = TALK / TRADE / CANCEL; Talk submenu = Introduction / Gossip / Den of Evil / Cancel.
	// V (i4, 2026-10-28): once the Den reward is paid the main menu gains "Reset Stat/Skill Points" (quest word 41 = 0x2002,
	// then 0x8002), so Talk/Trade sit one line higher. RespecRow is the pseudo quest the caller answers for it.
	{Akara, LevelMain, []Row{
		{"Talk", "V", LevelTalk, nil, 0},
		{"Trade", "V", "", nil, 0},
		{"Reset Stat/Skill Points", "V", "", &Cond{RespecRow, Started}, 0},
		{"Cancel", "V", "", nil, 0}}},
	{Akara, LevelTalk, []Row{
		{"Introduction", "V", "", nil, 0},
		{"Gossip", "V", "", nil, 0},
		{"Den of Evil", "V", "", &Cond{"den_of_evil", Needed}, Done},
		// V (screenshot, i4 2026-10-28): once Sisters' Burial Grounds is started the list gains this row, so the line count
		// (and with it every click position of the bottom-anchored box) depends on the quest states.
		{"Sisters' Burial Grounds", "V", "", &Cond{"sisters_burial_grounds", Started}, Done},
		{"Cancel", "V", "", nil, 0}}},

	{Kashya, LevelMain, []Row{{"Talk", "U", LevelTalk, nil, 0}, {"Hire", "U", "", nil, 0}, {"Cancel", "U", "", nil, 0}}},
	{Kashya, LevelTalk, []Row{{"Introduction", "U", "", nil, 0}, {"Gossip", "U", "", nil, 0}, {"Cancel", "U", "", nil, 0}}},

	// Charsi: the Malus turn-in is a Talk with the item in the pack (S); the Imbue row appears after (U).
	{Charsi, LevelMain, []Row{
		{"Talk", "U", LevelTalk, nil, 0}, {"Trade", "U", "", nil, 0},
		{"Imbue", "U", "", &Cond{"tools_of_the_trade", Started}, Done},
		{"Cancel", "U", "", nil, 0}}},
	{Charsi, LevelTalk, []Row{{"Introduction", "U", "", nil, 0}, {"Gossip", "U", "", nil, 0}, {"Cancel", "U", "", nil, 0}}},

	// Warriv: the travel row sits above Cancel (S, acts.md: "Go East"); text is not readable from memory.
	{Warriv, LevelMain, []Row{{"Talk", "U", LevelTalk, nil, 0}, {"Go East", "U", "", &Cond{"act1_done", Done}, 0}, {"Cancel", "U", "", nil, 0}}},
	{Warriv, LevelTalk, []Row{{"Introduction", "U", "", nil, 0}, {"Gossip", "U", "", nil, 0}, {"Cancel", "U", "", nil, 0}}},

	{CainTown, LevelMain, []Row{{"Talk", "U", LevelTalk, nil, 0}, {"Identify", "U", "", nil, 0}, {"Cancel", "U", "", nil, 0}}},
	{CainTown, LevelTalk, []Row{{"Introduction", "U", "", nil, 0}, {"Gossip", "U", "", nil, 0}, {"Cancel", "U", "", nil, 0}}},

	{Jerhyn, LevelMain, []Row{{"Talk", "U", LevelTalk, nil, 0}, {"Cancel", "U", "", nil, 0}}},
	{Jerhyn, LevelTalk, []Row{{"Introduction", "U", "", nil, 0}, {"Gossip", "U", "", nil, 0}, {"Cancel", "U", "", nil, 0}}},

	{Meshif, LevelMain, []Row{{"Talk", "U", LevelTalk, nil, 0}, {"Sail East", "U", "", &Cond{"act2_done", Done}, 0}, {"Cancel", "U", "", nil, 0}}},
	{Meshif, LevelTalk, []Row{{"Introduction", "U", "", nil, 0}, {"Gossip", "U", "", nil, 0}, {"Cancel", "U", "", nil, 0}}},

	{Tyrael, LevelMain, []Row{{"Talk", "U", LevelTalk, nil, 0}, {"Travel to Harrogath", "U", "", &Cond{"terrors_end", Done}, 0}, {"Cancel", "U", "", nil, 0}}},
	{Tyrael, LevelTalk, []Row{{"Introduction", "U", "", nil, 0}, {"Gossip", "U", "", nil, 0}, {"Cancel", "U", "", nil, 0}}},
}

// RespecRow is the pseudo quest id the status callback answers (Started = yes) for Akara's fourth main-menu line.
const RespecRow = "akara_respec"

type subKey struct {
	NPC          int
	Level, Label string
}

// subWhen limits a row's submenu to a quest status range [Want, until): outside it the click is a leaf (a speech, or
// the reward). Akara: the topic list (Introduction / Gossip / Den of Evil / ...) exists only while Den of Evil is
// Needed; at 0x0004 (talked), 0x001c (cleared, not reported), 0x001e (reward pending) and done her Talk plays a
// speech and pays the pending reward directly. V live on i4 (2026-10-28): 0x001e pays the skill point at the first
// Talk, 0x001c and 0x0004 show no list, Needed shows Talk / Introduction / Gossip / Den of Evil / Cancel.
var subWhen = map[subKey]struct {
	Cond  Cond
	Until Status
}{
	{Akara, LevelMain, "Talk"}: {Cond{"den_of_evil", Needed}, Started},
}

// HasSubmenu reports whether clicking the row opens its Next level in the current quest state.
func HasSubmenu(npc int, level string, row Row, status func(string) Status) bool {
	if row.Next == "" {
		return false
	}
	w, ok := subWhen[subKey{npc, level, row.Label}]
	if !ok || status == nil {
		return true
	}
	s := status(w.Cond.Quest)
	return s >= w.Cond.Want && (w.Until == 0 || s < w.Until)
}

var (
	ErrNoMenu = errors.New("quest: no dialog table for npc/level")
	ErrNoRow  = errors.New("quest: row not present in the dialog table")
	// ErrNoSubmenu: the path wants a submenu the quest state does not show (Akara's Talk after Den of Evil started).
	ErrNoSubmenu = errors.New("quest: the row opens no submenu in this quest state")
	ErrTurnIn    = errors.New("quest: item still in the pack after the turn-in")
)

// Visible returns the rows of a menu that are present for the given quest state (status may be nil: all rows).
func (m Menu) Visible(status func(quest string) Status) []Row {
	var out []Row
	for _, r := range m.Rows {
		if r.When != nil && status != nil {
			s := status(r.When.Quest)
			if s < r.When.Want || (r.Until != 0 && s >= r.Until) {
				continue
			}
		}
		out = append(out, r)
	}
	return out
}

// FindMenu looks up one level of one NPC.
func FindMenu(npc int, level string) (Menu, bool) {
	for _, m := range Menus {
		if m.NPC == npc && m.Level == level {
			return m, true
		}
	}
	return Menu{}, false
}

// RowIndex is the index from the top and the line count of the named row, for the current quest state.
func RowIndex(npc int, level, label string, status func(string) Status) (idx, n int, row Row, err error) {
	m, ok := FindMenu(npc, level)
	if !ok {
		return 0, 0, Row{}, fmt.Errorf("%w: npc %d level %q", ErrNoMenu, npc, level)
	}
	rows := m.Visible(status)
	for i, r := range rows {
		if r.Label == label {
			return i, len(rows), r, nil
		}
	}
	return 0, len(rows), Row{}, fmt.Errorf("%w: npc %d %q %q", ErrNoRow, npc, level, label)
}

// Dialog is the NPC dialog primitive. The real implementation lives in internal/town (approach, Interact,
// wait for UI flag 0x08, wait out the first-contact greeting) and clicks bottom-anchored lines.
type Dialog interface {
	// Talk opens the dialog of the NPC class and returns when the main menu is up.
	Talk(ctx context.Context, npcClass int) error
	// Choose clicks line row (0 = top) of a menu that has of lines in total (the box is bottom-anchored, so
	// the click point needs the count). A row that opens a submenu leaves the dialog open at that level.
	Choose(ctx context.Context, row, of int) error
	// Close leaves the dialog (Cancel row or Esc) and returns once UI flag 0x08 is clear.
	Close(ctx context.Context) error
}

// Pick clicks the labelled row of the NPC's menu at level and returns the row (for Next).
func Pick(ctx context.Context, d Dialog, npc int, level, label string, status func(string) Status) (Row, error) {
	idx, n, row, err := RowIndex(npc, level, label, status)
	if err != nil {
		return Row{}, err
	}
	return row, d.Choose(ctx, idx, n)
}

// PickPath walks labels through the levels: the first at LevelMain, each following label at the Next level of
// the one before. Example: PickPath(.., Akara, "Talk", "Den of Evil").
func PickPath(ctx context.Context, d Dialog, npc int, status func(string) Status, labels ...string) error {
	level := LevelMain
	for _, l := range labels {
		if level == "" {
			return fmt.Errorf("%w: %q after %v", ErrNoSubmenu, l, labels)
		}
		row, err := Pick(ctx, d, npc, level, l, status)
		if err != nil {
			return err
		}
		if !HasSubmenu(npc, level, row, status) {
			row.Next = ""
		}
		level = row.Next
	}
	return nil
}

// TurnIn describes handing a quest item over. In the real game most turn-ins happen on the Talk itself with
// the item in the pack (S); Path is the fallback row path when the item is still there after the Talk.
type TurnIn struct {
	NPC  int
	Item string   // item code that must leave the pack
	Path []string // labels for PickPath; may be empty
}

// Has reports whether the pack holds an item code.
type Has interface{ Has(code string) bool }

// DoTurnIn opens the NPC, lets the Talk do the hand-over, falls back to Path, closes, and proves the item left
// the pack (T3 evidence: the caller still records it as such).
func DoTurnIn(ctx context.Context, d Dialog, inv Has, status func(string) Status, t TurnIn) error {
	if err := d.Talk(ctx, t.NPC); err != nil {
		return err
	}
	defer d.Close(ctx)
	if !inv.Has(t.Item) {
		return nil
	}
	if len(t.Path) > 0 {
		if err := PickPath(ctx, d, t.NPC, status, t.Path...); err != nil {
			return err
		}
	}
	if inv.Has(t.Item) {
		return fmt.Errorf("%w: %s", ErrTurnIn, t.Item)
	}
	return nil
}
