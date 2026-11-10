package watch

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"d2lod-bot/internal/art"
)

// fakeArt serves made-up bytes; keys it does not know behave like the real store.
type fakeArt struct {
	imgs  map[string][]byte
	calls int
}

func (f *fakeArt) PNG(key string) ([]byte, error) {
	f.calls++
	if !art.ValidKey(key) {
		return nil, art.ErrBadKey
	}
	if b, ok := f.imgs[key]; ok {
		return b, nil
	}
	return nil, art.ErrNotFound
}

func artGet(h http.Handler, method, path string, hdr map[string]string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "/", nil)
	if u, err := url.Parse(path); err == nil {
		r.URL = u
	} else {
		r.URL = &url.URL{Path: path} // hostile paths a client could still send raw
	}
	for k, v := range hdr {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestArtImageRoute(t *testing.T) {
	api := NewArtAPI()
	f := &fakeArt{imgs: map[string][]byte{"item-invgsw": []byte("\x89PNGfake"), "panel-inv": []byte("panel")}}
	api.Bind(f)
	h := api.Handler()

	w := artGet(h, "GET", "/watch/art/item-invgsw.png", nil)
	if w.Code != 200 || w.Body.String() != "\x89PNGfake" {
		t.Fatalf("image: %d %q", w.Code, w.Body.String())
	}
	for k, want := range map[string]string{"Content-Type": "image/png", "X-Content-Type-Options": "nosniff", "Cache-Control": "private, max-age=86400"} {
		if got := w.Header().Get(k); got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}
	etag := w.Header().Get("ETag")
	if etag == "" {
		t.Fatal("no ETag")
	}
	if w := artGet(h, "GET", "/watch/art/item-invgsw.png", map[string]string{"If-None-Match": etag}); w.Code != 304 || w.Body.Len() != 0 {
		t.Errorf("conditional GET: %d", w.Code)
	}
	if w := artGet(h, "HEAD", "/watch/art/item-invgsw.png", nil); w.Code != 200 {
		t.Errorf("HEAD: %d", w.Code)
	}
	if w := artGet(h, "POST", "/watch/art/item-invgsw.png", nil); w.Code != 405 {
		t.Errorf("POST: %d", w.Code)
	}
}

// Bad keys never reach the source (no archive lookup, no file access): 404 straight from the grammar.
func TestArtBadKeys404WithoutLookup(t *testing.T) {
	api := NewArtAPI()
	f := &fakeArt{imgs: map[string][]byte{"item-invgsw": []byte("x")}}
	api.Bind(f)
	h := api.Handler()
	for _, p := range []string{
		"/watch/art/", "/watch/art/item-invgsw", "/watch/art/item-invgsw.gif", "/watch/art/item-.png",
		"/watch/art/..%2f..%2fetc%2fpasswd.png", "/watch/art/../../etc/passwd.png", "/watch/art/item-..%2fx.png",
		"/watch/art/item-%2e%2e.png", "/watch/art/ITEM-invgsw.png", "/watch/art/item-INVGSW.png",
		"/watch/art/item-invgsw.png/x", "/watch/art/slot-nope.png", "/watch/art/item-aaaaaaaaaaaaaaaaa.png",
		"/watch/art/item-a%00b.png", `/watch/art/item-a\b.png`, "/watch/art/item-a b.png",
	} {
		if w := artGet(h, "GET", p, nil); w.Code != 404 && w.Code != 301 && w.Code != 307 { // the mux redirects a path it had to clean
			t.Errorf("%s: %d", p, w.Code)
		}
	}
	if f.calls != 0 {
		t.Errorf("source consulted %d times for bad keys", f.calls)
	}
	if w := artGet(h, "GET", "/watch/art/item-nosuch.png", nil); w.Code != 404 {
		t.Errorf("missing art: %d", w.Code)
	}
}

type failArt struct{}

func (failArt) PNG(string) ([]byte, error) { return nil, errors.New("open /secret/path: boom") }

func TestArtAvailable(t *testing.T) {
	api := NewArtAPI()
	h := api.Handler()
	get := func() ArtReply {
		w := artGet(h, "GET", "/api/art", nil)
		if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("/api/art: %d %v", w.Code, w.Header())
		}
		var v ArtReply
		if err := jsonDecode(w.Body.String(), &v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	if v := get(); v.Available || v.Schema != "watch.art/1" || v.Panel != "doll" {
		t.Errorf("no source: %+v", v)
	}
	if w := artGet(h, "GET", "/watch/art/item-invgsw.png", nil); w.Code != 404 {
		t.Errorf("image without source: %d", w.Code)
	}
	api.Bind(failArt{})
	if v := get(); v.Available {
		t.Errorf("failing source: %+v", v)
	}
	if w := artGet(h, "GET", "/watch/art/panel-inv.png", nil); w.Code != 404 || strings.Contains(w.Body.String(), "secret") {
		t.Errorf("failure leaks detail: %d %s", w.Code, w.Body)
	}
	api.Bind(&fakeArt{imgs: map[string][]byte{"panel-inv": []byte("p")}})
	if v := get(); !v.Available {
		t.Errorf("working source: %+v", v)
	}
	api.Bind(nil)
	if v := get(); v.Available {
		t.Errorf("unbound: %+v", v)
	}
}

// Fun mounts the routes only when Art is set, on the viewer mux (behind the viewer's gate like the inventory page).
func TestFunMountsArt(t *testing.T) {
	s, err := NewSim("sorceress", 1)
	if err != nil {
		t.Fatal(err)
	}
	f, err := NewFun(FunConfig{Hub: s.Hub})
	if err != nil {
		t.Fatal(err)
	}
	mount := func() *http.ServeMux {
		m := http.NewServeMux()
		f.mountView(m, s.Hub.ViewHandler())
		return m
	}
	if w := artGet(mount(), "GET", "/api/art", nil); w.Code != 404 {
		t.Errorf("Art nil: %d", w.Code)
	}
	f.Art = NewArtAPI()
	f.Art.Bind(&fakeArt{imgs: map[string][]byte{"panel-inv": []byte("p")}})
	m := mount()
	if w := artGet(m, "GET", "/api/art", nil); w.Code != 200 || !strings.Contains(w.Body.String(), `"available":true`) {
		t.Errorf("/api/art: %d %s", w.Code, w.Body)
	}
	if w := artGet(m, "GET", "/watch/art/panel-inv.png", nil); w.Code != 200 {
		t.Errorf("image: %d", w.Code)
	}
}

func jsonDecode(s string, v any) error { return json.NewDecoder(strings.NewReader(s)).Decode(v) }

type progArt struct {
	fakeArt
	p art.Progress
}

func (f *progArt) Progress() art.Progress { return f.p }

// /api/art carries the extraction state and counts; unavailable art says so; a source without progress is ready.
func TestArtStateAndCounts(t *testing.T) {
	api := NewArtAPI()
	h := api.Handler()
	get := func() ArtReply {
		var v ArtReply
		if err := jsonDecode(artGet(h, "GET", "/api/art", nil).Body.String(), &v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	if v := get(); v.State != "unavailable" {
		t.Errorf("unbound: %+v", v)
	}
	pa := &progArt{fakeArt: fakeArt{imgs: map[string][]byte{"panel-inv": []byte("p")}}, p: art.Progress{State: art.StateExtracting, Done: 120, Total: 1500}}
	api.Bind(pa)
	if v := get(); !v.Available || v.State != "extracting" || v.Done != 120 || v.Total != 1500 {
		t.Errorf("extracting: %+v", v)
	}
	pa.p = art.Progress{State: art.StateReady, Done: 1500, Total: 1500, Missing: 3}
	if v := get(); v.State != "ready" || v.Missing != 3 {
		t.Errorf("ready: %+v", v)
	}
	api.Bind(&fakeArt{imgs: map[string][]byte{"panel-inv": []byte("p")}})
	if v := get(); v.State != "ready" || v.Total != 0 {
		t.Errorf("no progress side: %+v", v)
	}
}
