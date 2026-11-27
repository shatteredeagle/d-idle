package watch

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
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
