package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"d2lod-bot/internal/access"
)

// Same rule as the Wine wizard's validCharName (internal/setup/steps.go): letters only, 2 to 15. It also keeps
// anything key-shaped (digits and dashes) out of the one free-text field.
var nameRe = regexp.MustCompile(`^[A-Za-z]{2,15}$`)

func validCharName(n string) error {
	if !nameRe.MatchString(n) {
		return fmt.Errorf("character name %q must be 2 to 15 letters", n)
	}
	return nil
}

const (
	stateFile  = "wizard.json"
	envFile    = "wizard.env"
	accessFile = "access.json"
	schema     = 1

	// warmEstimate is the page's guess for the table warm-up (U: unmeasured; testerlive measures it).
	warmEstimate = 120 * time.Second
)

// userError is a failure the page shows in plain words together with one sentence on what to do about it.
type userError struct{ Msg, Fix string }

func (e *userError) Error() string { return e.Msg }

func uerr(msg, fix string) error { return &userError{msg, fix} }

// Saved is /data/wizard.json. It holds no secret (the watch token lives in its own 0600 file) but is written
// 0600 anyway. The done phase is written last: the entrypoint waits for exactly this file with phase "done".
type Saved struct {
	Schema      int    `json:"schema"`
	Phase       string `json:"phase"` // "inputs" or "done"
	CharName    string `json:"char_name,omitempty"`
	Class       string `json:"class,omitempty"`
	ClassIndex  int    `json:"class_index"`
	Goals       string `json:"goals,omitempty"`
	PatchSHA256 string `json:"patch_sha256,omitempty"`
	DoneAt      string `json:"done_at,omitempty"`
	TokenShown  bool   `json:"token_shown,omitempty"`
	// AheadAt is the time the tester ticked the go-ahead box (RFC 3339, UTC); empty means not ticked. Start needs it.
	AheadAt   string `json:"ahead_at,omitempty"`
	StartedAt string `json:"started_at,omitempty"` // when Start began the warm-up; survives a closed page or a restart
}

// Inputs is the only thing the browser sends. Unknown fields are rejected by the decoder.
type Inputs struct {
	CharName string `json:"char_name"`
	Class    string `json:"class"`
	Goals    string `json:"goals"`
}

// Extractor warms the game table cache (internal/data.Open in production).
type Extractor func(gameDir string) error

type Wizard struct {
	Data    string // /data
	Game    string // /game
	Tables  *Tables
	Extract Extractor
	Now     func() time.Time
	Access  *access.Store // /data/access.json: the passcode hash or the token choice, read by the watch gate too

	mu      sync.Mutex
	saved   Saved
	check   *Report
	running bool
	failure string
}

func NewWizard(data, game string, t *Tables, ex Extractor) (*Wizard, error) {
	w := &Wizard{Data: data, Game: game, Tables: t, Extract: ex, Now: time.Now, Access: access.NewStore(filepath.Join(data, accessFile))}
	b, err := os.ReadFile(filepath.Join(data, stateFile))
	switch {
	case err == nil:
		dec := json.NewDecoder(strings.NewReader(string(b)))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&w.saved); err != nil {
			return nil, fmt.Errorf("%s is damaged (%w): delete it to start the wizard again", stateFile, err)
		}
	case errors.Is(err, os.ErrNotExist):
	default:
		return nil, err
	}
	if w.saved.Schema == 0 {
		w.saved.Schema = schema
	}
	if w.saved.Phase != "done" && w.saved.Class != "" {
		if _, err := t.Class(w.saved.Class); err != nil {
			return nil, fmt.Errorf("%s: %w (edit or delete it)", stateFile, err)
		}
	}
	return w, nil
}

// Done reports whether the wizard finished in an earlier run (the entrypoint hook skips the page then).
func (w *Wizard) Done() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.saved.Phase == "done"
}

// View is the state the page renders. The phase is derived from /data and the check, never kept in the browser:
// ahead (go-ahead not ticked), data (game files), character, access, start, running, failed or done. The page may
// show an earlier step than the phase (Back), never a later one.
type View struct {
	Phase       string    `json:"phase"`
	Check       *Report   `json:"check"`
	Inputs      Inputs    `json:"inputs"`
	Error       string    `json:"error,omitempty"`
	Fix         string    `json:"fix,omitempty"` // one sentence on what to do about Error
	Classes     []Class   `json:"classes"`
	Presets     []Preset  `json:"presets"`
	Ahead       Ahead     `json:"ahead"`
	Access      string    `json:"access"` // "", "passcode" or "token" (the hash never leaves the server)
	Progress    *Progress `json:"progress,omitempty"`
	Interrupted bool      `json:"interrupted,omitempty"` // a start began earlier and was cut off (restart); press Start again
	Token       string    `json:"token,omitempty"`       // only in the first done view, and only for the token kind
	Saved       *Saved    `json:"-"`
}

type Ahead struct {
	OK bool   `json:"ok"`
	At string `json:"at,omitempty"`
}

// Progress is the warm-up as the start screen shows it. The step list is honest: only the first step is something
// this process does; the other two happen when the bot takes over, so they read "next".
type Progress struct {
	StartedAt   string      `json:"started_at"`
	ElapsedSec  int         `json:"elapsed_sec"`
	EstimateSec int         `json:"estimate_sec"`
	Percent     int         `json:"percent"` // capped at 95 until the job really finished
	Steps       []StepState `json:"steps"`
}

type StepState struct {
	Name  string `json:"name"`
	State string `json:"state"` // done, doing, next
}

func (w *Wizard) inputsLocked() Inputs {
	in := Inputs{CharName: w.saved.CharName, Class: w.saved.Class, Goals: w.saved.Goals}
	if in.Class == "" {
		in.Class = w.Tables.DefaultClass
	}
	if in.Goals == "" {
		in.Goals = w.Tables.DefaultPreset
	}
	return in
}

// Check (re)runs the data check; a failed one can be run again after the user fixes the mount.
func (w *Wizard) Check() Report {
	r := CheckGame(w.Game, w.Tables)
	w.mu.Lock()
	w.check = &r
	w.mu.Unlock()
	return r
}

// State builds the view. token is handed out once, in the first view after the wizard is done.
func (w *Wizard) State(token string) View {
	w.mu.Lock()
	need := w.check == nil
	w.mu.Unlock()
	if need && !w.Done() {
		w.Check()
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	v := View{Check: w.check, Inputs: w.inputsLocked(), Classes: w.Tables.Classes, Presets: w.Tables.Presets, Error: w.failure,
		Ahead: Ahead{OK: w.saved.AheadAt != "", At: w.saved.AheadAt}, Access: w.Access.Current().Kind}
	if w.failure != "" {
		v.Fix = "Check that the game folder is mounted and not being changed, then press Retry. If it fails again, use Report a problem."
	}
	switch {
	case w.saved.Phase == "done":
		v.Phase = "done"
		if !w.saved.TokenShown && v.Access == access.KindToken {
			v.Token = token
			w.saved.TokenShown = true
			_ = w.saveLocked()
		}
	case w.running:
		v.Phase = "running"
	case w.failure != "":
		v.Phase = "failed"
	case w.saved.AheadAt == "":
		v.Phase = "ahead"
	case w.check == nil || !w.check.OK:
		v.Phase = "data"
	case w.saved.CharName == "":
		v.Phase = "character"
	case v.Access == "":
		v.Phase = "access"
	default:
		v.Phase = "start"
		v.Interrupted = w.saved.StartedAt != ""
	}
	if w.running || w.failure != "" {
		v.Progress = w.progressLocked()
	}
	return v
}

func (w *Wizard) progressLocked() *Progress {
	p := &Progress{StartedAt: w.saved.StartedAt, EstimateSec: int(warmEstimate / time.Second), Steps: []StepState{}}
	t, err := time.Parse(time.RFC3339, w.saved.StartedAt)
	if err != nil {
		return p
	}
	el := w.Now().Sub(t)
	if el < 0 {
		el = 0
	}
	p.ElapsedSec = int(el / time.Second)
	p.Percent = min(95, int(100*el/warmEstimate))
	first := "doing"
	if w.failure != "" {
		first = "next"
	}
	p.Steps = []StepState{{"Reading the game data tables", first}, {"Starting the game engine", "next"}, {"Creating " + w.saved.CharName, "next"}}
	return p
}

// SetAhead stores or clears the go-ahead tick (with the time). Start refuses to run without it.
func (w *Wizard) SetAhead(ok bool) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.running || w.saved.Phase == "done" {
		return errBusy
	}
	switch {
	case !ok:
		w.saved.AheadAt = ""
	case w.saved.AheadAt == "":
		w.saved.AheadAt = w.Now().UTC().Format(time.RFC3339)
	}
	return w.saveLocked()
}

// AccessChoice is what the access screen sends. Passcode and Confirm are only used for the passcode kind.
type AccessChoice struct {
	Kind     string `json:"kind"`
	Passcode string `json:"passcode"`
	Confirm  string `json:"confirm"`
}

// SetAccess stores the access choice in access.json (0600). The passcode is hashed here and then dropped.
func (w *Wizard) SetAccess(c AccessChoice) (access.Record, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.running || w.saved.Phase == "done" {
		return access.Record{}, errBusy
	}
	var rec access.Record
	switch c.Kind {
	case access.KindPasscode:
		if err := access.ValidatePasscode(c.Passcode); err != nil {
			return rec, uerr(err.Error(), "Pick a longer passcode. There are no other rules; four random words work well.")
		}
		if c.Passcode != c.Confirm {
			return rec, uerr("The two passcodes are not the same.", "Type the same passcode in both boxes.")
		}
		var err error
		if rec, err = access.NewPasscode(c.Passcode, w.Now()); err != nil {
			return rec, err
		}
	case access.KindToken:
		rec = access.TokenRecord(w.Now())
	default:
		return rec, uerr("Choose a passcode or the generated token.", "Pick one of the two options.")
	}
	return rec, w.Access.Save(rec)
}

// SetInputs validates and stores the choices. It is refused while a run is active or after done.
func (w *Wizard) SetInputs(in Inputs) error {
	if err := validCharName(in.CharName); err != nil {
		return err
	}
	c, err := w.Tables.Class(in.Class)
	if err != nil {
		return err
	}
	if _, err := w.Tables.Preset(in.Goals); err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.running || w.saved.Phase == "done" {
		return errBusy
	}
	w.saved.Phase, w.saved.CharName, w.saved.Class, w.saved.ClassIndex, w.saved.Goals = "inputs", in.CharName, c.ID, c.D2DIndex, in.Goals
	w.failure = ""
	return w.saveLocked()
}

var errBusy = errors.New("setup is already running or finished")

// Start re-checks the data, then extracts the game tables in the background; on success it writes wizard.env
// and, last, wizard.json with phase done.
func (w *Wizard) Start() error {
	r := w.Check()
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.running || w.saved.Phase == "done" {
		return errBusy
	}
	switch {
	case w.saved.AheadAt == "":
		return uerr("Tick the go-ahead box first.", "Go back to the go-ahead step, read it and tick the box.")
	case !r.OK:
		return uerr("The game files are not ready: "+r.Summary, r.Fix)
	case w.saved.CharName == "":
		return uerr("Choose a hero name first.", "Go back to the hero step.")
	case w.Access.Current().Kind == "":
		return uerr("Choose how the watch screen is protected first.", "Go back to the access step.")
	}
	w.running, w.failure = true, ""
	w.saved.PatchSHA256 = r.PatchSHA256
	w.saved.StartedAt = w.Now().UTC().Format(time.RFC3339)
	_ = w.saveLocked() // so a closed page or a restart still knows when the warm-up began
	go w.run()
	return nil
}

func (w *Wizard) run() {
	err := w.Extract(w.Game)
	w.mu.Lock()
	defer w.mu.Unlock()
	w.running = false
	if err == nil {
		err = w.finishLocked()
	}
	if err != nil {
		w.failure = "Reading the game tables failed (" + err.Error() + ")."
	}
}

func (w *Wizard) finishLocked() error {
	s := w.saved
	// values are validated (letters, digits, "campaign:..." characters) before they reach a shell file
	env := fmt.Sprintf("D2D_NAME='%s'\nD2D_CLASS='%d'\nD2BOT_GOALS='%s'\n", s.CharName, s.ClassIndex, s.Goals)
	if err := writeAtomic(filepath.Join(w.Data, envFile), []byte(env)); err != nil {
		return err
	}
	w.saved.Phase, w.saved.DoneAt = "done", w.Now().UTC().Format(time.RFC3339)
	return w.saveLocked()
}

func (w *Wizard) saveLocked() error {
	b, _ := json.MarshalIndent(w.saved, "", "  ")
	return writeAtomic(filepath.Join(w.Data, stateFile), append(b, '\n'))
}

func writeAtomic(path string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
