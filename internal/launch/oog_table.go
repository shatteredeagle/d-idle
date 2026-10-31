package launch

// Out-of-game (menu) geometry for the 800x600 client.
//
// Coordinates are VERIFIED LIVE on the worker (2026-10-28, 1.14d, GE-Proton11-11) unless the entry says
// otherwise; the method for each is in docs/oog.md. Convention (VERIFIED: control structs of the main
// menu, char select and char create screens were dumped from memory and matched against screenshots): a
// control is (type, x, y, w, h) where (x, y) is the LEFT-BOTTOM corner in client pixels, so the click
// point is (x + w/2, y - h/2). Entries with Verified=false were not observed.

// Control types (observed: 1 edit, 2 image, 4 label, 5 scrollbar, 6 button).
const (
	CtlAny    = -1
	CtlEdit   = 1
	CtlImage  = 2
	CtlLabel  = 4
	CtlScroll = 5
	CtlButton = 6
)

// Rect is one control's geometry. Type CtlAny (-1) matches any control type.
type Rect struct{ Type, X, Y, W, H int }

// Center is the click point inside the control (bottom-anchored y).
func (r Rect) Center() (x, y int) { return r.X + r.W/2, r.Y - r.H/2 }

// Button is a named clickable control.
type Button struct {
	Name     string
	R        Rect
	Verified bool
}

// ScreenID names an out-of-game screen.
type ScreenID string

const (
	ScreenUnknown    ScreenID = "unknown"
	ScreenSplash     ScreenID = "splash"       // logo/copyright screens right after launch (5 controls, UNVERIFIED signature)
	ScreenMainMenu   ScreenID = "main-menu"    // Single Player / Battle.net / Other Multiplayer / Exit
	ScreenCharSelect ScreenID = "char-select"  // list of single-player characters
	ScreenCharCreate ScreenID = "char-create"  // ONE screen: class portraits, name box, expansion/hardcore boxes, OK
	ScreenDifficulty ScreenID = "difficulty"   // Normal / Nightmare / Hell (UNVERIFIED: never appeared for a Normal-only character)
	ScreenPopupOK    ScreenID = "popup-ok"     // centered message with one button (VERIFIED: name taken, button reads "Cancel")
	ScreenPopupYesNo ScreenID = "popup-yes-no" // confirmation (UNVERIFIED)

	// LAN screens (VERIFIED LIVE 2026-10-28 on i1/i2, docs/net/multiplayer.md section 9). The path is
	// Main menu -> Other Multiplayer -> TCP/IP Game -> Host Game | Join Game -> char select -> in game.
	// The Other Multiplayer screen also holds "Open Battle.net": that button is deliberately NOT in the
	// table, so no code can click it.
	ScreenMultiMenu  ScreenID = "multi-menu"  // Other Multiplayer: [Open Battle.net], TCP/IP Game, Cancel
	ScreenTCPOptions ScreenID = "tcp-options" // TCP/IP Options: Host Game, Join Game, Cancel, "Your IP Address is"
	ScreenTCPJoinIP  ScreenID = "tcp-join-ip" // modal "Enter Host IP Address to Join Game" over tcp-options
)

// Screen is how to recognise a screen and what can be clicked on it. A screen matches when every Sig
// rect is present in the control list and no Not rect is. Order in Screens matters: popups first.
type Screen struct {
	ID      ScreenID
	Sig     []Rect
	Not     []Rect
	Buttons []Button
}

var (
	BtnBottomLeftExit = Button{"exit", Rect{CtlButton, 33, 572, 128, 35}, true}
	BtnBottomRightOK  = Button{"ok", Rect{CtlButton, 627, 572, 128, 35}, true}
	BtnPopupOK        = Button{"popup-ok", Rect{CtlButton, 351, 337, 96, 32}, true}
	BtnPopupYes       = Button{"popup-yes", Rect{CtlButton, 421, 337, 96, 32}, false}
	BtnPopupNo        = Button{"popup-no", Rect{CtlButton, 281, 337, 96, 32}, false}
	rectNameBox       = Rect{CtlEdit, 318, 510, 157, 16}
	rectCharCreateBtn = Rect{CtlButton, 33, 528, 168, 60}
	rectCharDeleteBtn = Rect{CtlButton, 433, 528, 168, 60}
	rectSinglePlayer  = Rect{CtlButton, 264, 324, 272, 35}
	rectMainMenuExit  = Rect{CtlButton, 264, 568, 272, 35}
	rectNormalSP      = Rect{CtlAny, 264, 297, 272, 35}
	rectNightmareSP   = Rect{CtlAny, 264, 340, 272, 35}
	rectHellSP        = Rect{CtlAny, 264, 383, 272, 35}

	// LAN screens (control dumps matched against screenshots, docs/net/multiplayer.md section 9).
	rectMultiOpen    = Rect{CtlButton, 264, 310, 272, 35} // Open Battle.net: signature only, never clicked
	rectMultiTCP     = Rect{CtlButton, 264, 350, 272, 35}
	rectHostGame     = Rect{CtlButton, 265, 206, 272, 35}
	rectJoinGame     = Rect{CtlButton, 265, 264, 272, 35}
	rectTCPCancel    = Rect{CtlButton, 39, 571, 128, 35}
	rectJoinIPBox    = Rect{CtlEdit, 300, 268, 213, 20}
	rectJoinIPCancel = Rect{CtlButton, 281, 337, 96, 32}
	rectJoinIPOK     = Rect{CtlButton, 421, 337, 96, 32}

	// RectCharHeader is the label at the top of char select showing the SELECTED character's name
	// (VERIFIED: text "Testy"/"Botty"). The slot controls themselves carry no text.
	RectCharHeader = Rect{CtlLabel, 85, 78, 466, 42}
	// RectClassHeader is the label on the create screen showing the picked class ("Sorceress", ...).
	RectClassHeader = Rect{CtlLabel, 0, 180, 800, 100}
)

// Screens is the data-driven screen/button table.
var Screens = []Screen{
	// The join-IP modal is drawn over tcp-options and also looks like a two-button popup: first.
	{ID: ScreenTCPJoinIP, Sig: []Rect{rectJoinIPBox, rectJoinIPOK, rectJoinIPCancel},
		Buttons: []Button{
			{"ip-box", rectJoinIPBox, true},
			{"ok", rectJoinIPOK, true},
			{"cancel", rectJoinIPCancel, true},
		}},
	{ID: ScreenPopupYesNo, Sig: []Rect{BtnPopupYes.R, BtnPopupNo.R},
		Buttons: []Button{BtnPopupYes, BtnPopupNo}},
	{ID: ScreenPopupOK, Sig: []Rect{BtnPopupOK.R}, Not: []Rect{BtnPopupYes.R},
		Buttons: []Button{BtnPopupOK}},
	{ID: ScreenDifficulty, Sig: []Rect{rectNormalSP, rectNightmareSP, rectHellSP},
		Buttons: []Button{
			{"normal", rectNormalSP, false},
			{"nightmare", rectNightmareSP, false},
			{"hell", rectHellSP, false},
		}},
	{ID: ScreenCharCreate, Sig: []Rect{rectNameBox},
		Buttons: []Button{
			{"name-box", rectNameBox, true},
			{"expansion-checkbox", Rect{CtlButton, 319, 540, 15, 16}, true}, // checked by default; toggle UNVERIFIED
			{"hardcore-checkbox", Rect{CtlButton, 319, 560, 15, 16}, true},  // position verified, toggling not
			BtnBottomRightOK, BtnBottomLeftExit,
		}},
	{ID: ScreenCharSelect, Sig: []Rect{rectCharCreateBtn, rectCharDeleteBtn},
		Buttons: []Button{
			{"create", rectCharCreateBtn, true},
			{"delete", rectCharDeleteBtn, true},
			{"convert", Rect{CtlButton, 233, 528, 168, 60}, true},
			BtnBottomRightOK, BtnBottomLeftExit,
		}},
	{ID: ScreenMainMenu, Sig: []Rect{rectSinglePlayer, rectMainMenuExit},
		Buttons: []Button{
			{"single-player", rectSinglePlayer, true},
			{"battle-net", Rect{CtlButton, 264, 366, 272, 35}, true},
			{"other-multiplayer", Rect{CtlButton, 264, 433, 272, 35}, true},
			{"exit", rectMainMenuExit, true},
		}},
	{ID: ScreenTCPOptions, Sig: []Rect{rectHostGame, rectJoinGame, rectTCPCancel},
		Buttons: []Button{
			{"host", rectHostGame, true},
			{"join", rectJoinGame, true},
			{"cancel", rectTCPCancel, true},
		}},
	{ID: ScreenMultiMenu, Sig: []Rect{rectMultiOpen, rectMultiTCP, rectMainMenuExit},
		Buttons: []Button{
			{"tcp-ip", rectMultiTCP, true},
			{"cancel", rectMainMenuExit, true}, // same rect as main-menu Exit; here it goes back
		}},
	{ID: ScreenSplash, Sig: []Rect{{CtlImage, 0, 599, 800, 600}}, Not: []Rect{rectSinglePlayer, rectCharCreateBtn, rectNameBox}},
}

// Char select slots: 2 columns x 4 rows of 200x92 controls (VERIFIED: control list; Botty was slot 0 and
// Testy slot 1 after the second character was created, newest first). Slot i is column i%2, row i/2;
// the click point is the control centre.
const CharSlots = 8

// SlotClick returns the click point of char-select slot i (0..7).
func SlotClick(i int) (x, y int) {
	col, row := i%2, i/2
	return Rect{CtlLabel, 37 + 272*col, 178 + 93*row, 200, 92}.Center()
}

// ClassClick is where to click on the create screen to pick a class. VERIFIED LIVE: each point was clicked
// and the class header label then read back the class name (see docs/oog.md). The control list's
// portrait rects are NOT the hit areas for assassin; kolbot's numbers were wrong for most classes.
// A click made within ~2 s of a previous class click is often ignored, so callers must verify the label.
var ClassClick = map[string][2]int{
	"amazon":      {100, 245},
	"assassin":    {200, 300},
	"necromancer": {301, 241},
	"barbarian":   {400, 238},
	"paladin":     {521, 247},
	"sorceress":   {626, 261},
	"druid":       {720, 278},
}

// In-game Esc menu (VERIFIED LIVE by screenshot: three rows OPTIONS y~210, SAVE AND EXIT GAME y~261,
// RETURN TO GAME y~310; clicking (400,261) saved and returned to the main menu). It has no controls in
// the D2Win list (FirstControl stays 0 in game), so it is found through the UI flag array instead.
var (
	EscKey         = "esc"
	BtnSaveAndExit = Button{"save-and-exit", Rect{CtlAny, 300, 276, 200, 30}, true} // centre (400,261)
)

// FindScreen returns the table entry for id.
func FindScreen(id ScreenID) (Screen, bool) {
	for _, s := range Screens {
		if s.ID == id {
			return s, true
		}
	}
	return Screen{}, false
}

// Btn returns the named button of a screen.
func (s Screen) Btn(name string) (Button, bool) {
	for _, b := range s.Buttons {
		if b.Name == name {
			return b, true
		}
	}
	return Button{}, false
}
