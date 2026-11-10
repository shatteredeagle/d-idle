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

// ArtReply is the /api/art body. Panel names the paper-doll panel the art belongs to.
type ArtReply struct {
	Schema    string `json:"schema"`
	Available bool   `json:"available"`
	Panel     string `json:"panel"`
}

// Handler serves both routes; mount it on every ArtRoutes entry.
func (a *ArtAPI) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("/api/art", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, ArtReply{Schema: ArtSchema, Available: a.available(), Panel: "doll"})
	})
	m.HandleFunc("/watch/art/", a.image)
	return readOnly(m)
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
