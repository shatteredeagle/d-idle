package party

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"d2lod-bot/internal/event"
)

// Driver is one member's OOG side: the real one wraps launch.Flow (create/join a TCP/IP game, LeaveGame;
// the TCP/IP screens are not in the OOG table yet, docs/net T5) and cmd/oog. Fakes in tests.
type Driver interface {
	Create(ctx context.Context, g Game) error // host a TCP/IP game and be inside it
	Join(ctx context.Context, g Game) error   // join g.Host:4000 and be inside it
	Leave(ctx context.Context) error          // save and exit to the main menu
}

// FollowLeader keeps a member within a screen of the leader (real implementation later: nav + roster reads).
type FollowLeader interface {
	Follow(ctx context.Context, leader string) error
	Stop()
}

// Assist attacks what the leader attacks (real implementation later: combat).
type Assist interface {
	Assist(ctx context.Context, leader string) error
	Stop()
}

// Member is one bot in the party. Following and Assisting may be nil (no behaviour wired).
type Member struct {
	ID        string // fleet slot id, "i2"
	Character string // in-game name, what Follow/Assist target
	Role      Role
	Driver    Driver
	Following FollowLeader
	Assisting Assist
}

type State string

const (
	Idle      State = "idle"
	Forming   State = "forming"
	Running   State = "running"
	Handoff   State = "handoff"
	Disbanded State = "disbanded"
)

type MemberState string

const (
	Out    MemberState = "out" // not in the game
	In     MemberState = "in"
	Dead   MemberState = "dead" // in the game as a corpse or on the way back from town
	Left   MemberState = "left" // left on purpose; not rejoined automatically
	Failed MemberState = "failed"
)

var (
	ErrSize     = errors.New("party: size must be 2 to 4")
	ErrState    = errors.New("party: not allowed in this state")
	ErrNoMember = errors.New("party: no such member")
	ErrNoHost   = errors.New("party: no member can host")
	ErrNoGame   = errors.New("party: no current game")
)

// Config of a Party. Zero values pick defaults.
type Config struct {
	ID          string
	Difficulty  string     // default "normal"
	HostAddr    string     // default 127.0.0.1 (all bots on one machine)
	Dir         *Directory // default a private one
	Bus         *event.Bus // optional
	Sleep       func(context.Context, time.Duration) error
	JoinStagger time.Duration // pause between joiners (default 2 s)
	JoinTries   int           // per member (default 3)
	Backoff     time.Duration // first retry delay, doubled (default 3 s)
}

type slot struct {
	Member
	state MemberState
}

// Party is the orchestrator state machine. All methods are safe for concurrent use; operations run one at a
// time, Status never blocks behind them.
type Party struct {
	cfg Config

	opMu sync.Mutex // serialises operations

	mu      sync.Mutex // guards the fields below
	state   State
	slots   []*slot
	host    string // member id
	gen     int
	game    Game
	lastErr string
}

// New validates the composition: 2 to 4 members, unique ids and characters, no Solo, at most one leader, at
// least one member that can host and one that fights.
func New(cfg Config, members []Member) (*Party, error) {
	if cfg.ID == "" {
		cfg.ID = "party"
	}
	if cfg.Difficulty == "" {
		cfg.Difficulty = "normal"
	}
	if cfg.HostAddr == "" {
		cfg.HostAddr = "127.0.0.1"
	}
	if cfg.Dir == nil {
		cfg.Dir = NewDirectory()
	}
	if cfg.Sleep == nil {
		cfg.Sleep = func(ctx context.Context, d time.Duration) error {
			t := time.NewTimer(d)
			defer t.Stop()
			select {
			case <-t.C:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	if cfg.JoinStagger == 0 {
		cfg.JoinStagger = 2 * time.Second
	}
	if cfg.JoinTries == 0 {
		cfg.JoinTries = 3
	}
	if cfg.Backoff == 0 {
		cfg.Backoff = 3 * time.Second
	}
	if len(members) < 2 || len(members) > 4 {
		return nil, fmt.Errorf("%w (got %d)", ErrSize, len(members))
	}
	p := &Party{cfg: cfg, state: Idle}
	ids, chars := map[string]bool{}, map[string]bool{}
	leaders, hosts, fighters := 0, 0, 0
	for _, m := range members {
		if m.ID == "" || m.Driver == nil {
			return nil, errors.New("party: member needs an id and a driver")
		}
		if !m.Role.Valid() || m.Role == Solo {
			return nil, fmt.Errorf("party: member %s has role %q, which cannot be in a party", m.ID, m.Role)
		}
		if ids[m.ID] || (m.Character != "" && chars[m.Character]) {
			return nil, fmt.Errorf("party: duplicate member %s/%s", m.ID, m.Character)
		}
		ids[m.ID], chars[m.Character] = true, true
		pol := PolicyFor(m.Role)
		if m.Role == Leader {
			leaders++
		}
		if pol.CanHost {
			hosts++
		}
		if pol.Fights {
			fighters++
		}
		p.slots = append(p.slots, &slot{Member: m, state: Out})
	}
	if leaders > 1 {
		return nil, errors.New("party: more than one leader")
	}
	if hosts == 0 {
		return nil, ErrNoHost
	}
	if fighters == 0 {
		return nil, errors.New("party: nobody fights (all mules)")
	}
	return p, nil
}

func (p *Party) emit(member, state, msg string) {
	if p.cfg.Bus == nil {
		return
	}
	p.cfg.Bus.Publish(event.Event{V: event.Version, Instance: member, Kind: event.KindLifecycle, Source: "party",
		Data: event.Lifecycle{State: "party." + state, Reason: p.cfg.ID, Message: msg}})
}

func (p *Party) find(id string) *slot {
	for _, s := range p.slots {
		if s.ID == id {
			return s
		}
	}
	return nil
}

// pickHost returns the best member to host among those not excluded, by hostOrder, then member order.
func (p *Party) pickHost(skip map[string]bool) *slot {
	for _, r := range hostOrder {
		for _, s := range p.slots {
			if s.Role == r && !skip[s.ID] && s.state != Left && s.state != Failed {
				return s
			}
		}
	}
	return nil
}

// leaderName is who followers follow: the leader if it is in the game, else the host.
func (p *Party) leaderName() string {
	for _, s := range p.slots {
		if s.Role == Leader && (s.state == In || s.state == Dead) {
			return s.Character
		}
	}
	if h := p.find(p.host); h != nil {
		return h.Character
	}
	return ""
}

// Start forms the party: the host creates the game, then members join one by one (staggered). Members that
// cannot join become Failed; the party runs when the host and at least one other member are in.
func (p *Party) Start(ctx context.Context) error {
	p.opMu.Lock()
	defer p.opMu.Unlock()
	p.mu.Lock()
	if p.state != Idle {
		p.mu.Unlock()
		return fmt.Errorf("%w: %s", ErrState, p.state)
	}
	p.mu.Unlock()
	return p.form(ctx, nil, Forming)
}

// form runs one game generation. skip lists members not to use as host (a lost host); they are not joined
// either unless they are Out and not skipped... a skipped host is left Out and rejoins through Rejoin.
func (p *Party) form(ctx context.Context, skip map[string]bool, st State) error {
	p.mu.Lock()
	p.state = st
	h := p.pickHost(skip)
	if h == nil {
		p.state = Idle
		p.mu.Unlock()
		return ErrNoHost
	}
	p.gen++
	g := Game{Name: GameName(p.cfg.ID, p.gen), Password: fmt.Sprintf("%s%d", "pw", p.gen), Host: p.cfg.HostAddr,
		Difficulty: p.cfg.Difficulty, MaxPlayers: len(p.slots)}
	p.host, p.game = h.ID, g
	p.mu.Unlock()
	p.cfg.Dir.Publish(p.cfg.ID, g)
	p.emit(h.ID, "creating", "game "+g.Name)

	if err := h.Driver.Create(ctx, g); err != nil {
		p.cfg.Dir.Revoke(p.cfg.ID)
		p.mu.Lock()
		h.state, p.state, p.lastErr = Failed, Idle, err.Error()
		p.mu.Unlock()
		p.emit(h.ID, "create_failed", err.Error())
		return fmt.Errorf("party: host %s could not create the game: %w", h.ID, err)
	}
	p.mu.Lock()
	h.state = In
	p.mu.Unlock()

	joined := 0
	for _, s := range p.slots {
		if s == h || s.state == Left || s.state == Failed || skip[s.ID] {
			continue
		}
		if joined > 0 {
			if err := p.cfg.Sleep(ctx, p.cfg.JoinStagger); err != nil {
				return err
			}
		}
		if err := p.join(ctx, s); err != nil {
			p.emit(s.ID, "join_failed", err.Error())
			continue
		}
		joined++
	}
	if joined == 0 {
		_ = h.Driver.Leave(ctx)
		p.cfg.Dir.Revoke(p.cfg.ID)
		p.mu.Lock()
		h.state, p.state, p.lastErr = Out, Idle, "nobody joined"
		p.mu.Unlock()
		return errors.New("party: nobody joined the host's game")
	}
	p.mu.Lock()
	p.state = Running
	p.mu.Unlock()
	p.emit(h.ID, "running", fmt.Sprintf("game %s, %d in", g.Name, joined+1))
	p.startBehaviours(ctx)
	return nil
}

// join joins the current game with retries and backoff; marks the slot In or Failed.
func (p *Party) join(ctx context.Context, s *slot) error {
	g, ok := p.cfg.Dir.Lookup(p.cfg.ID)
	if !ok {
		return ErrNoGame
	}
	var err error
	d := p.cfg.Backoff
	for try := 1; try <= p.cfg.JoinTries; try++ {
		p.mu.Lock()
		s.state = Out
		p.mu.Unlock()
		if err = s.Driver.Join(ctx, g); err == nil {
			p.mu.Lock()
			s.state = In
			p.mu.Unlock()
			return nil
		}
		if try < p.cfg.JoinTries {
			if e := p.cfg.Sleep(ctx, d); e != nil {
				return e
			}
			d *= 2
		}
	}
	p.mu.Lock()
	s.state, p.lastErr = Failed, err.Error()
	p.mu.Unlock()
	return err
}

func (p *Party) startBehaviours(ctx context.Context) {
	p.mu.Lock()
	lead, host := p.leaderName(), p.host
	var run []*slot
	for _, s := range p.slots {
		if s.state == In && s.ID != host {
			run = append(run, s)
		}
	}
	p.mu.Unlock()
	for _, s := range run {
		pol := PolicyFor(s.Role)
		if pol.Follows && s.Following != nil && s.Character != lead {
			_ = s.Following.Follow(ctx, lead)
		}
		if pol.Assists && s.Assisting != nil && s.Character != lead {
			_ = s.Assisting.Assist(ctx, lead)
		}
	}
}

func (s *slot) stopBehaviours() {
	if s.Following != nil {
		s.Following.Stop()
	}
	if s.Assisting != nil {
		s.Assisting.Stop()
	}
}

func (p *Party) stopAll() {
	for _, s := range p.slots {
		s.stopBehaviours()
	}
}

// Leave takes a member out on purpose (it is not rejoined). The host leaving ends the game for everybody
// (V multiplayer.md), so that triggers a handoff when two or more members remain, else a disband.
func (p *Party) Leave(ctx context.Context, id string) error {
	p.opMu.Lock()
	defer p.opMu.Unlock()
	s := p.find(id)
	if s == nil {
		return ErrNoMember
	}
	p.mu.Lock()
	if p.state != Running {
		p.mu.Unlock()
		return fmt.Errorf("%w: %s", ErrState, p.state)
	}
	wasHost := id == p.host
	p.mu.Unlock()
	s.stopBehaviours()
	err := s.Driver.Leave(ctx)
	p.mu.Lock()
	s.state = Left
	p.mu.Unlock()
	p.emit(id, "left", "")
	if wasHost {
		return p.handoff(ctx, map[string]bool{id: true})
	}
	return err
}

// MemberDied records a death (the bot revives itself in town; the party does not leave the game).
// Followers stop following a dead leader until MemberRevived.
func (p *Party) MemberDied(id string) error {
	p.opMu.Lock()
	defer p.opMu.Unlock()
	s := p.find(id)
	if s == nil {
		return ErrNoMember
	}
	p.mu.Lock()
	if s.state != In {
		p.mu.Unlock()
		return fmt.Errorf("%w: member %s is %s", ErrState, id, s.state)
	}
	s.state = Dead
	isLead := s.Role == Leader
	p.mu.Unlock()
	if isLead {
		for _, o := range p.slots {
			if o != s {
				o.stopBehaviours()
			}
		}
	}
	p.emit(id, "member_died", "")
	return nil
}

// MemberRevived marks a dead member alive again and restarts behaviours if it is the leader.
func (p *Party) MemberRevived(ctx context.Context, id string) error {
	p.opMu.Lock()
	defer p.opMu.Unlock()
	s := p.find(id)
	if s == nil {
		return ErrNoMember
	}
	p.mu.Lock()
	if s.state != Dead {
		p.mu.Unlock()
		return fmt.Errorf("%w: member %s is %s", ErrState, id, s.state)
	}
	s.state = In
	p.mu.Unlock()
	p.emit(id, "member_revived", "")
	if s.Role == Leader {
		p.startBehaviours(ctx)
	}
	return nil
}

// MemberLost reports that a member's game or connection is gone (crash, disconnect). A lost host ends the
// game, so everybody is brought out and a new host creates a new game (handoff). Any other member becomes
// Out and comes back through Rejoin.
func (p *Party) MemberLost(ctx context.Context, id string) error {
	p.opMu.Lock()
	defer p.opMu.Unlock()
	s := p.find(id)
	if s == nil {
		return ErrNoMember
	}
	p.mu.Lock()
	if p.state != Running {
		p.mu.Unlock()
		return fmt.Errorf("%w: %s", ErrState, p.state)
	}
	wasHost := id == p.host
	s.state = Out
	p.mu.Unlock()
	s.stopBehaviours()
	p.emit(id, "member_lost", "")
	if wasHost {
		return p.handoff(ctx, map[string]bool{id: true})
	}
	return nil
}

// Rejoin brings an Out (or Failed) member back into the current game, with retries.
func (p *Party) Rejoin(ctx context.Context, id string) error {
	p.opMu.Lock()
	defer p.opMu.Unlock()
	s := p.find(id)
	if s == nil {
		return ErrNoMember
	}
	p.mu.Lock()
	ok := p.state == Running && (s.state == Out || s.state == Failed || s.state == Left)
	st := p.state
	p.mu.Unlock()
	if !ok {
		return fmt.Errorf("%w: party %s, member %s", ErrState, st, s.state)
	}
	if err := p.join(ctx, s); err != nil {
		return err
	}
	p.emit(id, "rejoined", "")
	p.startBehaviours(ctx)
	return nil
}

// handoff ends the current game (all members leave), then forms a new one with another host. The skipped
// members (the lost or departed host) stay Out/Left; the lost one comes back with Rejoin.
func (p *Party) handoff(ctx context.Context, skip map[string]bool) error {
	p.mu.Lock()
	p.state = Handoff
	p.mu.Unlock()
	p.emit(p.host, "handoff", "host gone, game ends for everybody")
	p.stopAll()
	p.cfg.Dir.Revoke(p.cfg.ID)
	for _, s := range p.slots {
		if s.state == In || s.state == Dead {
			_ = s.Driver.Leave(ctx) // best effort: the game is gone anyway
			p.mu.Lock()
			s.state = Out
			p.mu.Unlock()
		}
	}
	remaining := 0
	p.mu.Lock()
	for _, s := range p.slots {
		if !skip[s.ID] && s.state != Left && s.state != Failed {
			remaining++
		}
	}
	p.mu.Unlock()
	if remaining < 2 {
		p.mu.Lock()
		p.state = Disbanded
		p.mu.Unlock()
		p.emit("", "disbanded", "fewer than two members left")
		return nil
	}
	err := p.form(ctx, skip, Handoff)
	if err != nil {
		p.mu.Lock()
		p.state = Disbanded
		p.mu.Unlock()
		p.emit("", "disbanded", err.Error())
	}
	return err
}

// Disband makes everybody leave and ends the party.
func (p *Party) Disband(ctx context.Context) error {
	p.opMu.Lock()
	defer p.opMu.Unlock()
	p.stopAll()
	var first error
	for _, s := range p.slots {
		if s.state == In || s.state == Dead {
			if err := s.Driver.Leave(ctx); err != nil && first == nil {
				first = err
			}
		}
		p.mu.Lock()
		s.state = Out
		p.mu.Unlock()
	}
	p.cfg.Dir.Revoke(p.cfg.ID)
	p.mu.Lock()
	p.state = Disbanded
	p.mu.Unlock()
	p.emit("", "disbanded", "")
	return first
}

// MemberView and Snapshot are the status shapes (JSON for the dashboard party strip).
type MemberView struct {
	ID, Character string
	Role          Role
	State         MemberState
	Host          bool
}

type Snapshot struct {
	ID      string
	State   State
	Host    string
	Game    Game // password cleared
	Gen     int
	Members []MemberView
	Scaling Scaling // for the players currently in the game
	LastErr string  `json:",omitempty"`
}

func (p *Party) Status() Snapshot {
	p.mu.Lock()
	defer p.mu.Unlock()
	sn := Snapshot{ID: p.cfg.ID, State: p.state, Host: p.host, Game: p.game.Public(), Gen: p.gen, LastErr: p.lastErr}
	in := 0
	for _, s := range p.slots {
		sn.Members = append(sn.Members, MemberView{ID: s.ID, Character: s.Character, Role: s.Role, State: s.state, Host: s.ID == p.host})
		if s.state == In || s.state == Dead {
			in++
		}
	}
	sort.SliceStable(sn.Members, func(i, j int) bool { return sn.Members[i].ID < sn.Members[j].ID })
	sn.Scaling = ScalingFor(in)
	return sn
}
