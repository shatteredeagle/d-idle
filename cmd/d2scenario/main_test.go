package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"d2lod-bot/internal/scenario"
)

const dir = "../../tools/provingground/scenarios"

func TestPlanIsADryRun(t *testing.T) {
	var b bytes.Buffer
	if err := cmdPlan([]string{"-dir", dir, "-tier", "fast"}, &b); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"DRY RUN", "cain.scroll/Paladin/11", "town.stash-gold-withdraw"} {
		if !strings.Contains(b.String(), want) {
			t.Errorf("plan lacks %q:\n%s", want, b.String())
		}
	}
	b.Reset()
	if err := cmdPlan([]string{"-dir", dir, "-tier", "full", "-id", "cain.", "-class", "Sorceress", "-seed", "5", "-json"}, &b); err != nil {
		t.Fatal(err)
	}
	var p struct {
		Runs []struct{ ID, Class string }
	}
	if err := json.Unmarshal(b.Bytes(), &p); err != nil || len(p.Runs) == 0 {
		t.Fatalf("%v %s", err, b.String())
	}
	for _, r := range p.Runs {
		if r.Class != "Sorceress" || !strings.HasPrefix(r.ID, "cain.") {
			t.Fatalf("selector leaked %v", r)
		}
	}
}

func TestRunDryRunStartsNothing(t *testing.T) {
	var b bytes.Buffer
	out := filepath.Join(t.TempDir(), "o")
	if err := cmdRun([]string{"-dir", dir, "-tier", "full", "-dry-run", "-out", out}, &b); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), "DRY RUN") || !strings.Contains(b.String(), "d2dslot.sh run") {
		t.Fatal(b.String())
	}
	if !strings.Contains(b.String(), "needs the shipped sandbox d2d build") {
		t.Fatal("the sandbox scenario must be reported BLOCKED, not planned:\n" + b.String())
	}
}

func TestEvalExitCodes(t *testing.T) {
	st := scenario.State{Version: scenario.StateVersion, Source: "fixture", Has: []string{"items"}, Items: []scenario.StateItem{{Code: "bks", Where: "pack"}}, WritesAfterStart: 0}
	p := filepath.Join(t.TempDir(), "s.json")
	b, _ := json.Marshal(st)
	os.WriteFile(p, b, 0o644)
	var out bytes.Buffer
	if err := cmdEval([]string{"cain.scroll", "-dir", dir, "-state", p}, &out); err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	st.Items = nil
	b, _ = json.Marshal(st)
	os.WriteFile(p, b, 0o644)
	if err := cmdEval([]string{"cain.scroll", "-dir", dir, "-state", p}, &out); err == nil {
		t.Fatal("no scroll must be red")
	} else if e, ok := err.(exitErr); !ok || e.code != 3 {
		t.Fatalf("%v", err)
	}
}

func TestCoverageAndValidateOnShippedData(t *testing.T) {
	var b bytes.Buffer
	if err := cmdValidate([]string{"-dir", dir, "-board", "../../data/questboard", "-milestones", "../../tools/provingground/earned/milestones.json"}, &b); err != nil {
		t.Fatal(err)
	}
	b.Reset()
	if err := cmdCoverage([]string{"-dir", dir, "-board", "../../data/questboard"}, &b); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), "UNCOVERED") || !strings.Contains(b.String(), "quest.search_for_cain.scroll") {
		t.Fatal(b.String())
	}
}

func TestSetupPrintsLedgerReasons(t *testing.T) {
	var b bytes.Buffer
	if err := cmdSetup([]string{"cain.report", "-dir", dir, "-class", "Sorceress", "-seed", "3"}, &b); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), "debug quest set 4 0x2012") || !strings.Contains(b.String(), "CHEATED") {
		t.Fatal(b.String())
	}
}

func TestGateCheckAndDeterminismFiles(t *testing.T) {
	d := t.TempDir()
	write := func(name string, rs []scenario.Result) string {
		p := filepath.Join(d, name)
		f, _ := os.Create(p)
		scenario.WriteJSON(f, rs)
		f.Close()
		return p
	}
	g := scenario.Result{Run: scenario.Run{ID: "a", Class: "Paladin", Seed: 1}, Status: scenario.Green}
	r := g
	r.Status = scenario.Red
	var out bytes.Buffer
	if err := cmdGateCheck([]string{write("g.json", []scenario.Result{g})}, &out); err != nil {
		t.Fatal(err)
	}
	if err := cmdGateCheck([]string{write("r.json", []scenario.Result{r})}, &out); err == nil {
		t.Fatal("red gate passed")
	}
	if err := cmdDeterminism([]string{write("a.json", []scenario.Result{g}), write("b.json", []scenario.Result{r})}, &out); err == nil {
		t.Fatal("verdict change passed")
	}
}

func TestInstanceIDIsValidForTheBot(t *testing.T) {
	id := instanceID(scenario.Run{ID: "cain.scroll.sandbox", Class: "Sorceress", Seed: 1234567})
	if len(id) > 16 || strings.Trim(id, "abcdefghijklmnopqrstuvwxyz0123456789-") != "" {
		t.Fatalf("bad instance id %q", id)
	}
	if id != instanceID(scenario.Run{ID: "cain.scroll.sandbox", Class: "Sorceress", Seed: 1234567}) {
		t.Fatal("not stable")
	}
}

func TestBotFailureReadsTheExitLine(t *testing.T) {
	p := filepath.Join(t.TempDir(), "bot.log")
	os.WriteFile(p, []byte(`{"level":"INFO","msg":"x"}`+"\n"+`{"level":"ERROR","msg":"exit","err":"api server: boom"}`+"\n"), 0o644)
	if got := botFailure(p); got != "api server: boom" {
		t.Fatalf("%q", got)
	}
	os.WriteFile(p, []byte(`{"level":"INFO","msg":"x"}`+"\n"), 0o644)
	if botFailure(p) != "" {
		t.Fatal("clean log")
	}
}

func TestPointStatsFromDump(t *testing.T) {
	line := `{"v":1,"player":{"stat_pts":25,"skill_pts":5},"stats":{"base":{"str":25,"dex":20,"vit":25,"ene":15}},"skills":{"base_levels":[1,0,2,0]}}`
	got := pointStats(nil, line)
	want := map[string]int{"str": 25, "dex": 20, "vit": 25, "ene": 15, "statpts": 25, "skillpts": 5, "skillsspent": 3}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %d, want %d (%v)", k, got[k], v, got)
		}
	}
	if pointStats(nil, "err unknown verb") != nil || pointStats(nil, "{}") != nil {
		t.Error("stock d2d or an empty dump observes nothing")
	}
}

func TestBotConfigAndSkillLevels(t *testing.T) {
	dir := t.TempDir()
	if err := writeBotConfig(dir, scenario.Setup{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "config.json")); err == nil {
		t.Fatal("a config file without a build")
	}
	if err := writeBotConfig(dir, scenario.Setup{Build: "sorceress_normal30"}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "config.json"))
	if string(b) != `{"levelerBuild":"sorceress_normal30"}` {
		t.Fatalf("config %s", b)
	}
	line := `{"player":{"skill_pts":2},"skills":{"base_levels":[0,3,0,1]}}`
	if got := skillLevels(nil, line); len(got) != 4 || got[1] != 3 {
		t.Fatalf("levels %v", got)
	}
	if got := skillLevels(nil, "garbage"); got != nil {
		t.Fatalf("garbage %v", got)
	}
	if ps := pointStats(nil, line); ps["skillsspent"] != 4 || ps["skillpts"] != 2 {
		t.Fatalf("stats %v", ps)
	}
}

func TestParseBoardAndReply(t *testing.T) {
	body := []byte(`{"quests":[{"id":"q","chunks":[{"id":"a","state":"accepted","reason":{"code":"waits_for"}},{"id":"b","state":"ready"},{"id":"c","state":"blocked","reason":"gave_up"}]}]}`)
	got := parseBoard(body)
	if len(got) != 3 || got[0].Reason != "waits_for" || got[1].Reason != "" || got[2].Reason != "gave_up" || got[0].Quest != "q" {
		t.Fatalf("board %+v", got)
	}
	r := parseReply(1, "/api/quests/q/cancel", 200, []byte(`{"ok":true,"already":true,"changed":[{"quest":"q","chunk":"a"}]}`))
	if !r.OK || !r.Already || len(r.Changed) != 1 || r.Changed[0] != "q/a" || r.Index != 1 {
		t.Fatalf("reply %+v", r)
	}
}
