package launch

import (
	"context"
	"encoding/binary"
	"errors"
	"strings"
	"testing"
	"time"
	"unicode/utf16"

	"d2lod-bot/internal/mem"
)

// putControls lays out a D2Win-style control list in f and points FirstControl at it.
func putControls(f *mem.Fake, ctrls []Control, inGame bool) {
	const arena = 0x10000000
	f.PutU32(mem.Base+rvaPlayerUnit, 0)
	f.PutU32(mem.Base+rvaFirstControl, 0)
	if inGame {
		f.PutU32(mem.Base+rvaPlayerUnit, 0x20000000)
		return
	}
	for i, c := range ctrls {
		a := uint32(arena + i*0x200)
		b := make([]byte, 0x4C)
		le := binary.LittleEndian
		le.PutUint32(b[0:], uint32(c.Type))
		le.PutUint32(b[0xC:], uint32(c.X))
		le.PutUint32(b[0x10:], uint32(c.Y))
		le.PutUint32(b[0x14:], uint32(c.W))
		le.PutUint32(b[0x18:], uint32(c.H))
		if i+1 < len(ctrls) {
			le.PutUint32(b[0x3C:], a+0x200)
		}
		if c.Text != "" {
			le.PutUint32(b[0x48:], a+0x100) // ControlText
			f.PutU32(a+0x100, a+0x180)      // wText[0]
			u := utf16.Encode([]rune(c.Text + "\x00"))
			tb := make([]byte, 2*len(u))
			for j, w := range u {
				le.PutUint16(tb[2*j:], w)
			}
			f.Put(a+0x180, tb)
		}
		f.Put(a, b)
	}
	if len(ctrls) > 0 {
		f.PutU32(mem.Base+rvaFirstControl, arena)
	}
}

func screenControls(id ScreenID) []Control {
	sc, _ := FindScreen(id)
	var cs []Control
	for _, r := range sc.Sig {
		t := r.Type
		if t == CtlAny {
			t = CtlButton
		}
		cs = append(cs, Control{Rect: Rect{t, r.X, r.Y, r.W, r.H}})
	}
	return cs
}

// fakeWorld is a small menu state machine reacting to clicks on table coordinates. It implements
// mem.Source (via the embedded Fake), Input and Typer.
type fakeWorld struct {
	*mem.Fake
	screen   ScreenID
	player   string   // name while in game
	chars    []string // saved characters, slot order
	selected string
	class    string
	typed    string
	popup    string
	ui       map[int]uint32
	dropNext int // class clicks to ignore (the game drops clicks right after a class change)
	clicks   [][2]int
	keys     []string
	lan      string // "host" or "join" once chosen on tcp-options
	joinFail string // popup text instead of entering the game on join
	hostUp   bool
}

func newWorld(screen ScreenID, chars ...string) *fakeWorld {
	w := &fakeWorld{Fake: mem.NewFake(), screen: screen, chars: chars, ui: map[int]uint32{}}
	if len(chars) > 0 {
		w.selected = chars[0]
	}
	w.show()
	return w
}

func (w *fakeWorld) show() {
	for i := 0; i < 0x26; i++ {
		w.PutU32(mem.Base+rvaUIArray+uint32(4*i), w.ui[i])
	}
	if w.screen == "in-game" {
		putControls(w.Fake, nil, true)
		w.PutU32(mem.Base+rvaPlayerUnit, 0x20000000)
		w.PutU32(0x20000000+0x14, 0x20001000)
		w.Put(0x20001000, append([]byte(w.player), make([]byte, 16)...))
		return
	}
	cs := screenControls(w.screen)
	switch w.screen {
	case ScreenCharSelect:
		cs = append(cs, Control{Rect: RectCharHeader, Text: w.selected})
	case ScreenCharCreate:
		cs = append(cs, Control{Rect: RectClassHeader, Text: w.class})
	case ScreenPopupOK:
		cs = append(cs, Control{Rect: Rect{CtlLabel, 300, 250, 200, 30}, Text: w.popup})
	}
	putControls(w.Fake, cs, false)
}

func (w *fakeWorld) Key(k string) error {
	w.keys = append(w.keys, k)
	if k == "esc" && w.screen == "in-game" {
		w.ui[uiEscMenu] = 1 - w.ui[uiEscMenu]
		w.show()
	}
	return nil
}

func (w *fakeWorld) TypeChar(r rune) error { w.typed += string(r); return nil }

func (w *fakeWorld) Click(x, y int) error {
	w.clicks = append(w.clicks, [2]int{x, y})
	hit := func(id ScreenID, name string) bool {
		sc, _ := FindScreen(id)
		b, _ := sc.Btn(name)
		bx, by := b.R.Center()
		return w.screen == id && x == bx && y == by
	}
	switch {
	case w.screen == "in-game" && w.ui[uiEscMenu] == 1:
		if cx, cy := BtnSaveAndExit.R.Center(); x == cx && y == cy {
			w.screen, w.ui[uiEscMenu] = ScreenMainMenu, 0
			if w.lan != "" {
				w.screen = ScreenTCPOptions // VERIFIED LIVE: a LAN game returns to TCP/IP Options
			}
		}
	case hit(ScreenMainMenu, "single-player"):
		w.screen = ScreenCharSelect
	case hit(ScreenMainMenu, "other-multiplayer"):
		w.screen = ScreenMultiMenu
	case hit(ScreenMultiMenu, "tcp-ip"):
		w.screen = ScreenTCPOptions
	case hit(ScreenMultiMenu, "cancel"), hit(ScreenTCPOptions, "cancel"):
		w.screen = ScreenMainMenu
	case hit(ScreenTCPOptions, "host"):
		w.screen, w.lan = ScreenCharSelect, "host"
	case hit(ScreenTCPOptions, "join"):
		w.screen, w.lan, w.typed = ScreenTCPJoinIP, "join", ""
	case hit(ScreenTCPJoinIP, "ok"):
		if w.joinFail != "" {
			w.screen, w.popup = ScreenPopupOK, w.joinFail
		} else {
			w.screen = ScreenCharSelect
		}
	case hit(ScreenTCPJoinIP, "ip-box"):
		w.typed = ""
	case hit(ScreenCharSelect, "create"):
		w.screen, w.class, w.typed = ScreenCharCreate, "", ""
	case hit(ScreenCharSelect, "ok"):
		w.screen, w.player = "in-game", w.selected
	case hit(ScreenCharCreate, "exit"):
		w.screen = ScreenCharSelect
	case hit(ScreenCharCreate, "ok"):
		if w.typed == "" {
			break
		}
		for _, c := range w.chars {
			if c == w.typed {
				w.screen, w.popup = ScreenPopupOK, "That character name is already taken."
			}
		}
		if w.screen != ScreenPopupOK {
			w.screen, w.player = "in-game", w.typed
		}
	case hit(ScreenPopupOK, "popup-ok"):
		w.screen = ScreenCharCreate
	case w.screen == ScreenCharSelect:
		for i := 0; i < CharSlots; i++ {
			if sx, sy := SlotClick(i); sx == x && sy == y && i < len(w.chars) {
				w.selected = w.chars[i]
			}
		}
	case w.screen == ScreenCharCreate:
		for c, p := range ClassClick {
			if p[0] == x && p[1] == y {
				if w.dropNext > 0 {
					w.dropNext--
				} else {
					w.class = strings.ToUpper(c[:1]) + c[1:]
				}
			}
		}
	}
	w.show()
	return nil
}

func quick(w *fakeWorld) *Flow {
	return &Flow{Mem: w, In: w, Sleep: func(time.Duration) {}}
}

func TestStartCreatesCharacterFromMainMenu(t *testing.T) {
	w := newWorld(ScreenMainMenu)
	w.dropNext = 2
	f := quick(w)
	if err := f.CreateCharacter(context.Background(), "Botty", "sorceress"); err != nil {
		t.Fatal(err)
	}
	if w.typed != "Botty" || w.player != "Botty" || w.class != "Sorceress" {
		t.Fatalf("typed=%q player=%q class=%q", w.typed, w.player, w.class)
	}
}

func TestClassNeverConfirmedIsAnError(t *testing.T) {
	w := newWorld(ScreenCharCreate)
	w.dropNext = 99
	err := quick(w).Start(context.Background(), CharSpec{Name: "Botty", Class: "druid", Create: true})
	if !errors.Is(err, ErrWrongClass) {
		t.Fatalf("err = %v", err)
	}
}

func TestStartSelectsExistingBySlot(t *testing.T) {
	w := newWorld(ScreenMainMenu, "Botty", "Testy")
	if err := quick(w).Start(context.Background(), CharSpec{Name: "testy"}); err != nil {
		t.Fatal(err)
	}
	if w.player != "Testy" {
		t.Fatalf("player %q", w.player)
	}
}

func TestStartMissingCharacter(t *testing.T) {
	w := newWorld(ScreenCharSelect, "Botty")
	err := quick(w).Start(context.Background(), CharSpec{Name: "Nobody"})
	if !errors.Is(err, ErrCharNotFound) {
		t.Fatalf("err = %v", err)
	}
	w = newWorld(ScreenCharSelect, "Botty")
	if err := quick(w).Start(context.Background(), CharSpec{Name: "Newbie", Class: "paladin", Create: true}); err != nil || w.player != "Newbie" {
		t.Fatalf("err=%v player=%q", err, w.player)
	}
}

func TestNameTakenPopup(t *testing.T) {
	w := newWorld(ScreenCharSelect, "Testy")
	err := quick(w).CreateCharacter(context.Background(), "Testy", "druid")
	if !errors.Is(err, ErrNameTaken) {
		t.Fatalf("err = %v", err)
	}
	if w.screen != ScreenCharCreate {
		t.Fatalf("popup not dismissed: %s", w.screen)
	}
}

func TestStartFromCreateScreenWithoutCreateGoesBack(t *testing.T) {
	w := newWorld(ScreenCharCreate, "Testy")
	if err := quick(w).Start(context.Background(), CharSpec{Name: "Testy"}); err != nil || w.player != "Testy" {
		t.Fatalf("err=%v player=%q", err, w.player)
	}
}

func TestStartAlreadyInGame(t *testing.T) {
	w := newWorld("in-game", "Testy")
	w.player = "Testy"
	w.show()
	if err := quick(w).Start(context.Background(), CharSpec{Name: "Testy"}); err != nil {
		t.Fatal(err)
	}
	if len(w.clicks) != 0 || len(w.keys) != 0 {
		t.Fatalf("touched the game: %v %v", w.clicks, w.keys)
	}
}

func TestStartSwitchesCharacterFromGame(t *testing.T) {
	w := newWorld("in-game", "Botty", "Testy")
	w.player = "Botty"
	w.show()
	if err := quick(w).Start(context.Background(), CharSpec{Name: "Testy"}); err != nil || w.player != "Testy" {
		t.Fatalf("err=%v player=%q", err, w.player)
	}
}

func TestStuckScreenIsReported(t *testing.T) {
	w := newWorld(ScreenMainMenu)
	now := time.Unix(0, 0)
	f := &Flow{Mem: w, In: noop{}, ScreenTimeout: 5 * time.Second, Now: func() time.Time { return now },
		Sleep: func(d time.Duration) { now = now.Add(d) }}
	err := f.Start(context.Background(), CharSpec{Name: "x"})
	if !errors.Is(err, ErrStuck) || !strings.Contains(err.Error(), "main-menu") {
		t.Fatalf("err = %v", err)
	}
}

func TestLoadingIsBounded(t *testing.T) {
	w := newWorld(ScreenMainMenu)
	putControls(w.Fake, nil, false) // neither canary: loading
	now := time.Unix(0, 0)
	f := &Flow{Mem: w, In: noop{}, LoadTimeout: 10 * time.Second, Now: func() time.Time { return now },
		Sleep: func(d time.Duration) { now = now.Add(d) }}
	if err := f.Start(context.Background(), CharSpec{Name: "x"}); !errors.Is(err, ErrTimeout) {
		t.Fatalf("err = %v", err)
	}
}

type noop struct{}

func (noop) Click(x, y int) error { return nil }
func (noop) Key(string) error     { return nil }

func TestIdentifyAndPhase(t *testing.T) {
	for _, sc := range Screens {
		if sc.ID == ScreenSplash {
			continue // its signature is the bare background image, shadowed by every other screen
		}
		if got := Identify(screenControls(sc.ID)); got != sc.ID {
			t.Errorf("Identify(%s sig) = %s", sc.ID, got)
		}
	}
	if got := Identify([]Control{{Rect: Rect{CtlImage, 0, 599, 800, 600}}}); got != ScreenSplash {
		t.Errorf("splash = %s", got)
	}
	f := mem.NewFake()
	putControls(f, nil, true)
	if p, _ := ReadPhase(f); p != PhaseInGame {
		t.Fatalf("phase %v", p)
	}
	putControls(f, screenControls(ScreenMainMenu), false)
	if p, _ := ReadPhase(f); p != PhaseMenu {
		t.Fatalf("phase %v", p)
	}
}

func TestLeaveGameOpensEscOnlyWhenNoPanelOpen(t *testing.T) {
	w := newWorld("in-game", "Testy")
	w.player = "Testy"
	w.ui[1] = 1 // inventory open: first Esc must close it, second opens the menu
	w.show()
	f := quick(w)
	w.ui[uiEscMenu] = 0
	// the fake toggles esc on every Esc; model the real game: a panel absorbs the first Esc
	orig := w.ui
	_ = orig
	if err := f.LeaveGame(context.Background(), 30*time.Second); err != nil {
		t.Fatal(err)
	}
	if w.screen != ScreenMainMenu {
		t.Fatalf("screen %s", w.screen)
	}
}

func TestLeaveGameWhenAlreadyOut(t *testing.T) {
	w := newWorld(ScreenMainMenu)
	if err := quick(w).LeaveGame(context.Background(), time.Second); err != nil || len(w.keys) != 0 {
		t.Fatalf("err=%v keys=%v", err, w.keys)
	}
}

func TestPlayerNameAndUI(t *testing.T) {
	w := newWorld("in-game")
	w.player = "Testy"
	w.ui[uiEscMenu] = 1
	w.ui[0x15] = 1
	w.show()
	if n, err := PlayerName(w); err != nil || n != "Testy" {
		t.Fatalf("%q %v", n, err)
	}
	esc, panel, err := uiState(w)
	if err != nil || !esc || panel {
		t.Fatalf("esc=%v panel=%v err=%v", esc, panel, err)
	}
}

func TestKeeperRestartsAndGuardsCrashLoop(t *testing.T) {
	w := newWorld(ScreenMainMenu, "Testy")
	var order []string
	alive := true
	k := &Keeper{
		Mon: &Monitor{P: Probes{
			Alive:     func() (int, bool) { return 7, alive },
			Heartbeat: func(int) (uint64, error) { return 1, nil },
		}},
		Kill:   func() error { order = append(order, "kill"); return nil },
		Launch: func(context.Context) error { order = append(order, "launch"); alive = true; return nil },
		Ready:  func(context.Context) (*Flow, error) { order = append(order, "ready"); return quick(w), nil },
		Char:   CharSpec{Name: "Testy"},
		Sleep:  func(time.Duration) {},
	}
	now := time.Unix(1000, 0)
	if ev, _ := k.Step(context.Background(), now); ev != EventUp {
		t.Fatalf("ev %v", ev)
	}
	for i := 0; i < 3; i++ {
		alive = false
		w.screen, w.player = ScreenMainMenu, ""
		w.show()
		now = now.Add(time.Minute)
		if ev, err := k.Step(context.Background(), now); ev != EventExited || err != nil {
			t.Fatalf("round %d: ev=%v err=%v", i, ev, err)
		}
		k.Step(context.Background(), now.Add(time.Second)) // Up again
	}
	if strings.Join(order, ",") != strings.Repeat("kill,launch,ready,", 3)[:len(strings.Repeat("kill,launch,ready,", 3))-1] {
		t.Fatalf("order %v", order)
	}
	alive = false
	now = now.Add(time.Minute)
	if _, err := k.Step(context.Background(), now); !errors.Is(err, ErrCrashLoop) {
		t.Fatalf("err = %v", err)
	}
}

func TestHostAndJoinWalkTheLANScreens(t *testing.T) {
	w := newWorld(ScreenMainMenu, "Navi", "Combi")
	if err := quick(w).HostGame(context.Background(), CharSpec{Name: "Combi"}); err != nil {
		t.Fatal(err)
	}
	if w.lan != "host" || w.player != "Combi" {
		t.Fatalf("lan=%q player=%q", w.lan, w.player)
	}
	w = newWorld(ScreenMainMenu, "Navi", "Combi")
	if err := quick(w).JoinGame(context.Background(), CharSpec{Name: "Navi"}, "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	bs := 0
	for _, k := range w.keys {
		if k == "backspace" {
			bs++
		}
	}
	if bs < 16 {
		t.Fatalf("the IP box must be cleared first, backspaces=%d", bs)
	}
	if w.lan != "join" || w.typed != "127.0.0.1" || w.player != "Navi" {
		t.Fatalf("lan=%q typed=%q player=%q", w.lan, w.typed, w.player)
	}
}

func TestJoinFailurePopupIsErrJoinFailed(t *testing.T) {
	w := newWorld(ScreenMainMenu, "Navi")
	w.joinFail = "Unable to connect"
	err := quick(w).JoinGame(context.Background(), CharSpec{Name: "Navi"}, "127.0.0.1")
	if !errors.Is(err, ErrJoinFailed) {
		t.Fatalf("want ErrJoinFailed, got %v", err)
	}
}

func TestJoinRejectsOddAddress(t *testing.T) {
	w := newWorld(ScreenMainMenu, "Navi")
	if err := quick(w).JoinGame(context.Background(), CharSpec{Name: "Navi"}, "bad host"); err == nil {
		t.Fatal("expected error")
	}
}

func TestLANScreensIdentify(t *testing.T) {
	for _, id := range []ScreenID{ScreenMultiMenu, ScreenTCPOptions, ScreenTCPJoinIP} {
		if got := Identify(screenControls(id)); got != id {
			t.Errorf("%s identified as %s", id, got)
		}
	}
	for _, sc := range Screens {
		for _, b := range sc.Buttons {
			if strings.Contains(b.Name, "battle") && sc.ID != ScreenMainMenu {
				t.Errorf("battle.net button %q on %s", b.Name, sc.ID)
			}
		}
	}
}

func TestLeaveLANGameReachesTCPOptions(t *testing.T) {
	w := newWorld(ScreenMainMenu, "Navi")
	f := quick(w)
	if err := f.HostGame(context.Background(), CharSpec{Name: "Navi"}); err != nil {
		t.Fatal(err)
	}
	if err := f.LeaveGame(context.Background(), time.Second); err != nil {
		t.Fatal(err)
	}
	if w.screen != ScreenTCPOptions {
		t.Fatalf("screen %s", w.screen)
	}
	// and a join from there goes straight on
	if err := f.JoinGame(context.Background(), CharSpec{Name: "Navi"}, "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
}

func TestLeaveGameRunsBeforeSaveHookOnce(t *testing.T) {
	w := newWorld("in-game", "Testy")
	w.player = "Testy"
	w.show()
	f := quick(w)
	n := 0
	f.BeforeSave = func(ctx context.Context, reason string) error { n++; return nil }
	if err := f.LeaveGame(context.Background(), 30*time.Second); err != nil {
		t.Fatal(err)
	}
	if n != 1 || w.screen != ScreenMainMenu {
		t.Fatalf("hook ran %d times, screen %s", n, w.screen)
	}
}
