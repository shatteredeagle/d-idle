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
	stateFile = "wizard.json"
	envFile   = "wizard.env"
	schema    = 1
)

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

	mu      sync.Mutex
	saved   Saved
	check   *Report
	running bool
	failure string
}

func NewWizard(data, game string, t *Tables, ex Extractor) (*Wizard, error) {
	w := &Wizard{Data: data, Game: game, Tables: t, Extract: ex, Now: time.Now}
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
	return w, nil
}

// Done reports whether the wizard finished in an earlier run (the entrypoint hook skips the page then).
func (w *Wizard) Done() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.saved.Phase == "done"
}

// View is the state the page renders. The phase is derived from /data and the check, never kept in the browser.
type View struct {
	Phase   string   `json:"phase"` // data, character, start, running, failed, done
	Check   *Report  `json:"check"`
	Inputs  Inputs   `json:"inputs"`
	Error   string   `json:"error,omitempty"`
	Classes []Class  `json:"classes"`
	Presets []Preset `json:"presets"`
	Token   string   `json:"token,omitempty"` // only in the first done view
	Saved   *Saved   `json:"-"`
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
	v := View{Check: w.check, Inputs: w.inputsLocked(), Classes: w.Tables.Classes, Presets: w.Tables.Presets, Error: w.failure}
	switch {
	case w.saved.Phase == "done":
		v.Phase = "done"
		if !w.saved.TokenShown {
			v.Token = token
			w.saved.TokenShown = true
			_ = w.saveLocked()
		}
	case w.running:
		v.Phase = "running"
	case w.failure != "":
		v.Phase = "failed"
	case w.check == nil || !w.check.OK:
		v.Phase = "data"
	case w.saved.CharName == "":
		v.Phase = "character"
	default:
		v.Phase = "start"
	}
	return v
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
	if !c.Supported {
		return fmt.Errorf("%s is not supported in this build", c.Name)
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
	if !r.OK {
		return errors.New("the game files are not ready: " + r.Summary)
	}
	if w.saved.CharName == "" {
		return errors.New("choose a character name first")
	}
	w.running, w.failure = true, ""
	w.saved.PatchSHA256 = r.PatchSHA256
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
		w.failure = "Reading the game tables failed: " + err.Error()
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
