package selfupdate

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	ErrRevoked      = errors.New("this version was revoked")
	ErrDowngrade    = errors.New("refusing a release older than the floor (downgrade)")
	ErrIncompatible = errors.New("release is not compatible with this install")
	ErrNeedsImage   = errors.New("this update needs a new image: run docker compose pull")
	ErrNotSafe      = errors.New("the hero is not in a safe spot; try again or force")
	ErrNotReady     = errors.New("no verified update is staged")
	ErrConfirm      = errors.New("confirm does not match the staged version")
	ErrHashMismatch = errors.New("downloaded file does not match the manifest")
	ErrDisk         = errors.New("not enough free disk space for the update")
)

// States of the engine (design 3.6).
const (
	StateIdle        = "idle"
	StateChecking    = "checking"
	StateDownloading = "downloading"
	StateReady       = "ready"
	StateApplying    = "applying"
	StateError       = "error"
)

// Config wires the engine to the process. Everything that touches the game is a callback, so the package moves
// bytes only.
type Config struct {
	Layout       Layout
	Client       *Client
	Keys         []ed25519.PublicKey // trusted public keys (current, next)
	Running      string              // version of the running release
	ImageABI     int                 // label io.didle.image.abi of this image
	Local        Compat              // highest bot_api and d2d_ctl this install speaks; DataSchema must match exactly (U: no migrations yet)
	CheckEnabled bool                // false: no outbound call at all
	IdleSafe     func() bool         // nil: always safe
	Shutdown     func() error        // save the hero and exit; the supervisor restarts the container
	SelfTest     func(dir string) error
	DiskFree     func() (uint64, error) // nil: not checked
	Now          func() time.Time
}

// Available is a verified release newer than the running one.
type Available struct {
	Version    string `json:"version"`
	Notes      string `json:"notes"`
	Size       int64  `json:"size"`
	NeedsImage bool   `json:"needs_image"`
}

// Status is the body of GET /api/update/status.
type Status struct {
	State        string          `json:"state"`
	Step         string          `json:"step,omitempty"`
	Progress     float64         `json:"progress"`
	Running      string          `json:"running"`
	Available    *Available      `json:"available,omitempty"`
	LastCheck    time.Time       `json:"last_check"`
	LastError    string          `json:"last_error,omitempty"`
	CheckEnabled bool            `json:"check_enabled"`
	Rollback     *RollbackMarker `json:"rollback,omitempty"`
}

// Engine is the updater state machine. All methods are safe for concurrent use.
type Engine struct {
	cfg Config

	mu      sync.Mutex
	state   string
	step    string
	progr   float64
	avail   *Available
	remote  Remote
	man     Manifest
	lastErr string
	cancel  context.CancelFunc
}

// New builds an engine.
func New(cfg Config) *Engine {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Client != nil {
		cfg.Client.Offline = cfg.Client.Offline || !cfg.CheckEnabled
	}
	return &Engine{cfg: cfg, state: StateIdle}
}

// Status is a snapshot for the UI.
func (e *Engine) Status() Status {
	e.mu.Lock()
	defer e.mu.Unlock()
	st, _ := e.cfg.Layout.LoadState()
	s := Status{State: e.state, Step: e.step, Progress: e.progr, Running: e.cfg.Running, Available: e.avail,
		LastCheck: st.LastCheck, LastError: e.lastErr, CheckEnabled: e.cfg.CheckEnabled}
	if m, err := e.cfg.Layout.ReadMarker(); err == nil && m != nil && !m.Seen {
		s.Rollback = m
	}
	return s
}

func (e *Engine) fail(err error) error {
	e.mu.Lock()
	e.state, e.lastErr, e.step = StateError, err.Error(), ""
	e.mu.Unlock()
	return err
}

func (e *Engine) set(state, step string) {
	e.mu.Lock()
	e.state, e.step = state, step
	e.mu.Unlock()
}

// Check asks GitHub for a newer release, verifies it and records it as available. It downloads no asset
// beyond the manifest, its signature and REVOKED. Errors leave a previously found update untouched.
func (e *Engine) Check(ctx context.Context) error {
	e.mu.Lock()
	if e.state == StateDownloading || e.state == StateApplying || e.state == StateChecking {
		e.mu.Unlock()
		return fmt.Errorf("busy (%s)", e.state)
	}
	prev := e.state
	e.state, e.lastErr = StateChecking, ""
	e.mu.Unlock()
	restore := func(err error) error {
		e.mu.Lock()
		e.state, e.lastErr = prev, err.Error()
		if prev == StateError {
			e.state = StateIdle
		}
		e.mu.Unlock()
		return err
	}
	st, err := e.cfg.Layout.LoadState()
	if err != nil {
		return restore(err)
	}
	if e.cfg.Client == nil {
		return restore(ErrOffline)
	}
	r, etag, err := e.cfg.Client.Latest(ctx, st.ETag)
	if err != nil {
		if !errors.Is(err, ErrOffline) {
			st.LastCheck = e.cfg.Now()
			e.cfg.Layout.SaveState(st)
		}
		return restore(err)
	}
	st.LastCheck = e.cfg.Now()
	if r.NotModified {
		e.cfg.Layout.SaveState(st)
		e.mu.Lock()
		e.state = prev
		if prev == StateError || prev == StateChecking {
			e.state = StateIdle
		}
		e.mu.Unlock()
		return nil
	}
	m, err := e.evaluate(&st, r)
	if err != nil {
		e.cfg.Layout.SaveState(st) // floors learned from a verified REVOKED stay, the etag does not
		return restore(err)
	}
	st.ETag = etag
	if err := e.cfg.Layout.SaveState(st); err != nil {
		return restore(err)
	}
	return e.offer(r, m)
}

// offer records a verified release as available (or as nothing, when it is not newer).
func (e *Engine) offer(r Remote, m Manifest) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.remote, e.man = r, m
	if CompareVersions(m.Version, e.cfg.Running) <= 0 {
		e.avail, e.state = nil, StateIdle
		return nil
	}
	var size int64
	for _, f := range m.Files {
		size += f.Size
	}
	e.avail = &Available{Version: m.Version, Notes: m.Notes, Size: size, NeedsImage: m.MinImage > e.cfg.ImageABI}
	e.state = StateIdle
	return nil
}

// evaluate verifies a remote release and applies the policy checks. It updates st with floors learned from signed
// revocations, whether or not the release itself is acceptable.
func (e *Engine) evaluate(st *State, r Remote) (Manifest, error) {
	if err := Verify(e.cfg.Keys, r.Raw, r.Sig); err != nil {
		return Manifest{}, err
	}
	m, err := ParseManifest(r.Raw)
	if err != nil {
		return Manifest{}, err
	}
	raise := func(v string) {
		if v != "" && CompareVersions(v, st.VersionFloor) > 0 {
			st.VersionFloor = v
		}
	}
	raise(m.Revoke.MinVersion)
	var rev *Revoked
	if r.revokedRaw != nil {
		if err := Verify(e.cfg.Keys, r.revokedRaw, r.revokedSig); err != nil {
			return Manifest{}, fmt.Errorf("REVOKED: %w", err)
		}
		x, err := ParseRevoked(r.revokedRaw)
		if err != nil {
			return Manifest{}, err
		}
		rev = &x
		raise(x.MinVersion)
	}
	switch {
	case m.Revoke.Bans(m.Version) || (rev != nil && rev.Bans(m.Version)):
		return Manifest{}, fmt.Errorf("%w: %s", ErrRevoked, m.Version)
	case m.Seq < st.SeqFloor:
		return Manifest{}, fmt.Errorf("%w: seq %d below %d", ErrDowngrade, m.Seq, st.SeqFloor)
	case CompareVersions(m.Version, st.VersionFloor) < 0:
		return Manifest{}, fmt.Errorf("%w: %s below %s", ErrDowngrade, m.Version, st.VersionFloor)
	case st.IsBad(m.Version):
		return Manifest{}, fmt.Errorf("%s was rolled back before and is not retried", m.Version)
	}
	if m.MinImage <= e.cfg.ImageABI {
		if m.Compat.BotAPI > e.cfg.Local.BotAPI || m.Compat.D2DCtl > e.cfg.Local.D2DCtl || m.Compat.DataSchema != e.cfg.Local.DataSchema {
			return Manifest{}, fmt.Errorf("%w: release pairs %+v, install %+v", ErrIncompatible, m.Compat, e.cfg.Local)
		}
	}
	return m, nil
}

// Inbox looks for a release set in <root>/inbox and treats it like a checked release: same signature, floor,
// revocation and compatibility rules, no network. Press Download and Apply afterwards as usual.
func (e *Engine) Inbox() error {
	e.mu.Lock()
	if e.state == StateDownloading || e.state == StateApplying {
		s := e.state
		e.mu.Unlock()
		return fmt.Errorf("busy (%s)", s)
	}
	e.lastErr = ""
	e.mu.Unlock()
	st, err := e.cfg.Layout.LoadState()
	if err != nil {
		return err
	}
	r, err := LoadInbox(e.cfg.Layout.Inbox())
	if err != nil {
		return e.softFail(err)
	}
	m, err := e.evaluate(&st, r)
	e.cfg.Layout.SaveState(st)
	if err != nil {
		return e.softFail(err)
	}
	return e.offer(r, m)
}

func (e *Engine) softFail(err error) error {
	e.mu.Lock()
	e.lastErr = err.Error()
	e.mu.Unlock()
	return err
}

// Download stages and verifies the available release. It does not restart anything.
func (e *Engine) Download(ctx context.Context) error {
	e.mu.Lock()
	if e.avail == nil {
		e.mu.Unlock()
		return ErrNotReady
	}
	if e.avail.NeedsImage {
		e.mu.Unlock()
		return ErrNeedsImage
	}
	if e.state != StateIdle && e.state != StateError {
		s := e.state
		e.mu.Unlock()
		return fmt.Errorf("busy (%s)", s)
	}
	ctx, cancel := context.WithCancel(ctx)
	e.cancel = cancel
	m, r := e.man, e.remote
	e.state, e.step, e.progr, e.lastErr = StateDownloading, "download", 0, ""
	e.mu.Unlock()
	defer cancel()

	var total int64
	for _, f := range m.Files {
		total += f.Size
	}
	if e.cfg.DiskFree != nil {
		if free, err := e.cfg.DiskFree(); err == nil && free < uint64(total)*3 {
			return e.fail(ErrDisk)
		}
	}
	dir := filepath.Join(e.cfg.Layout.Staging(), m.Version)
	os.RemoveAll(dir)
	var done int64
	for _, f := range m.Files {
		base := done
		err := e.cfg.Client.Download(ctx, r, f, dir, func(n int64) {
			e.mu.Lock()
			if total > 0 {
				e.progr = float64(base+n) / float64(total)
			}
			e.mu.Unlock()
		})
		if err != nil {
			os.RemoveAll(dir)
			return e.fail(err)
		}
		done += f.Size
	}
	e.set(StateDownloading, "verify")
	if err := m.VerifyFiles(dir); err != nil {
		os.RemoveAll(dir)
		return e.fail(fmt.Errorf("%w: %v", ErrHashMismatch, err))
	}
	e.mu.Lock()
	e.state, e.step, e.progr = StateReady, "", 1
	e.mu.Unlock()
	return nil
}

// Cancel stops a running download and clears the staging area.
func (e *Engine) Cancel() {
	e.mu.Lock()
	if e.cancel != nil {
		e.cancel()
	}
	st := e.state
	v := e.man.Version
	if st == StateReady {
		e.state = StateIdle
	}
	e.mu.Unlock()
	if v != "" && st != StateApplying {
		os.RemoveAll(filepath.Join(e.cfg.Layout.Staging(), v))
	}
}

// Apply installs the staged release and asks the process to shut down; the boot shim then runs it. confirm must
// name the staged version. Unless force is set the hero has to be in a safe spot.
func (e *Engine) Apply(confirm string, force bool) error {
	e.mu.Lock()
	if e.state != StateReady {
		e.mu.Unlock()
		return ErrNotReady
	}
	m := e.man
	if confirm != m.Version {
		e.mu.Unlock()
		return ErrConfirm
	}
	if !force && e.cfg.IdleSafe != nil && !e.cfg.IdleSafe() {
		e.mu.Unlock()
		return ErrNotSafe
	}
	e.state, e.step = StateApplying, "unpack"
	e.mu.Unlock()

	l := e.cfg.Layout
	dir := filepath.Join(l.Staging(), m.Version)
	if err := m.VerifyFiles(dir); err != nil { // the disk may have changed since the download
		return e.fail(fmt.Errorf("%w: %v", ErrHashMismatch, err))
	}
	st, err := l.LoadState()
	if err != nil {
		return e.fail(err)
	}
	if err := l.InstallRelease(m, e.remote.Raw, e.remote.Sig, dir, e.cfg.SelfTest); err != nil {
		return e.fail(err)
	}
	old := l.CurrentVersion()
	if err := l.SwapCurrent(m.Version); err != nil {
		return e.fail(err)
	}
	st.Previous, st.Current, st.Pending, st.Boots = old, m.Version, m.Version, 0
	if m.Seq > st.SeqFloor {
		st.SeqFloor = m.Seq
	}
	if CompareVersions(m.Version, st.VersionFloor) > 0 {
		st.VersionFloor = m.Version
	}
	if err := l.SaveState(st); err != nil {
		e.undo(old, m.Version, "state not saved: "+err.Error())
		return e.fail(err)
	}
	l.Prune(m.Version, old)
	os.RemoveAll(dir)
	e.set(StateApplying, "restart")
	if e.cfg.Shutdown != nil {
		if err := e.cfg.Shutdown(); err != nil {
			e.undo(old, m.Version, "shutdown failed: "+err.Error())
			return e.fail(err)
		}
	}
	return nil
}

// undo points current back at old and records why.
func (e *Engine) undo(old, from, reason string) {
	l := e.cfg.Layout
	if old != "" {
		l.SwapCurrent(old)
	} else {
		os.Remove(l.Current())
	}
	st, _ := l.LoadState()
	st.Current, st.Previous, st.Pending = old, "", ""
	st.Bad = append(st.Bad, from)
	l.SaveState(st)
	l.WriteMarker(RollbackMarker{From: from, To: old, Reason: reason, At: e.cfg.Now()})
}

// Rollback goes back to the previous release by hand (or to the baked base when there is none) and restarts.
func (e *Engine) Rollback() error {
	e.mu.Lock()
	if e.state == StateApplying || e.state == StateDownloading {
		e.mu.Unlock()
		return fmt.Errorf("busy (%s)", e.state)
	}
	e.mu.Unlock()
	l := e.cfg.Layout
	st, err := l.LoadState()
	if err != nil {
		return err
	}
	cur := l.CurrentVersion()
	if cur == "" {
		return errors.New("already on the baked base release")
	}
	e.undo(st.Previous, cur, "manual rollback")
	e.set(StateApplying, "restart")
	if e.cfg.Shutdown != nil {
		return e.cfg.Shutdown()
	}
	return nil
}

// MarkSeen hides the rollback banner.
func (e *Engine) MarkSeen() error {
	m, err := e.cfg.Layout.ReadMarker()
	if err != nil || m == nil {
		return err
	}
	m.Seen = true
	return e.cfg.Layout.WriteMarker(*m)
}

// Run checks once after startDelay and then daily with up to 2 hours of jitter, until ctx ends. It does nothing
// when checks are switched off. It never downloads.
func (e *Engine) Run(ctx context.Context, startDelay time.Duration) {
	if !e.cfg.CheckEnabled {
		return
	}
	t := time.NewTimer(startDelay)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		e.Check(ctx) // a failure is quiet: it shows as "last check failed" and the next day retries
		t.Reset(24*time.Hour + time.Duration(rand.Int63n(int64(2*time.Hour))))
	}
}
