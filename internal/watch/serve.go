package watch

import (
	"context"
	"crypto/subtle"
	"d2lod-bot/internal/buildinfo"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
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
	Origins              string          // comma list; empty: the viewer's port on localhost
	Video                *Video          // game video on the viewer (/stream, /shot.jpg); nil: off
	Access               *Access         // passcode choice (access.json); nil: token only. A passcode record replaces the token gate.
	Senses               *Senses         // side-panel minimap (/api/senses); nil: off
	Fun                  *Fun            // return screen, tally strip and Codex (funwire.go); nil: the plain live screen only
	PalFeed              *PalFeed        // lossless picture feed (palfeed.go); nil: MJPEG only
	Update               Updater         // self-update status, banner and answer routes (update.go); nil: no update UI
	Notices              func() []string // standing operator warnings shown as a banner (notices.go); nil: none
	Profiler             *Profiler       // on-demand CPU profile routes on the answer listener (pprofapi.go); nil: off
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
	if o.Notices != nil {
		nh := NoticesHandler(o.Notices)
		for _, p := range NoticeRoutes {
			m.Handle(p, nh)
		}
		viewH = withScriptTag(viewH, `<script src="/watch/notices.js"></script>`)
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

// tokenCookieAge is how long the d2dview cookie lives: as long as a passcode session (internal/access). Without a
// lifetime it was a session cookie, and a bookmark of the clean address stopped working when the browser dropped it.
const tokenCookieAge = 30 * 24 * time.Hour

// tokenHint is the 401 body of a page load: the clean address carries no token, so a bookmark or home-screen icon
// whose cookie is gone must say what to do instead of a bare "unauthorized". It names no token and no address, and
// has no inline style (the page CSP blocks them; tools/uilint).
const tokenHint = `<!doctype html><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Sign-in needed</title><body>
<h1>This page needs its token again</h1>
<p>The browser no longer has the sign-in cookie for this page (it lasts 30 days, and a home-screen icon or a private
window keeps its own cookies).</p>
<p>Open the link you were given once more: the address of this page with <code>?t=YOUR-TOKEN</code> at the end.
The page then removes the token from the address and remembers this browser for 30 days.</p>
<p>A wrong or old token gives this page too.</p></body>
`

// tokenGate checks the bearer header, the ?t= (or ?token=) query or the d2dview cookie, as d2dview does. A valid query
// token sets the cookie for tokenCookieAge; a page load (GET that accepts HTML) is then redirected to the same address
// without it, so the token does not stay in the address bar, the history or a Referer. A wrong query token never becomes
// a cookie. Scripts that send it on an API call are served directly, and the handlers behind the gate never see it in
// the query (some copy the query into a redirect or a log line). A page load without credentials gets tokenHint.
func tokenGate(token string, next http.Handler) http.Handler {
	if token == "" {
		return next
	}
	setCookie := func(w http.ResponseWriter) {
		http.SetCookie(w, &http.Cookie{Name: "d2dview", Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode,
			MaxAge: int(tokenCookieAge / time.Second)})
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		q := r.URL.Query()
		qt := q.Get("t")
		if qt == "" {
			qt = q.Get("token")
		}
		fromQuery, fromCookie := qt != "", false
		if fromQuery {
			got = qt
		} else if c, err := r.Cookie("d2dview"); err == nil && got == "" {
			got, fromCookie = c.Value, true
		}
		page := r.Method == http.MethodGet && strings.Contains(r.Header.Get("Accept"), "text/html")
		h := w.Header()
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Cache-Control", "no-store") // the default; a handler that serves cacheable art sets its own
		if subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
			if page {
				h.Set("Content-Type", "text/html; charset=utf-8")
				h.Set("X-Content-Type-Options", "nosniff")
				w.WriteHeader(http.StatusUnauthorized)
				io.WriteString(w, tokenHint)
				return
			}
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if fromQuery {
			setCookie(w)
			q.Del("t")
			q.Del("token")
			u := *r.URL
			u.RawQuery = q.Encode()
			if page {
				// One leading slash only: "//host/x" in a Location is another site.
				loc := "/" + strings.TrimLeft(u.EscapedPath(), "/")
				if u.RawQuery != "" {
					loc += "?" + u.RawQuery
				}
				http.Redirect(w, r, loc, http.StatusSeeOther)
				return
			}
			r = r.Clone(r.Context())
			r.URL = &u
			r.RequestURI = u.RequestURI()
		} else if fromCookie && page {
			setCookie(w) // a page that is used keeps its 30 days
		}
		next.ServeHTTP(w, r)
	})
}
