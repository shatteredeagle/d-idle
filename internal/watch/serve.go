package watch

import (
	"context"
	"crypto/subtle"
	"d2lod-bot/internal/buildinfo"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
)

// Loopback reports whether addr (host:port) binds only the local machine.
func Loopback(addr string) bool {
	h, _, err := net.SplitHostPort(addr)
	if err != nil || h == "" {
		return false
	}
	ip := net.ParseIP(h)
	return h == "localhost" || (ip != nil && ip.IsLoopback())
}

// Origins lists the page origins the answer route accepts: the comma list flagVal, else the viewer's own port on
// localhost and the loopback address.
func Origins(flagVal, viewAddr string) []string {
	var out []string
	for _, o := range strings.Split(flagVal, ",") {
		if o = strings.TrimSpace(o); o != "" {
			out = append(out, o)
		}
	}
	if len(out) > 0 {
		return out
	}
	_, port, err := net.SplitHostPort(viewAddr)
	if err != nil {
		return nil
	}
	return []string{"http://localhost:" + port, "http://127.0.0.1:" + port}
}

// ServeOptions are the two listeners of a bot that serves its own screen (d2bot -watch). Same rules as d2dview -watch:
// the viewer carries only GET routes, every write goes through the separate answer listener, and a token is required
// unless the listener is loopback.
type ServeOptions struct {
	ViewAddr, AnswerAddr string
	Token                string
	Origins              string    // comma list; empty: the viewer's port on localhost
	Video                *Video    // game video on the viewer (/stream, /shot.jpg); nil: off
	Access               *Access   // passcode choice (access.json); nil: token only. A passcode record replaces the token gate.
	Senses               *Senses   // side-panel minimap (/api/senses); nil: off
	Fun                  *Fun      // return screen, tally strip and Codex (funwire.go); nil: the plain live screen only
	PalFeed              *PalFeed  // lossless picture feed (palfeed.go); nil: MJPEG only
	Update               Updater   // self-update status, banner and answer routes (update.go); nil: no update UI
	Profiler             *Profiler // on-demand CPU profile routes on the answer listener (pprofapi.go); nil: off
}

// Served names the addresses that were bound.
type Served struct{ View, Answer net.Addr }

// Serve binds both listeners and serves until ctx ends. Nothing is bound when an option is refused.
func (h *Hub) Serve(ctx context.Context, o ServeOptions) (Served, error) {
	if o.ViewAddr == "" || o.AnswerAddr == "" {
		return Served{}, errors.New("watch: the viewer and the answer address are both required")
	}
	if _, port, _ := net.SplitHostPort(o.ViewAddr); o.ViewAddr == o.AnswerAddr && port != "0" {
		return Served{}, errors.New("watch: the answer route needs its own address, apart from the viewer")
	}
	for _, a := range []string{o.ViewAddr, o.AnswerAddr} {
		if o.Token == "" && o.Access == nil && !Loopback(a) {
			return Served{}, fmt.Errorf("watch: %s is not loopback and needs a token", a)
		}
	}
	h.SetVideo(o.Video)
	h.SetPalFeed(o.PalFeed)
	h.SetBuild(buildinfo.Current())
	vl, err := net.Listen("tcp", o.ViewAddr)
	if err != nil {
		return Served{}, err
	}
	al, err := net.Listen("tcp", o.AnswerAddr)
	if err != nil {
		vl.Close()
		return Served{}, err
	}
	if _, port, _ := net.SplitHostPort(al.Addr().String()); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			h.SetAnswerPort(p)
		}
	}
	_, vport, _ := net.SplitHostPort(vl.Addr().String())
	viewURL := net.JoinHostPort(hostOf(o.ViewAddr), vport)
	m := http.NewServeMux()
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/watch", http.StatusFound)
	})
	for _, p := range ViewRoutes {
		if o.Fun != nil && o.Fun.owns(p) {
			continue
		}
		m.Handle(p, h.ViewHandler())
	}
	if o.Fun != nil {
		o.Fun.mountView(m, h.ViewHandler())
	}
	if o.Senses != nil {
		for _, p := range SensesRoutes {
			m.Handle(p, o.Senses.Handler())
		}
	}
	if o.Video != nil {
		for _, p := range VideoRoutes {
			m.Handle(p, h.ViewHandler())
		}
	}
	if o.PalFeed != nil {
		for _, p := range PalFeedRoutes {
			m.Handle(p, h.ViewHandler())
		}
	}
	for _, p := range LogRoutes {
		m.Handle(p, h.LogHandler(m)) // reads the routes above, so it comes after them
	}
	var viewH http.Handler = m
	if o.Update != nil {
		uh := UpdateHandler(o.Update)
		for _, p := range UpdateRoutes {
			m.Handle(p, uh)
		}
		viewH = withUpdateBanner(m)
	}
	if o.Fun != nil && o.Fun.OnView != nil {
		inner, seen := viewH, o.Fun.OnView
		viewH = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { seen(); inner.ServeHTTP(w, r) })
	}
	view := &http.Server{Handler: o.Access.viewGate(o.Token, viewH)}
	ansOpts := AnswerOptions{Token: o.Token, Origins: Origins(o.Origins, viewURL)}
	if o.Access != nil {
		ansOpts.Token = "" // the Access gate judges the token or the passcode cookie before the handler runs
	}
	ansH := h.AnswerHandler(ansOpts)
	if o.Fun != nil {
		ansH = o.Fun.wrapAnswer(ansH, ansOpts)
	}
	if o.Update != nil {
		um := http.NewServeMux()
		um.Handle("/", ansH)
		uah := UpdateAnswerHandler(o.Update, ansOpts)
		for _, p := range UpdatePostRoutes {
			um.Handle(p, uah)
		}
		ansH = um
	}
	if o.Profiler != nil {
		pm := http.NewServeMux()
		pm.Handle("/", ansH)
		ph := o.Profiler.Handler(ansOpts)
		for _, p := range ProfilerRoutes {
			pm.Handle(p, ph)
		}
		ansH = pm
	}
	if o.Access != nil {
		ansH = o.Access.answerGate(o.Token, ansH)
	}
	ans := &http.Server{Handler: ansH}
	go view.Serve(vl)
	go ans.Serve(al)
	go func() { <-ctx.Done(); view.Close(); ans.Close() }()
	return Served{View: vl.Addr(), Answer: al.Addr()}, nil
}

func hostOf(addr string) string { h, _, _ := net.SplitHostPort(addr); return h }

// tokenGate checks the bearer header, the ?t= query or the d2dview cookie, as d2dview does. A valid ?t= sets the cookie;
// a page load (GET that accepts HTML) is then redirected to the same address without t, so the token does not stay in
// the address bar, the history or a Referer. A wrong ?t= never becomes a cookie. Scripts that send ?t= on an API call
// are served directly.
func tokenGate(token string, next http.Handler) http.Handler {
	if token == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		q := r.URL.Query()
		fromQuery := q.Get("t") != ""
		if fromQuery {
			got = q.Get("t")
		} else if c, err := r.Cookie("d2dview"); err == nil && got == "" {
			got = c.Value
		}
		h := w.Header()
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Cache-Control", "no-store") // the default; a handler that serves cacheable art sets its own
		if subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if fromQuery {
			http.SetCookie(w, &http.Cookie{Name: "d2dview", Value: got, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode})
			if r.Method == http.MethodGet && strings.Contains(r.Header.Get("Accept"), "text/html") {
				u := *r.URL
				q.Del("t")
				u.RawQuery = q.Encode()
				http.Redirect(w, r, u.RequestURI(), http.StatusSeeOther)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
