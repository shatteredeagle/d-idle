package scenario

import (
	"fmt"
	"strings"
	"testing"
)

func pgRec(scn, class string, seed int64, i int, verdict, d1, commit, setup, branch string) LedgerRecord {
	return LedgerRecord{V: LedgerVersion, Run: fmt.Sprintf("job%02d/%s/%s/%d", i, scn, class, seed), Scenario: scn, Class: class, Seed: seed,
		Verdict: verdict, D1: d1, Setup: setup, BotCommit: commit, D2DSHA256: "ff", Host: "builder", Parallel: 2, Event: "nightly",
		Branch: branch, Started: fmt.Sprintf("2026-11-21T%02d:00:00Z", i)}
}

func pgLedger() []LedgerRecord {
	var recs []LedgerRecord
	for i := 0; i < 6; i++ { // Paladin: five equal parallel runs make it stable
		recs = append(recs, pgRec("town.resupply", "Paladin", 11, i, Green, "a", "c1", SetupSetup, "main"))
	}
	recs = append(recs, pgRec("town.resupply", "Necromancer", 11, 1, Green, "a", "c1", SetupCheated, "main"),
		pgRec("town.resupply", "Necromancer", 11, 2, Red, "b", "c1", SetupCheated, "main"), // same binaries: a flip
		pgRec("town.resupply", "Necromancer", 11, 3, Blocked, "", "c1", SetupCheated, "agent/x"))
	return recs
}

var pgTiers = TierFile{Version: TiersVersion, Tiers: map[string]TierEntry{"town.resupply": {Tier: TierT2, Source: "catalogue"}}}

func TestBuildPGPageGridStabilityFlips(t *testing.T) {
	p, err := BuildPGPage(pgLedger(), nil, pgTiers, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Cells) != 2 || p.Cells[0].Class != "Necromancer" || p.Cells[1].Class != "Paladin" {
		t.Fatalf("cells sorted by scenario then class: %+v", p.Cells)
	}
	n, pal := p.Cells[0], p.Cells[1]
	if pal.Stability != StabilityStable || pal.Tier != TierT2 || pal.Runs != 6 || pal.Flips != 0 || pal.Latest.Run != "job05/town.resupply/Paladin/11" {
		t.Fatalf("paladin %+v", pal)
	}
	if n.Stability != StabilityQuarantined || n.Flips != 1 || n.Runs != 3 || n.Latest.Verdict != Blocked {
		t.Fatalf("necromancer %+v", n)
	}
	if len(n.History) != 3 || n.History[0].Verdict != Blocked || !n.History[1].Flip || n.History[2].Flip {
		t.Fatalf("history newest first, flip on the changed run: %+v", n.History)
	}

	main, err := BuildPGPage(pgLedger(), nil, pgTiers, "main")
	if err != nil {
		t.Fatal(err)
	}
	if main.Cells[0].Runs != 2 || main.Cells[0].Latest.Verdict != Red || main.Cells[0].Stability != StabilityQuarantined {
		t.Fatalf("branch filter hides runs but stability is from the whole ledger: %+v", main.Cells[0])
	}
}

func TestBuildPGPageReleaseAndHistoryCap(t *testing.T) {
	var recs []LedgerRecord
	for i := 0; i < PGHistory+5; i++ {
		recs = append(recs, pgRec("cain.scroll", "Sorceress", 3, i%24, Green, "a", "c1", SetupSetup, "main"))
		recs[i].Started = fmt.Sprintf("2026-11-%02dT00:00:00Z", 1+i)
		recs[i].Run = fmt.Sprintf("r%02d", i)
	}
	p, err := BuildPGPage(recs, nil, TierFile{}, "main")
	if err != nil {
		t.Fatal(err)
	}
	c := p.Cells[0]
	if len(c.History) != PGHistory || c.History[0].Run != fmt.Sprintf("r%02d", PGHistory+4) || c.Runs != PGHistory+5 || c.Tier != "" {
		t.Fatalf("history capped, newest first, tier unknown stays empty: %d %s %d", len(c.History), c.History[0].Run, c.Runs)
	}
	if !strings.Contains(PGStatusDoc(p), "| cain.scroll | Sorceress | ? | stable |") {
		t.Fatalf("unknown tier shows ?:\n%s", PGStatusDoc(p))
	}
	if _, err := BuildPGPage(recs, []Release{{Scenario: "cain.scroll"}}, TierFile{}, ""); err == nil {
		t.Fatal("an invalid release is refused")
	}
}

func TestReleaseViewOnlyEarnedAndSetup(t *testing.T) {
	recs := pgLedger()
	recs = append(recs, pgRec("cain.scroll", "Paladin", 11, 7, Green, "a", "c2", SetupEarned, "main"))
	got := ReleaseView(recs, "c1")
	if len(got) != 1 || got[0].Class != "Paladin" || got[0].Run != "job05/town.resupply/Paladin/11" {
		t.Fatalf("only the latest SETUP run on c1, CHEATED dropped: %+v", got)
	}
	if got := ReleaseView(recs, "c2"); len(got) != 1 || got[0].Setup != SetupEarned {
		t.Fatalf("%+v", got)
	}
	if ReleaseView(recs, "") != nil {
		t.Fatal("no commit, no view")
	}
}

func TestPGStatusDocDeterministic(t *testing.T) {
	p1, _ := BuildPGPage(pgLedger(), nil, pgTiers, "main")
	recs := pgLedger()
	recs[0], recs[len(recs)-1] = recs[len(recs)-1], recs[0] // ledger order does not matter, start time does
	p2, _ := BuildPGPage(recs, nil, pgTiers, "main")
	d := PGStatusDoc(p1)
	if d != PGStatusDoc(p2) {
		t.Fatalf("not deterministic:\n%s\n---\n%s", d, PGStatusDoc(p2))
	}
	for _, want := range []string{"GENERATED", "Branch filter: `main`", "| town.resupply | Necromancer | T2 | quarantined | RED | CHEATED | 2 | 1 | b | c1 | builder |"} {
		if !strings.Contains(d, want) {
			t.Errorf("missing %q in\n%s", want, d)
		}
	}
	empty := PGStatusDoc(PGPage{Version: PGPageVersion})
	if !strings.Contains(empty, "(no runs recorded)") || !strings.Contains(empty, "all branches") {
		t.Fatalf("%s", empty)
	}
}
