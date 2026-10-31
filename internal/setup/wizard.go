package setup

import (
	"context"
	"d2lod-bot/internal/fleet"
	"errors"
	"fmt"
	"regexp"
	"sync"
	"time"

	"d2lod-bot/internal/store"
)

// StepStatus is the state of one wizard step.
type StepStatus string

const (
	StepPending     StepStatus = "pending"
	StepRunning     StepStatus = "running"
	StepWaitingUser StepStatus = "waiting-user" // an interactive window is open; the user must act in it
	StepDone        StepStatus = "done"
	StepFailed      StepStatus = "failed"
	StepSkipped     StepStatus = "skipped" // does not apply to the chosen mode
)

// Modes: where the game comes from. Always the user's own files, never a download by this app.
const (
	ModeStubs         = "stubs"          // the user's Blizzard downloader stubs; the stub downloads inside Wine
	ModeInstallerDirs = "installer-dirs" // folders the user already downloaded (Installer.exe + Tome)
	ModeGameDir       = "game-dir"       // an existing installed game folder, copied in
)

// Inputs are the user's choices. Paths only: there is deliberately no field for CD keys (they are typed
// into the installer window) and the HTTP decoder rejects unknown fields.
type Inputs struct {
	DataDir         string `json:"dataDir,omitempty"`
	Mode            string `json:"mode,omitempty"`
	StubD2          string `json:"stubD2,omitempty"`
	StubLoD         string `json:"stubLoD,omitempty"`
	InstallerD2Dir  string `json:"installerD2Dir,omitempty"`
	InstallerLoDDir string `json:"installerLoDDir,omitempty"`
	GameDirSrc      string `json:"gameDirSrc,omitempty"`
	PatchFile       string `json:"patchFile,omitempty"`
	Slots           int    `json:"slots,omitempty"`
	CharName        string `json:"charName,omitempty"`
	CharClass       string `json:"charClass,omitempty"`
	Profile         string `json:"profile,omitempty"` // handicap profile name of the first bot slot (fleet.Spec.Profile, bot.ProfileByName)
	Goals           string `json:"goals,omitempty"`   // goal presets of the first bot slot (fleet.Spec.Goals)
}

// StepState is the persisted progress of one step.
type StepState struct {
	Status   StepStatus `json:"status"`
	Error    string     `json:"error,omitempty"`
	Note     string     `json:"note,omitempty"` // what the user should do, or what was found
	Attempts int        `json:"attempts"`
	Started  time.Time  `json:"started,omitzero"`
	Finished time.Time  `json:"finished,omitzero"`
}

// State is the document persisted through internal/store (schema version 1).
type State struct {
	Inputs
	Steps   map[string]StepState `json:"steps"`
	Log     []string             `json:"log,omitempty"`
	Updated time.Time            `json:"updated,omitzero"`
}

// StepDef is one wizard step. Verify must be cheap-ish and side-effect free: it decides whether the step
// is already satisfied (so resuming, or a pre-installed game, skips work).
type StepDef struct {
	ID          string
	Title       string
	Interactive bool
	// Applies reports whether the step is part of the plan for these inputs (nil: always).
	Applies func(*State) bool
	// Ready checks that the inputs the step needs are present and sane; its error is shown to the user.
	Ready func(*State) error
	// Hold, when it returns text, stops the run before the step (no failure, step stays pending) so the user
	// can supply input; only honored when Wizard.HoldForInput is set (the web UI), never by the CLI.
	Hold   func(*State) string
	Verify func(ctx context.Context, s *State) error
	Do     func(ctx context.Context, s *State, r *Run) error
}

func (d StepDef) holdText(w *Wizard, s *State) string {
	if !w.HoldForInput || d.Hold == nil {
		return ""
	}
	return d.Hold(s)
}

// Run is what a step's Do gets to talk back with.
type Run struct{ w *Wizard }

// Logf records a progress line (scrubbed by Redact).
func (r *Run) Logf(f string, a ...any) { r.w.logf(f, a...) }

// WaitingUser marks the step as waiting for the user in a window; note tells them what to do.
func (r *Run) WaitingUser(note string) { r.w.setCurrent(StepWaitingUser, note) }

// ErrBusy is returned for operations that need the wizard idle.
var ErrBusy = errors.New("setup: a run is in progress")

// Wizard is the resumable state machine.
type Wizard struct {
	// Display is the X display of the installer/game windows (":99"); informational, for the UI.
	Display string
	// HoldForInput makes a run stop before steps whose Hold asks for input (the web UI sets it).
	HoldForInput bool
	// Now is the clock (tests).
	Now func() time.Time

	file  *store.File[State]
	steps []StepDef

	mu      sync.Mutex
	st      State
	cur     string // step being executed
	running bool
	cancel  context.CancelFunc
	done    chan struct{}
}

const maxLog = 200

// New loads the persisted state at path (or starts empty). A step left running by a crash goes back to
// pending, so the run resumes.
func New(path string, steps []StepDef) (*Wizard, error) {
	w := &Wizard{file: &store.File[State]{Path: path, Version: 1}, steps: steps}
	st, _, err := w.file.Load()
	if err != nil {
		return nil, err
	}
	if st.Steps == nil {
		st.Steps = map[string]StepState{}
	}
	for id, s := range st.Steps {
		if s.Status == StepRunning || s.Status == StepWaitingUser {
			s.Status, s.Note = StepPending, "interrupted; will resume"
			st.Steps[id] = s
		}
	}
	w.st = st
	return w, nil
}

func (w *Wizard) now() time.Time {
	if w.Now != nil {
		return w.Now()
	}
	return time.Now()
}

// save persists; caller holds mu.
func (w *Wizard) save() {
	w.st.Updated = w.now()
	_ = w.file.Save(w.st)
}

func (w *Wizard) logf(f string, a ...any) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.st.Log = append(w.st.Log, w.now().Format("15:04:05 ")+Redact(fmt.Sprintf(f, a...)))
	if n := len(w.st.Log); n > maxLog {
		w.st.Log = append([]string(nil), w.st.Log[n-maxLog:]...)
	}
	w.save()
}

func (w *Wizard) set(id string, f func(*StepState)) {
	s := w.st.Steps[id]
	f(&s)
	w.st.Steps[id] = s
	w.save()
}

func (w *Wizard) setCurrent(st StepStatus, note string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.cur != "" {
		w.set(w.cur, func(s *StepState) { s.Status, s.Note = st, Redact(note) })
	}
}

// StepView is one step as shown to the UI.
type StepView struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Interactive bool   `json:"interactive"`
	StepState
}

// View is the wizard document of GET /state.
type View struct {
	Inputs  Inputs     `json:"inputs"`
	Steps   []StepView `json:"steps"`
	Current string     `json:"current,omitempty"` // first step that is not done or skipped
	Running bool       `json:"running"`
	Done    bool       `json:"done"`
	Display string     `json:"display,omitempty"`
	Log     []string   `json:"log"`
	Updated time.Time  `json:"updated,omitzero"`
}

// View returns a consistent snapshot.
func (w *Wizard) View() View {
	w.mu.Lock()
	defer w.mu.Unlock()
	v := View{Inputs: w.st.Inputs, Running: w.running, Display: w.Display, Log: append([]string{}, w.st.Log...), Updated: w.st.Updated, Done: true}
	for _, d := range w.steps {
		s := w.st.Steps[d.ID]
		if s.Status == "" {
			s.Status = StepPending
		}
		if d.Applies != nil && !d.Applies(&w.st) {
			s.Status = StepSkipped
		}
		v.Steps = append(v.Steps, StepView{ID: d.ID, Title: d.Title, Interactive: d.Interactive, StepState: s})
		if s.Status != StepDone && s.Status != StepSkipped {
			v.Done = false
			if v.Current == "" {
				v.Current = d.ID
			}
		}
	}
	return v
}

// SetInputs merges the non-empty fields of in into the stored inputs. Not allowed while running.
func (w *Wizard) SetInputs(in Inputs) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.running {
		return ErrBusy
	}
	merge := func(dst *string, v string) {
		if v != "" {
			*dst = v
		}
	}
	t := &w.st.Inputs
	merge(&t.DataDir, in.DataDir)
	merge(&t.Mode, in.Mode)
	merge(&t.StubD2, in.StubD2)
	merge(&t.StubLoD, in.StubLoD)
	merge(&t.InstallerD2Dir, in.InstallerD2Dir)
	merge(&t.InstallerLoDDir, in.InstallerLoDDir)
	merge(&t.GameDirSrc, in.GameDirSrc)
	merge(&t.PatchFile, in.PatchFile)
	merge(&t.CharName, in.CharName)
	merge(&t.CharClass, in.CharClass)
	if len(in.Goals) > 200 {
		return errors.New("goals: too long")
	}
	if err := (&fleet.Spec{Character: "Wizbot", Profile: in.Profile, Goals: in.Goals}).Validate(); err != nil {
		return err
	}
	merge(&t.Profile, in.Profile)
	merge(&t.Goals, in.Goals)
	if in.Slots > 0 {
		t.Slots = in.Slots
	}
	w.save()
	return nil
}

// Inputs returns the stored inputs.
func (w *Wizard) Inputs() Inputs {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.st.Inputs
}

// Start runs the plan in the background until it finishes, fails, needs input or is cancelled.
func (w *Wizard) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return ErrBusy
	}
	ctx, w.cancel = context.WithCancel(ctx)
	w.running, w.done = true, make(chan struct{})
	w.mu.Unlock()
	go func() {
		err := w.run(ctx)
		w.mu.Lock()
		if err != nil {
			w.st.Log = append(w.st.Log, w.now().Format("15:04:05 ")+"stopped: "+Redact(err.Error()))
		}
		w.running, w.cur = false, ""
		w.save()
		close(w.done)
		w.mu.Unlock()
	}()
	return nil
}

// Wait blocks until the background run ends (tests, shutdown).
func (w *Wizard) Wait() {
	w.mu.Lock()
	d := w.done
	w.mu.Unlock()
	if d != nil {
		<-d
	}
}

// Cancel stops a running plan; the current step goes back to pending.
func (w *Wizard) Cancel() {
	w.mu.Lock()
	c := w.cancel
	w.mu.Unlock()
	if c != nil {
		c()
	}
}

// Run is Start and Wait; it returns the error that stopped the run.
func (w *Wizard) Run(ctx context.Context) error {
	if err := w.Start(ctx); err != nil {
		return err
	}
	w.Wait()
	for _, d := range w.View().Steps {
		if d.Status == StepFailed {
			return fmt.Errorf("setup: step %s failed: %s", d.ID, d.Error)
		}
	}
	return nil
}

// Reset forgets one step (or all with id ""), so it is verified and run again.
func (w *Wizard) Reset(id string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.running {
		return ErrBusy
	}
	if id == "" {
		w.st.Steps = map[string]StepState{}
	} else {
		known := false
		for _, d := range w.steps {
			known = known || d.ID == id
		}
		if !known {
			return fmt.Errorf("setup: unknown step %q", id)
		}
		delete(w.st.Steps, id)
	}
	w.save()
	return nil
}

func (w *Wizard) run(ctx context.Context) error {
	for _, d := range w.steps {
		if err := ctx.Err(); err != nil {
			return err
		}
		w.mu.Lock()
		snap := w.st
		w.mu.Unlock()
		if d.Applies != nil && !d.Applies(&snap) {
			w.mu.Lock()
			w.set(d.ID, func(s *StepState) { s.Status, s.Error = StepSkipped, "" })
			w.mu.Unlock()
			continue
		}
		if h := d.holdText(w, &snap); h != "" {
			w.mu.Lock()
			w.set(d.ID, func(s *StepState) { s.Status, s.Error, s.Note = StepPending, "", h })
			w.mu.Unlock()
			w.logf("%s: waiting for input: %s", d.ID, h)
			return nil
		}
		if d.Verify != nil && d.Verify(ctx, &snap) == nil {
			w.mu.Lock()
			w.set(d.ID, func(s *StepState) {
				if s.Status != StepDone {
					s.Finished = w.now()
				}
				s.Status, s.Error, s.Note = StepDone, "", "already satisfied"
				if s.Attempts > 0 {
					s.Note = ""
				}
			})
			w.mu.Unlock()
			continue
		}
		if d.Ready != nil {
			if err := d.Ready(&snap); err != nil {
				w.fail(d.ID, err)
				return err
			}
		}
		w.mu.Lock()
		w.cur = d.ID
		w.set(d.ID, func(s *StepState) {
			s.Status, s.Error, s.Note, s.Started, s.Finished = StepRunning, "", "", w.now(), time.Time{}
			s.Attempts++
		})
		w.mu.Unlock()
		w.logf("%s: start", d.ID)
		err := d.Do(ctx, &snap, &Run{w})
		if err == nil && d.Verify != nil {
			if verr := d.Verify(ctx, &snap); verr != nil {
				err = fmt.Errorf("verify: %w", verr)
			}
		}
		if err != nil {
			if ctx.Err() != nil { // cancelled: not a failure of the step
				w.mu.Lock()
				w.set(d.ID, func(s *StepState) { s.Status, s.Note = StepPending, "cancelled" })
				w.mu.Unlock()
				return ctx.Err()
			}
			w.fail(d.ID, err)
			return fmt.Errorf("%s: %w", d.ID, err)
		}
		w.mu.Lock()
		w.cur = ""
		w.set(d.ID, func(s *StepState) { s.Status, s.Error, s.Note, s.Finished = StepDone, "", "", w.now() })
		w.mu.Unlock()
		w.logf("%s: done", d.ID)
	}
	return nil
}

func (w *Wizard) fail(id string, err error) {
	w.mu.Lock()
	w.cur = ""
	w.set(id, func(s *StepState) {
		s.Status, s.Error, s.Finished = StepFailed, Redact(err.Error()), w.now()
		if s.Attempts == 0 {
			s.Attempts = 1
		}
	})
	w.mu.Unlock()
}

// Redact removes anything shaped like a Diablo II CD key from a log line. Keys never enter this package
// by design; this is the second belt. Over-redacting (any 16-character alphanumeric token) is fine.
func Redact(s string) string { return keyRe.ReplaceAllString(s, "[redacted]") }

var keyRe = regexp.MustCompile(`(?i)\b[a-z0-9]{4}-[a-z0-9]{6}-[a-z0-9]{4}\b|\b[a-z0-9]{16}\b|\b(?:\d{4}-){3}\d{4}\b`)
