package watch

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"net/http"
	"strings"
	"sync"

	idrevealdata "d2lod-bot/data/idreveal"
	"d2lod-bot/internal/goals"
)

// The identify reveal (TASK-157, docs/unlock-idreveal.md): when the hero picks up a magic or rare item a face-down card
// appears; the viewer flips it and only then sees the quality word, the colour, the sound and (when a reader exists) the
// affixes. The first magic or rare pickup opens it (ladder row "idreveal"); sets, uniques and crafted items get their
// cards after the first unique or set in the Codex (row "idreveal_later"). A face-down card carries no quality, no colour
// and no item name: the name of an identified item would give its rarity away.

const (
	// RevealRoute is GET (viewer); RevealFlipRoute is POST (answer listener). The rows are IDRevealID and IDRevealLaterID.
	RevealRoute     = "/api/reveal"
	RevealFlipRoute = "/api/reveal/flip"
	IDRevealID      = "idreveal"
	IDRevealLaterID = "idreveal_later"

	revealSchema = "watch.reveal/1"
	maxCards     = 30
)

// QualityCard is the wording and look of one quality.
type QualityCard struct {
	Label  string   `json:"label"`
	Colour string   `json:"colour"`
	Cue    [][2]int `json:"cue"`
	Text   []string `json:"text"`
}

// RevealTable is the parsed data/idreveal/reveal.json.
type RevealTable struct {
	Schema    string                       `json:"schema"`
	Doc       string                       `json:"doc"`
	Note      string                       `json:"note"`
	FaceDown  struct{ Title, Text string } `json:"face_down"`
	Qualities map[string]QualityCard       `json:"qualities"`
	First     []string                     `json:"first_cards"`
	Later     []string                     `json:"later_cards"`
}

// ParseRevealTable checks the table: each quality has a label, a colour class, a cue and 3 or more texts; the first and
// later lists name known qualities and do not overlap. Errors name the quality.
func ParseRevealTable(b []byte) (*RevealTable, error) {
	var t RevealTable
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&t); err != nil {
		return nil, fmt.Errorf("reveal table: %w", err)
	}
	if t.Schema != "idreveal.cards/1" {
		return nil, fmt.Errorf("reveal table: schema %q", t.Schema)
	}
	if t.FaceDown.Title == "" || t.FaceDown.Text == "" {
		return nil, errors.New("reveal table: face_down needs a title and a text")
	}
	for q, c := range t.Qualities {
		if c.Label == "" || !strings.HasPrefix(c.Colour, "q-") || len(c.Cue) == 0 || len(c.Text) < 3 {
			return nil, fmt.Errorf("reveal table: quality %q needs a label, a q-* colour, a cue and 3 or more texts", q)
		}
	}
	seen := map[string]bool{}
	for _, list := range [][]string{t.First, t.Later} {
		for _, q := range list {
			if _, ok := t.Qualities[q]; !ok {
				return nil, fmt.Errorf("reveal table: quality %q is listed but has no entry", q)
			}
			if seen[q] {
				return nil, fmt.Errorf("reveal table: quality %q is in both card lists", q)
			}
			seen[q] = true
		}
	}
	for q := range t.Qualities {
		if !seen[q] {
			return nil, fmt.Errorf("reveal table: quality %q is in neither card list", q)
		}
	}
	return &t, nil
}

// AffixSource reads the affixes of an identified item. No reader exists yet (docs/unlock-idreveal.md); a Fake serves the
// tests and nil means "not read", which the card says plainly.
type AffixSource interface {
	Affixes(name string) []string
}

// RevealCard is one card. Until Revealed, only ID, N, T and Revealed are filled.
type RevealCard struct {
	ID        string   `json:"id"`
	N         int      `json:"n"`
	T         float64  `json:"t"`
	Revealed  bool     `json:"revealed"`
	Name      string   `json:"name,omitempty"`
	Quality   string   `json:"quality,omitempty"`
	Label     string   `json:"label,omitempty"`
	Colour    string   `json:"colour,omitempty"`
	Text      string   `json:"text,omitempty"`
	Cue       [][2]int `json:"cue,omitempty"`
	Affixes   []string `json:"affixes,omitempty"`
	AffixNote string   `json:"affixNote,omitempty"`
}

// RevealView is GET /api/reveal. Locked: the teaser only.
type RevealView struct {
	Schema   string       `json:"schema"`
	Open     bool         `json:"open"`
	Title    string       `json:"title"`
	Teaser   string       `json:"teaser,omitempty"`
	Face     string       `json:"face,omitempty"` // title of a face-down card
	FaceText string       `json:"faceText,omitempty"`
	Later    bool         `json:"later"` // sets, uniques and crafted items get cards too
	Cards    []RevealCard `json:"cards"`
}

type hiddenCard struct {
	card    RevealCard // the public part (face down)
	name    string
	quality string
}

// Reveal holds the cards. Safe for concurrent use.
type Reveal struct {
	u       *Unlocks
	table   *RevealTable
	affixes AffixSource
	now     func() float64

	mu    sync.Mutex
	n     int
	seq   uint64
	cards []*hiddenCard
}

// NewReveal builds the reveal over the ladder. A ladder without the row "idreveal" or "idreveal_later" is an error
// naming it; table nil reads the embedded table; affixes may be nil.
func NewReveal(u *Unlocks, table *RevealTable, affixes AffixSource, now func() float64) (*Reveal, error) {
	for _, id := range []string{IDRevealID, IDRevealLaterID} {
		if _, err := u.Engine.Check(id); err != nil {
			return nil, err
		}
	}
	if table == nil {
		var err error
		if table, err = ParseRevealTable(idrevealdata.JSON); err != nil {
			return nil, err
		}
	}
	if now == nil {
		now = func() float64 { return 0 }
	}
	return &Reveal{u: u, table: table, affixes: affixes, now: now}, nil
}

// OnGoalEvent makes a card for every pickup whose quality has an entry (a Seq at or below the highest seen is a replay).
// Whether the viewer sees it is decided when the view is built.
func (rv *Reveal) OnGoalEvent(ev goals.Event) {
	if ev.Type != goals.EvDrop || ev.Item == nil {
		return
	}
	q := strings.ToLower(ev.Item.Quality)
	if _, ok := rv.table.Qualities[q]; !ok {
		return
	}
	rv.mu.Lock()
	defer rv.mu.Unlock()
	if ev.Seq != 0 {
		if ev.Seq <= rv.seq {
			return
		}
		rv.seq = ev.Seq
	}
	rv.n++
	c := &hiddenCard{card: RevealCard{ID: fmt.Sprintf("c%d", rv.n), N: rv.n, T: rv.now()}, name: safeText(ev.Item.Name, 80), quality: q}
	rv.cards = append(rv.cards, c)
	if len(rv.cards) > maxCards {
		rv.cards = append([]*hiddenCard(nil), rv.cards[len(rv.cards)-maxCards:]...)
	}
}

// visible says whether the viewer may see a card of this quality yet.
func (rv *Reveal) visible(quality string, first, later bool) bool {
	switch {
	case containsStr(rv.table.First, quality):
		return first
	case containsStr(rv.table.Later, quality):
		return later
	}
	return false
}

func (rv *Reveal) open() (first, later bool) {
	f, _ := rv.u.state(IDRevealID)
	l, _ := rv.u.state(IDRevealLaterID)
	return f.Open, l.Open
}

func (rv *Reveal) face(c *hiddenCard) RevealCard { return c.card }

func (rv *Reveal) faceUp(c *hiddenCard) RevealCard {
	q := rv.table.Qualities[c.quality]
	out := c.card
	out.Revealed, out.Name, out.Quality, out.Label, out.Colour, out.Cue = true, c.name, c.quality, q.Label, q.Colour, q.Cue
	h := fnv.New32a()
	h.Write([]byte(c.card.ID + "/" + c.quality))
	out.Text = q.Text[int(h.Sum32())%len(q.Text)]
	if rv.affixes != nil {
		out.Affixes = rv.affixes.Affixes(c.name)
	}
	if len(out.Affixes) == 0 {
		out.AffixNote = "The properties are not read from the game yet."
	}
	return out
}

// View is the payload; cards are newest last, face down ones carry nothing about the item.
func (rv *Reveal) View() RevealView {
	st, _ := rv.u.state(IDRevealID)
	v := RevealView{Schema: revealSchema, Open: st.Open, Title: st.Title, Cards: []RevealCard{}}
	if !st.Open {
		v.Teaser = st.Teaser
		return v
	}
	first, later := rv.open()
	v.Later, v.Face, v.FaceText = later, rv.table.FaceDown.Title, rv.table.FaceDown.Text
	rv.mu.Lock()
	defer rv.mu.Unlock()
	for _, c := range rv.cards {
		if !rv.visible(c.quality, first, later) {
			continue
		}
		if c.card.Revealed {
			v.Cards = append(v.Cards, rv.faceUp(c))
		} else {
			v.Cards = append(v.Cards, rv.face(c))
		}
	}
	return v
}

// Errors for Flip.
var (
	ErrNoCard      = errors.New("reveal: unknown card")
	ErrCardFlipped = errors.New("reveal: card already flipped")
)

// Flip turns a visible card face up. An id that is not a card the viewer can see (unknown, or hidden until the sets and
// uniques open) is ErrNoCard naming it, so ids cannot be probed.
func (rv *Reveal) Flip(id string) (RevealCard, error) {
	first, later := rv.open()
	rv.mu.Lock()
	defer rv.mu.Unlock()
	for _, c := range rv.cards {
		if c.card.ID != id || !rv.visible(c.quality, first, later) {
			continue
		}
		if c.card.Revealed {
			return rv.faceUp(c), fmt.Errorf("%w: %q", ErrCardFlipped, id)
		}
		c.card.Revealed = true
		return rv.faceUp(c), nil
	}
	return RevealCard{}, fmt.Errorf("%w %q", ErrNoCard, id)
}

// Handler serves GET /api/reveal.
func (rv *Reveal) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "GET only", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, 200, rv.View())
	})
}

type revealFlipBody struct {
	ID string `json:"id"`
}

type revealFlipReply struct {
	OK    bool        `json:"ok"`
	Code  string      `json:"code,omitempty"`
	Error string      `json:"error,omitempty"`
	Card  *RevealCard `json:"card,omitempty"`
}

// FlipHandler is POST /api/reveal/flip {id} on the answer listener, with the gate of the other unlock routes.
func (rv *Reveal) FlipHandler(o AnswerOptions) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !unlockPostGate(o, w, r) {
			return
		}
		var b revealFlipBody
		if err := decodeSmall(r, &b); err != nil || b.ID == "" {
			writeJSON(w, http.StatusBadRequest, revealFlipReply{Code: "body", Error: `body must be {"id": "<card>"}`})
			return
		}
		if st, _ := rv.u.state(IDRevealID); !st.Open {
			writeJSON(w, http.StatusNotFound, revealFlipReply{Code: "not-found", Error: ErrNoCard.Error() + " " + fmt.Sprintf("%q", b.ID)})
			return
		}
		c, err := rv.Flip(b.ID)
		switch {
		case err == nil:
			writeJSON(w, 200, revealFlipReply{OK: true, Card: &c})
		case errors.Is(err, ErrNoCard):
			writeJSON(w, http.StatusNotFound, revealFlipReply{Code: "not-found", Error: err.Error()})
		case errors.Is(err, ErrCardFlipped):
			writeJSON(w, http.StatusConflict, revealFlipReply{Code: "already-flipped", Error: err.Error(), Card: &c})
		default:
			writeJSON(w, http.StatusInternalServerError, revealFlipReply{Code: "internal", Error: err.Error()})
		}
	})
}
