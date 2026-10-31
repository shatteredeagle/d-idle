package fleet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"d2lod-bot/internal/event"
)

var (
	ErrNoSlot    = errors.New("fleet: no such slot")
	ErrDuplicate = errors.New("fleet: duplicate slot id or character")
	ErrLeased    = errors.New("fleet: slot is leased")
)

// SrcFleet is the event Source of everything published here.
const SrcFleet = "fleet"

// Backoff is the restart policy of one slot: Min doubling up to Max; reset after the game ran StableAfter.
type Backoff struct {
	Min, Max, StableAfter time.Duration
}

// Config of a Fleet. Runtime is required; everything else has a default.
type Config struct {
	Runtime Runtime
	// Bots, when set, runs a d2bot process per slot while its game is up (nil = game and helper only).
	Bots    BotRunner
	Probe   Probe // default OSProbe
	Limits  Limits
	Bus     *event.Bus
	DataDir string // disk admission is checked here (default ".")
	// Prefix maps a slot id to its Wine prefix directory (default DataDir/prefixes/<id>; paths.Layout has the real one).
	Prefix func(id string) string
	// StateFile, when set, persists the slot specs so Restore can re-add them after a host restart.
	StateFile string

	Stagger      time.Duration // pause after each start so Game.exe launches never overlap (default 5 s)
	PollEvery    time.Duration // liveness poll (default 5 s)
	DrainTimeout time.Duration // OOG leave-game budget on remove (default 30 s)
	Backoff      Backoff       // default 2 s .. 2 min, stable after 5 min
	GameWait     time.Duration // bot-owned game: how long d2bot may take to launch Game.exe (default 6 min)
	MaxRestarts  int           // consecutive restarts before a slot is Failed (0 = unlimited)

	Clock func() time.Time
	Sleep func(context.Context, time.Duration) error // tests inject a fast one
}

type slot struct {
	spec     Spec
	alloc    Alloc
	state    State
	restarts int
	lastErr  string
	since    time.Time
	lease    *Lease
	cancel   context.CancelFunc
	done     chan struct{}
	bot      *BotStatus         // last report of the slot's d2bot, nil when none ran
	kick     context.CancelFunc // bot-owned game: stops the current run (d2bot and with it the game); nil between runs
}

// Fleet supervises the slots of one host. Safe for concurrent use.
type Fleet struct {
	cfg Config

	mu      sync.Mutex
	slots   map[string]*slot
	changed chan struct{} // closed and replaced on every state change (WaitState)
	leaseN  uint64

	startMu sync.Mutex // serialises Game.exe launches (staggered, like launch.StartInstance)
	wg      sync.WaitGroup
}

func New(cfg Config) (*Fleet, error) {
	if cfg.Runtime == nil {
		return nil, errors.New("fleet: Runtime is required")
	}
	if cfg.Probe == nil {
		cfg.Probe = OSProbe{}
	}
	cfg.Limits = cfg.Limits.withDefaults()
	if cfg.DataDir == "" {
		cfg.DataDir = "."
	}
	if cfg.Prefix == nil {
		d := cfg.DataDir
		cfg.Prefix = func(id string) string { return filepath.Join(d, "prefixes", id) }
	}
	if cfg.Stagger == 0 {
		cfg.Stagger = 5 * time.Second
	}
	if cfg.PollEvery == 0 {
		cfg.PollEvery = 5 * time.Second
	}
	if cfg.DrainTimeout == 0 {
		cfg.DrainTimeout = 30 * time.Second
	}
	if cfg.Backoff.Min == 0 {
		cfg.Backoff = Backoff{Min: 2 * time.Second, Max: 2 * time.Minute, StableAfter: 5 * time.Minute}
	}
	if cfg.GameWait == 0 {
		// d2bot's restart is Shutdown 15 s + helper up to 60 s + Launch up to 90 s + login, and a loaded box stretches each
		// (S4, docs/stallfatal.md): 3 min was a thin margin, so the fleet only steps in after twice that.
		cfg.GameWait = 6 * time.Minute
	}
	if cfg.Clock == nil {
		cfg.Clock = time.Now
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
	if o, ok := cfg.Bots.(GameOwner); ok && o.OwnsGame() {
		if _, ok := cfg.Runtime.(Preparer); !ok {
			return nil, errors.New("fleet: the bot runner owns the game, so Runtime must implement Preparer")
		}
	}
	return &Fleet{cfg: cfg, slots: map[string]*slot{}, changed: make(chan struct{})}, nil
}

func (f *Fleet) emit(id string, st State, reason, msg string) {
	if f.cfg.Bus == nil {
		return
	}
	f.cfg.Bus.Publish(event.Event{V: event.Version, Instance: id, Kind: event.KindLifecycle, Source: SrcFleet,
		Data: event.Lifecycle{State: "slot." + string(st), Reason: reason, Message: msg}})
}

// set changes a slot's state under the lock, wakes waiters and publishes the event.
func (f *Fleet) set(s *slot, st State, reason string, err error) {
	f.mu.Lock()
	s.state, s.since = st, f.cfg.Clock()
	msg := ""
	if err != nil {
		s.lastErr = err.Error()
		msg = s.lastErr
	}
	close(f.changed)
	f.changed = make(chan struct{})
	id := s.alloc.ID
	f.mu.Unlock()
	f.emit(id, st, reason, msg)
}

// active counts slots that hold (or are about to hold) a running game.
func (f *Fleet) active() int {
	n := 0
	for _, s := range f.slots {
		if s.state != Failed {
			n++
		}
	}
	return n
}

// Add admits a slot, allocates its resources and starts its supervisor. It returns once the slot is
// registered (state starting); use WaitState to wait for running. A refusal is an *AdmissionError.
func (f *Fleet) Add(ctx context.Context, spec Spec) (View, error) {
	if err := spec.Validate(); err != nil {
		return View{}, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, s := range f.slots {
		if strings.EqualFold(s.spec.Character, spec.Character) {
			return View{}, fmt.Errorf("%w: character %s is already in slot %s (one save dir per character)", ErrDuplicate, spec.Character, s.alloc.ID)
		}
	}
	n := 0
	if spec.ID != "" {
		if _, taken := f.slots[spec.ID]; taken {
			return View{}, fmt.Errorf("%w: slot %s exists", ErrDuplicate, spec.ID)
		}
		fmt.Sscanf(spec.ID, "i%d", &n)
	} else {
		for i := 1; i <= 9; i++ {
			if _, taken := f.slots[fmt.Sprintf("i%d", i)]; !taken {
				n = i
				break
			}
		}
		if n == 0 {
			_, err := f.cfg.Limits.Check(f.cfg.Probe, 9, f.cfg.DataDir) // reports the slots refusal
			return View{}, err
		}
	}
	if _, aerr := f.cfg.Limits.Check(f.cfg.Probe, f.active(), f.cfg.DataDir); aerr != nil {
		return View{}, aerr
	}
	al := AllocFor(n, f.cfg.Prefix)
	spec.ID = al.ID
	sctx, cancel := context.WithCancel(context.Background())
	s := &slot{spec: spec, alloc: al, state: Starting, since: f.cfg.Clock(), cancel: cancel, done: make(chan struct{})}
	f.slots[al.ID] = s
	f.persistLocked()
	close(f.changed)
	f.changed = make(chan struct{})
	f.wg.Add(1)
	go f.supervise(sctx, s)
	f.emit(al.ID, Starting, "added", spec.Character)
	return f.viewLocked(s), nil
}

// supervise is the per-slot loop: bring the instance up, watch it, restart with backoff when it dies.
func (f *Fleet) supervise(ctx context.Context, s *slot) {
	defer f.wg.Done()
	defer close(s.done)
	bo := f.cfg.Backoff.Min
	for {
		f.set(s, Starting, "start", nil)
		err := f.bringUp(ctx, s)
		if ctx.Err() != nil {
			return
		}
		var ranFor time.Duration
		if err == nil {
			rctx, rcancel := context.WithCancel(ctx)
			f.mu.Lock()
			s.kick = rcancel
			f.mu.Unlock()
			stopBot := func() {}
			if f.botOwnsGame() {
				// d2bot launches the game as its own child (Yama: only an ancestor may read its memory)
				stopBot = f.startBot(rctx, s)
				err = f.awaitGame(rctx, s)
			}
			var t0 time.Time
			if err == nil {
				f.set(s, Running, "up", nil)
				t0 = f.cfg.Clock()
				if !f.botOwnsGame() {
					stopBot = f.startBot(rctx, s)
				}
				err = f.watch(rctx, s)
			}
			stopBot() // before any OOG leave or restart: d2bot and those both drive the game's input
			f.mu.Lock()
			s.kick = nil
			f.mu.Unlock()
			rcancel()
			if ctx.Err() != nil {
				return
			}
			if errors.Is(err, context.Canceled) {
				err = errors.New("restart requested")
			}
			if !t0.IsZero() {
				ranFor = f.cfg.Clock().Sub(t0)
			}
		}
		if ranFor >= f.cfg.Backoff.StableAfter {
			bo = f.cfg.Backoff.Min
			f.mu.Lock()
			s.restarts = 0
			f.mu.Unlock()
		}
		f.mu.Lock()
		s.restarts++
		n := s.restarts
		f.mu.Unlock()
		if f.cfg.MaxRestarts > 0 && n > f.cfg.MaxRestarts {
			f.set(s, Failed, "gave up", fmt.Errorf("%d restarts: %w", n-1, err))
			return
		}
		f.set(s, Restarting, fmt.Sprintf("backoff %s", bo), err)
		// clean leftovers (wineserver, Xvfb) so the next start is from scratch
		_ = f.cfg.Runtime.Stop(ctx, s.alloc)
		if f.cfg.Sleep(ctx, bo) != nil {
			return
		}
		bo = min(bo*2, f.cfg.Backoff.Max)
	}
}

func (f *Fleet) bringUp(ctx context.Context, s *slot) error {
	f.startMu.Lock()
	defer f.startMu.Unlock()
	if f.botOwnsGame() {
		// prefix, Xvfb and helper only; d2bot launches Game.exe and walks the menus itself
		if err := f.cfg.Runtime.(Preparer).Prepare(ctx, s.alloc, s.spec); err != nil {
			return fmt.Errorf("prepare: %w", err)
		}
		return f.cfg.Sleep(ctx, f.cfg.Stagger)
	}
	if err := f.cfg.Runtime.Start(ctx, s.alloc, s.spec); err != nil {
		return fmt.Errorf("start: %w", err)
	}
	if err := f.cfg.Runtime.Login(ctx, s.alloc, s.spec); err != nil {
		return fmt.Errorf("login: %w", err)
	}
	return f.cfg.Sleep(ctx, f.cfg.Stagger)
}

// botOwnsGame: the slot's d2bot launches and owns Game.exe (managed mode), so the Runtime only prepares.
func (f *Fleet) botOwnsGame() bool {
	o, ok := f.cfg.Bots.(GameOwner)
	return ok && o.OwnsGame()
}

// awaitGame waits until the game that d2bot launches is up, for at most GameWait.
func (f *Fleet) awaitGame(ctx context.Context, s *slot) error {
	for n := int(f.cfg.GameWait/f.cfg.PollEvery) + 1; n > 0; n-- {
		ok, err := f.cfg.Runtime.Alive(ctx, s.alloc)
		if err != nil {
			return fmt.Errorf("liveness: %w", err)
		}
		if ok {
			return nil
		}
		if err := f.cfg.Sleep(ctx, f.cfg.PollEvery); err != nil {
			return err
		}
	}
	return errors.New("d2bot did not bring the game up in time")
}

// startBot runs the slot's bot for as long as its game is up; the returned func stops it and waits.
func (f *Fleet) startBot(ctx context.Context, s *slot) (stop func()) {
	if f.cfg.Bots == nil {
		return func() {}
	}
	bctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		f.cfg.Bots.Run(bctx, s.alloc, s.spec, func(b BotStatus) {
			f.mu.Lock()
			s.bot = &b
			f.mu.Unlock()
		})
	}()
	return func() { cancel(); <-done }
}

// watch returns an error when the game is gone (or its liveness cannot be read), nil never: only ctx ends it otherwise.
func (f *Fleet) watch(ctx context.Context, s *slot) error {
	// Bot-owned game: d2bot itself stops and relaunches Game.exe (recycles, lost-game restarts), so a miss
	// only counts once the game has been absent for GameWait, the time d2bot is given to bring it back.
	var missSince time.Time
	for {
		if err := f.cfg.Sleep(ctx, f.cfg.PollEvery); err != nil {
			return err
		}
		ok, err := f.cfg.Runtime.Alive(ctx, s.alloc)
		if err != nil {
			return fmt.Errorf("liveness: %w", err)
		}
		if ok {
			missSince = time.Time{}
			continue
		}
		if !f.botOwnsGame() {
			return errors.New("game process gone")
		}
		now := f.cfg.Clock()
		if missSince.IsZero() {
			missSince = now
		}
		if now.Sub(missSince) >= f.cfg.GameWait {
			return errors.New("game process gone")
		}
	}
}

// RemoveOpts: Force removes a slot that somebody holds a lease on.
type RemoveOpts struct{ Force bool }

// Remove drains a slot: the supervisor stops, the game leaves via OOG (save and exit to the main menu), then
// game, wineserver and Xvfb are stopped and the slot's resources are freed. A failed OOG leave is reported
// in the returned error AFTER the stop still happened (the save is then whatever the game wrote last).
func (f *Fleet) Remove(ctx context.Context, id string, opt RemoveOpts) error {
	f.mu.Lock()
	s, ok := f.slots[id]
	if !ok {
		f.mu.Unlock()
		return ErrNoSlot
	}
	if l := f.liveLeaseLocked(s); l != nil && !opt.Force {
		f.mu.Unlock()
		return fmt.Errorf("%w by %s", ErrLeased, l.Owner)
	}
	s.lease = nil // draining slots hand out no leases and revoke the old one
	prev, cancel, done := s.state, s.cancel, s.done
	f.mu.Unlock()
	f.set(s, Draining, "remove", nil)
	cancel()
	<-done

	var leaveErr error
	// bot-owned game: d2bot (SIGTERM, --stop-game) already saved and left; the fleet holds no memory handle
	if (prev == Running || prev == Starting) && !f.botOwnsGame() {
		lctx, cancel := context.WithTimeout(ctx, f.cfg.DrainTimeout)
		leaveErr = f.cfg.Runtime.LeaveGame(lctx, s.alloc)
		cancel()
	}
	stopErr := f.cfg.Runtime.Stop(ctx, s.alloc)
	f.mu.Lock()
	delete(f.slots, id)
	f.persistLocked()
	close(f.changed)
	f.changed = make(chan struct{})
	f.mu.Unlock()
	f.emit(id, "removed", "removed", s.spec.Character)
	if stopErr != nil {
		return fmt.Errorf("fleet: stop %s: %w", id, stopErr)
	}
	if leaveErr != nil {
		return fmt.Errorf("fleet: %s was stopped without a clean leave-game: %w", id, leaveErr)
	}
	return nil
}

// Restart kills the game of a slot; the supervisor restarts it (counted, with backoff). A Failed slot gets a
// fresh supervisor.
func (f *Fleet) Restart(ctx context.Context, id, reason string) error {
	f.mu.Lock()
	s, ok := f.slots[id]
	if !ok {
		f.mu.Unlock()
		return ErrNoSlot
	}
	failed := s.state == Failed
	kick := s.kick
	if failed {
		s.restarts = 0
		s.done = make(chan struct{})
		sctx, cancel := context.WithCancel(context.Background())
		s.cancel = cancel
		f.wg.Add(1)
		go f.supervise(sctx, s)
	}
	f.mu.Unlock()
	f.emit(id, Restarting, "restart requested", reason)
	if failed {
		return nil
	}
	if kick != nil && f.botOwnsGame() {
		kick() // SIGTERM to d2bot, which stops its game; the supervisor then restarts the slot
		return nil
	}
	return f.cfg.Runtime.Stop(ctx, s.alloc)
}

func (f *Fleet) viewLocked(s *slot) View {
	v := View{ID: s.alloc.ID, Spec: s.spec, Alloc: s.alloc, State: s.state, Restarts: s.restarts, LastError: s.lastErr, Since: s.since}
	if l := f.liveLeaseLocked(s); l != nil {
		v.LeasedBy = l.Owner
	}
	if s.bot != nil {
		b := *s.bot
		v.Bot = &b
	}
	return v
}

// List returns all slots ordered by id.
func (f *Fleet) List() []View {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]View, 0, len(f.slots))
	for _, s := range f.slots {
		out = append(out, f.viewLocked(s))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Get returns one slot.
func (f *Fleet) Get(id string) (View, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.slots[id]
	if !ok {
		return View{}, ErrNoSlot
	}
	return f.viewLocked(s), nil
}

// Status is the host's capacity plus the slots; Capacity.Admit tells whether one more slot would be accepted now.
func (f *Fleet) Status() HostStatus {
	f.mu.Lock()
	n := f.active()
	f.mu.Unlock()
	c, _ := f.cfg.Limits.Check(f.cfg.Probe, n, f.cfg.DataDir)
	return HostStatus{Capacity: c, Slots: f.List()}
}

// WaitState blocks until slot id is in state st (error when it vanishes or ctx ends).
func (f *Fleet) WaitState(ctx context.Context, id string, st State) error {
	for {
		f.mu.Lock()
		s, ok := f.slots[id]
		ch := f.changed
		cur := State("")
		if ok {
			cur = s.state
		}
		f.mu.Unlock()
		if !ok {
			return ErrNoSlot
		}
		if cur == st {
			return nil
		}
		select {
		case <-ch:
		case <-ctx.Done():
			return fmt.Errorf("waiting for %s to be %s (now %s): %w", id, st, cur, ctx.Err())
		}
	}
}

// closeParallel bounds how many slots drain at once in Close.
const closeParallel = 8

// Close drains every slot (graceful remove) in parallel, so the container stop grace is sized for one slot, and
// waits for the supervisors.
func (f *Fleet) Close(ctx context.Context) error {
	var (
		mu    sync.Mutex
		first error
		wg    sync.WaitGroup
		sem   = make(chan struct{}, closeParallel)
	)
	var specs []Spec
	for _, v := range f.List() {
		specs = append(specs, v.Spec)
		sem <- struct{}{}
		wg.Add(1)
		go func(id string) {
			defer func() { <-sem; wg.Done() }()
			if err := f.Remove(ctx, id, RemoveOpts{Force: true}); err != nil {
				mu.Lock()
				if first == nil {
					first = err
				}
				mu.Unlock()
			}
		}(v.ID)
	}
	wg.Wait()
	f.wg.Wait()
	f.keepSpecs(specs) // a shutdown is not a removal: the slots come back on the next start (found in the container first run)
	return first
}

func (f *Fleet) keepSpecs(specs []Spec) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.cfg.StateFile == "" || len(specs) == 0 || len(f.slots) > 0 {
		return
	}
	sort.Slice(specs, func(i, j int) bool { return specs[i].ID < specs[j].ID })
	b, _ := json.MarshalIndent(stateFile{Slots: specs}, "", "  ")
	tmp := f.cfg.StateFile + ".tmp"
	if os.WriteFile(tmp, b, 0o644) == nil {
		_ = os.Rename(tmp, f.cfg.StateFile)
	}
}

type stateFile struct {
	Slots []Spec `json:"slots"`
}

func (f *Fleet) persistLocked() {
	if f.cfg.StateFile == "" {
		return
	}
	var sf stateFile
	for _, s := range f.slots {
		sf.Slots = append(sf.Slots, s.spec)
	}
	sort.Slice(sf.Slots, func(i, j int) bool { return sf.Slots[i].ID < sf.Slots[j].ID })
	b, _ := json.MarshalIndent(sf, "", "  ")
	tmp := f.cfg.StateFile + ".tmp"
	if os.WriteFile(tmp, b, 0o644) == nil {
		_ = os.Rename(tmp, f.cfg.StateFile)
	}
}

// Restore re-adds the specs saved in StateFile (each through admission, so a smaller host takes fewer).
// It returns the refusals; a missing file is not an error.
func (f *Fleet) Restore(ctx context.Context) []error {
	b, err := os.ReadFile(f.cfg.StateFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return []error{err}
	}
	var sf stateFile
	if err := json.Unmarshal(b, &sf); err != nil {
		return []error{err}
	}
	var errs []error
	for _, sp := range sf.Slots {
		if _, err := f.Add(ctx, sp); err != nil {
			errs = append(errs, fmt.Errorf("restore %s: %w", sp.Character, err))
		}
	}
	return errs
}
