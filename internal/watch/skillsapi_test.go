package watch

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"d2lod-bot/internal/data"
	"d2lod-bot/internal/skilltree"
)

type fakeSkills struct{ v skilltree.View }

func (f fakeSkills) View() skilltree.View { return f.v }

func skillsFixture(class, name string, lvl, unspent int) skilltree.View {
	sk := []*data.Skill{
		{ID: 36, Key: "Fire Bolt", Name: "Fire Bolt", Class: "sor", Page: 1, Row: 1, Column: 2, ReqLevel: 1, MaxLevel: 20, Mana: 5, LvlMana: 1, ManaShift: 7, MinMana: 1, ShortKey: "s36", LongKey: "l36"},
		{ID: 47, Key: "Fire Ball", Name: "Fire Ball", Class: "sor", Page: 1, Row: 5, Column: 2, ReqLevel: 18, MaxLevel: 20, ReqSkills: [3]string{"Fire Bolt"}},
		{ID: 97, Key: "Sacrifice", Name: "Sacrifice", Class: "pal", Page: 1, Row: 1, Column: 1, ReqLevel: 1, MaxLevel: 20},
		{ID: 98, Key: "Smite", Name: "Smite", Class: "pal", Page: 1, Row: 2, Column: 2, ReqLevel: 6, MaxLevel: 20, ReqSkills: [3]string{"Sacrifice"}},
	}
	pts := map[string]map[int]skilltree.Points{"sor": {36: {Base: 3, Bonus: 1}}, "pal": {97: {Base: 2}}}
	tx := func(k string) string { return map[string]string{"s36": "fake short", "l36": "fake long"}[k] }
	return skilltree.Build(skilltree.Tables{Skills: sk, Text: tx},
		skilltree.Hero{ClassCode: class, ClassName: name, Level: lvl, Unspent: unspent, Points: pts[class]}, nil)
}

// The /api/skills body is a public shape (the page and later clients read it): golden per class, fake tables only.
func TestSkillsViewGolden(t *testing.T) {
	for _, c := range []struct{ file, class, name string }{{"skills_paladin.json", "pal", "Paladin"}, {"skills_sorceress.json", "sor", "Sorceress"}} {
		api := NewSkillsAPI()
		api.Bind(fakeSkills{skillsFixture(c.class, c.name, 20, 3)})
		w := httptest.NewRecorder()
		api.Handler().ServeHTTP(w, httptest.NewRequest("GET", "/api/skills", nil))
		if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("%s: %d %v", c.file, w.Code, w.Header())
		}
		p := filepath.Join("testdata", c.file)
		if os.Getenv("UPDATE") != "" {
			os.WriteFile(p, w.Body.Bytes(), 0o644)
		}
		want, err := os.ReadFile(p)
		if err != nil || string(want) != w.Body.String() {
			t.Errorf("%s differs (UPDATE=1 rewrites it):\n%s", c.file, w.Body)
		}
		if !strings.Contains(w.Body.String(), `"allocate":false`) || !strings.Contains(w.Body.String(), `"schema":"watch.skills/1"`) {
			t.Errorf("%s: schema or allocate flag", c.file)
		}
	}
}

func TestSkillsRoutesAndStubs(t *testing.T) {
	api := NewSkillsAPI()
	h := api.Handler()
	get := func(p string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", p, nil))
		return w
	}
	if w := get("/api/skills"); w.Code != 503 {
		t.Errorf("unbound: %d", w.Code)
	}
	for p, ct := range map[string]string{"/watch/skills": "text/html", "/watch/skills.js": "javascript", "/watch/skills.css": "text/css"} {
		if w := get(p); w.Code != 200 || !strings.Contains(w.Header().Get("Content-Type"), ct) || w.Body.Len() < 100 {
			t.Errorf("%s: %d %s", p, w.Code, w.Header().Get("Content-Type"))
		}
	}
	if w := get("/watch/skills"); strings.Contains(w.Body.String(), "style=") || strings.Contains(w.Body.String(), "<style") {
		t.Error("the page must not carry inline styles (CSP)")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("POST", "/api/skills", strings.NewReader("{}")))
	if w.Code != 405 {
		t.Errorf("viewer POST: %d", w.Code)
	}

	post := api.PostHandler(AnswerOptions{Token: "tok", Origins: []string{"http://ok.test"}})
	auth := map[string]string{"Authorization": "Bearer tok"}
	if w := invPost(post, "/api/skills/allocate", `{"skill":36,"points":1}`, nil); w.Code != 401 {
		t.Errorf("no token: %d", w.Code)
	}
	if w := invPost(post, "/api/skills/allocate", `{"skill":36,"points":1}`, auth); w.Code != http.StatusNotImplemented || !strings.Contains(w.Body.String(), `"off"`) {
		t.Errorf("flag off: %d %s", w.Code, w.Body)
	}
	api.AllocateEnabled = true
	if w := invPost(post, "/api/skills/allocate", `{"skill":36,"points":1}`, auth); w.Code != http.StatusNotImplemented || !strings.Contains(w.Body.String(), "not_implemented") {
		t.Errorf("flag on, no allocator: %d %s", w.Code, w.Body)
	}
	// with the flag on and nothing to run it the view still says allocate:false, so the page shows no control
	api.Bind(fakeSkills{skillsFixture("pal", "Paladin", 10, 1)})
	if w := get("/api/skills"); !strings.Contains(w.Body.String(), `"allocate":false`) {
		t.Errorf("allocate must stay false without an allocator: %s", w.Body)
	}
}
