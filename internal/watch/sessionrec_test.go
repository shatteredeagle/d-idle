package watch

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/decide"
)

type sessRig struct {
	t    *testing.T
	cat  *decide.Catalogue
	st   *decide.Store
	clk  *decide.ManualClock
	wall time.Time
	v    Vitals
	view int
	ords []Order
	s    *Session
	cls  string
}

const secretTok = "tok-SECRET-0123456789abcdef"

func newSessRig(t *testing.T, class string) *sessRig {
	t.Helper()
	cat, err := decide.Shipped()
	if err != nil {
		t.Fatal(err)
	}
	r := &sessRig{t: t, cat: cat, clk: &decide.ManualClock{}, wall: time.Date(2026, 11, 8, 10, 0, 0, 0, time.UTC), cls: class}
	if r.st, err = decide.New(cat, "human", r.clk); err != nil {
		t.Fatal(err)
	}
	r.s, err = NewSession(SessionConfig{Root: t.TempDir(), Profile: "human", Commit: "abc1234", CatalogueVersion: cat.Version,
		Store: r.st, Cat: cat, Clock: r.clk, Now: func() time.Time { return r.wall }, Rand: bytes.NewReader([]byte("12345678abcdefgh")),
		Identity: func() (string, string) { return class, "build-" + class },
		Vitals:   func() Vitals { return r.v }, Viewers: func() int { return r.view }, Orders: func() []Order { return r.ords },
		Secrets: []string{secretTok}})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// advance moves both clocks and polls once a second like the launcher does.
func (r *sessRig) advance(d time.Duration) {
	for done := time.Duration(0); done < d; done += time.Second {
		r.clk.Advance(time.Second)
		r.wall = r.wall.Add(time.Second)
		if _, err := r.st.Tick(); err != nil {
			r.t.Fatal(err)
		}
		r.s.Poll()
	}
}

func (r *sessRig) ask(gold float64) decide.Decision {
	r.t.Helper()
	ctx, err := r.cat.Build("t1.repair", decide.Obs{Class: r.cls, Num: map[string]float64{"durability": 25, "gold": gold},
		Cat: map[string]string{"act": "1", "diff": "normal", "hc": "normal"}}, 1, nil)
	if err != nil {
		r.t.Fatal(err)
	}
	d, err := r.st.Ask(ctx)
	if err != nil {
		r.t.Fatal(err)
	}
	return d
}

func readLines(t *testing.T, path string) []map[string]any {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var out []map[string]any
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var m map[string]any
		if err := json.Unmarshal(sc.Bytes(), &m); err != nil {
			t.Fatalf("bad line %q: %v", sc.Text(), err)
		}
		out = append(out, m)
	}
	return out
}

func kinds(ls []map[string]any) string {
	var k []string
	for _, l := range ls {
		k = append(k, l["kind"].(string))
	}
	return strings.Join(k, ",")
}

// script plays the same 25 minutes for both builds: three questions, one answered, one faded, one timeline of orders.
func (r *sessRig) script() {
	r.v = Vitals{LifePct: 100, XP: 0}
	r.s.Poll()
	r.advance(30 * time.Second)
	d := r.ask(1000) // opens thought 1 (default acted on)
	if d.Thought == nil {
		r.t.Fatal("no thought")
	}
	r.view = 1
	r.advance(10 * time.Second)
	r.v = Vitals{LifePct: 40, XP: 1000, Credits: 2}
	if err := r.st.Answer(d.Thought.ID, d.Answer); err != nil {
		r.t.Fatal(err)
	}
	r.advance(5 * time.Second)
	r.ords = []Order{{ID: 1, Kind: OrderGo, Area: "Cold Plains", Minutes: 20}}
	r.advance(2 * time.Second)
	d2 := r.ask(9000) // a second key: opens thought 2, never answered, fades
	r.s.OnDeath()
	r.advance(5 * time.Minute) // fade (45 s), notes and the outcome windows run
	_ = d2
	if err := r.s.SaveSessionNotes("the first ten minutes were fun " + secretTok); err != nil {
		r.t.Fatal(err)
	}
	r.ords = append(r.ords, Order{ID: 2, Kind: OrderHome})
	r.v = Vitals{LifePct: 80, XP: 3600, Credits: 5}
	r.advance(12 * time.Minute)
	r.view = 0
	r.advance(10 * time.Second)
}

func TestSessionRecordAndSummary(t *testing.T) {
	for _, class := range []string{"sorceress", "paladin"} {
		t.Run(class, func(t *testing.T) {
			r := newSessRig(t, class)
			r.script()
			if err := r.s.Stop(); err != nil {
				t.Fatal(err)
			}
			if err := r.s.Stop(); err != nil { // idempotent
				t.Fatal(err)
			}
			rec := readLines(t, filepath.Join(r.s.Dir(), "session.jsonl"))
			if rec[0]["kind"] != "start" || rec[0]["class"] != class || rec[len(rec)-1]["kind"] != "stop" {
				t.Fatalf("record: %s", kinds(rec))
			}
			var sum Summary
			b, err := os.ReadFile(filepath.Join(r.s.Dir(), "summary.json"))
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(b, &sum); err != nil {
				t.Fatal(err)
			}
			if sum.Questions != 2 || sum.Answered != 1 || sum.Faded != 1 || sum.Orders != 2 || sum.Deaths != 1 {
				t.Errorf("counts: %+v", sum)
			}
			if got := sum.QuestionsPer10Min; len(got) != 2 || got[0] != 2 || got[1] != 0 {
				t.Errorf("questions per 10 min: %v", got)
			}
			if sum.LatencyMedianMS == nil || *sum.LatencyMedianMS != 10000 {
				t.Errorf("latency: %v", sum.LatencyMedianMS)
			}
			if sum.SecBetweenOrders == nil || *sum.SecBetweenOrders < 300 {
				t.Errorf("between orders: %v", sum.SecBetweenOrders)
			}
			if sum.StayedInTabMin < 17 || sum.StayedInTabMin > 18 {
				t.Errorf("in tab: %v", sum.StayedInTabMin)
			}
			if sum.Credits != 5 || sum.XPPerH <= 0 || sum.Class != class || sum.Commit != "abc1234" {
				t.Errorf("summary: %+v", sum)
			}
			if len(sum.First10) == 0 || sum.First10[0].TS > 600 {
				t.Errorf("timeline: %+v", sum.First10)
			}
			if !strings.Contains(sum.Notes, "[removed]") {
				t.Errorf("notes not scrubbed: %q", sum.Notes)
			}
			if md, _ := os.ReadFile(filepath.Join(r.s.Dir(), "summary.md")); !strings.Contains(string(md), "Questions: 2 asked") {
				t.Errorf("markdown:\n%s", md)
			}
			// the token is in none of the files
			ents, _ := os.ReadDir(r.s.Dir())
			for _, e := range ents {
				if f, _ := os.ReadFile(filepath.Join(r.s.Dir(), e.Name())); bytes.Contains(f, []byte(secretTok)) || bytes.Contains(f, []byte("SECRET")) {
					t.Errorf("secret in %s", e.Name())
				}
			}
		})
	}
}

func TestDatasetSchemaAndOutcomeJoin(t *testing.T) {
	r := newSessRig(t, "sorceress")
	r.script()
	if err := r.s.Stop(); err != nil {
		t.Fatal(err)
	}
	ls := readLines(t, filepath.Join(r.s.Dir(), "decisions.jsonl"))
	if ls[0]["kind"] != "header" || ls[0]["session"] != "3132333435363738" || ls[0]["v"].(float64) != 1 {
		t.Fatalf("header: %v", ls[0])
	}
	fires := map[int]map[string]any{}
	out := map[int]map[string]any{}
	ans := map[int]map[string]any{}
	for _, l := range ls[1:] {
		if l["v"].(float64) != 1 {
			t.Errorf("v: %v", l)
		}
		switch l["kind"] {
		case "fire":
			fires[int(l["id"].(float64))] = l
		case "outcome":
			out[int(l["of"].(float64))] = l
		case "answer":
			ans[int(l["of"].(float64))] = l
		}
	}
	if len(fires) != 2 {
		t.Fatalf("fires: %d", len(fires))
	}
	f := fires[1]
	for _, k := range []string{"id", "session", "commit", "t_game", "t_wall", "point", "ctx", "class", "build", "options", "default", "answer", "by", "latency_ms"} {
		if _, ok := f[k]; !ok {
			t.Errorf("fire misses %s: %v", k, f)
		}
	}
	if f["point"] != "t1.repair" || f["by"] != "default" || f["class"] != "sorceress" || f["build"] != "build-sorceress" ||
		f["ctx"].(map[string]any)["act"] != "1" || f["t_game"].(float64) != 30000+0 {
		t.Errorf("fire 1: %v", f)
	}
	if a := ans[1]; a["by"] != "player" || a["latency_ms"].(float64) != 10000 {
		t.Errorf("answer 1: %v", a)
	}
	if a := ans[2]; a["by"] != "timeout" || a["latency_ms"] != nil {
		t.Errorf("answer 2: %v", a)
	}
	// outcome lines reference the fire, never rewrite it; both windows are over (10 min) with the third firing's none
	for id := range fires {
		o, ok := out[id]
		if !ok {
			t.Fatalf("no outcome for fire %d", id)
		}
		if o["window_min"].(float64) != 10 || o["life_pct_min"] == nil || o["deaths"] == nil {
			t.Errorf("outcome %d: %v", id, o)
		}
	}
	if out[1]["deaths"].(float64) != 1 || out[1]["life_pct_min"].(float64) != 40 {
		t.Errorf("outcome 1: %v", out[1])
	}
}

func TestDatasetRepeatLines(t *testing.T) {
	r := newSessRig(t, "paladin")
	r.v = Vitals{LifePct: 100}
	r.s.Poll()
	d := r.ask(1000)
	_ = d
	for i := 0; i < 5; i++ { // repeats of the same key inside the dedupe window are counted, not logged
		r.advance(time.Second)
		r.ask(1000)
	}
	if err := r.s.Stop(); err != nil {
		t.Fatal(err)
	}
	var sum Summary
	b, _ := os.ReadFile(filepath.Join(r.s.Dir(), "summary.json"))
	_ = json.Unmarshal(b, &sum)
	if sum.FiresLogged != 1 || sum.FiresSuppressed != 5 || sum.Questions != 1 {
		t.Errorf("dedupe: %+v", sum)
	}
}

func TestExportScrubs(t *testing.T) {
	r := newSessRig(t, "sorceress")
	r.script()
	if err := r.s.Stop(); err != nil {
		t.Fatal(err)
	}
	// a hostile line: names, a path, an address, a token and an unknown field
	f, _ := os.OpenFile(filepath.Join(r.s.Dir(), "decisions.jsonl"), os.O_APPEND|os.O_WRONLY, 0o600)
	f.WriteString(`{"v":1,"kind":"fire","id":99,"session":"x","commit":"abc1234","t_wall":"2026-11-08T10:01:00Z","point":"t1.repair","ctx":{"act":"1","zed":"Zedname"},"class":"sorceress","build":"/home/jesse/x","options":["a","me@host"],"default":"a","answer":"a","by":"default","account":"jesse","note":"hello"}` + "\n")
	f.Close()
	var buf bytes.Buffer
	n, err := ExportSession(r.s.Dir(), &buf, ExportOptions{Names: []string{"Zedname", "human"}, Rand: bytes.NewReader([]byte("ABCDEFGHijklmnop"))})
	if err != nil {
		t.Fatal(err)
	}
	if n < 3 {
		t.Errorf("scrubbed %d", n)
	}
	out := buf.String()
	for _, bad := range []string{"jesse", "Zedname", "/home", "@host", "3132333435363738", "human", "note", "account", secretTok, "t_wall", "2026-11-08"} {
		if strings.Contains(out, bad) {
			t.Errorf("export leaks %q:\n%s", bad, out)
		}
	}
	if !strings.Contains(out, `"t_off_ms"`) || !strings.Contains(out, "4142434445464748") {
		t.Errorf("export lacks offsets or the new session id:\n%s", out)
	}
	var j struct{ V int }
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		if err := json.Unmarshal([]byte(l), &j); err != nil {
			t.Fatal(err)
		}
	}
	// a newer dataset version is refused
	os.WriteFile(filepath.Join(r.s.Dir(), "decisions.jsonl"), []byte(`{"v":2,"kind":"header"}`+"\n"), 0o600)
	if _, err := ExportSession(r.s.Dir(), &bytes.Buffer{}, ExportOptions{}); err == nil {
		t.Error("v2 accepted")
	}
}

// TestDatasetSizePerHour measures the dataset of a busy simulated hour (one question every 20 s, a quarter answered):
// the number docs/research/decision-dataset.md quotes (S: fixture, not a live session).
func TestDatasetSizePerHour(t *testing.T) {
	r := newSessRig(t, "sorceress")
	r.v = Vitals{LifePct: 100}
	r.s.Poll()
	for i := 0; i < 180; i++ {
		d := r.ask(float64(500 + i*37%9000))
		r.advance(20 * time.Second)
		if i%4 == 0 && d.Thought != nil {
			_ = r.st.Answer(d.Thought.ID, d.Answer)
		}
	}
	if err := r.s.Stop(); err != nil {
		t.Fatal(err)
	}
	var sum Summary
	b, _ := os.ReadFile(filepath.Join(r.s.Dir(), "summary.json"))
	_ = json.Unmarshal(b, &sum)
	t.Logf("fires %d, dataset %d bytes in %.0f min = %.0f bytes/h, record %d bytes", sum.FiresLogged, sum.DecisionBytes, sum.DurationS/60, sum.DecisionBytesPerH, sum.RecordBytes)
	if sum.DecisionBytesPerH <= 0 || sum.DecisionBytesPerH > 400_000 {
		t.Errorf("bytes per hour: %.0f", sum.DecisionBytesPerH)
	}
}

// TestSummaryAndDatasetGolden pins the files of the scripted session byte for byte (SESSION_GOLDEN=update rewrites).
func TestSummaryAndDatasetGolden(t *testing.T) {
	r := newSessRig(t, "sorceress")
	r.script()
	if err := r.s.Stop(); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"summary.json", "decisions.jsonl", "session.jsonl"} {
		got, _ := os.ReadFile(filepath.Join(r.s.Dir(), n))
		gp := filepath.Join("testdata", "session-"+n+".golden")
		if os.Getenv("SESSION_GOLDEN") == "update" {
			os.WriteFile(gp, got, 0o644)
		}
		want, err := os.ReadFile(gp)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s differs from %s:\n%s", n, gp, got)
		}
	}
}
