package watch

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// The layout manifest (docs/ui-design-system.md section 6.4, migration PR 14): watch.layout/1 says where each panel
// sits per breakpoint. web/layout.json is the default (the approved mockup layout); GET /api/layout serves the layout
// in force, POST /api/layout (answer listener, the same gate as the policy sheet) stores a per-profile copy that wins
// over the default, or resets to it. A file the mockup editor (docs/mockups/ui/editor.html) exports is accepted
// unchanged: the shell maps the editor's ids onto the real page (the "layout" block of web/shell.json).

const (
	LayoutRoute   = "/api/layout"
	layoutSchema  = "watch.layout/1"
	layoutMaxBody = 256 << 10 // a layout file is a few tens of KB; the answer route's 4 KiB would refuse it
)

// LayoutManifest is a watch.layout/1 file: the grid positions the layout editor exports.
type LayoutManifest struct {
	Schema      string                      `json:"schema"`
	Cols        int                         `json:"cols"`
	RowHeight   int                         `json:"rowHeight"`
	Density     string                      `json:"density,omitempty"`
	Theme       string                      `json:"theme,omitempty"`
	Breakpoints map[string]LayoutBreakpoint `json:"breakpoints"`
}

// LayoutBreakpoint is one breakpoint of a layout manifest.
type LayoutBreakpoint struct {
	Width  int           `json:"width"`
	Cols   int           `json:"cols"`
	Panels []LayoutPanel `json:"panels"`
}

// LayoutPanel is one placed panel.
type LayoutPanel struct {
	ID        string `json:"id"`
	Band      string `json:"band"`
	Col       int    `json:"col"`
	Row       int    `json:"row"`
	ColSpan   int    `json:"colSpan"`
	RowSpan   int    `json:"rowSpan"`
	Collapsed bool   `json:"collapsed"`
	Order     int    `json:"order"`
}

// layoutBreakpoints are the names the shell knows.
var layoutBreakpoints = []string{"phone", "tablet", "laptop", "wide", "overlay"}

// ParseLayout reads a layout manifest. Unknown keys are ignored (compatibility promise, section 0); a schema of a
// later major version and a panel outside the grid are errors.
func ParseLayout(b []byte) (LayoutManifest, error) {
	var l LayoutManifest
	if err := json.Unmarshal(b, &l); err != nil {
		return l, err
	}
	if l.Schema != layoutSchema {
		return l, fmt.Errorf("layout schema %q (this version reads %s)", l.Schema, layoutSchema)
	}
	for name, bp := range l.Breakpoints {
		cols := bp.Cols
		if cols == 0 {
			cols = l.Cols
		}
		for _, p := range bp.Panels {
			if p.ID == "" || strings.ContainsAny(p.ID, " /") || p.Col < 1 || p.ColSpan < 1 || p.Col+p.ColSpan-1 > cols || p.Row < 1 {
				return l, fmt.Errorf("layout %s: panel %q at col %d span %d row %d does not fit %d columns", name, p.ID, p.Col, p.ColSpan, p.Row, cols)
			}
		}
	}
	return l, nil
}

// Check is the strict check on top of ParseLayout (tools/uilint and the tests of the shipped files use it): known breakpoint names, no panel
// twice, no overlap within a breakpoint, unique order, and known panel ids when known is not nil.
func (l LayoutManifest) Check(known func(id string) bool) error {
	names := make([]string, 0, len(l.Breakpoints))
	for n := range l.Breakpoints {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		if !containsStr(layoutBreakpoints, name) {
			return fmt.Errorf("layout: unknown breakpoint %q", name)
		}
		seen, orders, cells := map[string]bool{}, map[int]string{}, map[[2]int]string{}
		for _, p := range l.Breakpoints[name].Panels {
			if known != nil && !known(p.ID) {
				return fmt.Errorf("layout %s: unknown panel id %q", name, p.ID)
			}
			if seen[p.ID] {
				return fmt.Errorf("layout %s: panel %q placed twice", name, p.ID)
			}
			seen[p.ID] = true
			if o, dup := orders[p.Order]; dup {
				return fmt.Errorf("layout %s: order %d used by %s and %s", name, p.Order, o, p.ID)
			}
			orders[p.Order] = p.ID
			rows := p.RowSpan
			if p.Collapsed || rows < 1 {
				rows = 1
			}
			for r := p.Row; r < p.Row+rows; r++ {
				for c := p.Col; c < p.Col+p.ColSpan; c++ {
					if o, hit := cells[[2]int{r, c}]; hit {
						return fmt.Errorf("layout %s: %s overlaps %s at row %d col %d", name, p.ID, o, r, c)
					}
					cells[[2]int{r, c}] = p.ID
				}
			}
		}
	}
	return nil
}

// LayoutConf is the "layout" block of web/shell.json: how the ids of a layout file map onto the real page. Aliases
// are editor ids that became another panel (the editor's "hero-status" card is the real "hero-vitals"); Compat are
// the pre-shell cards of watch.html (their data-lid); Reserved are ids a layout may name before the panel exists
// (the shell skips them).
type LayoutConf struct {
	Aliases  map[string]string `json:"aliases"`
	Compat   []string          `json:"compat"`
	Reserved []string          `json:"reserved"`
}

var shellLayoutConf = sync.OnceValues(func() (LayoutConf, error) {
	var c struct {
		Layout LayoutConf `json:"layout"`
	}
	b, err := webFS.ReadFile("web/shell.json")
	if err != nil {
		return c.Layout, err
	}
	err = json.Unmarshal(b, &c)
	return c.Layout, err
})

// LayoutKnownIDs returns the test for a layout entry id that resolves on the page: a panel manifest, an alias, a
// compat card or a reserved id.
func LayoutKnownIDs() (func(string) bool, error) {
	ms, err := Manifests()
	if err != nil {
		return nil, err
	}
	lc, err := shellLayoutConf()
	if err != nil {
		return nil, err
	}
	ids := map[string]bool{}
	for _, m := range ms {
		ids[m.ID] = true
	}
	for a := range lc.Aliases {
		ids[a] = true
	}
	for _, id := range lc.Compat {
		ids[id] = true
	}
	for _, id := range lc.Reserved {
		ids[id] = true
	}
	return func(id string) bool { return ids[id] }, nil
}

// layoutStore holds the layout in force: the stored per-profile copy when there is one, else web/layout.json. It has
// its own mutex and never touches the Hub's state.
type layoutStore struct {
	mu   sync.Mutex
	path string
	cur  []byte // the JSON GET serves; nil until first read
}

// SetLayoutPath names the per-profile file POST /api/layout writes (cmd/d2bot: <state dir>/layout-<profile>.json).
// Without a path a posted layout lives for the process only. A stored file that no longer parses is ignored, not
// deleted: the default is served until the next save.
func (h *Hub) SetLayoutPath(p string) {
	h.layout.mu.Lock()
	h.layout.path, h.layout.cur = p, nil
	h.layout.mu.Unlock()
}

// Layout returns the layout in force as JSON.
func (h *Hub) Layout() ([]byte, error) {
	h.layout.mu.Lock()
	defer h.layout.mu.Unlock()
	return h.layout.load()
}

func (s *layoutStore) load() ([]byte, error) {
	if s.cur != nil {
		return s.cur, nil
	}
	if s.path != "" {
		if b, err := os.ReadFile(s.path); err == nil {
			if _, err := ParseLayout(b); err == nil {
				s.cur = b
				return b, nil
			}
		}
	}
	b, err := webFS.ReadFile("web/layout.json")
	if err != nil {
		return nil, err
	}
	s.cur = b
	return b, nil
}

func (h *Hub) layoutHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := h.Layout()
		if err != nil {
			writeJSON(w, http.StatusServiceUnavailable, policyReply{Code: "off", Error: "no layout"})
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(b)
	}
}

type layoutBody struct {
	Layout json.RawMessage `json:"layout"`
	Reset  bool            `json:"reset"`
}

// postLayout serves POST /api/layout {"layout": <watch.layout/1>} or {"reset": true}. A layout naming a panel this
// version lacks is still stored (the shell skips the id), so a file from a later version survives a round trip.
func (h *Hub) postLayout(o AnswerOptions, w http.ResponseWriter, r *http.Request) {
	var b layoutBody
	if !layoutGate(o, w, r, &b) {
		return
	}
	fail := func(code int, c, msg string) { writeJSON(w, code, policyReply{Code: c, Error: msg}) }
	s := &h.layout
	s.mu.Lock()
	defer s.mu.Unlock()
	switch {
	case b.Reset:
		if s.path != "" {
			if err := os.Remove(s.path); err != nil && !errors.Is(err, os.ErrNotExist) {
				fail(http.StatusInternalServerError, "internal", "could not remove the stored layout")
				return
			}
		}
		s.cur = nil
	case len(b.Layout) > 0:
		// the loader's rule only: the editor's own exports can overlap, and the shell pushes an overlap down
		if _, err := ParseLayout(b.Layout); err != nil {
			fail(http.StatusBadRequest, "layout", safeText(err.Error(), 200))
			return
		}
		if s.path != "" {
			err := os.MkdirAll(filepath.Dir(s.path), 0o755)
			if err == nil {
				err = os.WriteFile(s.path, b.Layout, 0o644)
			}
			if err != nil {
				fail(http.StatusInternalServerError, "internal", "could not save the layout")
				return
			}
		}
		s.cur = b.Layout
	default:
		fail(http.StatusBadRequest, "body", `send {"layout": {...}} or {"reset": true}`)
		return
	}
	cur, err := s.load()
	if err != nil {
		fail(http.StatusInternalServerError, "internal", "no layout")
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write([]byte(`{"ok":true,"layout":`))
	w.Write(cur)
	w.Write([]byte(`}`))
}

// layoutGate is policyGate (policysheet.go) with a body limit that fits a layout file: origin list, POST only, token,
// JSON only, known fields.
func layoutGate(o AnswerOptions, w http.ResponseWriter, r *http.Request, dst any) bool {
	origin := r.Header.Get("Origin")
	allowed := origin != "" && containsStr(o.Origins, origin)
	if allowed {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Vary", "Origin")
	}
	fail := func(code int, c, msg string) { writeJSON(w, code, policyReply{Code: c, Error: msg}) }
	switch {
	case r.Method == http.MethodOptions && allowed:
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusNoContent)
		return false
	case r.Method != http.MethodPost:
		w.Header().Set("Allow", "POST, OPTIONS")
		fail(http.StatusMethodNotAllowed, "method", "POST only")
		return false
	case origin != "" && !allowed:
		fail(http.StatusForbidden, "origin", "origin not allowed")
		return false
	case !o.tokenOK(r):
		fail(http.StatusUnauthorized, "token", "unauthorized")
		return false
	}
	if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
		fail(http.StatusUnsupportedMediaType, "content-type", "send application/json")
		return false
	}
	dec := json.NewDecoder(io.LimitReader(r.Body, layoutMaxBody+1))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil || dec.More() {
		fail(http.StatusBadRequest, "body", `body must be one JSON object: {"layout": {...}} or {"reset": true}`)
		return false
	}
	return true
}
