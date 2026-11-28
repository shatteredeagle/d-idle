package d2dcontainer_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The adversarial corpus (card sec2, fable review 12 C4 and review 4 finding 1): every known secret shape in the
// places it turns up (log text, logfmt, env dumps, links, JSON values). Values are fake and built from parts.
var secCorpus = []struct{ name, in, secret string }{
	{"key lower dashed", `key abcd-1234-efgh-5678`, "1234-efgh"},
	{"key spaced", `key ABCD 1234 EFGH 5678`, "1234 EFGH"},
	{"key lower spaced labelled", `serial abcd 1234 efgh 5678 ok`, "1234 efgh"},
	{"key mixed case dashed", `Abcd1-Ef2gh-Ij3kl-Mn4op`, "Ef2gh"},
	{"passcode many words", `passcode: four random words here`, "random words"},
	{"passcode no separator", `passcode four random words here`, "random"},
	{"passcode logfmt words", `level=INFO passcode=four random words here`, "random"},
	{"password no separator", `the password correct horse battery staple`, "horse"},
	{"passcode in json text", `{"msg":"passcode four random words here","n":1}`, "random"},
	{"passcode escaped quote", `{"passcode":"ab\"cd efgh"}`, "efgh"},
	{"pin", `pin=4821`, "4821"},
	{"env dump token", `D2DVIEW_TOKEN=` + fakeTok[:24], fakeTok[:24]},
	{"env dump export", `export WATCH_TOKEN='` + fakeTok[:24] + `'`, fakeTok[:24]},
	{"env dump password", `DB_PASSWORD=hunter2 with space`, "hunter2"},
	{"env dump user", `USERNAME=alice`, "alice"},
	{"env dump home", `HOME=` + "/ho" + `me/alice`, "alice"},
	{"env dump hostname", `HOSTNAME=alicebox`, "alicebox"},
	{"link token", `http://localhost:8891/watch?t=` + fakeTok[:20], fakeTok[:20]},
	{"link token second", `GET /api/x?a=1&token=abc123def`, "abc123def"},
	{"link passcode", `GET /login?passcode=hunter2x`, "hunter2x"},
	{"link user info", `fetch https://alice:s3cr3t@example.org/x`, "s3cr3t"},
	{"link host", `open http://alicebox:8891/watch`, "alicebox"},
	{"json token nested", `{"a":{"b":{"watch_token":"` + fakeTok[:24] + `"}}}`, fakeTok[:24]},
	{"json cookie", `{"cookie":"d2dview=abc123def456"}`, "abc123def456"},
	{"json number secret", `{"pin":482193}`, "482193"},
	{"json unlabelled token", `{"note":"` + fakeTok + `"}`, fakeTok},
	{"header cookie", `Cookie: d2dview=abc123def456ghi789`, "abc123def456"},
	{"header authorization", `Authorization: Basic YWxpY2U6czNjcjN0`, "YWxpY2U6"},
	{"pem", `-----BEGIN OPENSSH PRIVATE KEY----- b3BlbnNzaA`, "b3BlbnNzaA"},
	{"terminal escape", "name \x1b[2Jgone", "\x1b"},
	{"nul run", "a\x00\x00\x00b", "\x00"},
}

func TestScrubAdversarialCorpus(t *testing.T) {
	var all strings.Builder
	for _, c := range secCorpus {
		all.WriteString(c.in + "\n")
	}
	lines := strings.Split(strings.TrimSuffix(scrub(t, all.String()), "\n"), "\n")
	if len(lines) != len(secCorpus) {
		t.Fatalf("%d lines in, %d out", len(secCorpus), len(lines))
	}
	for i, c := range secCorpus {
		if strings.Contains(strings.ToLower(lines[i]), strings.ToLower(c.secret)) {
			t.Errorf("%s: %q still contains %q", c.name, lines[i], c.secret)
		}
		if strings.HasPrefix(c.in, "{") && !json.Valid([]byte(lines[i])) {
			t.Errorf("%s: no longer JSON: %q", c.name, lines[i])
		}
	}
}

// What review 12 C4 found broken or hidden for no reason: JSON lines stay JSON, the build identity and plain facts stay.
func TestScrubKeepsShapeAndIdentity(t *testing.T) {
	hex40, hex64 := strings.Repeat("0123456789abcdef", 3)[:40], strings.Repeat("0123456789abcdef", 4)
	keep := []string{
		`commit ` + hex40,
		`bot_sha256=` + hex64,
		`{"build":"abc123def456","commit":"` + hex40 + `"}`,
		`game exe version 1.14.3.71`,
		`user_go=true dry=false`,
		`{"auth":{"mode":"cookie","ok":true},"n":2}`,
		`kills 1200 2300 4500 ok`,
		`build-paladin-zealot-lite loaded`,
		`BLOOD MOOR COLD PLAINS`,
		`key press Return then wait`,
	}
	for _, k := range keep {
		if out := strings.TrimSpace(scrub(t, k+"\n")); out != k {
			t.Errorf("changed:\n in  %q\n out %q", k, out)
		}
	}
	for _, j := range []string{
		`{"auth":{"mode":"token","ok":true},"left":1}`,
		`{"budget":{"token_left":2},"user_go":true,"session_id":null}`,
		`{"passcode":"a b c","list":[1,2],"token":12345}`,
	} {
		if out := strings.TrimSpace(scrub(t, j+"\n")); !json.Valid([]byte(out)) {
			t.Errorf("no longer JSON:\n in  %q\n out %q", j, out)
		}
	}
}

func scanKinds(t *testing.T, in string) (string, int) {
	t.Helper()
	cmd := exec.Command("sh", "tester-report.sh")
	cmd.Env = append(os.Environ(), "REPORT_SCAN_ONLY=1", "D2BOT_DATA="+t.TempDir())
	cmd.Stdin = strings.NewReader(in)
	out, err := cmd.CombinedOutput()
	if ee, ok := err.(*exec.ExitError); ok {
		return string(out), ee.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return string(out), 0
}

// The second line of defence has its own patterns: raw secrets are found, scrubbed text and plain log lines are not,
// and the answer never repeats the secret.
func TestLeftoverScan(t *testing.T) {
	raw := []struct{ in, secret string }{
		{`url http://x/watch?t=` + fakeTok[:20], fakeTok[:20]},
		{`Authorization: ` + fakeBearr, "abc123DEF456"},
		{`from ` + fakeMail, "alice"},
		{`open ` + fakeHome, "alice"},
		{`open C:\Users\alice\x`, "alice"},
		{`peer ` + fakeIP + `:1`, fakeIP},
		{`peer 100.101.102.103`, "101.102"},
		{`passcode: hunter2hunter2`, "hunter2"},
		{`{"password":"open sesame"}`, "sesame"},
		{`dial ` + fakeHost, "box1"},
		{`-----BEGIN RSA PRIVATE KEY-----`, "\x00"},
	}
	for _, c := range raw {
		out, code := scanKinds(t, c.in+"\n")
		if code != 3 || strings.TrimSpace(out) == "" {
			t.Errorf("%q: exit %d, kinds %q; want exit 3 and a kind", c.in, code, out)
		}
		if strings.Contains(out, c.secret) {
			t.Errorf("%q: the scan printed the secret: %q", c.in, out)
		}
	}
	var all strings.Builder
	for _, c := range secCorpus {
		all.WriteString(c.in + "\n")
	}
	all.WriteString("game version 10.0.19045.1\nlisten 0.0.0.0:8891 loop 127.0.0.1:99\npasscode accepted\n")
	if out, code := scanKinds(t, scrub(t, all.String())); code != 0 {
		t.Errorf("scrubbed corpus still has a finding (exit %d): %s", code, out)
	}
}

// A secret the scrubber cannot reach (a file name) stops the bundle: no tarball, exit 3, the kind is named.
func TestBundleRefusedOnLeftover(t *testing.T) {
	data, out := t.TempDir(), filepath.Join(t.TempDir(), "reports")
	dir := filepath.Join(data, "instances", fakeIP, "state", "ladder")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "journal.jsonl"), []byte(`{"q":"t0.first"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("sh", "tester-report.sh")
	cmd.Env = append(os.Environ(), "D2BOT_DATA="+data, "REPORT_OUT="+out, "D2D_SOCK="+filepath.Join(data, "none.sock"))
	b, err := cmd.CombinedOutput()
	ee, ok := err.(*exec.ExitError)
	if !ok || ee.ExitCode() != 3 {
		t.Fatalf("want exit 3, got %v\n%s", err, b)
	}
	if !strings.Contains(string(b), "NOT written: private address in a file name") || strings.Contains(string(b), fakeIP) {
		t.Errorf("message must name the kind and not the text: %s", b)
	}
	if left, _ := filepath.Glob(filepath.Join(out, "*.tar.gz")); len(left) != 0 {
		t.Errorf("a tarball was written: %v", left)
	}
}

// The bundle and its folder are private to the owner (review 4: default umask in the data volume).
func TestBundleIsOwnerOnly(t *testing.T) {
	data, out := t.TempDir(), filepath.Join(t.TempDir(), "reports")
	cmd := exec.Command("sh", "tester-report.sh")
	cmd.Env = append(os.Environ(), "D2BOT_DATA="+data, "REPORT_OUT="+out, "D2D_SOCK="+filepath.Join(data, "none.sock"))
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run: %v\n%s", err, b)
	}
	for _, p := range []string{out, strings.TrimSpace(string(b))} {
		st, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		if st.Mode().Perm()&0o077 != 0 {
			t.Errorf("%s has mode %v; want nothing for group and others", p, st.Mode().Perm())
		}
	}
}

// No rule backtracks badly: lines built to make a careless pattern explode finish in the time of a normal line.
// The ceiling is far above the plain number (about 0.1 s for all of them, V this box) and far below a blow-up.
func TestScrubLongLinesBounded(t *testing.T) {
	const n = 300 << 10
	rep := func(s string) string { return strings.Repeat(s, n/len(s)) }
	in := strings.Join([]string{
		rep("a."), rep("a:"), rep("a-"), rep("ab12"), rep("1."), rep("AB12 "), rep("abcd-"), rep(`\"`), rep("a@"), rep("/a"),
		"passcode " + rep("x "), `{"token":"` + rep(`\\`) + `"}`, "key " + rep("abcd 1234 "), rep("f:"), rep("user_"),
		"http://" + rep("a.") + "/x?t=" + rep("b&t="),
	}, "\n") + "\n"
	start := time.Now()
	out := scrub(t, in)
	d := time.Since(start)
	t.Logf("%d bytes in, %d out, %v", len(in), len(out), d)
	if strings.Count(out, "\n") != strings.Count(in, "\n") {
		t.Errorf("lines in %d, out %d", strings.Count(in, "\n"), strings.Count(out, "\n"))
	}
	if d > 10*time.Second {
		t.Errorf("scrub took %v for %d bytes: a rule backtracks", d, len(in))
	}
	if d > time.Second {
		t.Logf("slower than the 1 s target of the card (loaded box?)")
	}
}
