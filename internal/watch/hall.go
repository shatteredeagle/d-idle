package watch

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"

	"d2lod-bot/internal/currency"
	"d2lod-bot/internal/goals"
)

// The Hall of Fame (TASK-135, docs/hall-of-fame.md): a memorial of retired and dead heroes, kept as hall.retire and
// hall.death entries of the account ledger. Ledger and UI only: no charm item, no stat grant, no tally change.

// Hall routes: the page and JSON are GET on the viewer, the answer is POST on the answer listener.
const (
	HallPage      = "/watch/hall"
	HallRoute     = "/api/hall"
	HallPostRoute = "/api/hall/answer"
	// HallSectionID is the return screen section.
	HallSectionID = "hall"
	hallSchema    = "watch.hall/1"

	HallRetire = "retire"
	HallKeep   = "keep"

	// RetireQuestion is asked once Andariel is dead. The default is to keep playing.
	RetireQuestion = "Andariel is dead. Retire %s into the Hall of Fame, or keep playing?"
)

// HallRoutes are the GET routes of the viewer.
var HallRoutes = []string{HallPage, HallRoute}

// ---- titles (data) ---------------------------------------------------------------------------------------------------

//go:embed hall_titles.json
var shippedHallTitles []byte

// TitleWhen is the rule of a title: a status (retired or dead; empty: any) and, for a kill title, a monster name and the
// kills needed.
type TitleWhen struct {
	Status  string `json:"status,omitempty"`
	Monster string `json:"monster,omitempty"`
	Min     int    `json:"min,omitempty"`
}

// HallTitle is one row of the title table.
type HallTitle struct {
	ID      string    `json:"id"`
	Rank    int       `json:"rank"` // the highest earned rank is the hero's title
	Text    string    `json:"text"`
	Flavour string    `json:"flavour"`
	Classes []string  `json:"classes,omitempty"` // empty: every class
	When    TitleWhen `json:"when"`
}

// HallTitles is the title table.
type HallTitles struct {
	Version int         `json:"version"`
	Classes []string    `json:"classes"`
	Titles  []HallTitle `json:"titles"`
}

// ShippedHallTitles parses the embedded table.
func ShippedHallTitles() (*HallTitles, error) { return ParseHallTitles(shippedHallTitles) }

// ParseHallTitles reads and checks a table: every row has an id, text and rank, names only known classes, and has a
// status or a monster rule; ids are unique.
func ParseHallTitles(b []byte) (*HallTitles, error) {
	var t HallTitles
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&t); err != nil {
		return nil, fmt.Errorf("hall titles: %w", err)
	}
	seen := map[string]bool{}
	for _, r := range t.Titles {
		if r.ID == "" || r.Text == "" || r.Rank <= 0 {
			return nil, fmt.Errorf("hall titles: row %q needs id, text and rank", r.ID)
		}
		if seen[r.ID] {
			return nil, fmt.Errorf("hall titles: duplicate id %q", r.ID)
		}
		seen[r.ID] = true
		for _, c := range r.Classes {
			if !slices.Contains(t.Classes, c) {
				return nil, fmt.Errorf("hall titles: row %q names unknown class %q", r.ID, c)
			}
		}
		switch r.When.Status {
		case "", "retired", "dead":
		default:
			return nil, fmt.Errorf("hall titles: row %q has unknown status %q", r.ID, r.When.Status)
		}
		if r.When.Status == "" && (r.When.Monster == "" || r.When.Min <= 0) {
			return nil, fmt.Errorf("hall titles: row %q needs a status or a monster with min kills", r.ID)
		}
	}
	return &t, nil
}

// Known says whether class is in the table's class list.
func (t *HallTitles) Known(class string) error {
	if !slices.Contains(t.Classes, strings.ToLower(class)) {
		return fmt.Errorf("hall: unknown class %q (no title rows for it)", class)
	}
	return nil
}

// Earned lists the titles a hero holds, best first. An unknown class is an error naming it.
func (t *HallTitles) Earned(class, status string, killsBy map[string]int) ([]HallTitle, error) {
	class = strings.ToLower(class)
	if err := t.Known(class); err != nil {
		return nil, err
	}
	var out []HallTitle
	for _, r := range t.Titles {
		if len(r.Classes) > 0 && !slices.Contains(r.Classes, class) {
			continue
		}
		if r.When.Status != "" && r.When.Status != status {
			continue
		}
		if r.When.Monster != "" && killsBy[r.When.Monster] < r.When.Min {
			continue
		}
		out = append(out, r)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Rank > out[j].Rank })
	return out, nil
}

// notable names the monsters the table counts; the record keeps only those.
func (t *HallTitles) notable() map[string]bool {
	m := map[string]bool{}
	for _, r := range t.Titles {
		if r.When.Monster != "" {
			m[r.When.Monster] = true
		}
	}
	return m
}

// ---- records and the store -------------------------------------------------------------------------------------------

// HallRecord is the payload of a hall.* ledger entry. Status is derived from the entry kind, not stored.
type HallRecord struct {
	Name     string         `json:"name"`
	Class    string         `json:"class"`
	Level    int            `json:"level"`
	Kills    int            `json:"kills"`
	KillsBy  map[string]int `json:"killsBy,omitempty"` // only the monsters the title table counts
	Deaths   int            `json:"deaths"`
	GameSecs float64        `json:"gameSecs"`
	WallSecs float64        `json:"wallSecs"`
	BestItem string         `json:"bestItem,omitempty"`
	Cause    string         `json:"cause,omitempty"`
}

// HallLine is one hero on the page: one line each.
type HallLine struct {
	HallRecord
	Status  string   `json:"status"` // retired or dead
	Title   string   `json:"title"`
	Flavour string   `json:"flavour,omitempty"`
	Titles  []string `json:"titles"`
	Line    string   `json:"line"`
	Seq     int      `json:"seq"`
}

// HallView is GET /api/hall.
type HallView struct {
	Schema string     `json:"schema"`
	Heroes []HallLine `json:"heroes"`
	// Codex and Tier are the account-wide parts shown as a title line (no charm yet).
	Account string `json:"account"`
}

// HallStore reads the Hall from the ledger.
type HallStore struct {
	Ledger *currency.Ledger
	Titles *HallTitles
}

func hallKey(kind, id string) string { return "hall:" + kind + ":" + id }

// Record books a hero. It is idempotent: the same hero and kind is booked once, a replay returns added false.
func (s *HallStore) Record(kind, id string, r HallRecord) (added bool, err error) {
	if err := s.Titles.Known(r.Class); err != nil {
		return false, err
	}
	if id == "" || r.Name == "" {
		return false, errors.New("hall: a hero needs an id and a name")
	}
	if kind != currency.KindHallRetire && kind != currency.KindHallDeath {
		return false, fmt.Errorf("hall: unknown kind %q", kind)
	}
	r.Class = strings.ToLower(r.Class)
	data, err := json.Marshal(r)
	if err != nil {
		return false, err
	}
	_, err = s.Ledger.Append(currency.Entry{Kind: kind, Key: hallKey(kind, id), Character: r.Name, Data: data, Note: safeText(r.Cause, 120)})
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, currency.ErrDuplicate):
		return false, nil
	}
	return false, err
}

// View lists the heroes, oldest first. A hero booked as retired and later as dead shows once, as dead (the same hero may
// continue after a retire is booked elsewhere; the last word wins).
func (s *HallStore) View() (*HallView, error) {
	es, err := s.Ledger.Entries()
	if err != nil {
		return nil, err
	}
	v := &HallView{Schema: hallSchema, Heroes: []HallLine{}}
	byID := map[string]int{}
	for _, e := range es {
		var status string
		switch e.Kind {
		case currency.KindHallRetire:
			status = "retired"
		case currency.KindHallDeath:
			status = "dead"
		default:
			continue
		}
		var r HallRecord
		if err := json.Unmarshal(e.Data, &r); err != nil {
			return nil, fmt.Errorf("hall: entry %d: %w", e.Seq, err)
		}
		earned, err := s.Titles.Earned(r.Class, status, r.KillsBy)
		if err != nil {
			return nil, fmt.Errorf("hall: entry %d: %w", e.Seq, err)
		}
		l := HallLine{HallRecord: r, Status: status, Seq: e.Seq, Titles: []string{}}
		for _, t := range earned {
			l.Titles = append(l.Titles, t.Text)
		}
		if len(earned) > 0 {
			l.Title, l.Flavour = earned[0].Text, earned[0].Flavour
		}
		if l.BestItem == "" {
			l.BestItem = "-"
		}
		if l.Cause == "" {
			l.Cause = status
		}
		l.Line = hallLine(l)
		id := strings.TrimPrefix(strings.TrimPrefix(e.Key, "hall:retire:"), "hall:death:")
		if i, ok := byID[id]; ok {
			v.Heroes[i] = l
		} else {
			byID[id] = len(v.Heroes)
			v.Heroes = append(v.Heroes, l)
		}
	}
	retired, dead := 0, 0
	for _, h := range v.Heroes {
		if h.Status == "dead" {
			dead++
		} else {
			retired++
		}
	}
	v.Account = fmt.Sprintf("%d retired, %d fallen", retired, dead)
	return v, nil
}

func hallLine(l HallLine) string {
	title := l.Title
	if title == "" {
		title = "no title yet"
	}
	return fmt.Sprintf("%s the %s, level %d, %s: %d kills, %d deaths, %s game time (%s wall), best item %s, %s", l.Name, l.Class, l.Level,
		title, l.Kills, l.Deaths, fmtDur(l.GameSecs), fmtDur(l.WallSecs), l.BestItem, l.Cause)
}

// ---- the desk --------------------------------------------------------------------------------------------------------

// HallFacts is what only the bot knows about the hero. Permanent is the death event of the hardcore flag or the
// death-limit rule: no goals event carries it yet (U), so the bot fills it from its own rule, and a Fake sets it in tests.
type HallFacts struct {
	ID        string // stable per hero (character name plus creation), the idempotency key
	Name      string
	Class     string
	Level     int
	GameSecs  float64
	WallSecs  float64
	Permanent bool
	Cause     string
}

// HallConfig builds a HallDesk. Store, Facts and Stop are required. Stop is the bot's existing clean shutdown path.
type HallConfig struct {
	Store *HallStore
	Facts func() HallFacts
	Stop  func() error
	// StatePath keeps the counters and the question across a restart ("": in memory).
	StatePath string
}

type hallState struct {
	Kills   int            `json:"kills"`
	KillsBy map[string]int `json:"killsBy"`
	Deaths  int            `json:"deaths"`
	Best    string         `json:"best,omitempty"`
	BestRnk int            `json:"bestRank,omitempty"`
	Offered string         `json:"offered,omitempty"` // hero id the question was raised for
	Answer  string         `json:"answer,omitempty"`  // retire or keep
	Hero    string         `json:"hero,omitempty"`    // hero id the counters belong to
	StopOK  bool           `json:"stopOk,omitempty"`
}

// HallDesk counts what the Hall needs, raises the retire question and books the entries. Safe for concurrent use. It
// never retires by itself: no timer, no away rule, only an explicit answer.
type HallDesk struct {
	c  HallConfig
	mu sync.Mutex
	st hallState
}

// NewHallDesk checks the config and loads the saved state.
func NewHallDesk(c HallConfig) (*HallDesk, error) {
	if c.Store == nil || c.Store.Ledger == nil || c.Store.Titles == nil || c.Facts == nil || c.Stop == nil {
		return nil, errors.New("watch: the Hall desk needs a store, a facts reader and the stop path")
	}
	d := &HallDesk{c: c}
	d.st.KillsBy = map[string]int{}
	if c.StatePath != "" {
		if b, err := os.ReadFile(c.StatePath); err == nil {
			if err := json.Unmarshal(b, &d.st); err != nil {
				return nil, fmt.Errorf("watch: Hall state %s: %w", c.StatePath, err)
			}
			if d.st.KillsBy == nil {
				d.st.KillsBy = map[string]int{}
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}
	return d, nil
}

func (d *HallDesk) save() {
	if d.c.StatePath == "" {
		return
	}
	b, _ := json.Marshal(d.st)
	tmp := d.c.StatePath + ".tmp"
	if os.WriteFile(tmp, b, 0o644) == nil {
		os.Rename(tmp, d.c.StatePath)
	}
}

// OnGoalEvent counts kills, deaths and the best drop, raises the question at Andariel's death and books a permanent
// death. Counters restart when the hero id changes (a new hero).
func (d *HallDesk) OnGoalEvent(ev goals.Event) {
	f := d.c.Facts()
	d.mu.Lock()
	defer d.mu.Unlock()
	if f.ID != "" && d.st.Hero != f.ID {
		d.st = hallState{KillsBy: map[string]int{}, Hero: f.ID}
	}
	switch ev.Type {
	case goals.EvKill:
		d.st.Kills++
		if ev.Monster != "" {
			d.st.KillsBy[ev.Monster]++
		}
		if ev.Monster == "Andariel" && d.st.Offered == "" && f.ID != "" {
			d.st.Offered = f.ID
			d.save()
		} else if d.st.Kills%25 == 0 {
			d.save()
		}
	case goals.EvDeath:
		d.st.Deaths++
		if f.Permanent && f.ID != "" {
			d.bookLocked(currency.KindHallDeath, f, f.Cause)
		}
		d.save()
	case goals.EvDrop:
		if ev.Item != nil {
			if _, r := rankOf(AwayEvent{Item: ev.Item.Name, Quality: ev.Item.Quality, Type: ev.Item.Type}); r > d.st.BestRnk {
				d.st.Best, d.st.BestRnk = safeText(strings.TrimSpace(ev.Item.Quality+" "+ev.Item.Name), 60), r
			}
		}
	}
}

func (d *HallDesk) recordLocked(f HallFacts, cause string) HallRecord {
	by := map[string]int{}
	notable := d.c.Store.Titles.notable()
	for k, n := range d.st.KillsBy {
		if notable[k] {
			by[k] = n
		}
	}
	return HallRecord{Name: f.Name, Class: f.Class, Level: f.Level, Kills: d.st.Kills, KillsBy: by, Deaths: d.st.Deaths,
		GameSecs: f.GameSecs, WallSecs: f.WallSecs, BestItem: d.st.Best, Cause: cause}
}

func (d *HallDesk) bookLocked(kind string, f HallFacts, cause string) (bool, error) {
	added, err := d.c.Store.Record(kind, f.ID, d.recordLocked(f, cause))
	return added, err
}

// Question is the pending retire question, nil when none is open (Andariel not dead, or already answered).
type Question struct {
	Text    string   `json:"text"`
	Default string   `json:"default"` // keep playing
	Answers []string `json:"answers"`
	Hero    string   `json:"hero"`
}

// Pending returns the open question.
func (d *HallDesk) Pending() *Question {
	f := d.c.Facts()
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.st.Offered == "" || d.st.Offered != f.ID || d.st.Answer != "" {
		return nil
	}
	return &Question{Text: fmt.Sprintf(RetireQuestion, f.Name), Default: HallKeep, Answers: []string{HallRetire, HallKeep}, Hero: f.Name}
}

// Errors of Answer.
var (
	ErrNoQuestion = errors.New("watch: no retire question is open")
	ErrBadHall    = errors.New("watch: bad Hall answer")
)

// Answer takes the player's answer. "keep" closes the question and the hero plays on. "retire" books hall.retire (once;
// a replay adds nothing) and then asks the bot to stop through its shutdown path; if the stop fails the entry stays
// and a replay tries the stop again.
func (d *HallDesk) Answer(answer string) (changed bool, err error) {
	if answer != HallRetire && answer != HallKeep {
		return false, fmt.Errorf("%w: %q", ErrBadHall, answer)
	}
	f := d.c.Facts()
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.st.Offered == "" || d.st.Offered != f.ID {
		return false, ErrNoQuestion
	}
	if d.st.Answer != "" && d.st.Answer != answer {
		return false, fmt.Errorf("%w: already answered %q", ErrBadHall, d.st.Answer)
	}
	if answer == HallKeep {
		changed = d.st.Answer == ""
		d.st.Answer = HallKeep
		d.save()
		return changed, nil
	}
	added, err := d.bookLocked(currency.KindHallRetire, f, "retired")
	if err != nil {
		return false, err
	}
	d.st.Answer = HallRetire
	d.save()
	if added || !d.st.StopOK {
		if err := d.c.Stop(); err != nil {
			return added, fmt.Errorf("watch: retired, but the bot did not stop cleanly: %w", err)
		}
		d.st.StopOK = true
		d.save()
	}
	return added, nil
}

// Title is the hero's best title so far, "" for none. Used by the watch header.
func (d *HallDesk) Title() string {
	f := d.c.Facts()
	d.mu.Lock()
	defer d.mu.Unlock()
	if f.Class == "" {
		return ""
	}
	es, err := d.c.Store.Titles.Earned(f.Class, "", d.st.KillsBy)
	if err != nil || len(es) == 0 {
		return ""
	}
	return es[0].Text
}

// Section is the return screen's Hall section: the open question with its two buttons, else the Hall's account line.
func (d *HallDesk) Section() ReturnSection {
	return SectionFunc(func(*ReturnData) *SectionView {
		sv := &SectionView{ID: HallSectionID, Title: "Hall of Fame", After: true}
		if q := d.Pending(); q != nil {
			sv.Lines = append(sv.Lines, q.Text, "Retiring stops the bot. If you do nothing, the hero keeps playing.")
			sv.Buttons = append(sv.Buttons,
				SectionButton{Label: "Keep playing", Row: "hall", Option: HallKeep, Act: "hall"},
				SectionButton{Label: "Retire into the Hall", Row: "hall", Option: HallRetire, Act: "hall"})
		}
		if v, err := d.c.Store.View(); err == nil {
			sv.Lines = append(sv.Lines, "Hall: "+v.Account+". Open "+HallPage+" to read it.")
		} else {
			sv.Lines = append(sv.Lines, "The Hall could not be read: "+err.Error())
		}
		if t := d.Title(); t != "" {
			sv.Lines = append(sv.Lines, "Title so far: "+t)
		}
		return sv
	})
}

// ---- HTTP ------------------------------------------------------------------------------------------------------------

// Handler serves GET /watch/hall and GET /api/hall (viewer; the token gate sits in front of it).
func (d *HallDesk) Handler() http.Handler { return HallHandler(d.c.Store) }

// HallHandler serves the page and the JSON from a store alone.
func HallHandler(s *HallStore) http.Handler {
	m := http.NewServeMux()
	m.HandleFunc(HallRoute, func(w http.ResponseWriter, r *http.Request) {
		v, err := s.View()
		if err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, v)
	})
	m.HandleFunc(HallPage, func(w http.ResponseWriter, r *http.Request) {
		v, err := s.View()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			io.WriteString(w, "The Hall could not be read: "+html.EscapeString(err.Error()))
			return
		}
		w.Write(RenderHall(v))
	})
	return readOnly(m)
}

// RenderHall is the page: one row per hero.
func RenderHall(v *HallView) []byte {
	var b bytes.Buffer
	b.WriteString("<!doctype html><html lang=\"en\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width,initial-scale=1\"><title>Hall of Fame</title>")
	b.WriteString("<style>body{font:16px/1.4 system-ui,sans-serif;background:#14110f;color:#e8dcc8;margin:0;padding:1rem}table{border-collapse:collapse;width:100%}th,td{border-bottom:1px solid #3a322a;padding:.4rem .5rem;text-align:left}h1{font-size:1.3rem}.dead{color:#c98}.fl{color:#9a8f7c;font-style:italic}a{color:#d9b46a}</style></head><body>")
	b.WriteString("<p><a href=\"/watch/return\">Back to the return screen</a></p><h1>Hall of Fame</h1><p>" + html.EscapeString(v.Account) + "</p>")
	if len(v.Heroes) == 0 {
		b.WriteString("<p>No hero has retired or fallen yet.</p>")
	} else {
		b.WriteString("<table><thead><tr><th>Hero</th><th>Class</th><th>Level</th><th>Title</th><th>Kills</th><th>Deaths</th><th>Game time</th><th>Wall time</th><th>Best item</th><th>End</th></tr></thead><tbody>")
		for _, h := range v.Heroes {
			cls := ""
			if h.Status == "dead" {
				cls = " class=\"dead\""
			}
			title := html.EscapeString(h.Title)
			if h.Flavour != "" {
				title += " <span class=\"fl\">" + html.EscapeString(h.Flavour) + "</span>"
			}
			fmt.Fprintf(&b, "<tr%s><td>%s</td><td>%s</td><td>%d</td><td>%s</td><td>%d</td><td>%d</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>",
				cls, html.EscapeString(h.Name), html.EscapeString(h.Class), h.Level, title, h.Kills, h.Deaths,
				fmtDur(h.GameSecs), fmtDur(h.WallSecs), html.EscapeString(h.BestItem), html.EscapeString(h.Cause))
		}
		b.WriteString("</tbody></table>")
	}
	b.WriteString("</body></html>")
	return b.Bytes()
}

type hallBody struct {
	Answer string `json:"answer"`
}

type hallReply struct {
	OK      bool   `json:"ok"`
	Changed bool   `json:"changed,omitempty"`
	Code    string `json:"code,omitempty"`
	Error   string `json:"error,omitempty"`
}

// PostHandler is POST /api/hall/answer {answer: "retire"|"keep"} on the answer listener: origin list, token, JSON only.
func (d *HallDesk) PostHandler(o AnswerOptions) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := origin != "" && containsStr(o.Origins, origin)
		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
		}
		fail := func(code int, c, msg string) { writeJSON(w, code, hallReply{Code: c, Error: msg}) }
		switch {
		case r.Method == http.MethodOptions && allowed:
			w.Header().Set("Access-Control-Allow-Methods", "POST")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusNoContent)
			return
		case r.Method != http.MethodPost:
			w.Header().Set("Allow", "POST, OPTIONS")
			fail(http.StatusMethodNotAllowed, "method", "POST only")
			return
		case origin != "" && !allowed:
			fail(http.StatusForbidden, "origin", "origin not allowed")
			return
		case !o.tokenOK(r):
			fail(http.StatusUnauthorized, "token", "unauthorized")
			return
		}
		if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
			fail(http.StatusUnsupportedMediaType, "content-type", "send application/json")
			return
		}
		var b hallBody
		if err := decodeSmall(r, &b); err != nil {
			fail(http.StatusBadRequest, "body", "body must be one small JSON object with known fields")
			return
		}
		changed, err := d.Answer(b.Answer)
		switch {
		case err == nil:
			writeJSON(w, 200, hallReply{OK: true, Changed: changed})
		case errors.Is(err, ErrNoQuestion):
			fail(http.StatusNotFound, "no-question", err.Error())
		case errors.Is(err, ErrBadHall):
			fail(http.StatusConflict, "bad-answer", err.Error())
		default:
			fail(http.StatusInternalServerError, "hall", err.Error())
		}
	})
}

// ledgerDirState is the default state file next to the ledger.
func ledgerDirState(l *currency.Ledger) string { return filepath.Join(l.Dir(), "hall-state.json") }
