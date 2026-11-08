package watch

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	bestiarydata "d2lod-bot/data/bestiary"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/goals"
)

// Cain's notes, the bestiary (TASK-158, docs/unlock-bestiary.md): a card per monster kind that fills in as the hero kills
// it. 10 kills of any kind open the panel (ladder row "bestiary") and that kind's card; 50 kills of a kind add its resist
// details (row "bestiary_resists"). Kinds not met stay locked silhouettes that show only a family. View only: it counts the
// kill events the tally reads and reads the extracted tables through MonsterTables; it never reads the game.

const (
	// BestiaryRoute is GET (viewer). The two ladder rows gate the panel and the resist details.
	BestiaryRoute         = "/api/bestiary"
	BestiaryID            = "bestiary"
	BestiaryResistsID     = "bestiary_resists"
	BestiarySectionID     = "bestiary"
	bestiarySchema        = "watch.bestiary/1"
	BestiaryCardKills     = 10
	BestiaryResistKills   = 50
	bestiaryNotesSchema   = "bestiary.notes/1"
	bestiaryOpenedMaxKept = 200
)

// MonsterTables is the slice of data.DB the cards read. nil: cards carry the flavour only (tables false).
type MonsterTables interface {
	MonsterByName(name string) *data.Monster
}

// BestiaryNote is one row of data/bestiary/notes.json.
type BestiaryNote struct {
	Name    string `json:"name"`
	Family  string `json:"family"`
	Flavour string `json:"flavour"`
	Drops   string `json:"drops"`
	Tip     string `json:"tip"`
	TipTag  string `json:"tipTag"`
}

type bestiaryFile struct {
	Schema string         `json:"schema"`
	Doc    string         `json:"doc"`
	Note   string         `json:"note"`
	Kinds  []BestiaryNote `json:"kinds"`
}

// ParseBestiaryNotes checks the notes: unique names, every field filled, a tip tag of S or U. Errors name the kind.
func ParseBestiaryNotes(b []byte) ([]BestiaryNote, error) {
	var f bestiaryFile
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return nil, fmt.Errorf("bestiary notes: %w", err)
	}
	if f.Schema != bestiaryNotesSchema {
		return nil, fmt.Errorf("bestiary notes: schema %q", f.Schema)
	}
	seen := map[string]bool{}
	for _, k := range f.Kinds {
		key := strings.ToLower(k.Name)
		switch {
		case k.Name == "":
			return nil, fmt.Errorf("bestiary notes: a kind has no name")
		case seen[key]:
			return nil, fmt.Errorf("bestiary notes: duplicate kind %q", k.Name)
		case k.Family == "" || k.Flavour == "" || k.Drops == "" || k.Tip == "":
			return nil, fmt.Errorf("bestiary notes: kind %q needs a family, flavour, drops and tip", k.Name)
		case k.TipTag != "S" && k.TipTag != "U":
			return nil, fmt.Errorf("bestiary notes: kind %q: tipTag %q is not S or U", k.Name, k.TipTag)
		}
		seen[key] = true
	}
	return f.Kinds, nil
}

// BestiaryResists is the resistance percent per damage type; 100 or more is immune.
type BestiaryResists struct {
	Physical  int `json:"physical"`
	Magic     int `json:"magic"`
	Fire      int `json:"fire"`
	Lightning int `json:"lightning"`
	Cold      int `json:"cold"`
	Poison    int `json:"poison"`
}

// BestiaryDetails are the resists by difficulty and the names of the types the kind is immune to at Normal.
type BestiaryDetails struct {
	Normal    BestiaryResists `json:"normal"`
	Nightmare BestiaryResists `json:"nightmare"`
	Hell      BestiaryResists `json:"hell"`
}

// BestiaryCard is one kind as the viewer sees it. State: silhouette (not met), met (1 to 9 kills: a name and a count),
// card (10 kills: the page), full (50 kills: the resist details too). Fields a state does not reach are absent.
type BestiaryCard struct {
	ID      string           `json:"id"`
	State   string           `json:"state"`
	Family  string           `json:"family"`
	Name    string           `json:"name,omitempty"`
	Kills   int              `json:"kills,omitempty"`
	Next    int              `json:"next,omitempty"` // kills to the next step
	Type    string           `json:"type,omitempty"`
	Attack  []string         `json:"attack,omitempty"`
	Drops   string           `json:"drops,omitempty"`
	Flavour string           `json:"flavour,omitempty"`
	Tip     string           `json:"tip,omitempty"`
	TipTag  string           `json:"tipTag,omitempty"`
	Resists *BestiaryDetails `json:"resists,omitempty"`
}

// BestiaryView is GET /api/bestiary. Locked: open false, the teaser, no cards.
type BestiaryView struct {
	Schema string         `json:"schema"`
	Open   bool           `json:"open"`
	Title  string         `json:"title"`
	Teaser string         `json:"teaser,omitempty"`
	Tables bool           `json:"tables"` // false: no extracted tables, cards carry flavour only
	Met    int            `json:"met"`
	Total  int            `json:"total"`
	Cards  []BestiaryCard `json:"cards"`
}

type bestiaryOpened struct {
	t    float64
	name string
	step string
}

// Bestiary holds the kill counts. Safe for concurrent use.
type Bestiary struct {
	u      *Unlocks
	notes  []BestiaryNote
	mons   []*data.Monster // per note, nil without tables
	tables bool
	now    func() float64

	mu     sync.Mutex
	seq    uint64
	kills  map[string]int // lower-case kind name; saved to path when set (SetStore)
	path   string
	opened []bestiaryOpened
}

const bestiaryStoreSchema = "bestiary.kills/1"

type bestiaryStore struct {
	Schema string         `json:"schema"`
	Kills  map[string]int `json:"kills"`
}

// SetStore keeps the kill counts in the file at path (loaded now, rewritten on every counted kill), so a restart goes on
// from the old counts. A missing file starts empty; an unreadable or foreign one is an error and nothing is changed.
func (b *Bestiary) SetStore(path string) error {
	raw, err := os.ReadFile(path)
	var st bestiaryStore
	switch {
	case os.IsNotExist(err):
	case err != nil:
		return fmt.Errorf("bestiary store: %w", err)
	default:
		if err := json.Unmarshal(raw, &st); err != nil || st.Schema != bestiaryStoreSchema {
			return fmt.Errorf("bestiary store %s: not a %s file", path, bestiaryStoreSchema)
		}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.path = path
	for k, n := range st.Kills {
		if n > b.kills[k] {
			b.kills[k] = n
		}
	}
	return nil
}

// saveLocked writes the counts atomically. A failed write loses the save only; the counts stay in memory.
func (b *Bestiary) saveLocked() {
	if b.path == "" {
		return
	}
	raw, err := json.Marshal(bestiaryStore{Schema: bestiaryStoreSchema, Kills: b.kills})
	if err != nil {
		return
	}
	tmp := b.path + ".tmp"
	if os.MkdirAll(filepath.Dir(b.path), 0o700) != nil || os.WriteFile(tmp, raw, 0o600) != nil || os.Rename(tmp, b.path) != nil {
		return
	}
}

// FilterBestiaryNotes returns the shipped kinds that exist in the monster table and the names that do not (a kind the
// table lacks is skipped, not fatal). tables nil: every kind, none skipped.
func FilterBestiaryNotes(tables MonsterTables) (notes []BestiaryNote, skipped []string, err error) {
	all, err := ParseBestiaryNotes(bestiarydata.JSON)
	if err != nil {
		return nil, nil, err
	}
	if tables == nil {
		return all, nil, nil
	}
	notes = []BestiaryNote{} // never nil: nil means "read the embedded file" to NewBestiary
	for _, n := range all {
		if tables.MonsterByName(n.Name) == nil {
			skipped = append(skipped, n.Name)
			continue
		}
		notes = append(notes, n)
	}
	return notes, skipped, nil
}

// NewBestiary builds the bestiary over the ladder. A ladder without the row "bestiary" or "bestiary_resists" is an error
// naming the row. notes nil reads the embedded file. With tables, every kind must exist in the monster table (an error
// names the kind); tables nil gives flavour-only cards.
func NewBestiary(u *Unlocks, notes []BestiaryNote, tables MonsterTables, now func() float64) (*Bestiary, error) {
	for _, id := range []string{BestiaryID, BestiaryResistsID} {
		if _, err := u.Engine.Check(id); err != nil {
			return nil, err
		}
	}
	if notes == nil {
		var err error
		if notes, err = ParseBestiaryNotes(bestiarydata.JSON); err != nil {
			return nil, err
		}
	}
	if now == nil {
		now = func() float64 { return 0 }
	}
	b := &Bestiary{u: u, notes: notes, mons: make([]*data.Monster, len(notes)), tables: tables != nil, now: now, kills: map[string]int{}}
	if tables != nil {
		for i, n := range notes {
			m := tables.MonsterByName(n.Name)
			if m == nil {
				return nil, fmt.Errorf("bestiary: kind %q is not in the monster table", n.Name)
			}
			b.mons[i] = m
		}
	}
	return b, nil
}

func bestiaryID(name string) string { return strings.ReplaceAll(strings.ToLower(name), " ", "_") }

// OnGoalEvent counts a kill of a listed kind. An event with a Seq at or below the highest seen is a replay.
func (b *Bestiary) OnGoalEvent(ev goals.Event) {
	if ev.Type != goals.EvKill || ev.Monster == "" {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if ev.Seq != 0 {
		if ev.Seq <= b.seq {
			return
		}
		b.seq = ev.Seq
	}
	key := strings.ToLower(ev.Monster)
	idx := -1
	for i, n := range b.notes {
		if strings.EqualFold(n.Name, ev.Monster) {
			idx = i
			break
		}
	}
	if idx < 0 {
		return
	}
	b.kills[key]++
	b.saveLocked()
	switch b.kills[key] {
	case BestiaryCardKills:
		b.record(b.notes[idx].Name, "card")
	case BestiaryResistKills:
		b.record(b.notes[idx].Name, "resists")
	}
}

func (b *Bestiary) record(name, step string) {
	b.opened = append(b.opened, bestiaryOpened{t: b.now(), name: name, step: step})
	if len(b.opened) > bestiaryOpenedMaxKept {
		b.opened = append([]bestiaryOpened(nil), b.opened[len(b.opened)-bestiaryOpenedMaxKept:]...)
	}
}

func bestiaryResistsOf(m *data.Monster, d data.Difficulty) BestiaryResists {
	r := m.Res[d]
	return BestiaryResists{r[data.Physical], r[data.Magic], r[data.Fire], r[data.Lightning], r[data.Cold], r[data.Poison]}
}

func bestiaryAttack(m *data.Monster) []string {
	var out []string
	if m.Melee {
		out = append(out, "melee")
	}
	if m.Ranged {
		out = append(out, "ranged")
	}
	if m.Flying {
		out = append(out, "flies")
	}
	for _, e := range m.Elem {
		if e.Type != "" && e.Pct > 0 {
			out = append(out, e.Type+" hits")
		}
	}
	return out
}

// View is the payload. The resist details appear only when the kind has 50 kills and the ladder row is open.
func (b *Bestiary) View() BestiaryView {
	feed, _ := b.u.state(BestiaryID)
	resists, _ := b.u.state(BestiaryResistsID)
	v := BestiaryView{Schema: bestiarySchema, Open: feed.Open, Title: feed.Title, Tables: b.tables, Total: len(b.notes), Cards: []BestiaryCard{}}
	if !feed.Open {
		v.Teaser = feed.Teaser
		return v
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	for i, n := range b.notes {
		k := b.kills[strings.ToLower(n.Name)]
		c := BestiaryCard{ID: bestiaryID(n.Name), State: "silhouette", Family: n.Family}
		if k > 0 {
			v.Met++
			c.State, c.Name, c.Kills, c.Next = "met", n.Name, k, BestiaryCardKills-k
		}
		if k >= BestiaryCardKills {
			c.State, c.Next = "card", BestiaryResistKills-k
			c.Drops, c.Flavour, c.Tip, c.TipTag = n.Drops, n.Flavour, n.Tip, n.TipTag
			if m := b.mons[i]; m != nil {
				c.Type, c.Attack = m.Type, bestiaryAttack(m)
			}
		}
		if k >= BestiaryResistKills && resists.Open {
			c.State, c.Next = "full", 0
			if m := b.mons[i]; m != nil {
				c.Resists = &BestiaryDetails{bestiaryResistsOf(m, data.Normal), bestiaryResistsOf(m, data.Nightmare), bestiaryResistsOf(m, data.Hell)}
			}
		}
		v.Cards = append(v.Cards, c)
	}
	return v
}

// Handler serves GET /api/bestiary.
func (b *Bestiary) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "GET only", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, 200, b.View())
	})
}

// Section is the return screen's "New in Cain's notes": the cards and resist pages that opened since the viewer left.
// Hidden while the bestiary is locked or nothing opened in the window.
func (b *Bestiary) Section() ReturnSection {
	return SectionFunc(func(d *ReturnData) *SectionView {
		if st, _ := b.u.state(BestiaryID); !st.Open {
			return nil
		}
		b.mu.Lock()
		defer b.mu.Unlock()
		var lines []string
		for _, o := range b.opened {
			if o.t < d.Since {
				continue
			}
			if o.step == "card" {
				lines = append(lines, "A new card: "+o.name+".")
			} else {
				lines = append(lines, "Resist notes for "+o.name+".")
			}
		}
		if len(lines) == 0 {
			return nil
		}
		return &SectionView{ID: BestiarySectionID, Title: "New in Cain's notes", Lines: lines}
	})
}
