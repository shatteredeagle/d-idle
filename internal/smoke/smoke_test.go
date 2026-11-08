package smoke

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func rules(t *testing.T) *Rules {
	t.Helper()
	r, err := LoadRules("../../data/smoke/rules.json")
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// synth builds a log: n minute lines whose clvl follows clvlAt, kills grow 4 a minute, plus extra lines.
func synth(n int, clvlAt func(m int) int, extra ...string) string {
	var b strings.Builder
	for m := 1; m <= n; m++ {
		fmt.Fprintf(&b, `{"time":"t","level":"INFO","msg":"minute","clvl":%d,"kills":%d,"deaths":0}`+"\n", clvlAt(m), 4*m)
	}
	for _, e := range extra {
		b.WriteString(e + "\n")
	}
	return b.String()
}

func clvl(v20, v45 int) func(int) int {
	return func(m int) int {
		switch {
		case m >= 45:
			return v45
		case m >= 20:
			return v20
		}
		return 1
	}
}

func TestJudge(t *testing.T) {
	r := rules(t)
	q := func(k string) string { return `{"level":"INFO","msg":"question","key":"` + k + `"}` }
	cases := []struct {
		name, class, log string
		writes           int
		want             string // "" = pass, else a substring of the failures
	}{
		{"paladin pass", "paladin", synth(45, clvl(3, 5)), 0, ""},
		{"sorceress pass", "Sorceress", synth(45, clvl(3, 4), q("a"), q("b")), 0, ""},
		{"paladin low at 45", "paladin", synth(45, clvl(3, 3)), 0, "clvl 3 at minute 45, need 4"},
		{"sorceress ok where paladin fails", "sorceress", synth(45, clvl(3, 3)), 0, ""},
		{"low at 20", "paladin", synth(45, clvl(1, 5)), 0, "at minute 20"},
		{"short run", "paladin", synth(30, clvl(3, 5)), 0, "short run"},
		{"panic", "paladin", synth(45, clvl(3, 5), "panic: runtime error: index out of range"), 0, "crash"},
		{"error event", "paladin", synth(45, clvl(3, 5), `{"level":"ERROR","msg":"bad"}`), 0, "error event"},
		{"no kills", "paladin", strings.ReplaceAll(synth(45, clvl(3, 5)), `"kills":180`, `"kills":0`), 0, "kills 0"},
		{"equip spam", "paladin", synth(45, clvl(3, 5), strings.Repeat(`{"level":"WARN","msg":"leveler: slot click refused"}`+"\n", 6)), 0, "equip refusals"},
		{"aura spam", "paladin", synth(45, clvl(3, 5), strings.Repeat(`{"level":"INFO","msg":"combat: aura did not come up"}`+"\n", 4)), 0, "aura"},
		{"too many questions", "paladin", synth(45, clvl(3, 5), strings.Repeat(q("x")+"\n", 1)+strings.Join(func() []string {
			var s []string
			for i := 0; i < 13; i++ {
				s = append(s, q(fmt.Sprint("k", i)))
			}
			return s
		}(), "\n")), 0, "questions, max"},
		{"question key repeats", "paladin", synth(45, clvl(3, 5), q("same"), q("same")), 0, `key "same"`},
		{"writes", "paladin", synth(45, clvl(3, 5)), 2, "writes after start: 2"},
		{"writes unknown", "paladin", synth(45, clvl(3, 5)), -1, "unknown"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, err := Judge(r, Input{Class: c.class, Commit: "abcdef0123456789", Log: strings.NewReader(c.log), Writes: c.writes})
			if err != nil {
				t.Fatal(err)
			}
			got := strings.Join(res.Fails, "; ")
			if c.want == "" && !res.Pass || c.want != "" && !strings.Contains(got, c.want) {
				t.Fatalf("want %q, got pass=%v fails=%q", c.want, res.Pass, got)
			}
			if c.want != "" && res.Pass {
				t.Fatal("expected a failure")
			}
		})
	}
}

func TestAllowListAndLine(t *testing.T) {
	r := rules(t)
	r.ErrorAllow = []string{"known benign"}
	log := synth(45, clvl(3, 5), `{"level":"ERROR","msg":"known benign thing"}`)
	res, err := Judge(r, Input{Class: "paladin", Commit: "abcdef0123456789", Log: strings.NewReader(log), Writes: -1})
	if err != nil {
		t.Fatal(err)
	}
	if res.Pass { // writes unknown still fails
		t.Fatal("writes unknown must fail")
	}
	res, _ = Judge(r, Input{Class: "paladin", Commit: "abcdef0123456789", Log: strings.NewReader(log), Writes: 0})
	if !res.Pass {
		t.Fatalf("allow-listed error failed: %v", res.Fails)
	}
	want := "SMOKE PASS paladin abcdef0123 clvl@45=5 kills=180 deaths=0 notes=no-session-record"
	if res.Line() != want {
		t.Fatalf("line %q", res.Line())
	}
}

func TestUnknownClass(t *testing.T) {
	_, err := Judge(rules(t), Input{Class: "bard", Log: strings.NewReader("")})
	if err == nil || !strings.Contains(err.Error(), `"bard"`) {
		t.Fatalf("err %v", err)
	}
}

func TestSession(t *testing.T) {
	r := rules(t)
	dir := t.TempDir()
	sub := filepath.Join(dir, "20261108T100000Z-human")
	os.MkdirAll(sub, 0o755)
	for _, f := range r.SessionFiles {
		os.WriteFile(filepath.Join(sub, f), []byte("ok\n"), 0o600)
	}
	run := func() *Result {
		res, err := Judge(r, Input{Class: "paladin", Log: strings.NewReader(synth(45, clvl(3, 5))), SessionDir: dir, Secrets: []string{"s3cr3ttoken"}, Writes: 0})
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	if res := run(); !res.Pass {
		t.Fatalf("clean record failed: %v", res.Fails)
	}
	os.WriteFile(filepath.Join(sub, "summary.md"), []byte("leak s3cr3ttoken"), 0o600)
	if res := run(); res.Pass || !strings.Contains(res.Fails[0], "secret") {
		t.Fatalf("secret not caught: %v", res.Fails)
	}
	os.Remove(filepath.Join(sub, "summary.json"))
	if res := run(); !strings.Contains(strings.Join(res.Fails, ";"), "missing: summary.json") {
		t.Fatalf("missing file not caught: %v", res.Fails)
	}
}

func TestRecordSmall(t *testing.T) {
	res, _ := Judge(rules(t), Input{Class: "paladin", Log: strings.NewReader(synth(45, clvl(3, 5), strings.Repeat("panic: x\n", 100))), Writes: 0})
	if len(res.Samples) > 12 || len(res.Record("s", "gate")) > 8000 || !strings.Contains(res.Record("s", "gate"), "SMOKE FAIL") {
		t.Fatal("record not small or missing verdict")
	}
}
