package currency

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func newTally(t *testing.T, char string) (*Tally, *clock) {
	t.Helper()
	l, c := newLedger(t, "didle:0123abcd")
	ta, err := NewTally(l, DefaultTallyRows(), char, "r1")
	if err != nil {
		t.Fatal(err)
	}
	return ta, c
}

func bal(t *testing.T, ta *Tally) int64 {
	t.Helper()
	b, err := ta.L.Balance("tally")
	if err != nil {
		t.Fatal(err)
	}
	return b.Spendable
}

func TestDefaultTallyRowsValid(t *testing.T) { DefaultTallyRows() }

func TestKillsFirstsBossAndReplay(t *testing.T) {
	// Two class/monster sets stand for two builds: tally is account scoped and does not care which hero kills.
	for _, hero := range []string{"Sorc", "Pally"} {
		ta, _ := newTally(t, hero)
		run := func() {
			for i := 1; i <= 50; i++ {
				if _, err := ta.Credit(TallyEvent{Kind: EvKill, ID: fmt.Sprint("r1:", i), Subject: "Fallen", Class: "normal"}); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := ta.Credit(TallyEvent{Kind: EvKill, ID: "r1:51", Subject: "Blood Raven", Class: "boss"}); err != nil {
				t.Fatal(err)
			}
		}
		run()
		// 50 kills = 2 batches (2), first Fallen 5, first Blood Raven 5, boss 15 (the 51st boss kill has weight 0)
		if got := bal(t, ta); got != 27 {
			t.Fatalf("%s: balance %d, want 27", hero, got)
		}
		// Replay the whole journal into a fresh Tally on the same ledger: nothing is booked twice.
		ta2, err := NewTally(ta.L, ta.Rows, hero, "r1")
		if err != nil {
			t.Fatal(err)
		}
		ta = ta2
		run()
		if got := bal(t, ta); got != 27 {
			t.Fatalf("%s: replay changed the balance to %d", hero, got)
		}
	}
}

func TestOnceRowsAndTrip(t *testing.T) {
	ta, _ := newTally(t, "Sorc")
	for _, ev := range []TallyEvent{
		{EvQuest, "q1", "den_of_evil/0", ""}, {EvQuest, "q2", "den_of_evil/0", ""},
		{EvWaypoint, "w1", "Cold Plains", ""}, {EvArea, "a1", "Cold Plains", ""},
		{EvTrip, "t1", "1", ""}, {EvTrip, "t1", "1", ""}, {EvTrip, "t2", "2", ""},
	} {
		if _, err := ta.Credit(ev); err != nil {
			t.Fatal(err)
		}
	}
	if got := bal(t, ta); got != 10+10+5+2+2 {
		t.Errorf("balance %d, want 29", got)
	}
}

func TestCreditErrors(t *testing.T) {
	ta, _ := newTally(t, "Sorc")
	if _, err := ta.Credit(TallyEvent{Kind: EvKill, ID: "1", Subject: "x", Class: "champion"}); !errors.Is(err, ErrUnknownClass) || !strings.Contains(err.Error(), "champion") {
		t.Errorf("unknown class: %v", err)
	}
	if _, err := ta.Credit(TallyEvent{Kind: EvKill, Subject: "x", Class: "normal"}); err == nil {
		t.Error("an event without id was accepted")
	}
	if _, err := ta.Credit(TallyEvent{Kind: "dance", ID: "1", Subject: "x"}); err == nil {
		t.Error("unknown kind accepted")
	}
	van, _ := newLedger(t, VanillaMode)
	tv, err := NewTally(van, DefaultTallyRows(), "Sorc", "r1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tv.Credit(TallyEvent{Kind: EvQuest, ID: "1", Subject: "q"}); !errors.Is(err, ErrVanilla) {
		t.Errorf("vanilla: %v", err)
	}
}

func TestQuote(t *testing.T) {
	rows := DefaultTallyRows()
	for _, tc := range []struct {
		area  string
		level int
		price int64
	}{
		{"Cold Plains", 6, 0}, {"Cold Plains", 30, 0}, {"cold plains", 5, 20}, {"Cold Plains", 3, 20},
		{"Cold Plains", 2, 40}, {"Burial Grounds", 2, 40}, {"Burial Grounds", 1, 80}, {"Forgotten Tower", 1, 80},
	} {
		q, err := rows.Quote(tc.area, tc.level)
		if err != nil || q.Price != tc.price {
			t.Errorf("%s at %d: %+v %v, want price %d", tc.area, tc.level, q, err, tc.price)
		}
		if (q.Price > 0) != (q.Text != "") {
			t.Errorf("%s at %d: warning text %q does not match price", tc.area, tc.level, q.Text)
		}
	}
	if _, err := rows.Quote("Moo Moo Farm", 5); !errors.Is(err, ErrUnknownZone) || !strings.Contains(err.Error(), "Moo Moo Farm") {
		t.Errorf("unknown zone: %v", err)
	}
	if _, err := rows.Quote("Cold Plains", 0); err == nil {
		t.Error("level 0 accepted")
	}
}

func TestCharterSpend(t *testing.T) {
	ta, _ := newTally(t, "Sorc")
	q, _ := ta.Rows.Quote("Cold Plains", 5) // 20
	if _, err := ta.Charter(q, "order-1"); !errors.Is(err, ErrInsufficient) {
		t.Fatalf("empty tally: %v", err)
	}
	if bal(t, ta) != 0 {
		t.Fatal("a refused spend changed the balance")
	}
	if _, err := ta.L.Grant("tally", "grant", "g1", 25, "test"); err != nil {
		t.Fatal(err)
	}
	if _, err := ta.Charter(q, "order-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := ta.Charter(q, "order-1"); !errors.Is(err, ErrDuplicate) {
		t.Errorf("same order paid twice: %v", err)
	}
	if got := bal(t, ta); got != 5 {
		t.Errorf("balance %d, want 5", got)
	}
	if e, err := ta.Charter(Quote{}, ""); err != nil || e.Seq != 0 {
		t.Errorf("a free trip booked something: %+v %v", e, err)
	}
}

func TestTallyRowsErrorsNameTheRow(t *testing.T) {
	reg := Default()
	for _, tc := range []struct{ from, to, want string }{
		{`"source": "tally_kills"`, `"source": "nope"`, `source "nope"`},
		{`"event": "boss"`, `"event": "boss2"`, `no credit row for event "boss"`},
		{`"area": "Blood Moor", "min": 1, "max": 8`, `"area": "Blood Moor", "min": 9, "max": 8`, `zone "Blood Moor"`},
		{`"currency": "tally"`, `"currency": "gold"`, `unknown currency`},
		{`"mult": 2`, `"mult": 0`, `mult > 0`},
		{`currency.tallyrows/1`, `currency.tallyrows/2`, `schema`},
	} {
		src := string(defaultTallyJSON)
		if !strings.Contains(src, tc.from) {
			t.Fatalf("fixture %q missing", tc.from)
		}
		_, err := ParseTallyRows([]byte(strings.Replace(src, tc.from, tc.to, 1)), reg)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%q: err %v, want %q", tc.to, err, tc.want)
		}
	}
}

func TestSummaryForWindow(t *testing.T) {
	ta, c := newTally(t, "Sorc")
	start := c.t
	ta.Credit(TallyEvent{Kind: EvQuest, ID: "1", Subject: "den"})
	c.t = c.t.Add(time.Hour)
	mid := c.t
	ta.Credit(TallyEvent{Kind: EvWaypoint, ID: "2", Subject: "Cold Plains"})
	q, _ := ta.Rows.Quote("Cold Plains", 5)
	if _, err := ta.Charter(q, "o1"); err != nil {
		t.Fatal(err)
	}
	s, err := ta.SummaryFor(Window{From: mid})
	if err != nil {
		t.Fatal(err)
	}
	if s.Credited != 10 || s.Spent != 20 || s.Net != -10 || len(s.Lines) != 2 {
		t.Errorf("late window: %+v", s)
	}
	all, _ := ta.SummaryFor(Window{From: start, To: mid})
	if all.Credited != 10 || all.Spent != 0 || all.Balance != 0 {
		t.Errorf("early window: %+v", all)
	}
}
