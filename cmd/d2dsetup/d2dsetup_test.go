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
)

const testToken = "0123456789abcdef0123456789abcdef"

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

const goodInputs = `{"char_name":"Sorcy","class":"sorceress","goals":"campaign:normal_act1"}`

func TestHappyPath(t *testing.T) {
	f := newFixture(t, smallTables)
	if v := f.state(t); v.Phase != "character" || !v.Check.OK || v.Inputs.Class != "sorceress" || v.Inputs.Goals != "campaign:normal_act1" {
		t.Fatalf("first state %+v", v)
	}
	if _, err := os.Stat(filepath.Join(f.data, stateFile)); err == nil {
		t.Fatal("state file written before Start")
	}
	if code, body, _ := f.do(t, "PUT", "/api/inputs", goodInputs); code != 200 || !strings.Contains(body, `"start"`) {
		t.Fatalf("inputs %d %s", code, body)
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
	// a second start or new inputs after done are refused
	if code, _, _ := f.do(t, "POST", "/api/start", ""); code != 409 {
		t.Errorf("start after done %d", code)
	}
	if code, _, _ := f.do(t, "PUT", "/api/inputs", goodInputs); code != 409 {
		t.Errorf("inputs after done %d", code)
	}
}

func TestFailureAndRerun(t *testing.T) {
	f := newFixture(t, smallTables)
	os.Remove(filepath.Join(f.game, "d2exp.mpq"))
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
	// extraction fails: phase failed, then a retry succeeds
	f.exErr = errors.New("bad archive")
	f.do(t, "PUT", "/api/inputs", goodInputs)
	f.do(t, "POST", "/api/start", "")
	v = f.waitPhase(t, "failed")
	if !strings.Contains(v.Error, "bad archive") {
		t.Errorf("error %q", v.Error)
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
	if f.state(t).Phase != "character" {
		t.Error("rejected input moved the wizard")
	}
}

func TestTokenEnforcement(t *testing.T) {
	f := newFixture(t, smallTables)
	noAuth := []string{"X-No-Auth", "1", "Authorization", ""}
	for _, p := range []string{"/", "/setup.html", "/app.js", "/style.css", "/api/state"} {
		if code, _, _ := f.do(t, "GET", p, "", noAuth...); code != 401 {
			t.Errorf("GET %s without token: %d", p, code)
		}
	}
	for _, c := range [][]string{{"PUT", "/api/inputs"}, {"POST", "/api/start"}, {"POST", "/api/check"}} {
		if code, _, _ := f.do(t, c[0], c[1], "", noAuth...); code != 401 {
			t.Errorf("%s %s without token: %d", c[0], c[1], code)
		}
	}
	if code, _, _ := f.do(t, "GET", "/api/state", "", "Authorization", "Bearer wrong"); code != 401 {
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
	f.do(t, "PUT", "/api/inputs", goodInputs)
	// restart before Start: inputs survive, the wizard is at the start step
	f.build(t, smallTables)
	if v := f.state(t); v.Phase != "start" || v.Inputs.CharName != "Sorcy" {
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
	if code, body, _ := f.do(t, "PUT", "/api/inputs", `{"char_name":"Necro","class":"necromancer","goals":"levelto:12"}`); code != 200 {
		t.Fatalf("%d %s", code, body)
	}
	f.do(t, "POST", "/api/start", "")
	f.waitPhase(t, "done")
	env, _ := os.ReadFile(filepath.Join(f.data, envFile))
	if string(env) != "D2D_NAME='Necro'\nD2D_CLASS='1'\nD2BOT_GOALS='levelto:12'\n" {
		t.Errorf("env %q", env)
	}
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
	if strings.Contains(string(idx), "<script>") || strings.Contains(string(idx), " onclick") {
		t.Error("inline script in index.html (the CSP forbids it)")
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
