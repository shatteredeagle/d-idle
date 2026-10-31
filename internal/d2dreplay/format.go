// Package d2dreplay records and plays back runs of the lockstep d2d fork: a replay file is the seed, the start
// recipe, the ordered control verbs with their tick numbers and state-hash checkpoints, nothing else (no game
// data, no paths, no hosts). Contract: openspec/changes/replay/specs/replay/spec.md; evidence: docs/d2d-replay.md.
package d2dreplay

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode"
)

// Version is the format version written in the header ("replay"). A reader refuses a different major.
const Version = 1

// HashV1 is the checkpoint hash algorithm: the detloop journal checkpoint (sha256 over the replies to
// CheckpointVerbs, see Checkpoint). The name travels in the header so a different implementation can refuse.
const HashV1 = "detloop-journal-v1"

// TickMS is the virtual time of one lockstep tick (docs/d2d-fork.md section 3: each stepped frame adds 40 ms, V).
const TickMS = 40

// Header is the first line of a replay file.
type Header struct {
	Replay  int      `json:"replay"`
	Created string   `json:"created"` // RFC 3339 UTC
	Hash    string   `json:"hash"`    // checkpoint algorithm, HashV1
	Every   int      `json:"every"`   // checkpoint interval in ticks
	D2D     Build    `json:"d2d"`
	Start   *Recipe  `json:"start,omitempty"` // fresh character
	Save    *SaveRef `json:"save,omitempty"`  // or: a saved character
	Meta    Meta     `json:"meta"`
}

// Build identifies the d2d binary and its non-path arguments.
type Build struct {
	SHA256 string   `json:"sha256,omitempty"`
	Args   []string `json:"args,omitempty"` // e.g. --lockstep --toggle autoloot=off,autoparty=off --no-save
}

// Recipe creates the throwaway character (d2d --start-* flags, docs/d2d-spike.md section 3).
type Recipe struct {
	Seed     int64  `json:"seed"`
	Class    int    `json:"class"` // d2d UI order, 1 = Necromancer
	Name     string `json:"name"`
	Screen   string `json:"screen"` // "ingame"
	Hardcore bool   `json:"hardcore,omitempty"`
	Classic  bool   `json:"classic,omitempty"`
}

// SaveRef names a start save by content only.
type SaveRef struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
	Seed   int64  `json:"seed"`
}

// Meta is free text about the run; nothing in it is needed for playback.
type Meta struct {
	Bot   string `json:"bot,omitempty"`
	Area  string `json:"area,omitempty"`
	Class string `json:"class,omitempty"`
	Note  string `json:"note,omitempty"`
	Tool  string `json:"tool,omitempty"`
	Fresh bool   `json:"fresh,omitempty"` // recipe-created character (vs. a save)
}

// Entry is one body line. Exactly one of Verb, Checkpoint, Restart or End is set (Kind says which).
type Entry struct {
	// verb: tick before the verb, the line, wall ms since the previous entry, the "err" reply if any
	T    int    `json:"t,omitempty"`
	Verb string `json:"v,omitempty"`
	W    int    `json:"w,omitempty"`
	Err  string `json:"e,omitempty"`
	// checkpoint
	C    int    `json:"c,omitempty"`
	H    string `json:"h,omitempty"`
	Lv   int    `json:"lv,omitempty"`
	Area int    `json:"area,omitempty"`
	Life string `json:"life,omitempty"`
	// restart marker (upstream d2d replaced) at tick T
	R bool `json:"r,omitempty"`
	// trailer
	End    bool `json:"end,omitempty"`
	Ticks  int  `json:"ticks,omitempty"`
	Verbs  int  `json:"verbs,omitempty"`
	WallMS int  `json:"wall_ms,omitempty"`
}

// Kind of an entry: "verb", "checkpoint", "restart", "end" or "" (empty line).
func (e Entry) Kind() string {
	switch {
	case e.End:
		return "end"
	case e.R:
		return "restart"
	case e.H != "":
		return "checkpoint"
	case e.Verb != "":
		return "verb"
	}
	return ""
}

// Writer streams a replay file: gzip of JSON lines, flushed after every checkpoint so a killed recorder leaves a
// readable prefix.
type Writer struct {
	f   *os.File
	gz  *gzip.Writer
	bw  *bufio.Writer
	n   int
	err error
}

// Create writes the header to path (gzip JSON lines).
func Create(path string, h Header) (*Writer, error) {
	if h.Replay == 0 {
		h.Replay = Version
	}
	if h.Hash == "" {
		h.Hash = HashV1
	}
	if h.Created == "" {
		h.Created = time.Now().UTC().Format(time.RFC3339)
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	w := &Writer{f: f, gz: gzip.NewWriter(f)}
	w.bw = bufio.NewWriter(w.gz)
	if err := w.write(h); err != nil {
		f.Close()
		return nil, err
	}
	return w, nil
}

func (w *Writer) write(v any) error {
	if w.err != nil {
		return w.err
	}
	b, err := json.Marshal(v)
	if err == nil {
		_, err = w.bw.Write(append(b, '\n'))
	}
	if err != nil {
		w.err = err
	}
	w.n++
	return err
}

// Add appends one entry.
func (w *Writer) Add(e Entry) error { return w.write(e) }

// Flush pushes buffered lines through gzip to the file.
func (w *Writer) Flush() error {
	if err := w.bw.Flush(); err != nil {
		return err
	}
	return w.gz.Flush()
}

// Close writes the trailer (if given) and closes the file.
func (w *Writer) Close(end *Entry) error {
	if end != nil {
		end.End = true
		w.write(*end)
	}
	err := w.bw.Flush()
	if e := w.gz.Close(); err == nil {
		err = e
	}
	if e := w.f.Close(); err == nil {
		err = e
	}
	if w.err != nil {
		return w.err
	}
	return err
}

// Replay is a whole file in memory.
type Replay struct {
	Header  Header
	Entries []Entry
	End     *Entry // nil when the file has no trailer (recorder killed)
	// Truncated says the gzip stream ended early (the tail was lost); Entries holds the readable prefix.
	Truncated bool
}

// ErrVersion is returned for a header of another major version.
var ErrVersion = errors.New("d2dreplay: unsupported replay version")

// Read loads a replay file. A truncated gzip tail is tolerated (Truncated set) so a crashed recording is still usable.
func Read(path string) (*Replay, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ReadFrom(f)
}

// Limits of the reader: a replay file is untrusted input (it may come from anyone).
var (
	maxDecompressed int64 = 256 << 20 // gzip bomb guard
	maxEntries            = 2_000_000
)

const (
	maxLine     = 1 << 20
	maxStep     = 1_000_000     // ticks of one "step n"
	maxTick     = 1_000_000_000 // upper bound of any tick field
	maxVerbLen  = 4096
	maxEveryVal = 10_000_000
)

// errTooLarge is returned when the decompressed stream passes maxDecompressed.
var errTooLarge = errors.New("d2dreplay: decompressed replay too large")

type limitedReader struct {
	r io.Reader
	n int64
}

func (l *limitedReader) Read(p []byte) (int, error) {
	if l.n <= 0 {
		return 0, errTooLarge
	}
	if int64(len(p)) > l.n {
		p = p[:l.n]
	}
	n, err := l.r.Read(p)
	l.n -= int64(n)
	return n, err
}

func isHex(s string, n int) bool {
	if len(s) != n {
		return false
	}
	for _, c := range s {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

// playbackVerbs are the first tokens a replay may contain: the mutating control verbs the tap records (U: the
// list follows internal/d2dctl/fake and the recorded fixtures; a verb of a newer fork is refused until added).
// "quit" is recorded but never replayed (play skips it).
var playbackVerbs = map[string]bool{
	"step": true, "click": true, "rclick": true, "key": true, "text": true, "move": true, "wheel": true,
	"cmd": true, "debug": true, "save": true, "lockstep": true, "pad": true, "quit": true,
}

// CleanLine refuses a control line that could be read as more than one command: any control character (newline,
// NUL, tab, ...), a Unicode line separator, invalid UTF-8.
func CleanLine(line string) error {
	for _, r := range line {
		if unicode.IsControl(r) || r == '\u2028' || r == '\u2029' || r == unicode.ReplacementChar {
			return fmt.Errorf("verb %q has a control character", line)
		}
	}
	return nil
}

// ValidVerb checks one recorded control line before it is stored or sent to d2d (which has no auth): one line, no
// control characters, a known mutating verb, no path-like tokens, a sane step count.
func ValidVerb(line string) error {
	if len(line) > maxVerbLen {
		return errors.New("verb too long")
	}
	if err := CleanLine(line); err != nil {
		return err
	}
	f := strings.Fields(line)
	if len(f) == 0 {
		return errors.New("blank verb")
	}
	if !playbackVerbs[f[0]] {
		return fmt.Errorf("verb %q is not a replayable control verb", f[0])
	}
	for _, x := range f[1:] {
		if strings.ContainsAny(x, "/\\") || strings.Contains(x, "..") || strings.HasPrefix(x, "~") {
			return fmt.Errorf("verb %q has a path-like token", line)
		}
	}
	if f[0] == "step" {
		if _, ok := StepCount(line); !ok {
			return fmt.Errorf("bad step count in %q (1..%d ticks)", line, maxStep)
		}
	}
	return nil
}

func (e Entry) validate() error {
	if e.T < 0 || e.T > maxTick || e.C < 0 || e.C > maxTick || e.W < 0 || e.W > maxTick {
		return errors.New("tick or wall field out of range")
	}
	switch e.Kind() {
	case "verb":
		return ValidVerb(e.Verb)
	case "checkpoint":
		if !isHex(e.H, 64) {
			return errors.New("checkpoint hash is not 64 hex digits")
		}
	}
	return nil
}

func (h Header) validate() error {
	if h.Every < 0 || h.Every > maxEveryVal {
		return fmt.Errorf("checkpoint interval %d out of range", h.Every)
	}
	if h.D2D.SHA256 != "" && !isHex(h.D2D.SHA256, 64) {
		return errors.New("d2d.sha256 is not 64 hex digits")
	}
	if h.Save != nil && h.Save.SHA256 != "" && !isHex(h.Save.SHA256, 64) {
		return errors.New("save.sha256 is not 64 hex digits")
	}
	return nil
}

// ReadFrom parses a gzip JSON-lines replay stream and validates every field (see ValidVerb, the limits above). A
// stream cut inside its last line is a truncated recording; any other bad line is an error.
func ReadFrom(r io.Reader) (*Replay, error) {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("d2dreplay: not a replay file: %w", err)
	}
	sc := bufio.NewScanner(&limitedReader{gzr, maxDecompressed})
	sc.Buffer(make([]byte, 1<<16), maxLine)
	if !sc.Scan() {
		if err := sc.Err(); err != nil {
			return nil, fmt.Errorf("d2dreplay: %w", err)
		}
		return nil, errors.New("d2dreplay: empty file")
	}
	rp := &Replay{}
	if err := json.Unmarshal(sc.Bytes(), &rp.Header); err != nil {
		return nil, fmt.Errorf("d2dreplay: bad header: %w", err)
	}
	if rp.Header.Replay != Version {
		return nil, fmt.Errorf("%w: %d (this tool reads %d)", ErrVersion, rp.Header.Replay, Version)
	}
	if err := rp.Header.validate(); err != nil {
		return nil, fmt.Errorf("d2dreplay: bad header: %w", err)
	}
	lineNo := 1
	var bad error // a bad line is fatal unless the stream ends right after it with a read error (cut tail)
	for sc.Scan() {
		lineNo++
		if bad != nil {
			return nil, bad
		}
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var e Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			bad = fmt.Errorf("d2dreplay: line %d: %w", lineNo, err)
			continue
		}
		if err := e.validate(); err != nil {
			return nil, fmt.Errorf("d2dreplay: line %d: %w", lineNo, err)
		}
		if e.End {
			end := e
			rp.End = &end
			bad = nil
			break
		}
		if len(rp.Entries) >= maxEntries {
			return nil, fmt.Errorf("d2dreplay: more than %d entries", maxEntries)
		}
		rp.Entries = append(rp.Entries, e)
	}
	if err := sc.Err(); err != nil {
		if errors.Is(err, errTooLarge) || errors.Is(err, bufio.ErrTooLong) {
			return nil, fmt.Errorf("d2dreplay: %w", err)
		}
		rp.Truncated = true // the gzip stream ended early
	} else if bad != nil {
		return nil, bad
	}
	if rp.End == nil {
		rp.Truncated = true
	}
	return rp, nil
}

// Checkpoints returns the checkpoint entries in order.
func (r *Replay) Checkpoints() []Entry {
	var out []Entry
	for _, e := range r.Entries {
		if e.Kind() == "checkpoint" {
			out = append(out, e)
		}
	}
	return out
}

// Ticks is the virtual tick count of the body (sum of the step counts).
func (r *Replay) Ticks() int {
	t := 0
	for _, e := range r.Entries {
		if n, ok := StepCount(e.Verb); ok {
			t += n
		}
	}
	return t
}

// StepCount returns n for a "step n [render]" line.
func StepCount(line string) (int, bool) {
	f := strings.Fields(line)
	if len(f) < 2 || f[0] != "step" {
		return 0, false
	}
	var n int
	if _, err := fmt.Sscan(f[1], &n); err != nil || n < 0 || n > maxStep {
		return 0, false
	}
	return n, true
}

// readDebug lists the "debug" sub-verbs that only read (docs/d2dctl.md, docs/d2d-view.md readOnly).
var readDebug = map[string]bool{
	"players": true, "level": true, "warps": true, "portals": true, "presets": true, "objects": true,
	"corpses": true, "layout": true, "nearby": true, "grid": true, "blocked": true, "attack": true,
	"passives": true, "points": true, "roster": true, "net": true, "tradestate": true, "itemat": true,
}

// Mutating says whether a control line can change the game and must be replayed. Read verbs are forwarded but
// not stored; "screenshot" is a read (it only draws) and would carry a path.
func Mutating(line string) bool {
	f := strings.Fields(line)
	if len(f) == 0 {
		return false
	}
	switch f[0] {
	case "state", "monsters", "npcs", "ground", "menu", "items", "info", "ping", "clock", "dump", "screenshot":
		return false
	case "debug":
		if len(f) < 2 {
			return false
		}
		if readDebug[f[1]] {
			return false
		}
		if f[1] == "quest" && len(f) >= 4 && f[3] == "show" {
			return false
		}
		return true
	}
	return true
}
