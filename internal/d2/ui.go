package d2

import "d2lod-bot/internal/mem"

// UI panel flags: DWORD array at RVA 0x3A27C0, index < 0x26. VERIFIED LIVE: GetUiVar_I (RVA 0x538D0)
// decodes to `cmp ecx,0x26; jb ok; ...; ok: mov eax,[0x7a27c0+ecx*4]; ret`, and toggling panels with
// the d2input helper flipped exactly the matching index (see docs/mem.md).
const (
	rvaUIFlags = 0x3A27C0
	uiCount    = 0x26
)

// UI indexes. VERIFIED LIVE by key toggle: Inventory (i), Character (c), SkillTree (t), Quest (q),
// EscMenu (seen open after Esc), MiniPanel (set at rest). The rest come from D2BS Constants.h and are UNVERIFIED.
const (
	UIInventory = 0x01
	UICharacter = 0x02
	UISkillTree = 0x04
	UIChat      = 0x05
	UINPCMenu   = 0x08
	UIEscMenu   = 0x09
	UIAutomap   = 0x0A
	UIShop      = 0x0C
	UIAltItems  = 0x0D
	UIQuest     = 0x0F
	UIWaypoint  = 0x14
	UIMiniPanel = 0x15
	UIParty     = 0x16
	UITrade     = 0x17
	UIStash     = 0x19
	UICube      = 0x1A
	UIBelt      = 0x1F
	UIMerc      = 0x24
)

// UIState is the raw panel flag array.
type UIState [uiCount]uint32

func (u *UIState) Open(idx int) bool { return idx >= 0 && idx < uiCount && u[idx] != 0 }

// UIKind says what a UI flag index means. This table is the ONE place for panel-flag semantics (docs/archive/review/findings.md
// F9): callers ask BlockingPanelOpen / EscClosableOpen and never keep their own index lists. An index that is not in the
// table is UIUnknown and never counts as a panel, so a newly found always-on flag cannot make the bot see a panel forever;
// UnknownOpen lists them for diagnostics.
type UIKind uint8

const (
	UIUnknown UIKind = iota
	// UIHUD flags never block clicks and Esc does not clear them: always-on HUD state (0x15 mini panel, 0x22, 0x23, 0x21 seen
	// after `h`), the red stat / skill point buttons 6 and 7 (set while points are unspent, VERIFIED LIVE: level 2, no panel
	// open, screenshot checked), the quest-log button 0x11 (stays set after a quest update, VERIFIED LIVE docs/town.md),
	// overlays (automap 0x0A, alt item labels 0x0D: U) and the belt 0x1F (U).
	UIHUD
	// UIPanel flags are panels, dialogs or menus that block world clicks and that Esc closes (the list the input layer
	// verified: docs/input.md; Esc menu 0x09 VERIFIED LIVE).
	UIPanel
)

var uiKinds = func() (k [uiCount]UIKind) {
	for _, i := range []int{0, 6, 7, 0x11, UIMiniPanel, 0x21, 0x22, 0x23, UIAutomap, UIAltItems, UIBelt} {
		k[i] = UIHUD
	}
	for _, i := range []int{UIInventory, UICharacter, 3, UISkillTree, UIChat, UINPCMenu, UIEscMenu, UIShop, UIQuest,
		UIWaypoint, UIParty, UITrade, UIStash, UICube, UIMerc} {
		k[i] = UIPanel
	}
	return
}()

// UIKindOf classifies a flag index (UIUnknown outside the array).
func UIKindOf(idx int) UIKind {
	if idx < 0 || idx >= uiCount {
		return UIUnknown
	}
	return uiKinds[idx]
}

// Indexes returns the open flag indexes of the given kind.
func (u *UIState) Indexes(kind UIKind) []int {
	var out []int
	for i := range u {
		if u[i] != 0 && uiKinds[i] == kind {
			out = append(out, i)
		}
	}
	return out
}

// BlockingPanelOpen reports whether a panel, dialog or menu that blocks world clicks (and that Esc closes) is open.
func (u *UIState) BlockingPanelOpen() bool { return len(u.Indexes(UIPanel)) > 0 }

// AnyPanelOpen is BlockingPanelOpen, kept for existing callers.
func (u *UIState) AnyPanelOpen() bool { return u.BlockingPanelOpen() }

// EscClosableOpen is true while something is open that Esc dismisses (what input.CloseAll loops on).
func (u *UIState) EscClosableOpen() bool { return u.BlockingPanelOpen() }

// UnknownOpen lists set flags the table does not know yet.
func (u *UIState) UnknownOpen() []int { return u.Indexes(UIUnknown) }

func (r *Reader) UI() (UIState, error) {
	var u UIState
	b, err := r.bytes(mem.Base+rvaUIFlags, uiCount*4)
	if err != nil {
		return u, err
	}
	for i := range u {
		u[i] = le32(b[i*4:])
	}
	return u, nil
}
