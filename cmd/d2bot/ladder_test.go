package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/api"
	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/goals/sim"
	"d2lod-bot/internal/ladder"
	"d2lod-bot/internal/watch"
)

// fakeEnable mirrors bot.(*Runner).EnableLadderWith on a virtual clock the test steps.
func fakeEnable(t *testing.T, clk *watch.SimClock, called *int) enableFunc {
	return func(dir, profile string, wire bot.LadderWire) (*ladder.Ladder, error) {
		*called++
		cat, err := decide.Shipped()
		if err != nil {
			return nil, err
		}
		st, err := decide.New(cat, profile, clk)
		if err != nil {
			return nil, err
		}
		sink, src, err := wire(st, cat, clk)
		if err != nil {
			return nil, err
		}
		l, err := ladder.New(ladder.Config{Cat: cat, Store: st, Clock: clk, Sink: sink, Answers: src})
		if err != nil {
			return nil, err
		}
		ladder.Install(l)
		t.Cleanup(func() { ladder.Install(nil) })
		return l, nil
	}
}

func goalManager(t *testing.T) *goals.Manager {
	t.Helper()
	m, err := goals.New(goals.Options{Runner: sim.New(sim.Options{Seed: 1}), Path: filepath.Join(t.TempDir(), "goals.json")})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func httpDo(t *testing.T, method, url string, body any, out any) int {
	t.Helper()
	var rd *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	} else {
		rd = bytes.NewReader(nil)
	}
	req, _ := http.NewRequest(method, url, rd)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if out != nil {
		json.NewDecoder(resp.Body).Decode(out)
	}
	return resp.StatusCode
}

var testBuilds = []struct {
	name string
	hero ladder.Hero
	life float64
}{
	{"sorceress", ladder.Hero{Class: d2.Sorceress, Gold: 600, AreaClass: "field", Durability: 25}, 30},
	{"paladin", ladder.Hero{Class: d2.Paladin, Hardcore: true, Gold: 600, AreaClass: "field", Durability: 25}, 45},
}

func askRetreat(t *testing.T, l *ladder.Ladder, h ladder.Hero, life float64) {
	t.Helper()
	extra := decide.Obs{Num: map[string]float64{"life": life}, Cat: map[string]string{"packdanger": "2", "kit": "OK", "held": "hp"}}
	if _, err := l.Decide("t0.retreat.drink", h, extra, ""); err != nil {
		t.Fatal(err)
	}
	l.Pump()
}

// With -ladder -watch the Hub receives the bot's thoughts; a person answers on the separate answer route and the order
// box reaches the goal manager; a scripted profile answers by itself. Two builds.
func TestLadderWatchOnTheBot(t *testing.T) {
	for _, b := range testBuilds {
		for _, scripted := range []string{"", "cautious"} {
			t.Run(b.name+"/"+scripted, func(t *testing.T) {
				clk := &watch.SimClock{}
				called := 0
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				m := goalManager(t)
				lo := ladderOptions{on: true, profile: scripted, watch: true, viewAddr: "127.0.0.1:0", answerAddr: "127.0.0.1:0"}
				rig, err := startLadder(ctx, lo, "", "", fakeEnable(t, clk, &called), m, func() goals.Status { return goals.Status{State: goals.RunnerRunning, Activity: "Fighting"} }, func() string { return "Cold Plains" })
				if err != nil || rig == nil || rig.Hub == nil || called != 1 {
					t.Fatalf("%v %v %d", rig, err, called)
				}
				view, ans := "http://"+rig.Served.View.String(), "http://"+rig.Served.Answer.String()
				askRetreat(t, rig.Ladder, b.hero, b.life)
				var tv watch.ThoughtsView
				httpDo(t, "GET", view+"/api/thoughts", nil, &tv)
				if len(tv.Thoughts) != 1 {
					t.Fatalf("the Hub did not receive the thought: %+v", tv)
				}
				if scripted == "" {
					choice := tv.Thoughts[0].Choices[0].ID
					if code := httpDo(t, "POST", ans+"/api/answer", map[string]any{"thought": tv.Thoughts[0].ID, "answer": choice}, nil); code != 200 {
						t.Fatalf("answer route: %d", code)
					}
					if code := httpDo(t, "POST", view+"/api/answer", map[string]any{"thought": tv.Thoughts[0].ID, "answer": choice}, nil); code == 200 {
						t.Fatal("the viewer accepted a write")
					}
				} else {
					clk.Advance(40 * time.Second)
					rig.Ladder.Pump()
				}
				if n := rig.Ladder.Stats().Answered; scripted != "" && n == 0 {
					t.Errorf("the scripted profile did not answer: %+v", rig.Ladder.Stats())
				}
				if scripted == "" {
					httpDo(t, "GET", view+"/api/thoughts", nil, &tv)
					if len(tv.Thoughts) != 0 {
						t.Errorf("answered thought still open: %+v", tv.Thoughts)
					}
				}
				// An order becomes an area goal for its duration; cancelling removes it.
				var rep struct {
					Order *watch.Order `json:"order"`
				}
				if code := httpDo(t, "POST", ans+"/api/orders", map[string]any{"kind": "go", "area": "Dark Wood", "minutes": 15}, &rep); code != 200 || rep.Order == nil {
					t.Fatalf("order: %d %+v", code, rep)
				}
				var found bool
				for _, g := range m.Goals() {
					found = found || (g.Type == goals.FarmArea && g.Params.Area == "Dark Wood" && g.Params.UntilMins == 15)
				}
				if !found {
					t.Fatalf("no farm_area goal: %+v", m.Goals())
				}
				if code := httpDo(t, "POST", ans+"/api/orders/cancel", map[string]any{"id": rep.Order.ID}, nil); code != 200 {
					t.Fatalf("cancel: %d", code)
				}
				for _, g := range m.Goals() {
					if g.Params.Area == "Dark Wood" {
						t.Errorf("the cancelled order's goal is still queued: %+v", g)
					}
				}
			})
		}
	}
}

// Without the flags nothing is made: no enable call, no state directory, no listener.
func TestLadderOffDoesNothing(t *testing.T) {
	dir := t.TempDir()
	called := 0
	rig, err := startLadder(context.Background(), ladderOptions{viewAddr: "127.0.0.1:0", answerAddr: "127.0.0.1:0"}, "", filepath.Join(dir, "state"),
		fakeEnable(t, &watch.SimClock{}, &called), nil, nil, nil)
	if rig != nil || err != nil || called != 0 || ladder.Active() != nil {
		t.Fatalf("%v %v %d", rig, err, called)
	}
	if ents, _ := os.ReadDir(dir); len(ents) != 0 {
		t.Fatalf("files made: %v", ents)
	}
	// -ladder alone with a scripted player: a store but no listener.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	rig, err = startLadder(context.Background(), ladderOptions{on: true, profile: "bold", viewAddr: l.Addr().String(), answerAddr: l.Addr().String()}, "", "",
		fakeEnable(t, &watch.SimClock{}, &called), nil, nil, nil)
	if err != nil || rig == nil || rig.Hub != nil || rig.Served.View != nil {
		t.Fatalf("scripted without -watch must open no listener: %+v %v", rig, err)
	}
}

func TestLadderFlagRules(t *testing.T) {
	base := []string{"--sim"}
	for _, c := range []struct {
		args []string
		bad  string
	}{
		{[]string{"--watch"}, "needs --ladder"},
		{[]string{"--ladder-profile", "bold"}, "needs --ladder"},
		{[]string{"--ladder", "--watch"}, "--attach or --managed"},
	} {
		if _, err := parseFlags(append(append([]string{}, base...), c.args...)); err == nil || !strings.Contains(err.Error(), c.bad) {
			t.Errorf("%v: %v", c.args, err)
		}
	}
	for _, c := range []struct {
		args []string
		bad  string
	}{
		{[]string{"--attach", "--ladder"}, "someone to answer"},
		{[]string{"--attach", "--ladder", "--ladder-profile", "holdout:x"}, "whole number"},
		{[]string{"--attach", "--ladder", "--ladder-profile", "nobody"}, "unknown answer profile"},
	} {
		if _, err := parseFlags(c.args); err == nil || !strings.Contains(err.Error(), c.bad) {
			t.Errorf("%v: %v", c.args, err)
		}
	}
	for _, args := range [][]string{{"--attach", "--ladder", "--watch"}, {"--attach", "--ladder", "--ladder-profile", "holdout:101"}, {"--attach"}} {
		if _, err := parseFlags(args); err != nil {
			t.Errorf("%v: %v", args, err)
		}
	}
	if o, _ := parseFlags([]string{"--attach"}); o.ladder.on || o.ladder.watch {
		t.Error("ladder and watch must default to off")
	}
}

// The token rule of -watch: a non-loopback listener needs one, and nothing is bound when it is refused.
func TestWatchServeTokenRule(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	called := 0
	lo := ladderOptions{on: true, watch: true, viewAddr: "0.0.0.0:0", answerAddr: "127.0.0.1:0"}
	if _, err := startLadder(ctx, lo, "", "", fakeEnable(t, &watch.SimClock{}, &called), goalManager(t), nil, nil); err == nil || !strings.Contains(err.Error(), "token") {
		t.Fatalf("want a token error, got %v", err)
	}
	if ladder.Active() != nil {
		t.Error("the ladder stayed installed after the screen was refused")
	}
	rig, err := startLadder(ctx, lo, "s3cret", "", fakeEnable(t, &watch.SimClock{}, &called), goalManager(t), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, port, _ := net.SplitHostPort(rig.Served.View.String())
	url := "http://127.0.0.1:" + port + "/api/thoughts"
	if code := httpDo(t, "GET", url, nil, nil); code != 401 {
		t.Errorf("no token: %d", code)
	}
	if code := httpDo(t, "GET", url+"?t=s3cret", nil, nil); code != 200 {
		t.Errorf("token: %d", code)
	}
}

func TestLadderPostKeepsTheFlagAcrossATuningFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tuning.json")
	first := env.DefaultTuning()
	tn0, err := bot.NewTuner(&first, bot.Config{}, path)
	if err != nil {
		t.Fatal(err)
	}
	f := 20.0
	if err := tn0.Edit(api.TuneEdit{Overrides: map[string]*float64{"combat.fightRadius": &f}}); err != nil { // writes tuning.json
		t.Fatal(err)
	}
	for _, c := range []struct {
		on   bool
		want int
	}{{true, 1}, {false, 0}} {
		tune := env.DefaultTuning()
		tn, err := bot.NewTuner(&tune, bot.Config{}, path, ladderPost(ladderOptions{on: c.on}))
		if err != nil || tn == nil {
			t.Fatal(err)
		}
		if tune.Ladder.Enabled != c.want {
			t.Fatalf("flag %v: ladder.enabled %d, want %d", c.on, tune.Ladder.Enabled, c.want)
		}
	}
}

// Drills name their run and get their own store; an unnamed run keeps the shared one; "shared" is the explicit
// continuation (t0drill Result 2, defect 7).
func TestLadderStorePerRun(t *testing.T) {
	st := filepath.Join("state", "i5")
	shared := filepath.Join(st, "ladder")
	cases := []struct {
		name, run, flag string
		on              bool
		want            string
		bad             bool
	}{
		{"unnamed run", "", "", true, shared, false},
		{"named run", "t0live-b", "", true, filepath.Join(shared, "runs", "t0live-b"), false},
		{"flag wins", "t0live-b", "mine", true, filepath.Join(shared, "runs", "mine"), false},
		{"continue", "t0live-b", "shared", true, shared, false},
		{"off ignores both", "a/b", "", false, shared, false},
		{"bad name", "../x", "", true, "", true},
	}
	for _, c := range cases {
		t.Setenv("D2BOT_RUN", c.run)
		got, err := ladderDir(st, ladderOptions{on: c.on, store: c.flag})
		if (err != nil) != c.bad || got != c.want {
			t.Errorf("%s: got %q, %v want %q", c.name, got, err, c.want)
		}
	}
}
