package launch

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf16"

	"d2lod-bot/internal/mem"
)

// RVAs of the two canaries, relative to the Game.exe image base. Both VERIFIED live in wave 1
// (docs/d2-1.14d-memory.md 2.1; PLAN.md): PlayerUnit non-zero in game, FirstControl non-zero in menus.
const (
	rvaPlayerUnit   = 0x3A6A70
	rvaFirstControl = 0x3D55BC
)

// Phase is the coarse client state from the two canaries (D2BS ClientState logic, V D2Helpers.cpp:125-147).
type Phase int

const (
	PhaseUnknown Phase = iota // neither pointer set: loading, or between screens
	PhaseMenu                 // FirstControl set, no player
	PhaseInGame               // player set, no controls
	PhaseBoth                 // both set: transitional, treat as loading
)

func (p Phase) String() string {
	return [...]string{"unknown", "menu", "in-game", "both"}[p]
}

// ReadPhase reads the canaries.
func ReadPhase(s mem.Source) (Phase, error) {
	pl, err := mem.U32(s, mem.Base+rvaPlayerUnit)
	if err != nil {
		return PhaseUnknown, err
	}
	fc, err := mem.U32(s, mem.Base+rvaFirstControl)
	if err != nil {
		return PhaseUnknown, err
	}
	switch {
	case pl != 0 && fc == 0:
		return PhaseInGame, nil
	case pl == 0 && fc != 0:
		return PhaseMenu, nil
	case pl != 0 && fc != 0:
		return PhaseBoth, nil
	}
	return PhaseUnknown, nil
}

// Control is one entry of the D2Win control list.
type Control struct {
	Rect
	Text string // first text line, "" if none
}

// ReadControls walks the control list from FirstControl. Layout VERIFIED LIVE (dumped on the main menu,
// char select and create screens, matched to screenshots): +0 type, +0xC x, +0x10 y (bottom), +0x14 w,
// +0x18 h, +0x3C next, +0x48 ControlText whose +0 is a wchar_t* of the first text line. The "disabled"
// bits (+8, +0x44) did not behave as D2BS documents (+8 was 5 on every control), so Disabled is not set.
func ReadControls(s mem.Source) ([]Control, error) {
	p, err := mem.U32(s, mem.Base+rvaFirstControl)
	if err != nil {
		return nil, err
	}
	var out []Control
	for n := 0; p != 0 && n < 1024; n++ {
		var b [0x4C]byte
		if err := s.ReadAt(b[:], p); err != nil {
			return out, err
		}
		u := func(o int) uint32 { return uint32(b[o]) | uint32(b[o+1])<<8 | uint32(b[o+2])<<16 | uint32(b[o+3])<<24 }
		c := Control{
			Rect: Rect{int(u(0)), int(u(0xC)), int(u(0x10)), int(u(0x14)), int(u(0x18))},
		}
		if t := u(0x48); t != 0 {
			if tp, err := mem.U32(s, t); err == nil && tp != 0 {
				c.Text = readUTF16(s, tp, 64)
			}
		}
		out = append(out, c)
		p = u(0x3C)
	}
	return out, nil
}

func readUTF16(s mem.Source, addr uint32, max int) string {
	var u []uint16
	for i := 0; i < max; i++ {
		w, err := mem.U16(s, addr+uint32(2*i))
		if err != nil || w == 0 {
			break
		}
		u = append(u, w)
	}
	return string(utf16.Decode(u))
}

func has(cs []Control, r Rect) bool {
	for _, c := range cs {
		if (r.Type == CtlAny || c.Type == r.Type) && c.X == r.X && c.Y == r.Y && c.W == r.W && c.H == r.H {
			return true
		}
	}
	return false
}

// Identify returns the first table screen whose signature is present in cs, else ScreenUnknown.
func Identify(cs []Control) ScreenID {
next:
	for _, sc := range Screens {
		for _, r := range sc.Sig {
			if !has(cs, r) {
				continue next
			}
		}
		for _, r := range sc.Not {
			if has(cs, r) {
				continue next
			}
		}
		return sc.ID
	}
	return ScreenUnknown
}

// Input is what the flow needs from internal/input (a thin adapter satisfies it): a left click at client
// pixels and a named key press (names as accepted by cmd/d2input: a-z, 0-9, enter, esc, ...).
// If the value also implements Typer, text is typed with WM_CHAR (VERIFIED LIVE to fill the name box,
// case preserved); otherwise lower-case key presses are used (UNVERIFIED for edit boxes).
type Input interface {
	Click(x, y int) error
	Key(name string) error
}

// Typer is the optional WM_CHAR path (input.Client.TypeChar).
type Typer interface{ TypeChar(r rune) error }

// CharSpec describes the character the flow should end up in.
type CharSpec struct {
	Name       string // letters/digits only
	Class      string // key of ClassClick; used only when the character is created
	Create     bool   // create the character when it is not in the list
	ForceNew   bool   // always create (CreateCharacter): a name clash is an error, never a selection
	Hardcore   bool
	Classic    bool   // true: untick the expansion box (default is LoD); toggle UNVERIFIED
	Difficulty string // "normal", "nightmare" or "hell" for the difficulty screen (UNVERIFIED screen)

	// LAN: "" = single player; LANHost = Other Multiplayer -> TCP/IP -> Host Game; LANJoin = ... -> Join Game
	// and type HostIP. Open Battle.net is never used (it is not in the table).
	LAN    string
	HostIP string // LANJoin only: dotted IPv4 typed into the join dialog
}

// LAN modes of CharSpec.
const (
	LANHost = "host"
	LANJoin = "join"
)

// Errors of the flow. Use errors.Is.
var (
	ErrStuck        = errors.New("stuck on screen")
	ErrCharNotFound = errors.New("character not in the list (first 8 slots)")
	ErrNameTaken    = errors.New("character name already taken")
	ErrWrongClass   = errors.New("class not selected after clicking")
	ErrTimeout      = errors.New("timed out")
	ErrJoinFailed   = errors.New("could not join the LAN game")
)

// Flow drives menus to "in a game with the wanted character" and back out.
type Flow struct {
	Mem   mem.Source
	In    Input
	Sleep func(time.Duration) // default time.Sleep; tests inject a no-op
	Now   func() time.Time    // default time.Now
	Log   func(string)
	// BeforeSave, if set, runs just before LeaveGame clicks Save and Exit (the character is written then). The bot
	// installs cheat.Env.GuardSave here when the cheat package is attached, so inflated max life is never saved.
	// An error is logged and the save goes on.
	BeforeSave func(ctx context.Context, reason string) error

	// ScreenTimeout bounds how long one unchanged screen is tolerated (default 20 s). Loading is bounded
	// by LoadTimeout (default 60 s: the main menu appears ~10 s after launch, entering a game <2 s).
	ScreenTimeout time.Duration
	LoadTimeout   time.Duration
	// ClassSettle is the wait after a class click (default 2.5 s; VERIFIED: ~1 s is often too short and
	// the click is dropped). ClassTries bounds the retries (default 4).
	ClassSettle time.Duration
	ClassTries  int
}

func (f *Flow) logf(format string, a ...any) {
	if f.Log != nil {
		f.Log(fmt.Sprintf(format, a...))
	}
}

func (f *Flow) sleep(d time.Duration) {
	if f.Sleep != nil {
		f.Sleep(d)
		return
	}
	time.Sleep(d)
}

func (f *Flow) now() time.Time {
	if f.Now != nil {
		return f.Now()
	}
	return time.Now()
}

func dur(d, def time.Duration) time.Duration {
	if d == 0 {
		return def
	}
	return d
}

func (f *Flow) clickBtn(id ScreenID, name string) error {
	sc, ok := FindScreen(id)
	if !ok {
		return fmt.Errorf("no screen %s in table", id)
	}
	b, ok := sc.Btn(name)
	if !ok {
		return fmt.Errorf("no button %s on screen %s", name, id)
	}
	x, y := b.R.Center()
	f.logf("click %s/%s at (%d,%d)", id, name, x, y)
	return f.In.Click(x, y)
}

func (f *Flow) typeText(s string) error {
	t, canChar := f.In.(Typer)
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			return fmt.Errorf("character name %q: only letters and digits are typed", s)
		}
		var err error
		if canChar {
			err = t.TypeChar(r)
		} else {
			err = f.In.Key(strings.ToLower(string(r)))
		}
		if err != nil {
			return err
		}
		f.sleep(60 * time.Millisecond)
	}
	return nil
}

// typeRaw types s with WM_CHAR; every rune must be in allowed (the IP box takes digits and dots).
func (f *Flow) typeRaw(s, allowed string) error {
	t, ok := f.In.(Typer)
	if !ok {
		return errors.New("input cannot type text (no TypeChar)")
	}
	for _, r := range s {
		if !strings.ContainsRune(allowed, r) {
			return fmt.Errorf("cannot type %q", r)
		}
		if err := t.TypeChar(r); err != nil {
			return err
		}
		f.sleep(60 * time.Millisecond)
	}
	return nil
}

// labelAt returns the text of the label control with geometry r ("" if absent).
func labelAt(cs []Control, r Rect) string {
	for _, c := range cs {
		if c.Rect == r {
			return strings.TrimSpace(c.Text)
		}
	}
	return ""
}

// popupText joins the text of all labels (the popup message lines).
func popupText(cs []Control) string {
	var t []string
	for _, c := range cs {
		if c.Type == CtlLabel && strings.TrimSpace(c.Text) != "" {
			t = append(t, strings.TrimSpace(c.Text))
		}
	}
	return strings.Join(t, " | ")
}

// PlayerName reads the name of the current character (PlayerUnit -> +0x14 PlayerData -> name at +0;
// VERIFIED LIVE: "Testy", "Botty"). It fails when not in a game.
func PlayerName(s mem.Source) (string, error) {
	pl, err := mem.U32(s, mem.Base+rvaPlayerUnit)
	if err != nil {
		return "", err
	}
	if pl == 0 {
		return "", errors.New("no player")
	}
	pd, err := mem.U32(s, pl+0x14)
	if err != nil || pd == 0 {
		return "", fmt.Errorf("player data: %v", err)
	}
	var b [16]byte
	if err := s.ReadAt(b[:], pd); err != nil {
		return "", err
	}
	n := 0
	for n < len(b) && b[n] != 0 {
		n++
	}
	return string(b[:n]), nil
}

// UI flag array (VERIFIED LIVE, docs/mem.md Q1): DWORD[0x26] at RVA 0x3A27C0, [9] = Esc menu. Indexes
// 0, 0x15, 0x22, 0x23 read 1 at rest in town and are not panels.
const (
	rvaUIArray = 0x3A27C0
	uiEscMenu  = 9
)

// uiState returns whether the Esc menu is open and whether any other panel is open.
func uiState(s mem.Source) (esc, panel bool, err error) {
	for i := 0; i < 0x26; i++ {
		v, e := mem.U32(s, mem.Base+rvaUIArray+uint32(4*i))
		if e != nil {
			return false, false, e
		}
		switch {
		case i == uiEscMenu:
			esc = v != 0
		case i == 0 || i == 0x15 || i == 0x22 || i == 0x23:
		case v != 0:
			panel = true
		}
	}
	// While the Esc menu is open the baseline flags read 0 and nothing else is open (VERIFIED).
	return esc, panel && !esc, nil
}

// Start navigates from wherever the game currently is (nothing but a splash/loading screen, main menu,
// char select, create screen, a popup, or even inside a game with another character) to being in a game
// with ch. Every wait is bounded; surprises return an error naming the screen. It never clicks blindly in
// a loop: a screen that does not change after ScreenTimeout is ErrStuck.
func (f *Flow) Start(ctx context.Context, ch CharSpec) error {
	screenLimit := dur(f.ScreenTimeout, 20*time.Second)
	loadLimit := dur(f.LoadTimeout, 60*time.Second)
	var last ScreenID
	lastChange := f.now()
	loadingSince := f.now()
	created, leaves, ipTyped := false, 0, false
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		ph, err := ReadPhase(f.Mem)
		if err != nil {
			return err
		}
		switch ph {
		case PhaseInGame:
			name, nerr := PlayerName(f.Mem)
			if nerr != nil {
				f.sleep(300 * time.Millisecond) // pointer set before PlayerData is filled
				continue
			}
			if ch.Name == "" || strings.EqualFold(name, ch.Name) {
				f.sleep(300 * time.Millisecond)
				if again, e := PlayerName(f.Mem); e == nil && strings.EqualFold(again, name) {
					return nil
				}
				continue
			}
			if leaves++; leaves > 2 {
				return fmt.Errorf("in game as %q, wanted %q, leaving did not help", name, ch.Name)
			}
			f.logf("in game as %q, leaving to switch to %q", name, ch.Name)
			if err := f.LeaveGame(ctx, 30*time.Second); err != nil {
				return err
			}
			last, lastChange = "", f.now()
			continue
		case PhaseUnknown, PhaseBoth:
			if f.now().Sub(loadingSince) > loadLimit {
				return fmt.Errorf("%w: loading screen for %s", ErrTimeout, loadLimit)
			}
			f.sleep(500 * time.Millisecond)
			continue
		}
		loadingSince = f.now()
		cs, err := ReadControls(f.Mem)
		if err != nil {
			return err
		}
		id := Identify(cs)
		if id != last {
			last, lastChange = id, f.now()
			f.logf("screen: %s", id)
		} else if f.now().Sub(lastChange) > screenLimit {
			return fmt.Errorf("%w %s for %s", ErrStuck, id, screenLimit)
		}
		switch id {
		case ScreenMainMenu:
			if ch.LAN != "" {
				err = f.clickBtn(id, "other-multiplayer")
			} else {
				err = f.clickBtn(id, "single-player")
			}
		case ScreenMultiMenu:
			if ch.LAN == "" {
				err = f.clickBtn(id, "cancel")
			} else {
				err = f.clickBtn(id, "tcp-ip")
			}
		case ScreenTCPOptions:
			switch ch.LAN {
			case LANHost:
				err = f.clickBtn(id, "host")
			case LANJoin:
				if ipTyped {
					// back here after the dialog: the join did not start. Failure popups are handled below.
					return fmt.Errorf("%w: back on the TCP/IP screen after entering %s", ErrJoinFailed, ch.HostIP)
				}
				err = f.clickBtn(id, "join")
			default:
				err = f.clickBtn(id, "cancel")
			}
		case ScreenTCPJoinIP:
			if ch.LAN != LANJoin {
				err = f.clickBtn(id, "cancel")
				break
			}
			if !ipTyped {
				if err = f.clickBtn(id, "ip-box"); err != nil {
					break
				}
				// The box keeps the previous address and appends to it (VERIFIED LIVE: "127.0.0.1127.0.0.1"
				// gave "Cannot Connect"; the buffer holds more than the 15 visible characters), so empty it first (32 backspaces; 100 cost ~90 s of helper round trips).
				for i := 0; i < 32 && err == nil; i++ {
					err = f.In.Key("backspace")
				}
				if err != nil {
					break
				}
				f.sleep(300 * time.Millisecond)
				if err = f.typeRaw(ch.HostIP, ".0123456789"); err != nil {
					break
				}
				f.sleep(300 * time.Millisecond)
				if err = f.clickBtn(id, "ok"); err == nil {
					ipTyped = true
				}
			}
		case ScreenCharSelect:
			if ch.ForceNew {
				err = f.clickBtn(id, "create")
				break
			}
			var found bool
			if found, err = f.selectChar(cs, ch.Name); err != nil {
				break
			}
			switch {
			case found:
				err = f.clickBtn(id, "ok")
			case ch.Create:
				err = f.clickBtn(id, "create")
			default:
				return fmt.Errorf("%w: %q", ErrCharNotFound, ch.Name)
			}
		case ScreenCharCreate:
			if !ch.Create && !ch.ForceNew {
				// Arrived here by mistake: go back rather than create anything.
				err = f.clickBtn(id, "exit")
				break
			}
			if created {
				break // OK was clicked; the next screen change is awaited by the stuck timer
			}
			if err = f.fillCreate(ch); err == nil {
				created = true
			}
		case ScreenDifficulty:
			d := strings.ToLower(ch.Difficulty)
			if d == "" {
				d = "normal"
			}
			err = f.clickBtn(id, d)
		case ScreenPopupYesNo:
			if ch.Hardcore {
				err = f.clickBtn(id, "popup-yes")
			} else {
				_ = f.clickBtn(id, "popup-no")
				return errors.New("unexpected yes/no popup: " + popupText(cs))
			}
		case ScreenPopupOK:
			txt := popupText(cs)
			_ = f.clickBtn(id, "popup-ok")
			f.sleep(500 * time.Millisecond)
			if ch.LAN == LANJoin {
				return fmt.Errorf("%w: %s", ErrJoinFailed, txt)
			}
			if strings.Contains(strings.ToLower(txt), "taken") {
				return fmt.Errorf("%w: %s", ErrNameTaken, txt)
			}
			return fmt.Errorf("menu popup: %s", txt)
		default:
			// splash or an unknown transition: wait; the stuck timer reports it.
		}
		if err != nil {
			return err
		}
		f.sleep(800 * time.Millisecond)
	}
}

// HostGame leaves to the menus as needed and hosts a TCP/IP game with the character: Other Multiplayer ->
// TCP/IP Game -> Host Game -> character -> in game (VERIFIED LIVE: no name, password or difficulty dialog
// for a Normal-only character). The game exists while this client stays in it.
func (f *Flow) HostGame(ctx context.Context, ch CharSpec) error {
	ch.LAN = LANHost
	return f.Start(ctx, ch)
}

// JoinGame joins the TCP/IP game of hostIP (127.0.0.1 on one machine) with the character.
func (f *Flow) JoinGame(ctx context.Context, ch CharSpec, hostIP string) error {
	ch.LAN, ch.HostIP = LANJoin, hostIP
	return f.Start(ctx, ch)
}

// EnterGame is the older name of Start.
func (f *Flow) EnterGame(ctx context.Context, ch CharSpec) error { return f.Start(ctx, ch) }

// CreateCharacter creates a new softcore expansion character of the class and enters its game. It fails
// with ErrNameTaken when the name exists (VERIFIED popup "That character name is already taken").
func (f *Flow) CreateCharacter(ctx context.Context, name, class string) error {
	return f.Start(ctx, CharSpec{Name: name, Class: class, Create: true, ForceNew: true, Difficulty: "normal"})
}

// selectChar makes the named character the selected one on char select (the header label shows the
// selected name) by clicking slots; it reports whether it is selected. At most CharSlots clicks.
func (f *Flow) selectChar(cs []Control, name string) (bool, error) {
	if strings.EqualFold(labelAt(cs, RectCharHeader), name) {
		return true, nil
	}
	for i := 0; i < CharSlots; i++ {
		x, y := SlotClick(i)
		if err := f.In.Click(x, y); err != nil {
			return false, err
		}
		f.sleep(400 * time.Millisecond)
		now, err := ReadControls(f.Mem)
		if err != nil {
			return false, err
		}
		if strings.EqualFold(labelAt(now, RectCharHeader), name) {
			f.logf("selected %q in slot %d", name, i)
			return true, nil
		}
	}
	return false, nil
}

// fillCreate picks the class (verified through the class header label), types the name, sets the check
// boxes and clicks OK.
func (f *Flow) fillCreate(ch CharSpec) error {
	pos, ok := ClassClick[strings.ToLower(ch.Class)]
	if !ok {
		return fmt.Errorf("unknown class %q", ch.Class)
	}
	tries := f.ClassTries
	if tries == 0 {
		tries = 4
	}
	picked := false
	for i := 0; i < tries && !picked; i++ {
		if err := f.In.Click(pos[0], pos[1]); err != nil {
			return err
		}
		f.sleep(dur(f.ClassSettle, 2500*time.Millisecond))
		cs, err := ReadControls(f.Mem)
		if err != nil {
			return err
		}
		picked = strings.EqualFold(labelAt(cs, RectClassHeader), ch.Class)
	}
	if !picked {
		return fmt.Errorf("%w: %s", ErrWrongClass, ch.Class)
	}
	if err := f.clickBtn(ScreenCharCreate, "name-box"); err != nil {
		return err
	}
	f.sleep(300 * time.Millisecond)
	if err := f.typeText(ch.Name); err != nil {
		return err
	}
	if ch.Classic {
		if err := f.clickBtn(ScreenCharCreate, "expansion-checkbox"); err != nil {
			return err
		}
	}
	if ch.Hardcore {
		if err := f.clickBtn(ScreenCharCreate, "hardcore-checkbox"); err != nil {
			return err
		}
	}
	f.sleep(300 * time.Millisecond)
	return f.clickBtn(ScreenCharCreate, "ok")
}

// LeaveGame saves and exits the current game to the MAIN MENU (VERIFIED LIVE: Game.exe keeps running and
// lands on the main menu, not char select). It opens the Esc menu only when no panel is open (an Esc with
// nothing open OPENS it; with a panel open it closes the panel), clicks Save and Exit once, and waits
// for PhaseMenu and then for the main menu (or, after a LAN game, the TCP/IP Options screen) to be identifiable. Already out of the game = nil.
func (f *Flow) LeaveGame(ctx context.Context, timeout time.Duration) error {
	deadline := f.now().Add(timeout)
	if ph, err := ReadPhase(f.Mem); err == nil && ph == PhaseMenu {
		return nil
	}
	clicked := false
	for f.now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return err
		}
		ph, err := ReadPhase(f.Mem)
		if err != nil {
			return err
		}
		if ph == PhaseMenu {
			break
		}
		if !clicked {
			esc, panel, err := uiState(f.Mem)
			if err != nil {
				return err
			}
			switch {
			case esc:
				if f.BeforeSave != nil {
					if err := f.BeforeSave(ctx, "save and exit"); err != nil {
						f.logf("before-save guard failed: %v", err)
					}
				}
				x, y := BtnSaveAndExit.R.Center()
				f.logf("click save-and-exit at (%d,%d)", x, y)
				if err := f.In.Click(x, y); err != nil {
					return err
				}
				clicked = true
			default:
				_ = panel // an open panel is closed by the same Esc; the next pass then sees no panel and opens the menu
				if err := f.In.Key(EscKey); err != nil {
					return err
				}
			}
			f.sleep(600 * time.Millisecond)
			continue
		}
		f.sleep(300 * time.Millisecond)
	}
	if ph, _ := ReadPhase(f.Mem); ph != PhaseMenu {
		return fmt.Errorf("%w: still in game %s after Save and Exit", ErrTimeout, timeout)
	}
	for f.now().Before(deadline.Add(10 * time.Second)) {
		cs, err := ReadControls(f.Mem)
		// A LAN game returns to the TCP/IP Options screen, not the main menu (VERIFIED LIVE).
		if id := Identify(cs); err == nil && (id == ScreenMainMenu || id == ScreenTCPOptions || id == ScreenMultiMenu) {
			return nil
		}
		f.sleep(300 * time.Millisecond)
	}
	return fmt.Errorf("%w: main menu not reached after leaving", ErrTimeout)
}
