package questboard

import (
	"fmt"
	"sort"
	"strings"

	questboarddata "d2lod-bot/data/questboard"
)

// Reason codes: a closed list, like the survival why codes. The board shows the sentence, never a number
// except where the game or the old level guide says one (design 3.2; the level in level_under is the legacy
// floor, shown because the user asked for it side by side).
const (
	ReasonLevelUnder    = "level_under"    // below the legacy floor for this class family
	ReasonNoMerc        = "no_merc"        // no hireling, or a dead one
	ReasonKitPotions    = "kit_potions"    // fewer healing potions than the kit minimum
	ReasonKitResist     = "kit_resist"     // lowest resist under the difficulty's minimum
	ReasonMarginLow     = "margin_low"     // the weakest encounter (boss or zone) is a close fight
	ReasonMarginWall    = "margin_wall"    // the weakest encounter would likely kill the hero
	ReasonOffenceUnread = "offence_unread" // the hero's kill speed is not read, the margin is discounted
	ReasonLegacyOnly    = "legacy_only"    // no margin data for this boss or zone: the old floor stands, warn never wall
	ReasonNoData        = "no_data"        // neither a margin nor a floor: unknown
)

// Thresholds is data/questboard/readiness.json.
type Thresholds struct {
	Schema      string              `json:"schema"`
	Dials       map[string]DialLine `json:"dials"`
	DefaultDial string              `json:"defaultDial"`
	Kit         KitMin              `json:"kit"`
	Tags        map[string]string   `json:"tags,omitempty"`
}

// DialLine is the margin line pair of one risk dial: Enter is the comfortable line, Stay the lowest line at
// which the encounter is still acceptable.
type DialLine struct {
	Enter float64 `json:"enter"`
	Stay  float64 `json:"stay"`
}

// KitMin is the kit floor that adds a reason (it never changes the word: items are inside the margin).
type KitMin struct {
	MinPotions int            `json:"minPotions"`
	MinResist  map[string]int `json:"minResist"`
}

// LoadThresholds parses and validates the embedded readiness data.
func LoadThresholds() (*Thresholds, error) { return ParseThresholds(questboarddata.Readiness) }

// ParseThresholds decodes strictly and checks every dial and difficulty is present and ordered.
func ParseThresholds(raw []byte) (*Thresholds, error) {
	var t Thresholds
	if err := strict(raw, &t); err != nil {
		return nil, fmt.Errorf("questboard: readiness: %w", err)
	}
	if t.Schema != "quests.readiness/1" {
		return nil, fmt.Errorf("questboard: readiness schema %q, want quests.readiness/1", t.Schema)
	}
	for _, d := range []string{"cautious", "balanced", "bold"} {
		l, ok := t.Dials[d]
		if !ok {
			return nil, fmt.Errorf("questboard: readiness: dial %q missing", d)
		}
		if l.Stay <= 0 || l.Enter < l.Stay {
			return nil, fmt.Errorf("questboard: readiness: dial %q needs 0 < stay <= enter", d)
		}
	}
	if _, ok := t.Dials[t.DefaultDial]; !ok {
		return nil, fmt.Errorf("questboard: readiness: defaultDial %q is not a dial", t.DefaultDial)
	}
	for _, d := range Difficulties {
		if _, ok := t.Kit.MinResist[d]; !ok {
			return nil, fmt.Errorf("questboard: readiness: kit.minResist lacks difficulty %q", d)
		}
	}
	return &t, nil
}

// Dial returns the lines of a dial; an unknown dial is an error naming it.
func (t *Thresholds) Dial(id string) (DialLine, error) {
	if id == "" {
		id = t.DefaultDial
	}
	l, ok := t.Dials[id]
	if !ok {
		ids := make([]string, 0, len(t.Dials))
		for k := range t.Dials {
			ids = append(ids, k)
		}
		sort.Strings(ids)
		return DialLine{}, fmt.Errorf("questboard: unknown dial %q (known: %s)", id, strings.Join(ids, ", "))
	}
	return l, nil
}

// Encounter is one margin the word rests on: the quest's boss, or the zone on the way. Best and Low are the
// survival margins (power.Result, corrected by the ledger k); Known false means no margin could be computed
// (an unread hero, a boss missing from the tables) and the encounter is ignored.
type Encounter struct {
	Name          string
	Kind          string // boss | zone
	Best, Low     float64
	Known         bool
	OffenceUnread bool
	// Area is the area a boss encounter stands in (0: not known); the bot keeps a parked boss chunk's area out of the
	// leveling picks with it.
	Area int
}

// Hero is the hero as readiness needs it (the adapter fills it from power.Hero).
type Hero struct {
	Class        string
	Level        int
	MercHired    bool
	MercAlive    bool
	Potions      int
	LowestResist int
	ResistKnown  bool
}

// Reason is one cause with its code and the sentence fragment it adds.
type Reason struct {
	Code string `json:"code"`
	Text string `json:"text"`
}

// Debug is the margin behind the word, for tools. The UI never renders it (design 3.2).
type Debug struct {
	Encounter string  `json:"encounter"`
	Best      float64 `json:"best"`
	Low       float64 `json:"low"`
	Enter     float64 `json:"enter"`
	Stay      float64 `json:"stay"`
}

// Readiness is the API-ready answer for one chunk: the word, the reasons, the old floor beside it and the
// human sentence. Unknown keeps the chunk `available`, never offered (S1 semantics: Word is empty).
type Readiness struct {
	Word        Word     `json:"word"`
	Reasons     []Reason `json:"reasons"`
	LegacyFloor int      `json:"legacyFloor"` // 0 = none for this quest and family
	Unknown     bool     `json:"unknown"`
	FromLegacy  bool     `json:"fromLegacy"` // no margin data: auto-start must not trust this word
	Sentence    string   `json:"sentence"`
	Debug       *Debug   `json:"debug,omitempty"`
}

// AssessInput is everything Assess reads.
type AssessInput struct {
	Dial       string // "" = the default dial
	Diff       int
	Hero       Hero
	Encounters []Encounter
	Floor      int // legacy floor for the hero's class family
	HasFloor   bool
	// LegacyWall: levels under the floor at which a floor-only judgment reads too hard (0: warn only, never too hard).
	LegacyWall int
}

// Assess is a pure function: it never changes zone choice, writes nothing and reads no game.
func (t *Thresholds) Assess(in AssessInput) (Readiness, error) {
	line, err := t.Dial(in.Dial)
	if err != nil {
		return Readiness{}, err
	}
	if in.Diff < 0 || in.Diff >= len(Difficulties) {
		return Readiness{}, fmt.Errorf("questboard: difficulty index %d out of range", in.Diff)
	}
	r := Readiness{}
	if in.HasFloor {
		r.LegacyFloor = in.Floor
	}

	var worst *Encounter
	worstRank := -1
	unread := false
	for i := range in.Encounters {
		e := &in.Encounters[i]
		if !e.Known {
			continue
		}
		if e.OffenceUnread {
			unread = true
		}
		if rk := rank(wordOf(e.Best, e.Low, line)); rk > worstRank || (rk == worstRank && worst != nil && e.Best < worst.Best) {
			worst, worstRank = e, rk
		}
	}

	switch {
	case worst != nil:
		r.Word = wordOf(worst.Best, worst.Low, line)
		r.Debug = &Debug{Encounter: worst.Name, Best: worst.Best, Low: worst.Low, Enter: line.Enter, Stay: line.Stay}
	case in.HasFloor:
		// Warn, never wall (ADR 0011): the old floor says risky under it, and its safe is not trusted for auto-start. With LegacyWall a
		// hero further under the floor than that reads too hard (a clvl 4 Paladin started the Cain scroll at floor 9 and died).
		r.FromLegacy = true
		r.Word = ReadySafe
		if in.Hero.Level < in.Floor {
			r.Word = ReadyRisky
			if in.LegacyWall > 0 && in.Hero.Level < in.Floor-in.LegacyWall {
				r.Word = ReadyTooHard
			}
		}
	default:
		r.Unknown = true
		r.Reasons = []Reason{{ReasonNoData, "No margin and no level guide for this one."}}
		r.Sentence = "Readiness is not known yet."
		return r, nil
	}

	r.Reasons = t.reasons(in, r, worst, unread)
	r.Sentence = sentence(r, in, worst)
	return r, nil
}

func wordOf(best, low float64, l DialLine) Word {
	switch {
	case low >= l.Enter:
		return ReadySafe
	case best >= l.Stay:
		return ReadyRisky
	}
	return ReadyTooHard
}

func rank(w Word) int {
	switch w {
	case ReadySafe:
		return 0
	case ReadyRisky:
		return 1
	}
	return 2
}

// reasons lists the causes, fixable ones first (a merc, kit), then the level, then the margin itself.
func (t *Thresholds) reasons(in AssessInput, r Readiness, worst *Encounter, unread bool) []Reason {
	var out []Reason
	h := in.Hero
	if !h.MercHired || !h.MercAlive {
		txt := "have no merc"
		if h.MercHired {
			txt = "have a dead merc"
		}
		out = append(out, Reason{ReasonNoMerc, txt})
	}
	if h.Potions < t.Kit.MinPotions {
		out = append(out, Reason{ReasonKitPotions, "are short of potions"})
	}
	if h.ResistKnown && h.LowestResist < t.Kit.MinResist[Difficulties[in.Diff]] {
		out = append(out, Reason{ReasonKitResist, "have a weak resist"})
	}
	if in.HasFloor && h.Level < in.Floor {
		out = append(out, Reason{ReasonLevelUnder, fmt.Sprintf("are %s under the recommended %d", levels(in.Floor-h.Level), in.Floor)})
	}
	if worst != nil && r.Word == ReadyRisky {
		out = append(out, Reason{ReasonMarginLow, fmt.Sprintf("face a close fight against %s", worst.Name)})
	}
	if worst != nil && r.Word == ReadyTooHard {
		out = append(out, Reason{ReasonMarginWall, fmt.Sprintf("would likely lose to %s", worst.Name)})
	}
	if unread {
		out = append(out, Reason{ReasonOffenceUnread, "have an unread attack, so the guess is cautious"})
	}
	if r.FromLegacy {
		out = append(out, Reason{ReasonLegacyOnly, "are judged by the old level guide only"})
	}
	return out
}

func levels(n int) string {
	switch n {
	case 1:
		return "one level"
	case 2:
		return "two levels"
	case 3:
		return "three levels"
	}
	return fmt.Sprintf("%d levels", n)
}

// sentence is the human line: 'You are two levels under the recommended 17 and have no merc: risky'.
func sentence(r Readiness, in AssessInput, worst *Encounter) string {
	word := strings.ReplaceAll(string(r.Word), "_", " ")
	if len(r.Reasons) == 0 {
		if worst != nil {
			return fmt.Sprintf("You out-fight %s comfortably: %s", worst.Name, word)
		}
		return "You are ready: " + word
	}
	parts := make([]string, len(r.Reasons))
	for i, rs := range r.Reasons {
		parts[i] = rs.Text
	}
	return "You " + joinAnd(parts) + ": " + word
}

func joinAnd(p []string) string {
	// "are X" then "have Y": keep each verb, join with commas and a final "and".
	switch len(p) {
	case 1:
		return p[0]
	case 2:
		return p[0] + " and " + p[1]
	}
	return strings.Join(p[:len(p)-1], ", ") + " and " + p[len(p)-1]
}
