package watch

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"d2lod-bot/internal/skilltree"
)

// The Skills page (docs/skilltrees.md): GET /api/skills (the hero's class trees, one cached body), the page and its
// two assets. Read only. POST /api/skills/allocate is reserved: it answers 501 today (feature flag skills.allocate,
// off by default) so a later allocate control only needs an implementation behind the same route.

// skillsViewTTL is how long one built body is served again; several open pages share one build.
var skillsViewTTL = 2 * time.Second

var (
	SkillsRoutes     = []string{"/api/skills", "/watch/skills", "/watch/skills.js", "/watch/skills.css"}
	SkillsPostRoutes = []string{"/api/skills/allocate"}
)

// SkillsSource is what the routes need from the bot side (cmd/d2bot/skillswire.go).
type SkillsSource interface {
	View() skilltree.View
}

// SkillsAllocator is the future write side. Nothing implements it yet; with the flag on and no allocator the route
// still answers 501.
type SkillsAllocator interface {
	RequestAllocate(skill, points int) (any, error)
}

// SkillsAPI holds the source once it exists.
type SkillsAPI struct {
	mu    sync.Mutex
	src   SkillsSource
	alloc SkillsAllocator
	// AllocateEnabled is the skills.allocate flag, off by default.
	AllocateEnabled bool

	cmu    sync.Mutex // held while the view is built
	cached []byte
	at     time.Time
}

func NewSkillsAPI() *SkillsAPI { return &SkillsAPI{} }

func (a *SkillsAPI) Bind(s SkillsSource) {
	a.mu.Lock()
	a.src = s
	a.mu.Unlock()
	a.cmu.Lock()
	a.cached = nil
	a.cmu.Unlock()
}

func (a *SkillsAPI) source() SkillsSource { a.mu.Lock(); defer a.mu.Unlock(); return a.src }

func (a *SkillsAPI) body(s SkillsSource) ([]byte, error) {
	a.cmu.Lock()
	defer a.cmu.Unlock()
	if a.cached != nil && time.Since(a.at) < skillsViewTTL {
		return a.cached, nil
	}
	v := s.View()
	v.Allocate = a.AllocateEnabled && a.alloc != nil // the page shows a control only when the action can run
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	a.cached, a.at = append(b, '\n'), time.Now()
	return a.cached, nil
}

// Handler serves the viewer GET routes.
func (a *SkillsAPI) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("/api/skills", func(w http.ResponseWriter, r *http.Request) {
		s := a.source()
		if s == nil {
			writeJSON(w, http.StatusServiceUnavailable, policyReply{Code: "off", Error: "the skills page is off or the hero is not known yet"})
			return
		}
		b, err := a.body(s)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, policyReply{Code: "internal", Error: "could not build the skill trees"})
			return
		}
		h := w.Header()
		h.Set("Content-Type", "application/json; charset=utf-8")
		h.Set("Cache-Control", "no-store")
		h.Set("X-Content-Type-Options", "nosniff")
		w.Write(b)
	})
	m.HandleFunc("/watch/skills", func(w http.ResponseWriter, r *http.Request) {
		pactHdr(w, "text/html; charset=utf-8")
		io.WriteString(w, skillsPage)
	})
	static := func(file, ctype string) {
		m.HandleFunc("/watch/"+file, func(w http.ResponseWriter, r *http.Request) {
			b, err := webFS.ReadFile("web/" + file)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			pactHdr(w, ctype)
			w.Write(b)
		})
	}
	static("skills.js", "text/javascript; charset=utf-8")
	static("skills.css", "text/css; charset=utf-8")
	return readOnly(m)
}

type skillsBody struct {
	Skill  int `json:"skill"`
	Points int `json:"points"`
}

// PostHandler serves POST /api/skills/allocate with the shared gates of the answer routes. Today it answers 501.
func (a *SkillsAPI) PostHandler(o AnswerOptions) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b skillsBody
		if !policyGate(o, w, r, &b) {
			return
		}
		if !a.AllocateEnabled {
			writeJSON(w, http.StatusNotImplemented, policyReply{Code: "off", Error: "allocating skill points is not available: the skills.allocate flag is off and the page is read only"})
			return
		}
		a.mu.Lock()
		al := a.alloc
		a.mu.Unlock()
		if al == nil {
			writeJSON(w, http.StatusNotImplemented, policyReply{Code: "not_implemented", Error: "allocating skill points is not implemented yet (docs/skilltrees.md, Future: allocation)"})
			return
		}
		res, err := al.RequestAllocate(b.Skill, b.Points)
		if err != nil {
			writeJSON(w, http.StatusConflict, policyReply{Code: "refused", Error: "the request was refused"})
			return
		}
		writeJSON(w, http.StatusOK, res)
	})
}
