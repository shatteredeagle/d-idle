package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func write(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

const why = "## Why\n\nThis change exists because the evidence in the log says the shared socket was driven by two agents at once.\n\n## What Changes\n\n- one thing\n"

func card(id, status, owns, excl string) string {
	return "# Tasks\n\n```yaml\nchange: " + id + "\nstatus: " + status + "\nagent: " + id + "\nbranch: agent/" + id +
		"\nowns:" + owns + "\nexclusive:" + excl + "\ngates:\n  - \"go test passes\"  # comment\nmerge:\nresult:\n```\n\n## 1. Work\n\n- [ ] 1.1 x\n"
}

func addCard(t *testing.T, root, id, status, owns, excl string) {
	write(t, root, "openspec/changes/"+id+"/proposal.md", why)
	write(t, root, "openspec/changes/"+id+"/tasks.md", card(id, status, owns, excl))
}

func has(fs []Finding, check, sub string) bool {
	for _, f := range fs {
		if f.Check == check && strings.Contains(f.Msg, sub) {
			return true
		}
	}
	return false
}

func TestParseHeader(t *testing.T) {
	f, err := parseHeader("x\n```yaml\nchange: a # c\nowns: [x/**, \"y/z.go\"]\nexclusive:\n  - r1\n  - r2\nresult: \"quoted # not comment\"\n```\n")
	if err != nil {
		t.Fatal(err)
	}
	if f["change"][0] != "a" || len(f["owns"]) != 2 || f["owns"][1] != "y/z.go" || len(f["exclusive"]) != 2 || f["result"][0] != "quoted # not comment" {
		t.Fatalf("bad parse: %#v", f)
	}
	if _, err := parseHeader("no block"); err == nil {
		t.Fatal("want error without block")
	}
	if _, err := parseHeader("```yaml\na: b\n"); err == nil {
		t.Fatal("want error for unterminated block")
	}
}

func TestGlobs(t *testing.T) {
	yes := [][2]string{{"internal/town/**", "internal/town/x.go"}, {"cmd/d2bot/d2d*.go", "cmd/d2bot/d2dwire.go"}, {"docs/d2d-*.md", "docs/d2d-wire.md"}, {"internal/a/", "internal/a/b/c.go"}}
	for _, c := range yes {
		if !globsOverlap(c[0], c[1]) || !globRegexp(c[0]).MatchString(c[1]) {
			t.Errorf("%v should overlap and match", c)
		}
	}
	if globsOverlap("cmd/d2bot/d2d*.go", "cmd/d2bot/behaviors.go") || globsOverlap("internal/town/**", "internal/townx/**") {
		t.Error("disjoint globs reported as overlapping")
	}
	if globRegexp("cmd/*.go").MatchString("cmd/a/b.go") {
		t.Error("* must not cross a slash")
	}
}

func TestCardsAndOverlap(t *testing.T) {
	root := t.TempDir()
	addCard(t, root, "alpha", "in-progress", "\n  - internal/town/**", "\n  - d2d-control-socket")
	addCard(t, root, "beta", "in-progress", "\n  - internal/town/x.go", "\n  - d2d-control-socket")
	addCard(t, root, "gamma", "proposed", "\n  - docs/g.md", " []")
	addCard(t, root, "Bad_ID", "done", "", "")
	cards, fs := loadCards(root)
	fs = append(fs, checkCards(cards)...)
	fs = append(fs, checkOverlap(cards)...)
	for _, want := range [][2]string{{"overlap", "alpha owns"}, {"overlap", "exclusive resource"}, {"card", "kebab-case"}, {"card", "archive it"}} {
		if !has(fs, want[0], want[1]) {
			t.Errorf("missing finding %v in %v", want, fs)
		}
	}
	if has(fs, "overlap", "gamma") {
		t.Error("gamma is disjoint")
	}
}

func TestCardProblems(t *testing.T) {
	root := t.TempDir()
	write(t, root, "openspec/changes/short/proposal.md", "## Why\n\ntoo short\n")
	write(t, root, "openspec/changes/short/tasks.md", "# Tasks\n\nno header\n")
	write(t, root, "openspec/changes/archive/2026-10-30-old/proposal.md", why)
	write(t, root, "openspec/changes/archive/2026-10-30-old/tasks.md", card("old", "archived", "", ""))
	cards, fs := loadCards(root)
	fs = append(fs, checkCards(cards)...)
	for _, want := range [][2]string{{"card", "no yaml header"}, {"card", "needs merge: and result:"}} {
		if !has(fs, want[0], want[1]) {
			t.Errorf("missing %v in %v", want, fs)
		}
	}
	if len(cards) != 2 || cards[0].ID != "old" || !cards[0].Archived {
		t.Fatalf("archive prefix not stripped: %+v", cards)
	}
}

func TestSpecRules(t *testing.T) {
	good := "# X\n\n## Purpose\nThis capability defines something long enough to pass the fifty character rule.\n\n## Requirements\n\n### Requirement: A\nThe bot SHALL do it.\n\n#### Scenario: S\n- **WHEN** x\n- **THEN** y\n"
	if fs := checkSpecText("g", good, false); len(fs) != 0 {
		t.Fatalf("good spec flagged: %v", fs)
	}
	bad := "## Purpose\nTBD\n\n## Requirements\n\n### Requirement: A\nThe bot does it.\n\n### Requirement: A\nThe bot SHALL x.\n\n#### Scenario: S\n"
	fs := checkSpecText("b", bad, false)
	for _, sub := range []string{"Purpose", "no SHALL", "duplicate", "no #### Scenario", "empty scenario"} {
		if !has(fs, "spec", sub) {
			t.Errorf("missing %q in %v", sub, fs)
		}
	}
	delta := "## MODIFIED Requirements\n\n### Requirement: A\nThe bot SHALL x.\n\n#### Scenario: S\n- **WHEN** a\n- **THEN** b\n\n## REMOVED Requirements\n\n### Requirement: B\n**Reason**: r\n"
	if fs := checkSpecText("d", delta, true); len(fs) != 0 {
		t.Fatalf("good delta flagged: %v", fs)
	}
	if fs := checkSpecText("d", "# nothing\n", true); !has(fs, "spec", "no ADDED") {
		t.Fatalf("delta without sections not flagged: %v", fs)
	}
}

func TestOwnersSync(t *testing.T) {
	root := t.TempDir()
	addCard(t, root, "alpha", "in-progress", "\n  - internal/a/**", " []")
	write(t, root, "docs/OWNERS", "# c\ninternal/a/** alpha alpha\ninternal/b/** bob ghost\nbroken\n")
	cards, _ := loadCards(root)
	fs := checkOwners(root, cards)
	if !has(fs, "owners", "ghost") || !has(fs, "owners", "needs glob") {
		t.Fatalf("expected ghost and malformed line: %v", fs)
	}
	write(t, root, "docs/OWNERS", "internal/other/** alpha alpha\n")
	if fs := checkOwners(root, cards); !has(fs, "owners", "no such line") || !has(fs, "owners", "internal/other") {
		t.Fatalf("expected missing line: %v", fs)
	}
	addCard(t, root, "beta", "in-progress", "\n  - internal/a/x.go", " []")
	cards, _ = loadCards(root)
	write(t, root, "docs/OWNERS", "internal/** main -\ninternal/a/** alpha alpha\ninternal/a/x.go beta beta\n")
	if fs := checkOwners(root, cards); !has(fs, "owners", "overlaps") || has(fs, "owners", "main") {
		t.Fatalf("expected owner overlap: %v", fs)
	}
}

func TestMerges(t *testing.T) {
	root := t.TempDir()
	addCard(t, root, "alpha", "in-progress", "\n  - a/**", " []")
	cards, _ := loadCards(root)
	old := gitMerges
	defer func() { gitMerges = old }()
	gitMerges = func(string, string, int) ([]Merge, error) {
		return []Merge{
			{"h1", "10-30 09:00", "Merge agent/alpha: done", ""},
			{"h2", "10-30 09:01", "Merge agent/ghost: x", ""},
			{"h3", "10-30 09:02", "Merge things", ""},
			{"h4", "10-30 09:03", "Merge things", "Change: alpha\n"},
			{"h5", "10-30 09:04", "Merge master into agent/x", ""},
		}, nil
	}
	fs := checkMerges(root, cards, "", 10)
	if len(fs) != 2 || !has(fs, "merge", "ghost") || !has(fs, "merge", "h3") {
		t.Fatalf("unexpected: %v", fs)
	}
}

func TestStatusBlockRoundTrip(t *testing.T) {
	root := t.TempDir()
	addCard(t, root, "alpha", "in-progress", "\n  - a/**", "\n  - r1")
	write(t, root, "openspec/specs/s/spec.md", "## Purpose\nx\n## Requirements\n### Requirement: A\nThe bot SHALL x.\n#### Scenario: S\n- **WHEN** a\n")
	write(t, root, "docs/plan/STATUS.md", "# Where we are\n\nheadline stays\n")
	cards, _ := loadCards(root)
	old := gitMerges
	defer func() { gitMerges = old }()
	gitMerges = func(string, string, int) ([]Merge, error) {
		return []Merge{{"h1", "10-30 09:00", "Merge agent/alpha: done", ""}}, nil
	}
	if fs := checkStatus(root, cards); !has(fs, "status", "no generated block") {
		t.Fatalf("want missing block: %v", fs)
	}
	if err := writeStatus(root, cards, 5); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(statusPath(root))
	if !strings.Contains(string(b), "headline stays") || !strings.Contains(string(b), "Merge agent/alpha: done") || !strings.Contains(string(b), "| [alpha]") {
		t.Fatalf("block content wrong:\n%s", b)
	}
	if fs := checkStatus(root, cards); len(fs) != 0 {
		t.Fatalf("fresh block flagged: %v", fs)
	}
	// new merges alone must not make the check stale; a changed card must
	gitMerges = func(string, string, int) ([]Merge, error) { return nil, nil }
	if err := writeStatus(root, cards, 5); err != nil {
		t.Fatal(err)
	}
	addCard(t, root, "beta", "proposed", "\n  - b/**", " []")
	cards, _ = loadCards(root)
	if fs := checkStatus(root, cards); !has(fs, "status", "out of date") {
		t.Fatalf("want stale: %v", fs)
	}
	if err := writeStatus(root, cards, 5); err != nil {
		t.Fatal(err)
	}
	b2, _ := os.ReadFile(statusPath(root))
	if strings.Count(string(b2), beginMark) != 1 || !strings.Contains(string(b2), "headline stays") {
		t.Fatalf("rewrite duplicated or lost text:\n%s", b2)
	}
}

func TestScaffoldPassesCheck(t *testing.T) {
	root := t.TempDir()
	if err := scaffold(root, "delta-one", "", "", []string{"x/**"}, []string{"instance:i2"}); err != nil {
		t.Fatal(err)
	}
	if err := scaffold(root, "delta-one", "", "", nil, nil); err == nil {
		t.Fatal("second scaffold must fail")
	}
	if err := scaffold(root, "Bad", "", "", nil, nil); err == nil {
		t.Fatal("bad id must fail")
	}
	cards, fs := loadCards(root)
	fs = append(fs, checkCards(cards)...)
	if len(cards) != 1 || cards[0].Owns()[0] != "x/**" || cards[0].Exclusive()[0] != "instance:i2" {
		t.Fatalf("scaffold header wrong: %+v", cards[0].Fields)
	}
	// only the TODO Why text length can trip: it is >= 50 chars, so the card is clean
	if len(fs) != 0 {
		t.Fatalf("fresh scaffold has findings: %v", fs)
	}
}

func TestStrictDates(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "openspec"), 0o755)
	os.WriteFile(filepath.Join(root, "openspec", ".strict.yaml"), []byte("cards_overlap_fatal_from: 2026-11-03 # c\nmerges_diff_fatal_from: 2026-11-06\nallow_merges:\n  - abc1234 # old\n"), 0o644)
	s := loadStrict(root)
	fs := []Finding{{"overlap", "x"}, {"merge", "y"}, {"openspec-cli", "z"}}
	day := func(d string) time.Time { x, _ := time.Parse("2006-01-02", d); return x }
	for _, c := range []struct {
		d string
		n int
	}{{"2026-11-02", 0}, {"2026-11-03", 1}, {"2026-11-05", 1}, {"2026-11-06", 2}} {
		if bad, _ := s.fatal(fs, day(c.d)); len(bad) != c.n {
			t.Errorf("%s: %d fatal, want %d", c.d, len(bad), c.n)
		}
	}
	if !s.allowed("abc1234") || !s.allowed("abc12") || s.allowed("def") || s.allowed("") {
		t.Error("allowlist match wrong")
	}
	if !loadStrict(t.TempDir()).CardsFrom.IsZero() {
		t.Error("missing file must mean no strict dates")
	}
}
