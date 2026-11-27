// Package fake is an in-process d2d control channel server for tests: same line protocol, answers taken
// from fixtures. It models only what tests need (see Fixtures); reply layouts follow docs/d2d-spike.md.
package fake

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Fixtures are raw payload lines (tab separated like the real channel) per verb.
type Fixtures struct {
	State         string // the one "state" line
	NPCs          []string
	Monsters      []string
	Ground        []string
	Menu          []string
	Items         []string
	Corpses       []string
	Players       []string
	Warps         []string
	Presets       []string
	Objects       []string
	Portals       []string
	Grid          []string
	Level         string // rest of "debug level"
	LevelMap      string // the JSON line of "debug levelmap" (empty: the verb is missing, answers err)
	LevelMonsters string // the JSON line of "debug levelmonsters" (empty: the verb is missing, answers err)
	WarpTriggers  string // the JSON line of "debug warptriggers" (empty: the verb is missing, answers err)
	// Minions, SkillState and CombatLog are the JSON lines of the combat read verbs (dlv-reads; empty: the verb answers
	// "err read combat off", like a fork started without --read-combat).
	Minions, SkillState, CombatLog string
	Quest                          string // rest of "debug quest <q> show"
	Attack                         string
	Passives                       string
	// Caps is the fork's "caps" list (comma separated); empty: the verb is unknown, like stock d2d.
	Caps string
	// Realtime starts the fork free-running (clock says lockstep=0); "lockstep on|off" then switches it, "lockstep" is
	// an unknown verb when NoLockstepVerb is set (a build without the runtime switch).
	Realtime, NoLockstepVerb bool
	// ItemReply is the first payload line of "item ..." (e.g. "equip id=5 slot=4 displaced=-1 at=-1,-1" or "noop slot=4 at=-1,-1"); ItemErr is an "err" text instead.
	ItemReply, ItemErr string
	// World answers the fork's second set of input verbs (buy, sell, waypoint, portal, pickup) by their first word: a text
	// starting "err " is an error reply, anything else the JSON line before "ok". A verb without an entry is unknown, like
	// an older build.
	World map[string]string
	// Hold answers "cmd skillhold status" in order, one entry per call, the last one repeating (e.g. "held=3 casts=1
	// frames=15 state=running"); nil: the hold verb is unknown, like a build without it. HoldErr refuses the start.
	Hold    []string
	HoldErr string
}

// Options model misbehaviour.
type Options struct {
	Delay     time.Duration            // added before every reply
	SlowVerbs map[string]time.Duration // extra delay per first token (or "debug <sub>")
	DropAfter int                      // close the connection without replying to the Nth command (0 = off)
	// NoQuirk turns off the "cmd move/interact ignored until a click after a warp" quirk (V on real d2d).
	NoQuirk bool
}

// Entry is one received command.
type Entry struct {
	Line    string
	Ignored bool // swallowed by the kick quirk (still answered "ok", like the real thing)
}

// Server is the fake. Create with New, stop with Close.
type Server struct {
	Fix  Fixtures
	Opts Options

	path string
	ln   net.Listener

	mu      sync.Mutex
	log     []Entry
	cur     net.Conn
	frame   int
	tick    int  // lockstep virtual ticks (step verb)
	lsOn    bool // the lockstep switch (Fixtures.Realtime starts it off)
	needs   bool // quirk state: a click is required
	quit    bool
	cmds    int
	skills  []UseSkill
	wg      sync.WaitGroup
	closing bool
}

// New listens on path (a unix socket; the directory must exist and should be private).
func New(path string, fix Fixtures, opts Options) (*Server, error) {
	_ = os.Remove(path)
	ln, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	s := &Server{Fix: fix, Opts: opts, path: path, ln: ln, frame: 100, lsOn: !fix.Realtime}
	s.wg.Add(1)
	go s.accept()
	return s, nil
}

// Path is the socket path.
func (s *Server) Path() string { return s.path }

// Close stops the server and drops the client.
func (s *Server) Close() {
	s.mu.Lock()
	s.closing = true
	if s.cur != nil {
		s.cur.Close()
	}
	s.mu.Unlock()
	s.ln.Close()
	s.wg.Wait()
	os.Remove(s.path)
}

// DropClient closes the current client's connection (a crashed or restarted d2d).
func (s *Server) DropClient() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cur != nil {
		s.cur.Close()
	}
}

// Log returns the received commands.
func (s *Server) Log() []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Entry(nil), s.log...)
}

// UseSkill is a "cmd skill" line as d2d's devctl reads it: every field after the verb is a number read with C atoi or
// stof, so a word reads as 0 (V source read: apps/d2d/devctl/verbs.hpp, the same parser on stock upstream and the fork).
type UseSkill struct {
	Skill int
	X, Y  float64
	Unit  int  // -1 when absent
	Left  bool // field 6 read as a number, nonzero
	Item  int  // -1 when absent
}

// ParseSkill reads the fields of "cmd skill <id> <x> <y> [unit] [hand] [item]" as d2d does; ok is false with fewer
// than five fields (d2d then answers no cast).
func ParseSkill(f []string) (u UseSkill, ok bool) {
	if len(f) < 5 || f[0] != "cmd" || f[1] != "skill" {
		return u, false
	}
	arg := func(i, fallback int) int {
		if i < len(f) {
			return atoi(f[i])
		}
		return fallback
	}
	x, _ := strconv.ParseFloat(f[3], 32)
	y, _ := strconv.ParseFloat(f[4], 32)
	return UseSkill{Skill: arg(2, 0), X: x, Y: y, Unit: arg(5, -1), Left: arg(6, 0) != 0, Item: arg(7, -1)}, true
}

// atoi is C atoi: an optional sign and the leading digits, 0 when there are none.
func atoi(s string) int {
	i, neg, n := 0, false, 0
	if i < len(s) && (s[i] == '-' || s[i] == '+') {
		neg = s[i] == '-'
		i++
	}
	for ; i < len(s) && s[i] >= '0' && s[i] <= '9'; i++ {
		n = n*10 + int(s[i]-'0')
	}
	if neg {
		return -n
	}
	return n
}

// Skills returns the "cmd skill" casts received, parsed as d2d parses them.
func (s *Server) Skills() []UseSkill {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]UseSkill(nil), s.skills...)
}

// Lines returns the received command lines only.
func (s *Server) Lines() []string {
	var out []string
	for _, e := range s.Log() {
		out = append(out, e.Line)
	}
	return out
}

// Quit reports whether "quit" was received.
func (s *Server) Quit() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.quit
}

func (s *Server) accept() {
	defer s.wg.Done()
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		s.mu.Lock()
		busy := s.cur != nil
		if !busy {
			s.cur = conn
		}
		s.mu.Unlock()
		if busy {
			conn.Write([]byte("err busy\n"))
			conn.Close()
			continue
		}
		s.wg.Add(1)
		go s.serve(conn)
	}
}

func (s *Server) serve(conn net.Conn) {
	defer s.wg.Done()
	defer func() {
		conn.Close()
		s.mu.Lock()
		s.cur = nil
		s.mu.Unlock()
	}()
	rd := bufio.NewReaderSize(conn, 64<<10+2)
	for {
		b, err := rd.ReadSlice('\n')
		if errors.Is(err, bufio.ErrBufferFull) {
			conn.Write([]byte("err line too long\n"))
			return
		}
		if err != nil {
			return
		}
		line := strings.TrimRight(string(b), "\r\n")
		s.mu.Lock()
		s.cmds++
		drop := s.Opts.DropAfter > 0 && s.cmds == s.Opts.DropAfter
		s.mu.Unlock()
		if drop {
			return
		}
		reply, ignored, quit := s.handle(line)
		s.mu.Lock()
		s.log = append(s.log, Entry{Line: line, Ignored: ignored})
		s.mu.Unlock()
		if d := s.delayFor(line); d > 0 {
			time.Sleep(d)
		}
		if _, err := conn.Write([]byte(reply)); err != nil {
			return
		}
		if quit {
			return
		}
	}
}

func (s *Server) delayFor(line string) time.Duration {
	d := s.Opts.Delay
	f := strings.Fields(line)
	if len(f) > 0 {
		d += s.Opts.SlowVerbs[f[0]]
	}
	if len(f) > 1 && f[0] == "debug" {
		d += s.Opts.SlowVerbs["debug "+f[1]]
	}
	return d
}

func payload(lines []string) string {
	var sb strings.Builder
	for _, l := range lines {
		sb.WriteString(l + "\n")
	}
	return sb.String()
}

func (s *Server) handle(line string) (reply string, ignored, quit bool) {
	f := strings.Fields(line)
	if len(f) == 0 {
		return "err empty command\n", false, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	switch f[0] {
	case "ping":
		return "ok\n", false, false
	case "step":
		n := 1
		if len(f) > 1 {
			fmt.Sscan(f[1], &n)
		}
		s.tick += n
		return fmt.Sprintf("frame=%d game_ms=%d tick=%d\nok\n", s.frame+s.tick, s.tick*40, s.tick), false, false
	case "lockstep":
		if s.Fix.NoLockstepVerb {
			return "err unknown verb 'lockstep'\n", false, false
		}
		if len(f) != 2 || (f[1] != "on" && f[1] != "off") {
			return "err usage: lockstep on|off\n", false, false
		}
		s.lsOn = f[1] == "on"
		return "ok\n", false, false
	case "clock":
		ls := 0
		if s.lsOn {
			ls = 1
		}
		return fmt.Sprintf("lockstep=%d scale=1 frame=%d stepped=%d game_ms=%d tick=%d pending=0 render_steps=0\nok\n", ls, s.frame+s.tick, s.tick, s.tick*40, s.tick), false, false
	case "caps":
		if s.Fix.Caps == "" {
			return "err unknown verb caps\n", false, false
		}
		return "caps=" + s.Fix.Caps + "\nok\n", false, false
	case "item":
		if s.Fix.ItemErr != "" {
			return "err " + s.Fix.ItemErr + "\n", false, false
		}
		return s.Fix.ItemReply + "\nok\n", false, false
	case "buy", "sell", "waypoint", "portal", "pickup":
		reply, ok := s.Fix.World[f[0]]
		if !ok {
			return fmt.Sprintf("err unknown verb '%s'\n", f[0]), false, false
		}
		if strings.HasPrefix(reply, "err ") {
			return reply + "\n", false, false
		}
		return reply + "\nok\n", false, false
	case "info":
		s.frame += 7
		return fmt.Sprintf("w=800 h=600 frame=%d\nok\n", s.frame), false, false
	case "state":
		return s.Fix.State + "\nok\n", false, false
	case "screenshot":
		if len(f) < 2 {
			return "err usage: screenshot <path>\n", false, false
		}
		return "ok 12345\n", false, false
	case "quit":
		s.quit = true
		return "ok\n", false, true
	case "save", "rclick", "text", "move", "wheel":
		return "ok\n", false, false
	case "click":
		s.needs = false
		return "ok\n", false, false
	case "key":
		if len(f) < 2 || strings.EqualFold(f[1], "shift") {
			return "err unknown key\n", false, false
		}
		return "ok\n", false, false
	case "npcs":
		return payload(s.Fix.NPCs) + "ok\n", false, false
	case "monsters":
		return payload(s.Fix.Monsters) + "ok\n", false, false
	case "ground":
		return payload(s.Fix.Ground) + "ok\n", false, false
	case "menu":
		return payload(s.Fix.Menu) + "ok\n", false, false
	case "items":
		return payload(s.Fix.Items) + "ok\n", false, false
	case "cmd":
		return s.cmd(f)
	case "debug":
		return s.debug(f)
	}
	return fmt.Sprintf("err unknown verb '%s'\n", f[0]), false, false
}

func (s *Server) cmd(f []string) (string, bool, bool) {
	if len(f) < 2 {
		return "err usage: cmd <what> ...\n", false, false
	}
	switch f[1] {
	case "move", "interact":
		if s.needs && !s.Opts.NoQuirk {
			return "ok\n", true, false
		}
		return "ok\n", false, false
	case "skillhold":
		return s.skillHold(f), false, false
	case "skill":
		if u, ok := ParseSkill(f); ok { // handle holds s.mu
			s.skills = append(s.skills, u)
		}
		return "ok\n", false, false
	case "pickup", "resurrect", "stat", "skillpt", "select", "belt", "use":
		return "ok\n", false, false
	}
	return fmt.Sprintf("err unknown cmd '%s'\n", f[1]), false, false
}

// skillHold is the fork's hold verb: the start answers at once, "status" reads the hold (Fixtures.Hold).
func (s *Server) skillHold(f []string) string {
	switch {
	case s.Fix.Hold == nil:
		return "err unknown cmd 'skillhold'\n"
	case len(f) == 3 && f[2] == "status":
		st := s.Fix.Hold[0]
		if len(s.Fix.Hold) > 1 {
			s.Fix.Hold = s.Fix.Hold[1:]
		}
		return st + "\nok\n"
	case len(f) < 7:
		return "err usage: cmd skillhold <id> <x> <y> <unit> <frames> [left]\n"
	case s.Fix.HoldErr != "":
		return "err skillhold: " + s.Fix.HoldErr + "\n"
	}
	hand := "right"
	if len(f) > 7 && f[7] != "0" {
		hand = "left"
	}
	return fmt.Sprintf("ok hold=%s skill=%s %s\n", f[6], f[2], hand)
}

func (s *Server) debug(f []string) (string, bool, bool) {
	if len(f) < 2 {
		return "err usage: debug <what>\n", false, false
	}
	switch f[1] {
	case "players":
		return payload(s.Fix.Players) + "ok\n", false, false
	case "level":
		return "ok " + s.Fix.Level + "\n", false, false
	case "quest":
		if len(f) >= 4 && f[3] == "show" {
			return "ok " + s.Fix.Quest + "\n", false, false
		}
		return "ok\n", false, false
	case "corpses":
		return payload(s.Fix.Corpses) + "ok\n", false, false
	case "warps":
		return payload(s.Fix.Warps) + "ok\n", false, false
	case "presets":
		return payload(s.Fix.Presets) + "ok\n", false, false
	case "objects":
		return payload(s.Fix.Objects) + "ok\n", false, false
	case "levelmap":
		if s.Fix.LevelMap == "" {
			return "err unknown debug verb\n", false, false
		}
		return s.Fix.LevelMap + "\nok\n", false, false
	case "warptriggers":
		if s.Fix.WarpTriggers == "" {
			return "err unknown debug verb\n", false, false
		}
		return s.Fix.WarpTriggers + "\nok\n", false, false
	case "levelmonsters":
		if s.Fix.LevelMonsters == "" {
			return "err unknown debug verb\n", false, false
		}
		return s.Fix.LevelMonsters + "\nok\n", false, false
	case "minions", "skillstate", "combatlog":
		line := map[string]string{"minions": s.Fix.Minions, "skillstate": s.Fix.SkillState, "combatlog": s.Fix.CombatLog}[f[1]]
		if line == "" {
			return "err read combat off: debug " + f[1] + " needs d2d --read-combat\n", false, false
		}
		return line + "\nok\n", false, false
	case "portals":
		return payload(s.Fix.Portals) + "ok\n", false, false
	case "grid":
		return payload(s.Fix.Grid) + "ok\n", false, false
	case "attack":
		return "ok " + s.Fix.Attack + "\n", false, false
	case "passives":
		return "ok " + s.Fix.Passives + "\n", false, false
	case "blocked":
		return "ok 0\n", false, false
	case "goto", "enter", "warp", "difficulty":
		s.needs = true
		if f[1] == "goto" {
			return "ok 30.5 20.5\n", false, false
		}
		return "ok\n", false, false
	case "die", "kill", "stat", "statpts", "skillpts", "clearinv", "points", "skill":
		return "ok\n", false, false
	case "equip":
		return "ok quantity=1\n", false, false
	}
	return fmt.Sprintf("err unknown verb 'debug %s'\n", f[1]), false, false
}

// TempSocket returns a short socket path in a private temp dir (unix paths are limited to about 100
// bytes, so t.TempDir() can be too long); cleanup removes it.
func TempSocket() (path string, cleanup func(), err error) {
	dir, err := os.MkdirTemp("", "d2dctl")
	if err != nil {
		return "", nil, err
	}
	return filepath.Join(dir, "d.sock"), func() { os.RemoveAll(dir) }, nil
}

// Sample returns fixtures shaped like the spike transcript (Den of Evil, Necromancer clvl 1, Rogue camp).
func Sample() Fixtures {
	return Fixtures{
		State: "screen=ingame class=Necromancer name=Spike difficulty=0 cam=30.70,13.70 walking=0 gold=0 str=15 life=45/45 mana=25/25 stamina=15360/15360 level=1 exp=0 pmode=NU gamemenu=0:0 merc=-",
		NPCs: []string{
			"Kashya\t812\t484\t1\t0\t35.7\t13.5",
			"Deckard Cain\t300\t200\t0\t3\t28.0\t10.0",
		},
		Monsters: []string{
			"fallen1\t12.10\t28.70\t752\t436\t2/2\tNU\t-\t-\tlvl1\tFallen\t#2\t-",
			"fallens1\t13.00\t29.50\t760\t440\t7/9\tA1\tchampion\t7,26\tlvl2\tFallen Shaman\t#5\tA1:3",
		},
		Ground:   []string{"gld\t8800 Gold\t371\t331\t#1\t40.9\t11.5"},
		Menu:     []string{"Kashya\t812\t450", "talk\t812\t468", "cancel\t812\t498"},
		Items:    []string{"[wnd loc=1 slot=4 q=2 panel=0 id=1 size=1x2 at=0,0]", "  Wand", "  One-Hand Damage: 2 to 4"},
		Corpses:  []string{"2\t40.10\t22.10\t3"},
		Players:  []string{"1\tSpike\t30.70\t13.70\t-1\t-1\t0"},
		Warps:    []string{"0\t67.0\t28.0\t8"},
		Presets:  []string{"0\t51\t30.0\t10.0\t0\tWaypoint", "1\t17\t12.0\t11.0\t1\tChest"},
		Objects:  []string{"0\t20.0\t21.0\tchest\t0\tlocked\tNU"},
		Portals:  []string{"0\t2\t40.1\t22.1", "1\t1\t30.7\t14.3"},
		Grid:     []string{"....#", "..##."},
		Level:    "1 30.70 13.70 57 41 864 912",
		Quest:    "bits=0x0014 den=3 skillpts=0 states=3,0,0,0,0,0",
		Attack:   "dmg=2-4 ar=80 def=6 block=0 dr=0%+0 mdr=0 res=0/0/0/0 cb=0",
		Passives: "11=5 12=3/wand",
	}
}
