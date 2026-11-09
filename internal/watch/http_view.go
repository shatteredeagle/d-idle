package watch

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"time"
)

//go:embed web
var webFS embed.FS

// Routes of the read-only side (mounted on d2dview). Every one is GET (HEAD) only. The game video routes (VideoRoutes)
// are not in this list: d2dview has its own /stream, so only the bot's own listener (Serve) mounts them.
var ViewRoutes = []string{"/watch", "/watch/", "/api/thoughts", "/api/journal", "/api/rules", "/api/uptime", "/api/orders", "/api/orders/map",
	"/api/why", WhylineRoute, "/api/config", "/api/events", "/api/currency"}

// VideoRoutes are served by ViewHandler only when a Video is set (Serve mounts them for d2bot -watch-video).
var VideoRoutes = []string{"/stream", "/shot.jpg"}

// pageCSP: scripts and styles only from this origin, no inline code; the page talks to its own origin and, for the
// answer route, to another port of the same host, so connect-src allows http(s) generally.
const pageCSP = "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self' http: https:; base-uri 'none'; form-action 'none'; frame-ancestors 'none'"

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v) // encoding/json escapes <, > and & as < etc.
}

func queryOf(r *http.Request) string {
	if r.URL.RawQuery == "" {
		return ""
	}
	return "?" + r.URL.RawQuery
}

// ViewHandler serves the screen and its GET endpoints. It refuses every method but GET and HEAD, and it never writes
// to the store: faded thoughts are recorded by the bot side (Tick), not by a read.
func (h *Hub) ViewHandler() http.Handler {
	m := http.NewServeMux()
	sub, _ := fs.Sub(webFS, "web")
	asset := func(name, ctype string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			b, err := fs.ReadFile(sub, name)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", ctype)
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Content-Security-Policy", pageCSP)
			w.Header().Set("Cache-Control", "no-store")
			w.Write(b)
		}
	}
	page := asset("watch.html", "text/html; charset=utf-8")
	m.HandleFunc("/watch", page)
	m.HandleFunc("/watch/", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/watch/":
			page(w, r)
		case "/watch/video", "/watch/game":
			// The video used to be the "Game" tab; it now sits at the top of /watch. Old links land there, token kept.
			http.Redirect(w, r, "/watch"+queryOf(r), http.StatusFound)
		case "/watch/app.js":
			asset("app.js", "text/javascript; charset=utf-8")(w, r)
		case "/watch/app.css", "/watch/tokens.css", "/watch/reset.css", "/watch/shell.css", "/watch/components.css", "/watch/utilities.css":
			asset(strings.TrimPrefix(r.URL.Path, "/watch/"), "text/css; charset=utf-8")(w, r)
		case "/watch/shell.js":
			asset("shell.js", "text/javascript; charset=utf-8")(w, r)
		default:
			if !servePanels(w, r, asset) {
				http.NotFound(w, r)
			}
		}
	})
	m.Handle(WhylineRoute, h.why.Handler())
	m.HandleFunc("/api/thoughts", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, h.Thoughts()) })
	m.HandleFunc("/api/journal", func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit <= 0 || limit > 1000 {
			limit = 200
		}
		writeJSON(w, 200, h.Journal(limit, r.URL.Query().Get("kind")))
	})
	m.HandleFunc("/api/rules", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, h.Rules()) })
	m.HandleFunc("/api/uptime", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, h.Uptime()) })
	m.HandleFunc("/api/orders", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, h.OrdersView()) })
	m.HandleFunc("/api/orders/map", func(w http.ResponseWriter, r *http.Request) {
		act, _ := strconv.Atoi(r.URL.Query().Get("act"))
		writeJSON(w, 200, h.OrdersMap(act))
	})
	m.HandleFunc("/api/currency", func(w http.ResponseWriter, r *http.Request) {
		v := h.Currency()
		if v == nil {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, 200, v)
	})
	m.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, h.Config()) })
	m.HandleFunc("/api/why", func(w http.ResponseWriter, r *http.Request) {
		var v WhyView
		var err error
		if k := r.URL.Query().Get("key"); k != "" {
			v, err = h.Why(k)
		} else {
			v, err = h.WhyLine(r.URL.Query().Get("line"))
		}
		if err != nil {
			writeJSON(w, 404, map[string]any{"schema": whySchema, "error": err.Error()})
			return
		}
		writeJSON(w, 200, v)
	})
	m.HandleFunc("/api/events", h.events)
	video := func(serve func(*Video, http.ResponseWriter, *http.Request)) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if h.video == nil {
				http.NotFound(w, r)
				return
			}
			serve(h.video, w, r)
		}
	}
	m.HandleFunc("/stream", video((*Video).handleStream))
	m.HandleFunc("/shot.jpg", video((*Video).handleShot))
	return readOnly(m)
}

// readOnly refuses everything but GET and HEAD before any handler runs.
func readOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "read-only: the watch screen answers on the answer route, not here", http.StatusMethodNotAllowed)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// SSEKeepAlive is the comment-line interval that keeps proxies from closing an idle stream.
var SSEKeepAlive = 15 * time.Second

// events is the Server-Sent Events stream: one "change" event per Hub change, carrying only the sequence number. The page
// then re-reads the views it shows. A page that cannot use EventSource polls /api/uptime and compares "seq".
func (h *Hub) events(w http.ResponseWriter, r *http.Request) {
	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "no streaming", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	seq, ch := h.Changed()
	fmt.Fprintf(w, "retry: 2000\nevent: hello\ndata: {\"schema\":\"watch.event/1\",\"seq\":%d}\n\n", seq)
	fl.Flush()
	ka := time.NewTicker(SSEKeepAlive)
	defer ka.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ka.C:
			fmt.Fprint(w, ": keepalive\n\n")
		case <-ch:
			seq, ch = h.Changed()
			fmt.Fprintf(w, "event: change\nid: %d\ndata: {\"schema\":\"watch.event/1\",\"seq\":%d}\n\n", seq, seq)
		}
		fl.Flush()
	}
}

// RunTicker fades thoughts on the Hub's clock every period until ctx ends. It belongs to the bot side (it writes).
func (h *Hub) RunTicker(ctx context.Context, period time.Duration) {
	t := time.NewTicker(period)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			h.Tick()
		}
	}
}
