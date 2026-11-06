package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/access"
)

func init() { access.Iterations = 1000 }

const testToken = "tok-for-tests-only-AAAAAAAAAAAAAAAA"

// small tables: the same shape as tables.json with tiny size bounds, so fake files stay tiny.
const smallTables = `{
 "files": [
  {"name": "d2data.mpq", "required": true, "min_size": 64, "why": "base"},
  {"name": "d2exp.mpq", "required": true, "min_size": 64, "why": "lod"},
  {"name": "d2char.mpq", "required": true, "min_size": 64, "why": "chars"},
  {"name": "Patch_D2.mpq", "required": true, "min_size": 64, "why": "patch"},
  {"name": "d2music.mpq", "why": "music"}
 ],
 "patch_sha256": [],
 "classes": [
  {"id": "sorceress", "name": "Sorceress", "d2d_index": 4, "supported": true},
  {"id": "necromancer", "name": "Necromancer", "d2d_index": 1}
 ],
 "default_class": "sorceress",
 "presets": [{"id": "campaign:normal_act1", "name": "Act 1"}],
 "default_preset": "campaign:normal_act1"
}`

func mpq(t *testing.T, dir, name string, size int) {
	t.Helper()
	b := make([]byte, size)
	copy(b, "MPQ\x1a")
	if err := os.WriteFile(filepath.Join(dir, name), b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func goodGame(t *testing.T) string {
	d := t.TempDir()
	for _, n := range []string{"d2data.mpq", "d2exp.mpq", "d2char.mpq", "patch_d2.mpq"} { // mixed case on purpose
		mpq(t, d, n, 128)
	}
	return d
}

func tables(t *testing.T, raw string) *Tables {
	tb, err := LoadTables([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	return tb
}

func status(r Report, name string) FileCheck {
	for _, f := range r.Files {
		if f.Name == name {
			return f
		}
	}
	return FileCheck{}
}

func TestDefaultTablesLoad(t *testing.T) {
	tb := DefaultTables()
	c, err := tb.Class("sorceress")
	if err != nil || !c.Supported || c.D2DIndex != 4 {
		t.Fatalf("sorceress %+v %v", c, err)
	}
	for _, id := range []string{"barbarian", "necromancer", "paladin", "amazon", "druid", "assassin"} {
		if c, err := tb.Class(id); err != nil || c.Supported {
			t.Errorf("%s %+v %v", id, c, err)
		}
	}
	if _, err := tb.Class("wizard"); err == nil || !strings.Contains(err.Error(), `"wizard"`) {
		t.Errorf("unknown class error %v", err)
	}
	for _, bad := range []string{`{"bogus":1}`, strings.Replace(smallTables, `"default_class": "sorceress"`, `"default_class": "necromancer"`, 1), strings.Replace(smallTables, `"campaign:normal_act1", "name"`, `"campaign:nope", "name"`, 1)} {
		if _, err := LoadTables([]byte(bad)); err == nil {
			t.Errorf("accepted %.60s", bad)
		}
	}
}

func TestCheckGame(t *testing.T) {
	tb := tables(t, smallTables)
	r := CheckGame(goodGame(t), tb)
	if !r.OK || r.Version != "unverified" || r.PatchSHA256 == "" {
		t.Fatalf("good: %+v", r)
	}
	if f := status(r, "d2music.mpq"); f.Status != StatusInfo {
		t.Errorf("optional: %+v", f)
	}

	d := goodGame(t)
	os.Remove(filepath.Join(d, "d2exp.mpq"))
	r = CheckGame(d, tb)
	if r.OK || status(r, "d2exp.mpq").Status != StatusFail || !strings.Contains(status(r, "d2exp.mpq").Message, "d2exp.mpq is missing") || !strings.Contains(r.Summary, "d2exp.mpq") {
		t.Errorf("missing: %+v", r)
	}

	d = goodGame(t)
	mpq(t, d, "d2char.mpq", 10) // too small
	if f := status(CheckGame(d, tb), "d2char.mpq"); f.Status != StatusFail || !strings.Contains(f.Message, "too small") {
		t.Errorf("small: %+v", f)
	}

	d = goodGame(t)
	os.WriteFile(filepath.Join(d, "d2data.mpq"), bytes.Repeat([]byte("x"), 200), 0o644) // wrong header
	if f := status(CheckGame(d, tb), "d2data.mpq"); f.Status != StatusFail || !strings.Contains(f.Message, "not a Diablo II archive") {
		t.Errorf("header: %+v", f)
	}

	d = goodGame(t)
	os.Chmod(filepath.Join(d, "d2data.mpq"), 0)
	if os.Geteuid() != 0 { // root reads anything
		if f := status(CheckGame(d, tb), "d2data.mpq"); f.Status != StatusFail || !strings.Contains(f.Message, "cannot be read") {
			t.Errorf("unreadable: %+v", f)
		}
	}

	// optional file that is broken is a warning, not a failure
	d = goodGame(t)
	os.WriteFile(filepath.Join(d, "d2music.mpq"), []byte("junk"), 0o644)
	if r := CheckGame(d, tb); !r.OK || status(r, "d2music.mpq").Status != StatusWarn {
		t.Errorf("optional broken: %+v", r)
	}

	// nothing there / D2R-like folder / no folder
	e := t.TempDir()
	os.WriteFile(filepath.Join(e, "Diablo II Resurrected Launcher.exe"), nil, 0o644)
	if r := CheckGame(e, tb); r.OK || !strings.Contains(r.Summary, "Resurrected") {
		t.Errorf("d2r: %+v", r)
	}
	if r := CheckGame(filepath.Join(e, "nope"), tb); r.OK || !strings.Contains(r.Summary, "Mount your Diablo II folder") {
		t.Errorf("nodir: %+v", r)
	}
}

func TestVersionPin(t *testing.T) {
	d := goodGame(t)
	b, _ := os.ReadFile(filepath.Join(d, "patch_d2.mpq"))
	sum := sha256.Sum256(b)
	pinned := strings.Replace(smallTables, `"patch_sha256": []`, `"patch_sha256": ["`+hex.EncodeToString(sum[:])+`"]`, 1)
	if r := CheckGame(d, tables(t, pinned)); !r.OK || r.Version != "1.14d" {
		t.Errorf("match: %+v", r)
	}
	other := strings.Replace(smallTables, `"patch_sha256": []`, `"patch_sha256": ["`+strings.Repeat("0", 64)+`"]`, 1)
	if r := CheckGame(d, tables(t, other)); r.OK || r.Version != "mismatch" || !strings.Contains(r.Summary, "1.14d") {
		t.Errorf("mismatch: %+v", r)
	}
}

// The six verdict states of the game files screen (mock screen 3) plus the folder that cannot be read.
func TestFileVerdicts(t *testing.T) {
	tb := tables(t, smallTables)
	pin := func(d string) *Tables {
		b, _ := os.ReadFile(filepath.Join(d, "patch_d2.mpq"))
		sum := sha256.Sum256(b)
		return tables(t, strings.Replace(smallTables, `"patch_sha256": []`, `"patch_sha256": ["`+hex.EncodeToString(sum[:])+`"]`, 1))
	}
	cases := []struct {
		name    string
		setup   func(d string)
		tb      func(d string) *Tables
		verdict string
		ok      bool
		issue   string // issue of the first failing file
	}{
		{"all good and pinned", func(string) {}, pin, "ok", true, ""},
		{"hash not pinned yet", func(string) {}, func(string) *Tables { return tb }, "unconfirmed", true, ""},
		{"missing patch", func(d string) { os.Remove(filepath.Join(d, "patch_d2.mpq")) }, func(string) *Tables { return tb }, "missing", false, "missing"},
		{"too small", func(d string) { mpq(t, d, "d2exp.mpq", 10) }, func(string) *Tables { return tb }, "too_small", false, "too_small"},
		{"not an archive", func(d string) { os.WriteFile(filepath.Join(d, "d2exp.mpq"), make([]byte, 128), 0o644) }, func(string) *Tables { return tb }, "not_mpq", false, "not_mpq"},
		{"two kinds of problem", func(d string) {
			os.Remove(filepath.Join(d, "d2char.mpq"))
			mpq(t, d, "d2exp.mpq", 10)
		}, func(string) *Tables { return tb }, "problems", false, ""},
		{"wrong version", func(string) {}, func(string) *Tables {
			return tables(t, strings.Replace(smallTables, `"patch_sha256": []`, `"patch_sha256": ["`+strings.Repeat("0", 64)+`"]`, 1))
		}, "wrong_version", false, ""},
		{"Resurrected folder", func(d string) {
			es, _ := os.ReadDir(d)
			for _, e := range es {
				os.Remove(filepath.Join(d, e.Name()))
			}
			os.WriteFile(filepath.Join(d, "Diablo II Resurrected.exe"), []byte("MZ"), 0o644)
		}, func(string) *Tables { return tb }, "resurrected", false, ""},
	}
	for _, c := range cases {
		d := goodGame(t)
		c.setup(d)
		r := CheckGame(d, c.tb(d))
		if r.Verdict != c.verdict || r.OK != c.ok {
			t.Errorf("%s: verdict %q ok %v (%s)", c.name, r.Verdict, r.OK, r.Summary)
		}
		if !r.OK && r.Fix == "" || c.verdict == "unconfirmed" && (r.Fix == "" || r.PatchSHA256 == "") {
			t.Errorf("%s: no fix hint / fingerprint: %+v", c.name, r)
		}
		if c.issue != "" {
			found := false
			for _, f := range r.Files {
				if f.Status == StatusFail && f.Issue == c.issue {
					found = true
				}
			}
			if !found {
				t.Errorf("%s: no file with issue %s: %+v", c.name, c.issue, r.Files)
			}
		}
	}
	if r := CheckGame(filepath.Join(t.TempDir(), "nope"), tb); r.Verdict != "no_folder" || r.OK || r.Fix == "" || strings.Contains(r.Summary, t.TempDir()) {
		t.Errorf("no folder: %+v", r)
	}
}

func TestResetAccess(t *testing.T) {
	dir := t.TempDir()
	st := access.NewStore(filepath.Join(dir, accessFile))
	rec, _ := access.NewPasscode("old passcode", time.Now())
	st.Save(rec)
	oldCookie := rec.Session()

	// default: back to the generated token, which is created when missing; the old login stops working
	var out bytes.Buffer
	if err := resetAccess([]string{"-data", dir}, strings.NewReader(""), &out); err != nil {
		t.Fatal(err)
	}
	if got := st.Current(); got.Kind != access.KindToken || got.SessionOK(oldCookie) {
		t.Errorf("after token reset: %+v", got)
	}
	tok, err := os.ReadFile(filepath.Join(dir, "watch.token"))
	if err != nil || len(strings.TrimSpace(string(tok))) != 32 || strings.Contains(out.String(), strings.TrimSpace(string(tok))) {
		t.Errorf("token file %q %v; the command must name the file, not print the token: %s", tok, err, out.String())
	}

	// -passcode reads one line from standard input and never echoes it
	out.Reset()
	if err := resetAccess([]string{"-data", dir, "-passcode"}, strings.NewReader("brand new words\n"), &out); err != nil {
		t.Fatal(err)
	}
	if got := st.Current(); got.Kind != access.KindPasscode || !got.Verify("brand new words") || got.Verify("old passcode") || strings.Contains(out.String(), "brand") {
		t.Errorf("after passcode reset: %+v / %s", got, out.String())
	}
	if fi, _ := os.Stat(filepath.Join(dir, accessFile)); fi.Mode().Perm() != 0o600 {
		t.Errorf("mode %v", fi.Mode())
	}
	for name, in := range map[string]string{"short": "abc\n", "empty": "", "blank line": "\n"} {
		if err := resetAccess([]string{"-data", dir, "-passcode"}, strings.NewReader(in), &out); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	if got := st.Current(); !got.Verify("brand new words") {
		t.Error("a rejected reset changed the passcode")
	}
	if err := resetAccess([]string{"-data", dir, "extra"}, strings.NewReader(""), &out); err == nil {
		t.Error("stray argument accepted")
	}
}

func TestNameValidation(t *testing.T) {
	for _, ok := range []string{"Ab", "Sorcy", "Abcdefghijklmno"} {
		if validCharName(ok) != nil {
			t.Errorf("%q rejected", ok)
		}
	}
	for _, bad := range []string{"", "A", "Abcdefghijklmnop", "Sor cy", "Sorc1", "ABCD-1234-EFGH", "../x", "a'b", "<b>hi</b>"} {
		if validCharName(bad) == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

type fixture struct {
	data, game string
	w          *Wizard
	srv        *httptest.Server
	extracted  chan struct{}
	exErr      error
}

func newFixture(t *testing.T, raw string) *fixture {
	t.Helper()
	f := &fixture{data: t.TempDir(), game: goodGame(t), extracted: make(chan struct{}, 8)}
	f.build(t, raw)
	return f
}

func (f *fixture) build(t *testing.T, raw string) {
	w, err := NewWizard(f.data, f.game, tables(t, raw), func(string) error { f.extracted <- struct{}{}; return f.exErr })
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewServer(w, testToken)
	if err != nil {
		t.Fatal(err)
	}
	f.w = w
	if f.srv != nil {
		f.srv.Close()
	}
	f.srv = httptest.NewServer(s)
	t.Cleanup(f.srv.Close)
}

func (f *fixture) do(t *testing.T, method, path, body string, hdr ...string) (int, string, http.Header) {
	t.Helper()
	req, _ := http.NewRequest(method, f.srv.URL+path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for i := 0; i+1 < len(hdr); i += 2 {
		req.Header.Set(hdr[i], hdr[i+1])
	}
	if req.Header.Get("Authorization") == "" && req.Header.Get("X-No-Auth") == "" {
		req.Header.Set("Authorization", "Bearer "+testToken)
	}
	c := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b), resp.Header
}

func (f *fixture) state(t *testing.T) View {
	t.Helper()
	code, body, _ := f.do(t, "GET", "/api/state", "")
	var v View
	if code != 200 || json.Unmarshal([]byte(body), &v) != nil {
		t.Fatalf("state %d %s", code, body)
	}
	return v
}

func (f *fixture) waitPhase(t *testing.T, want string) View {
	t.Helper()
	for i := 0; i < 200; i++ {
		if v := f.state(t); v.Phase == want {
			return v
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("phase never %s (now %s)", want, f.state(t).Phase)
	return View{}
}

// ready walks the first four steps with the given access choice (a JSON body for PUT /api/access).
func (f *fixture) ready(t *testing.T, accessBody string) {
	t.Helper()
	for _, c := range [][3]string{{"PUT", "/api/ahead", `{"ok":true}`}, {"PUT", "/api/inputs", goodInputs}, {"PUT", "/api/access", accessBody}} {
		if code, body, _ := f.do(t, c[0], c[1], c[2]); code != 200 {
			t.Fatalf("%s %s: %d %s", c[0], c[1], code, body)
		}
	}
}

const tokenChoice = `{"kind":"token"}`
const passChoice = `{"kind":"passcode","passcode":"four plain words","confirm":"four plain words"}`

const goodInputs = `{"char_name":"Sorcy","class":"sorceress","goals":"campaign:normal_act1"}`

func TestHappyPath(t *testing.T) {
	f := newFixture(t, smallTables)
	if v := f.state(t); v.Phase != "ahead" || v.Ahead.OK || !v.Check.OK || v.Inputs.Class != "sorceress" || v.Inputs.Goals != "campaign:normal_act1" || v.Access != "" {
		t.Fatalf("first state %+v", v)
	}
	if _, err := os.Stat(filepath.Join(f.data, stateFile)); err == nil {
		t.Fatal("state file written before the first answer")
	}
	// every step in order: go-ahead, game files (already fine), hero, access, start
	steps := []struct{ method, path, body, phase string }{
		{"PUT", "/api/ahead", `{"ok":true}`, "character"},
		{"PUT", "/api/inputs", goodInputs, "access"},
		{"PUT", "/api/access", tokenChoice, "start"},
	}
	for _, c := range steps {
		if code, body, _ := f.do(t, c.method, c.path, c.body); code != 200 || !strings.Contains(body, `"phase":"`+c.phase+`"`) {
			t.Fatalf("%s %s: %d %s (want phase %s)", c.method, c.path, code, body, c.phase)
		}
	}
	if code, _, _ := f.do(t, "POST", "/api/start", ""); code != 202 {
		t.Fatalf("start %d", code)
	}
	v := f.waitPhase(t, "done")
	if v.Token != testToken {
		t.Errorf("token not handed out once: %q", v.Token)
	}
	if again := f.state(t); again.Token != "" {
		t.Error("token shown twice")
	}
	if len(f.extracted) != 1 {
		t.Errorf("extract ran %d times", len(f.extracted))
	}
	b, err := os.ReadFile(filepath.Join(f.data, stateFile))
	if err != nil {
		t.Fatal(err)
	}
	var s Saved
	if err := json.Unmarshal(b, &s); err != nil || s.Phase != "done" || s.CharName != "Sorcy" || s.ClassIndex != 4 || s.Goals != "campaign:normal_act1" || s.PatchSHA256 == "" {
		t.Errorf("wizard.json %s %v", b, err)
	}
	if _, err := time.Parse(time.RFC3339, s.AheadAt); err != nil || s.StartedAt == "" {
		t.Errorf("go-ahead date / start time not stored: %+v", s)
	}
	if fi, _ := os.Stat(filepath.Join(f.data, stateFile)); fi.Mode().Perm() != 0o600 {
		t.Errorf("mode %v", fi.Mode())
	}
	env, _ := os.ReadFile(filepath.Join(f.data, envFile))
	if string(env) != "D2D_NAME='Sorcy'\nD2D_CLASS='4'\nD2BOT_GOALS='campaign:normal_act1'\n" {
		t.Errorf("wizard.env %q", env)
	}
	if strings.Contains(string(b), testToken) {
		t.Error("token inside wizard.json")
	}
	// a second start, new inputs, a new go-ahead or a new access choice after done are refused
	for _, c := range [][3]string{{"POST", "/api/start", ""}, {"PUT", "/api/inputs", goodInputs}, {"PUT", "/api/ahead", `{"ok":false}`}, {"PUT", "/api/access", passChoice}} {
		if code, _, _ := f.do(t, c[0], c[1], c[2]); code != 409 {
			t.Errorf("%s %s after done %d", c[0], c[1], code)
		}
	}
}

// The passcode kind: no token in the done view, only a hash in access.json (0600), and this browser is signed in.
func TestHappyPathPasscode(t *testing.T) {
	f := newFixture(t, smallTables)
	f.do(t, "PUT", "/api/ahead", `{"ok":true}`)
	f.do(t, "PUT", "/api/inputs", goodInputs)
	code, body, h := f.do(t, "PUT", "/api/access", passChoice)
	if code != 200 || !strings.Contains(body, `"access":"passcode"`) || strings.Contains(body, "four plain") || strings.Contains(body, "hash") {
		t.Fatalf("access %d %s", code, body)
	}
	ck := h.Get("Set-Cookie")
	for _, want := range []string{"d2dpass=", "HttpOnly", "SameSite=Strict", "Path=/"} {
		if !strings.Contains(ck, want) {
			t.Errorf("cookie %q lacks %s", ck, want)
		}
	}
	if strings.Contains(ck, "Secure") {
		t.Error("Secure cookie on plain HTTP would never be sent back")
	}
	raw, _ := os.ReadFile(filepath.Join(f.data, accessFile))
	if strings.Contains(string(raw), "four plain") || !strings.Contains(string(raw), `"kind": "passcode"`) {
		t.Errorf("access.json %s", raw)
	}
	if fi, _ := os.Stat(filepath.Join(f.data, accessFile)); fi.Mode().Perm() != 0o600 {
		t.Errorf("access.json mode %v", fi.Mode())
	}
	// the session cookie alone opens the setup page again, without the token
	cookie := strings.SplitN(ck, ";", 2)[0]
	if code, _, _ := f.do(t, "GET", "/api/state", "", "X-No-Auth", "1", "Authorization", "", "Cookie", cookie); code != 200 {
		t.Errorf("passcode cookie: %d", code)
	}
	if code, _, _ := f.do(t, "GET", "/api/state", "", "X-No-Auth", "1", "Authorization", "", "Cookie", "d2dpass=forged"); code != 401 {
		t.Errorf("forged passcode cookie: %d", code)
	}
	f.do(t, "POST", "/api/start", "")
	if v := f.waitPhase(t, "done"); v.Token != "" || v.Access != "passcode" {
		t.Errorf("done view %+v", v)
	}
	// the stored record verifies the passcode (what the watch gate will do)
	if !f.w.Access.Current().Verify("four plain words") {
		t.Error("stored hash does not verify")
	}
}

func TestGoAheadEnforcedAndStored(t *testing.T) {
	f := newFixture(t, smallTables)
	f.do(t, "PUT", "/api/inputs", goodInputs)
	f.do(t, "PUT", "/api/access", tokenChoice)
	// everything else is ready, the tick is missing
	code, body, _ := f.do(t, "POST", "/api/start", "")
	if code != 400 || !strings.Contains(body, "go-ahead") || !strings.Contains(body, `"fix"`) || len(f.extracted) != 0 {
		t.Fatalf("start without go-ahead: %d %s", code, body)
	}
	if v := f.state(t); v.Phase != "ahead" {
		t.Errorf("phase %s", v.Phase)
	}
	now := time.Date(2026, 11, 6, 15, 4, 5, 0, time.UTC)
	f.w.Now = func() time.Time { return now }
	f.do(t, "PUT", "/api/ahead", `{"ok":true}`)
	if v := f.state(t); v.Ahead.At != "2026-11-06T15:04:05Z" || v.Phase != "start" {
		t.Errorf("tick not stored with date: %+v", v)
	}
	// ticking again keeps the first date; unticking clears it and blocks Start again
	now = now.Add(time.Hour)
	f.do(t, "PUT", "/api/ahead", `{"ok":true}`)
	if v := f.state(t); v.Ahead.At != "2026-11-06T15:04:05Z" {
		t.Errorf("date overwritten: %+v", v.Ahead)
	}
	f.do(t, "PUT", "/api/ahead", `{"ok":false}`)
	if code, _, _ := f.do(t, "POST", "/api/start", ""); code != 400 {
		t.Errorf("start after untick %d", code)
	}
	// it is remembered after a restart
	f.do(t, "PUT", "/api/ahead", `{"ok":true}`)
	f.build(t, smallTables)
	if v := f.state(t); !v.Ahead.OK || v.Phase != "start" || v.Access != "token" {
		t.Errorf("resume: %+v", v)
	}
	if code, _, _ := f.do(t, "PUT", "/api/ahead", `{"ok":true,"x":1}`); code != 400 {
		t.Errorf("unknown field %d", code)
	}
}

func TestAccessChoiceRules(t *testing.T) {
	f := newFixture(t, smallTables)
	for name, body := range map[string]string{
		"too short":  `{"kind":"passcode","passcode":"12345","confirm":"12345"}`,
		"mismatch":   `{"kind":"passcode","passcode":"abcdef","confirm":"abcdeg"}`,
		"no kind":    `{"kind":""}`,
		"other kind": `{"kind":"none"}`,
		"unknown":    `{"kind":"token","password":"x"}`,
	} {
		code, resp, _ := f.do(t, "PUT", "/api/access", body)
		if code != 400 {
			t.Errorf("%s: %d %s", name, code, resp)
		}
		if name == "too short" && (!strings.Contains(resp, "too short") || !strings.Contains(resp, `"fix"`)) {
			t.Errorf("short text %s", resp)
		}
	}
	if _, err := os.Stat(filepath.Join(f.data, accessFile)); err == nil {
		t.Fatal("a rejected choice was stored")
	}
	// no composition rules: six equal letters are fine; the choice can be changed until Start
	if code, _, _ := f.do(t, "PUT", "/api/access", `{"kind":"passcode","passcode":"aaaaaa","confirm":"aaaaaa"}`); code != 200 {
		t.Errorf("simple passcode %d", code)
	}
	if code, _, _ := f.do(t, "PUT", "/api/access", tokenChoice); code != 200 || f.state(t).Access != "token" {
		t.Errorf("switch to token %d", code)
	}
}

// Back on the page is a client matter; the server must let every earlier answer be changed until Start.
func TestEarlierAnswersCanBeChanged(t *testing.T) {
	f := newFixture(t, smallTables)
	f.ready(t, tokenChoice)
	if code, _, _ := f.do(t, "PUT", "/api/inputs", `{"char_name":"Other","class":"sorceress","goals":"campaign:normal_act1"}`); code != 200 {
		t.Fatalf("change hero %d", code)
	}
	os.Remove(filepath.Join(f.game, "d2char.mpq"))
	if code, body, _ := f.do(t, "POST", "/api/check", ""); code != 200 || !strings.Contains(body, `"phase":"data"`) || !strings.Contains(body, `"verdict":"missing"`) {
		t.Errorf("recheck %d %s", code, body)
	}
	if v := f.state(t); v.Inputs.CharName != "Other" || v.Access != "token" || !v.Ahead.OK {
		t.Errorf("answers lost: %+v", v)
	}
}

func TestFailureAndRerun(t *testing.T) {
	f := newFixture(t, smallTables)
	os.Remove(filepath.Join(f.game, "d2exp.mpq"))
	f.do(t, "PUT", "/api/ahead", `{"ok":true}`)
	v := f.state(t)
	if v.Phase != "data" || v.Check.OK {
		t.Fatalf("missing file: %+v", v)
	}
	// start is refused with the plain message, inputs can still be stored
	if code, body, _ := f.do(t, "POST", "/api/start", ""); code != 400 || !strings.Contains(body, "d2exp.mpq") {
		t.Errorf("start %d %s", code, body)
	}
	// the user fixes the mount; the check is re-run
	mpq(t, f.game, "D2Exp.MPQ", 128)
	code, body, _ := f.do(t, "POST", "/api/check", "")
	if code != 200 || !strings.Contains(body, `"phase":"character"`) {
		t.Fatalf("recheck %d %s", code, body)
	}
	// extraction fails: phase failed with a plain message and a fix hint, then a retry succeeds
	f.exErr = errors.New("bad archive")
	f.ready(t, tokenChoice)
	f.do(t, "POST", "/api/start", "")
	v = f.waitPhase(t, "failed")
	if !strings.Contains(v.Error, "bad archive") || v.Fix == "" || v.Progress == nil {
		t.Errorf("error %q fix %q", v.Error, v.Fix)
	}
	if _, err := os.Stat(filepath.Join(f.data, envFile)); err == nil {
		t.Error("wizard.env written after a failed run")
	}
	f.exErr = nil
	f.do(t, "POST", "/api/start", "")
	f.waitPhase(t, "done")
}

func TestInputsRejected(t *testing.T) {
	f := newFixture(t, smallTables)
	for name, body := range map[string]string{
		"cd key field":     `{"char_name":"Sorcy","class":"sorceress","goals":"campaign:normal_act1","cd_key":"ABCD"}`,
		"key as name":      `{"char_name":"ABCD-1234-EFGH-5678","class":"sorceress","goals":"campaign:normal_act1"}`,
		"unsupported":      `{"char_name":"Necro","class":"necromancer","goals":"campaign:normal_act1"}`,
		"unknown class":    `{"char_name":"Wizzy","class":"wizard","goals":"campaign:normal_act1"}`,
		"free-text goals":  `{"char_name":"Sorcy","class":"sorceress","goals":"campaign:normal_act1;rm"}`,
		"not json":         `[`,
		"trailing garbage": `{"char_name":"Sorcy"} x`,
	} {
		if code, body, _ := f.do(t, "PUT", "/api/inputs", body); code != 400 {
			t.Errorf("%s: %d %s", name, code, body)
		}
	}
	if code, body, _ := f.do(t, "PUT", "/api/inputs", `{"char_name":"Necro","class":"necromancer","goals":"campaign:normal_act1"}`); !strings.Contains(body, "not supported in this build") {
		t.Errorf("unsupported text %d %s", code, body)
	}
	if f.state(t).Phase != "ahead" {
		t.Error("rejected input moved the wizard")
	}
}

func TestTokenEnforcement(t *testing.T) {
	f := newFixture(t, smallTables)
	noAuth := []string{"X-No-Auth", "1", "Authorization", ""}
	for _, p := range []string{"/", "/setup.html", "/app.js", "/onboard.css", "/style.css", "/api/state"} {
		if code, _, _ := f.do(t, "GET", p, "", noAuth...); code != 401 {
			t.Errorf("GET %s without token: %d", p, code)
		}
	}
	for _, c := range [][]string{{"PUT", "/api/inputs"}, {"PUT", "/api/ahead"}, {"PUT", "/api/access"}, {"POST", "/api/start"}, {"POST", "/api/check"}} {
		if code, _, _ := f.do(t, c[0], c[1], "", noAuth...); code != 401 {
			t.Errorf("%s %s without token: %d", c[0], c[1], code)
		}
	}
	if code, _, _ := f.do(t, "GET", "/api/state", "", "Authorization", "Bearer "+strings.Repeat("x", 32)); code != 401 {
		t.Errorf("wrong token %d", code)
	}
	if code, _, _ := f.do(t, "GET", "/api/state", "", "X-No-Auth", "1", "Authorization", "", "Cookie", "d2dview=wrong"); code != 401 {
		t.Errorf("wrong cookie %d", code)
	}
	// ?t= sets the watch screen's cookie and redirects without the token in the URL
	code, _, h := f.do(t, "GET", "/setup.html?t="+testToken, "", noAuth...)
	if code != 303 || strings.Contains(h.Get("Location"), "t=") || !strings.Contains(h.Get("Set-Cookie"), "d2dview="+testToken) || !strings.Contains(h.Get("Set-Cookie"), "HttpOnly") {
		t.Errorf("t= redirect %d %v", code, h)
	}
	if code, body, h := f.do(t, "GET", "/", "", "X-No-Auth", "1", "Authorization", "", "Cookie", "d2dview="+testToken); code != 200 || !strings.Contains(body, "tester setup") || !strings.Contains(h.Get("Content-Security-Policy"), "default-src 'self'") {
		t.Errorf("cookie page %d", code)
	}
	// cross-origin and wrong content type are refused for writes
	if code, _, _ := f.do(t, "PUT", "/api/inputs", goodInputs, "Origin", "http://evil.example"); code != 403 {
		t.Errorf("foreign origin %d", code)
	}
	host := strings.TrimPrefix(f.srv.URL, "http://")
	if code, _, _ := f.do(t, "PUT", "/api/inputs", goodInputs, "Origin", "http://"+host); code != 200 {
		t.Errorf("same origin %d", code)
	}
	if code, _, _ := f.do(t, "PUT", "/api/inputs", goodInputs, "Content-Type", "text/plain"); code != 415 {
		t.Errorf("content type %d", code)
	}
	if _, err := NewServer(f.w, "short"); err == nil {
		t.Error("short token accepted")
	}
}

func TestResumeAfterRestart(t *testing.T) {
	f := newFixture(t, smallTables)
	f.ready(t, tokenChoice)
	// restart before Start: every answer survives, the wizard is at the start step
	f.build(t, smallTables)
	if v := f.state(t); v.Phase != "start" || v.Inputs.CharName != "Sorcy" || !v.Ahead.OK || v.Access != "token" || v.Interrupted {
		t.Fatalf("resume inputs: %+v", v)
	}
	f.do(t, "POST", "/api/start", "")
	f.waitPhase(t, "done")
	// restart after done: Done() is true (the entrypoint hook skips the page), token already shown
	f.build(t, smallTables)
	if !f.w.Done() {
		t.Fatal("done lost")
	}
	if v := f.state(t); v.Phase != "done" || v.Token != "" {
		t.Errorf("after done %+v", v)
	}
	// damaged state is an error that names the file
	os.WriteFile(filepath.Join(f.data, stateFile), []byte("{"), 0o600)
	if _, err := NewWizard(f.data, f.game, f.w.Tables, nil); err == nil || !strings.Contains(err.Error(), stateFile) {
		t.Errorf("damaged: %v", err)
	}
	os.WriteFile(filepath.Join(f.data, stateFile), []byte(`{"schema":1,"cd_key":"x"}`), 0o600)
	if _, err := NewWizard(f.data, f.game, f.w.Tables, nil); err == nil {
		t.Error("unknown field in wizard.json accepted")
	}
}

// A second profile: another class table (Necromancer supported and default, two presets) drives the same code.
func TestSecondProfile(t *testing.T) {
	raw := strings.NewReplacer(
		`"default_class": "sorceress"`, `"default_class": "necromancer"`,
		`{"id": "necromancer", "name": "Necromancer", "d2d_index": 1}`, `{"id": "necromancer", "name": "Necromancer", "d2d_index": 1, "supported": true}`,
		`"presets": [{"id": "campaign:normal_act1", "name": "Act 1"}]`, `"presets": [{"id": "campaign:normal_act1", "name": "Act 1"}, {"id": "levelto:12", "name": "Level 12"}]`,
	).Replace(smallTables)
	f := newFixture(t, raw)
	if v := f.state(t); v.Inputs.Class != "necromancer" || len(v.Presets) != 2 {
		t.Fatalf("profile state %+v", v)
	}
	f.do(t, "PUT", "/api/ahead", `{"ok":true}`)
	if code, body, _ := f.do(t, "PUT", "/api/inputs", `{"char_name":"Necro","class":"necromancer","goals":"levelto:12"}`); code != 200 {
		t.Fatalf("%d %s", code, body)
	}
	f.do(t, "PUT", "/api/access", passChoice)
	f.do(t, "POST", "/api/start", "")
	f.waitPhase(t, "done")
	env, _ := os.ReadFile(filepath.Join(f.data, envFile))
	if string(env) != "D2D_NAME='Necro'\nD2D_CLASS='1'\nD2BOT_GOALS='levelto:12'\n" {
		t.Errorf("env %q", env)
	}
}

func TestInterruptedStartIsResumable(t *testing.T) {
	f := newFixture(t, smallTables)
	f.ready(t, tokenChoice)
	f.w.mu.Lock()
	f.w.saved.StartedAt = "2026-11-06T15:00:00Z" // a start that was cut off by a restart
	f.w.saveLocked()
	f.w.mu.Unlock()
	f.build(t, smallTables)
	v := f.state(t)
	if v.Phase != "start" || !v.Interrupted || v.Progress != nil {
		t.Fatalf("interrupted: %+v", v)
	}
	f.do(t, "POST", "/api/start", "")
	f.waitPhase(t, "done")
}

// The warm-up progress comes from the stored start time, so a closed page or a second browser sees the same numbers.
func TestProgressSurvivesClosedPage(t *testing.T) {
	f := newFixture(t, smallTables)
	f.ready(t, tokenChoice)
	block := make(chan struct{})
	f.w.Extract = func(string) error { <-block; return nil }
	now := time.Date(2026, 11, 6, 15, 0, 0, 0, time.UTC)
	f.w.Now = func() time.Time { return now }
	if code, _, _ := f.do(t, "POST", "/api/start", ""); code != 202 {
		t.Fatal(code)
	}
	now = now.Add(30 * time.Second)
	v := f.state(t) // "another browser": a fresh request with no page state
	if v.Phase != "running" || v.Progress == nil || v.Progress.ElapsedSec != 30 || v.Progress.EstimateSec != 120 || v.Progress.Percent != 25 || len(v.Progress.Steps) != 3 || v.Progress.Steps[0].State != "doing" {
		t.Fatalf("progress %+v", v.Progress)
	}
	now = now.Add(10 * time.Minute)
	if v := f.state(t); v.Progress.Percent != 95 {
		t.Errorf("percent must cap below 100 while running: %d", v.Progress.Percent)
	}
	if code, _, _ := f.do(t, "POST", "/api/start", ""); code != 409 {
		t.Errorf("second start while running %d", code)
	}
	close(block)
	f.waitPhase(t, "done")
}

func TestPageIsXSSSafe(t *testing.T) {
	b, err := pageFS.ReadFile("page/app.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"innerHTML", "outerHTML", "insertAdjacentHTML", "document.write", "eval(", "cd_key", "cdkey"} {
		if strings.Contains(string(b), bad) {
			t.Errorf("app.js contains %q", bad)
		}
	}
	idx, _ := pageFS.ReadFile("page/index.html")
	if strings.Contains(string(idx), "<script>") || strings.Contains(string(idx), " onclick") || strings.Contains(string(idx), "style=") {
		t.Error("inline script or style in index.html (the CSP forbids it)")
	}
	css, _ := pageFS.ReadFile("page/onboard.css")
	for name, text := range map[string]string{"app.js": string(b), "index.html": string(idx), "onboard.css": string(css)} {
		if strings.Contains(text, "http://") && name != "app.js" || strings.Contains(text, "https://") || strings.Contains(text, "@import") {
			t.Errorf("%s reaches for an external URL", name)
		}
	}
	for _, want := range []string{"textContent", "aria-current", "role: 'alert'", "aria-label"} {
		if !strings.Contains(string(b), want) {
			t.Errorf("app.js lacks %q", want)
		}
	}
	if strings.Contains(string(b), "style=") || strings.Contains(string(b), "'style'") {
		t.Error("app.js sets a style attribute (blocked by the CSP); use el.style")
	}
}

func TestTokenFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sub", "watch.token")
	tok, created, err := loadOrCreateToken(p)
	if err != nil || !created || len(tok) != 32 {
		t.Fatalf("%q %v %v", tok, created, err)
	}
	if fi, _ := os.Stat(p); fi.Mode().Perm() != 0o600 {
		t.Errorf("mode %v", fi.Mode())
	}
	tok2, created, err := loadOrCreateToken(p)
	if err != nil || created || tok2 != tok {
		t.Errorf("reload %q %v %v", tok2, created, err)
	}
	os.WriteFile(p, []byte("short\n"), 0o600)
	if _, _, err := loadOrCreateToken(p); err == nil {
		t.Error("short token file accepted")
	}
}

func TestRealExtractorFailsOnFakeArchives(t *testing.T) {
	err := tableExtractor(filepath.Join(t.TempDir(), "tables"))(goodGame(t))
	if err == nil {
		t.Fatal("fake MPQs extracted")
	}
}
