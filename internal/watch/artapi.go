package watch

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"sync/atomic"

	"d2lod-bot/internal/art"
)

// Game art routes (docs/paperdoll-art-build.md). The art is decoded at request time from the user's own game
// archives by internal/art and never leaves the viewer origin through any embedded or stored file.
var ArtRoutes = []string{"/api/art", "/watch/art/"}

// ArtSchema is the schema of the GET /api/art reply.
const ArtSchema = "watch.art/1"

// ArtSource is what the routes need from internal/art (*art.Store implements it).
type ArtSource interface {
	PNG(key string) ([]byte, error)
}

// ArtAPI serves /api/art and /watch/art/<key>.png. The source is late-bound; without one every image is 404 and
// /api/art says available:false, so the page keeps its drawn cells.
type ArtAPI struct {
	src atomic.Pointer[ArtSource]
}

func NewArtAPI() *ArtAPI { return &ArtAPI{} }

// Bind sets the source; nil clears it.
func (a *ArtAPI) Bind(s ArtSource) {
	if s == nil {
		a.src.Store(nil)
		return
	}
	a.src.Store(&s)
}

func (a *ArtAPI) source() ArtSource {
	if p := a.src.Load(); p != nil {
		return *p
	}
	return nil
}

// ArtProgress is the optional second side of an ArtSource: the state of the pre-extraction of every icon
// (*art.Store implements it). Without it the reply carries no counts.
type ArtProgress interface {
	Progress() art.Progress
}

// ArtReply is the /api/art body. Panel names the paper-doll panel the art belongs to. State is unavailable (no art,
// drawn cells stay), extracting (icons are being prepared in the background; art already works, Done of Total are
// ready), ready or idle (no extraction was started). Fields added to watch.art/1: state, done, total, missing.
type ArtReply struct {
	Schema    string `json:"schema"`
	Available bool   `json:"available"`
	Panel     string `json:"panel"`
	State     string `json:"state"`
	Done      int    `json:"done,omitempty"`
	Total     int    `json:"total,omitempty"`
	Missing   int    `json:"missing,omitempty"`
}

// Handler serves both routes; mount it on every ArtRoutes entry.
func (a *ArtAPI) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("/api/art", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, a.reply())
	})
	m.HandleFunc("/watch/art/", a.image)
	return readOnly(m)
}

func (a *ArtAPI) reply() ArtReply {
	r := ArtReply{Schema: ArtSchema, Available: a.available(), Panel: "doll", State: "unavailable"}
	if !r.Available {
		return r
	}
	r.State = art.StateReady
	if p, ok := a.source().(ArtProgress); ok {
		pr := p.Progress()
		r.State, r.Done, r.Total, r.Missing = pr.State, pr.Done, pr.Total, pr.Missing
	}
	return r
}

// available is true when the source can serve the panel frame, the one image every page view needs.
func (a *ArtAPI) available() bool {
	s := a.source()
	if s == nil {
		return false
	}
	_, err := s.PNG("panel-inv")
	return err == nil
}

func (a *ArtAPI) image(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/watch/art/")
	key, ok := strings.CutSuffix(name, ".png")
	if !ok || !art.ValidKey(key) { // the key grammar is the only thing that reaches an archive lookup
		http.NotFound(w, r)
		return
	}
	s := a.source()
	if s == nil {
		http.NotFound(w, r)
		return
	}
	b, err := s.PNG(key)
	if err != nil { // missing art and decode trouble look the same to the page; the reason stays out of the reply
		http.NotFound(w, r)
		return
	}
	sum := sha256.Sum256(b)
	etag := `"` + hex.EncodeToString(sum[:8]) + `"`
	h := w.Header()
	h.Set("Content-Type", "image/png")
	h.Set("Cache-Control", "private, max-age=86400")
	h.Set("ETag", etag)
	h.Set("X-Content-Type-Options", "nosniff")
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Write(b)
}
