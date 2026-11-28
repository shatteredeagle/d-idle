package watch

import (
	"bytes"
	"errors"
	"log"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDifficultySection(t *testing.T) {
	cfg := &DifficultyConfig{}
	sec := DifficultySection(cfg)
	build := func(o DifficultyOffer) *SectionView {
		cfg.Offer = func() DifficultyOffer { return o }
		return sec.Build(&ReturnData{})
	}
	if build(DifficultyOffer{}) != nil {
		t.Fatal("no target must hide the section")
	}
	v := build(DifficultyOffer{Target: "nightmare", Missing: []string{"clvl 20, need 38"}})
	if len(v.Buttons) != 0 || !strings.Contains(strings.Join(v.Lines, "|"), "clvl 20, need 38") {
		t.Fatalf("not ready: %+v", v)
	}
	v = build(DifficultyOffer{Target: "hell", Ready: true, Line: "ready for hell (U): waiting for your go"})
	if len(v.Buttons) != 1 || v.Buttons[0].Label != "Ready for Hell: Go" || v.Buttons[0].Act != "difficulty" || v.Buttons[0].Row != "hell" {
		t.Fatalf("ready: %+v", v)
	}
	v = build(DifficultyOffer{Target: "nightmare", Ready: true, Pending: true})
	if len(v.Buttons) != 0 || !strings.Contains(v.Lines[0], "recorded") {
		t.Fatalf("pending: %+v", v)
	}
}

func TestDifficultyPost(t *testing.T) {
	ready := false
	var got []string
	cfg := &DifficultyConfig{Go: func(tg string) (string, error) {
		got = append(got, tg)
		if !ready {
			return "", errors.New("not ready for " + tg + ": clvl 20, need 38")
		}
		return "ok", nil
	}}
	h := DifficultyPostHandler(cfg, AnswerOptions{Token: "tok", Origins: []string{"http://ok"}})
	do := func(token, ct, body string) (int, string) {
		req := httptest.NewRequest("POST", DifficultyPostRoute, strings.NewReader(body))
		req.Header.Set("Content-Type", ct)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w.Code, w.Body.String()
	}
	if c, _ := do("", "application/json", `{"target":"nightmare"}`); c != 401 || len(got) != 0 {
		t.Errorf("no token: %d", c)
	}
	if c, _ := do("tok", "application/json", `{"target":"normal"}`); c != 400 || len(got) != 0 {
		t.Errorf("bad target: %d", c)
	}
	if c, b := do("tok", "application/json", `{"target":"nightmare"}`); c != 409 || !strings.Contains(b, "clvl 20, need 38") {
		t.Errorf("not ready: %d %s", c, b)
	}
	ready = true
	if c, _ := do("tok", "application/json", `{"target":"nightmare"}`); c != 200 {
		t.Errorf("ready: %d", c)
	}
}

// The page door writes an audit line for every go (no address in it) and has an hourly budget.
func TestDifficultyPostAuditAndBudget(t *testing.T) {
	var buf bytes.Buffer
	old := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(old)
	now := time.Date(2026, 11, 27, 12, 0, 0, 0, time.UTC)
	n := 0
	cfg := &DifficultyConfig{Now: func() time.Time { return now }, Go: func(string) (string, error) {
		n++
		if n == 1 {
			return "", errors.New("a measured run is in progress")
		}
		return "ok", nil
	}}
	h := DifficultyPostHandler(cfg, AnswerOptions{Token: "tok"})
	do := func() int {
		req := httptest.NewRequest("POST", DifficultyPostRoute, strings.NewReader(`{"target":"hell"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer tok")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w.Code
	}
	if c := do(); c != 409 {
		t.Fatalf("refused go: %d", c)
	}
	if l := buf.String(); !strings.Contains(l, "action=go-difficulty quest=hell") || !strings.Contains(l, "ok=false") || !strings.Contains(l, "measured run") || strings.Contains(l, "192.0.2.") {
		t.Fatalf("audit line of a refusal: %q", l)
	}
	for i := 1; i < difficultyGoPerHour; i++ {
		if c := do(); c != 200 {
			t.Fatalf("go %d: %d", i, c)
		}
	}
	if c := do(); c != 429 || n != difficultyGoPerHour {
		t.Fatalf("over the budget: %d after %d calls", c, n)
	}
	if !strings.Contains(buf.String(), "code=rate_limited") {
		t.Error("the refusal by budget is not in the audit line")
	}
	now = now.Add(time.Hour)
	if c := do(); c != 200 {
		t.Fatalf("an hour later: %d", c)
	}
}
