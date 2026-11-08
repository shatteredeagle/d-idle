package currency

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Tally rows: how observed events credit tally and what a charter costs. Everything here is data in tallyrows.json;
// the sources they pay through are rows of registry.json. Values are the design's guesses (U), see docs/tally-strip.md.

const (
	TallyRowsSchema = "currency.tallyrows/1"

	EvKill     = "kill"     // Subject: monster name, Class: a class of the kill weights table
	EvArea     = "area"     // Subject: the area entered
	EvQuest    = "quest"    // Subject: quest id with difficulty; send it once the quest is done
	EvWaypoint = "waypoint" // Subject: the area whose waypoint was lit
	EvTrip     = "trip"     // Subject: a trip label; the hero came back to town
)

var (
	ErrUnknownZone   = errors.New("tally: area is not in the charter zone table")
	ErrUnknownClass  = errors.New("tally: unknown monster class")
	ErrCharterNeeded = errors.New("tally: this trip needs a charter")
)

type KillRow struct {
	Source  string `json:"source"`
	Per     int64  `json:"per"`    // weighted kills per credit
	Amount  int64  `json:"amount"` // tally per credit
	Weights []struct {
		Class  string `json:"class"`
		Weight int64  `json:"weight"`
	} `json:"weights"`
}

// CreditRow pays Amount tally once per account for the event's subject, except Event "trip" which pays per trip.
type CreditRow struct {
	Event  string `json:"event"`
	Source string `json:"source"`
	Amount int64  `json:"amount"`
	Why    string `json:"why"` // %s is the subject
}

type CharterTier struct {
	UpToAbove int64 `json:"up_to_above"` // applies while the hero is at most this many levels below the zone's minimum
	Mult      int64 `json:"mult"`
}

type CharterRow struct {
	Base  int64         `json:"base"`
	Tiers []CharterTier `json:"tiers"`
}

// TripFloor is the least activity that makes an expedition pay its "trip" row: the trip pays when ANY one field is
// reached (0 = that field is not a reason). A walk in and out with nothing done pays nothing.
type TripFloor struct {
	Kills int64 `json:"kills"`
	Drops int64 `json:"drops"`
	XP    int64 `json:"xp"`
}

func (f TripFloor) met(kills, drops, xp int64) bool {
	return f.Kills > 0 && kills >= f.Kills || f.Drops > 0 && drops >= f.Drops || f.XP > 0 && xp >= f.XP
}

// Zone is the recommended clvl range of an area.
type Zone struct {
	Area string `json:"area"`
	Min  int    `json:"min"`
	Max  int    `json:"max"`
}

// TallyRows is the validated tally table.
type TallyRows struct {
	Schema    string      `json:"schema"`
	Currency  string      `json:"currency"`
	Kill      KillRow     `json:"kill"`
	Credits   []CreditRow `json:"credits"`
	TripFloor TripFloor   `json:"trip_floor"`
	Charter   CharterRow  `json:"charter"`
	Zones     []Zone      `json:"zones"`
}

//go:embed tallyrows.json
var defaultTallyJSON []byte

// DefaultTallyRows is the table shipped with the bot, checked against the shipped registry.
func DefaultTallyRows() *TallyRows {
	r, err := ParseTallyRows(defaultTallyJSON, Default())
	if err != nil {
		panic("currency: embedded tallyrows.json is invalid: " + err.Error())
	}
	return r
}

// ParseTallyRows decodes (unknown fields are an error) and validates against a registry. Messages name the row.
func ParseTallyRows(b []byte, reg *Registry) (*TallyRows, error) {
	var t TallyRows
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&t); err != nil {
		return nil, fmt.Errorf("tally rows: %w", err)
	}
	return &t, t.Validate(reg)
}

func (t *TallyRows) Validate(reg *Registry) error {
	if t.Schema != TallyRowsSchema {
		return fmt.Errorf("tally rows: schema %q, want %q", t.Schema, TallyRowsSchema)
	}
	c, err := reg.Get(t.Currency)
	if err != nil {
		return fmt.Errorf("tally rows: %w", err)
	}
	if c.Storage != StorageLedger {
		return fmt.Errorf("tally rows: currency %q is not a ledger currency", t.Currency)
	}
	src := func(row, id string) error {
		s, ok := reg.Source(id)
		if !ok || s.Currency != t.Currency {
			return fmt.Errorf("tally rows: %s: source %q is missing from the registry or pays another currency", row, id)
		}
		return nil
	}
	if err := src("kill", t.Kill.Source); err != nil {
		return err
	}
	if t.Kill.Per <= 0 || t.Kill.Amount <= 0 || len(t.Kill.Weights) == 0 {
		return errors.New("tally rows: kill: needs per > 0, amount > 0 and a weights table")
	}
	for _, w := range t.Kill.Weights {
		if w.Class == "" || w.Weight < 0 {
			return fmt.Errorf("tally rows: kill weight %q: bad class or negative weight", w.Class)
		}
	}
	seen := map[string]bool{}
	for _, r := range t.Credits {
		if seen[r.Event] || r.Amount <= 0 || !strings.Contains(r.Why, "%s") {
			return fmt.Errorf("tally rows: credit %q: duplicate, non-positive amount or a why without %%s", r.Event)
		}
		seen[r.Event] = true
		if err := src("credit "+r.Event, r.Source); err != nil {
			return err
		}
	}
	for _, need := range []string{"first_monster", "first_area", "boss", "quest", "waypoint", "trip"} {
		if !seen[need] {
			return fmt.Errorf("tally rows: no credit row for event %q", need)
		}
	}
	if f := t.TripFloor; f.Kills < 0 || f.Drops < 0 || f.XP < 0 || f.Kills+f.Drops+f.XP == 0 {
		return errors.New("tally rows: trip_floor: needs at least one positive field and no negative one")
	}
	if t.Charter.Base <= 0 || len(t.Charter.Tiers) == 0 {
		return errors.New("tally rows: charter: needs base > 0 and tiers")
	}
	var last int64
	for _, ti := range t.Charter.Tiers {
		if ti.UpToAbove <= last || ti.Mult <= 0 {
			return fmt.Errorf("tally rows: charter tier up_to_above %d: must rise and have mult > 0", ti.UpToAbove)
		}
		last = ti.UpToAbove
	}
	zs := map[string]bool{}
	for _, z := range t.Zones {
		if z.Area == "" || zs[strings.ToLower(z.Area)] || z.Min < 1 || z.Max < z.Min {
			return fmt.Errorf("tally rows: zone %q: empty, duplicate or a bad range %d-%d", z.Area, z.Min, z.Max)
		}
		zs[strings.ToLower(z.Area)] = true
	}
	if len(t.Zones) == 0 {
		return errors.New("tally rows: no zones")
	}
	return nil
}

func (t *TallyRows) credit(event string) CreditRow {
	for _, r := range t.Credits {
		if r.Event == event {
			return r
		}
	}
	return CreditRow{}
}

// Quote is the price of a trip to an area for a hero of a level.
type Quote struct {
	Area  string `json:"area"`
	Min   int    `json:"min"`
	Max   int    `json:"max"`
	Level int    `json:"level"`
	Above int    `json:"above"` // levels the hero is below the zone's minimum; 0 = in range or over-levelled
	Price int64  `json:"price"` // 0 = free
	Text  string `json:"text"`  // the hero's warning; empty when free
}

// Quote prices a trip. In range (or over-levelled) is free. An area missing from the table is an error naming it.
func (t *TallyRows) Quote(area string, level int) (Quote, error) {
	if level < 1 {
		return Quote{}, fmt.Errorf("tally: hero level %d is not known yet", level)
	}
	for _, z := range t.Zones {
		if !strings.EqualFold(z.Area, strings.TrimSpace(area)) {
			continue
		}
		q := Quote{Area: z.Area, Min: z.Min, Max: z.Max, Level: level}
		if level >= z.Min {
			return q, nil
		}
		q.Above = z.Min - level
		for _, ti := range t.Charter.Tiers {
			if int64(q.Above) <= ti.UpToAbove {
				q.Price = t.Charter.Base * ti.Mult
				break
			}
		}
		if q.Price == 0 { // beyond the last tier: the last tier prices it, never a free trip
			q.Price = t.Charter.Base * t.Charter.Tiers[len(t.Charter.Tiers)-1].Mult
		}
		q.Text = fmt.Sprintf("I am level %d and %s is meant for level %d to %d. A charter costs %d tally; I will be careful, but I think it is too hard for me yet.",
			level, z.Area, z.Min, z.Max, q.Price)
		return q, nil
	}
	return Quote{}, fmt.Errorf("%w: %q", ErrUnknownZone, area)
}

// TallyEvent is an observed event reduced to what tally needs. ID must be unique and stable for the event (for
// example run plus the event's sequence number): replaying the same journal then books nothing twice.
type TallyEvent struct {
	Kind, ID, Subject, Class string
	// Activity of the trip (EvTrip only): kills, drops and XP seen since the hero left town.
	Kills, Drops, XP int64
}

// Tally credits and spends the tally of one profile. Safe for concurrent use.
type Tally struct {
	L         *Ledger
	Rows      *TallyRows
	Character string
	Run       string

	mu      sync.Mutex
	seen    map[string]bool // keys already in the ledger, so a kill does not re-read the whole file
	kills   int64           // weighted kills since this process started
	counted map[string]bool // kill event ids already counted, so a replay in this process adds nothing
}

// NewTally seeds the booked keys from the ledger.
func NewTally(l *Ledger, rows *TallyRows, character, run string) (*Tally, error) {
	if l == nil || rows == nil {
		return nil, errors.New("tally: needs a ledger and rows")
	}
	if err := rows.Validate(l.Registry()); err != nil {
		return nil, err
	}
	es, err := l.Entries()
	if err != nil {
		return nil, err
	}
	t := &Tally{L: l, Rows: rows, Character: character, Run: run, seen: map[string]bool{}, counted: map[string]bool{}}
	for _, e := range es {
		if e.Key != "" {
			t.seen[e.Key] = true
		}
	}
	return t, nil
}

type candidate struct {
	key, source, why string
	amount           int64
}

// Credit books what the event earns and returns the new entries. Duplicates are skipped silently (that is the
// idempotency); any other ledger error is returned. Vanilla mode returns ErrVanilla.
func (t *Tally) Credit(ev TallyEvent) ([]Entry, error) {
	if ev.ID == "" || ev.Subject == "" {
		return nil, fmt.Errorf("tally: event %q needs an id and a subject", ev.Kind)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	var cs []candidate
	once := func(event, keyPart string) {
		r := t.Rows.credit(event)
		cs = append(cs, candidate{"first/" + keyPart, r.Source, fmt.Sprintf(r.Why, ev.Subject), r.Amount})
	}
	switch ev.Kind {
	case EvKill:
		w := int64(-1)
		for _, x := range t.Rows.Kill.Weights {
			if x.Class == ev.Class {
				w = x.Weight
			}
		}
		if w < 0 {
			return nil, fmt.Errorf("%w: %q (kill weights table)", ErrUnknownClass, ev.Class)
		}
		once("first_monster", "monster/"+ev.Subject)
		if ev.Class == "boss" {
			r := t.Rows.credit("boss")
			cs = append(cs, candidate{"boss/" + ev.Subject, r.Source, fmt.Sprintf(r.Why, ev.Subject), r.Amount})
		}
		if t.counted[ev.ID] {
			break
		}
		t.counted[ev.ID] = true
		before := t.kills / t.Rows.Kill.Per
		t.kills += w
		if t.kills/t.Rows.Kill.Per > before {
			cs = append(cs, candidate{"kills/" + t.Character + "/" + ev.ID, t.Rows.Kill.Source,
				fmt.Sprintf("%d kills", t.Rows.Kill.Per), t.Rows.Kill.Amount})
		}
	case EvArea:
		once("first_area", "area/"+ev.Subject)
	case EvQuest:
		r := t.Rows.credit("quest")
		cs = append(cs, candidate{"quest/" + ev.Subject, r.Source, fmt.Sprintf(r.Why, ev.Subject), r.Amount})
	case EvWaypoint:
		r := t.Rows.credit("waypoint")
		cs = append(cs, candidate{"waypoint/" + ev.Subject, r.Source, fmt.Sprintf(r.Why, ev.Subject), r.Amount})
	case EvTrip:
		if !t.Rows.TripFloor.met(ev.Kills, ev.Drops, ev.XP) {
			break
		}
		r := t.Rows.credit("trip")
		cs = append(cs, candidate{"trip/" + t.Character + "/" + ev.ID, r.Source, fmt.Sprintf(r.Why, ev.Subject), r.Amount})
	default:
		return nil, fmt.Errorf("tally: unknown event kind %q", ev.Kind)
	}
	var out []Entry
	for _, c := range cs {
		if t.seen[c.key] {
			continue
		}
		e, err := t.L.Append(Entry{Kind: KindCredit, Currency: t.Rows.Currency, Source: c.source, Key: c.key,
			Amount: c.amount, Character: t.Character, Run: t.Run, Note: c.why})
		if err != nil && !errors.Is(err, ErrDuplicate) {
			return out, err
		}
		t.seen[c.key] = true
		if err == nil {
			out = append(out, e)
		}
	}
	return out, nil
}

// Charter pays for a trip: it books the quote's price as a spend under key (use the order id), refusing a short
// balance. A free quote books nothing. Call it only after the order is placed.
func (t *Tally) Charter(q Quote, key string) (Entry, error) {
	if q.Price == 0 {
		return Entry{}, nil
	}
	if key == "" {
		return Entry{}, errors.New("tally: a charter needs a key")
	}
	return t.L.Append(Entry{Kind: KindSpend, Currency: t.Rows.Currency, Amount: q.Price, Key: "charter/" + key,
		Upgrade: "charter", Character: t.Character, Run: t.Run, Note: "charter for " + q.Area})
}

// Window is a half-open time range [From, To); a zero To means open-ended.
type Window struct{ From, To time.Time }

// Line is one ledger movement in a summary.
type Line struct {
	TS     string `json:"ts"`
	Amount int64  `json:"amount"` // negative for a spend
	Why    string `json:"why"`
}

// Summary is what the strip and the return screen show for a window.
type Summary struct {
	Balance  int64  `json:"balance"`
	Sealed   int64  `json:"sealed"`
	Credited int64  `json:"credited"`
	Spent    int64  `json:"spent"`
	Net      int64  `json:"net"`
	Lines    []Line `json:"lines"`
}

// SummaryFor totals the movements of the window (ledger time stamps have second resolution). The balance is the
// current spendable balance, not the balance at the window's end.
func (t *Tally) SummaryFor(w Window) (Summary, error) {
	s := Summary{Lines: []Line{}}
	b, err := t.L.Balance(t.Rows.Currency)
	if err != nil {
		return s, err
	}
	s.Balance, s.Sealed = b.Spendable, b.Sealed
	es, err := t.L.Entries()
	if err != nil {
		return s, err
	}
	for _, e := range es {
		if e.Currency != t.Rows.Currency {
			continue
		}
		ts, err := time.Parse(time.RFC3339, e.TS)
		if err != nil || ts.Before(w.From) || (!w.To.IsZero() && !ts.Before(w.To)) {
			continue
		}
		switch e.Kind {
		case KindCommit, KindCredit, KindGrant:
			s.Credited += e.Amount - e.Sealed
			s.Lines = append(s.Lines, Line{e.TS, e.Amount - e.Sealed, e.Note})
		case KindSpend:
			s.Spent += e.Amount
			s.Lines = append(s.Lines, Line{e.TS, -e.Amount, e.Note})
		}
	}
	s.Net = s.Credited - s.Spent
	return s, nil
}
