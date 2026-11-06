package watch

import (
	"html/template"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"d2lod-bot/internal/access"
	"d2lod-bot/web"
)

// Access wires the passcode choice into the two listeners: the record file (shared with d2dsetup) and the
// failed-login limiter. A nil Access keeps the token-only behaviour of tokenGate.
type Access struct {
	Store   *access.Store
	Limiter *access.Limiter
}

// SessionCookie is the passcode login cookie. HttpOnly and SameSite=Strict; Secure is not set because the screen is
// plain HTTP on a home network (a Secure cookie would never be sent), so the cookie can be read by anyone who can
// watch the traffic of that network: the guide says never to publish the port on the internet.
const SessionCookie = "d2dpass"

const sessionAge = 30 * 24 * time.Hour

// NewSessionCookie is the cookie that proves a passcode login for rec (also used by d2dsetup after the passcode was set).
func NewSessionCookie(rec access.Record) *http.Cookie {
	return &http.Cookie{Name: SessionCookie, Value: rec.Session(), Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: int(sessionAge / time.Second)}
}

// ClientKey is the rate-limit key: the peer address without the port. Proxy headers are not trusted.
func ClientKey(r *http.Request) string {
	h, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return h
}

// viewGate guards the viewer. With a passcode chosen it needs the session cookie (login form at /login); with the
// token kind, or no file yet, it is tokenGate. With neither a token nor a record on a listener that needs one it would
// let everything through, so Serve refuses that case up front.
func (a *Access) viewGate(token string, next http.Handler) http.Handler {
	tok := tokenGate(token, next)
	if a == nil {
		return tok
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := a.Store.Current()
		if rec.Kind != access.KindPasscode {
			if token == "" {
				http.Error(w, "unauthorized: access is not set up yet", http.StatusUnauthorized)
				return
			}
			tok.ServeHTTP(w, r)
			return
		}
		switch r.URL.Path {
		case "/login":
			a.login(w, r, rec)
			return
		case "/login.css":
			loginCSS(w, r)
			return
		case "/logout":
			http.SetCookie(w, &http.Cookie{Name: SessionCookie, Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteStrictMode})
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		if c, err := r.Cookie(SessionCookie); err == nil && rec.SessionOK(c.Value) {
			next.ServeHTTP(w, r)
			return
		}
		if r.Method == http.MethodGet && !strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/stream" && r.URL.Path != "/shot.jpg" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		http.Error(w, "unauthorized: sign in at /login", http.StatusUnauthorized)
	})
}

// answerGate guards the answer listener the same way; its replies are JSON. Preflights pass (browsers send them
// without cookies) and are judged by the origin check inside the handler.
func (a *Access) answerGate(token string, next http.Handler) http.Handler {
	if a == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := a.Store.Current()
		if r.Method != http.MethodOptions {
			var ok bool
			switch {
			case rec.Kind == access.KindPasscode:
				c, err := r.Cookie(SessionCookie)
				ok = err == nil && rec.SessionOK(c.Value)
			case token != "":
				ok = AnswerOptions{Token: token}.tokenOK(r)
			}
			if !ok {
				writeJSON(w, http.StatusUnauthorized, reply{Code: "token", Error: "unauthorized"})
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

var loginTmpl = template.Must(template.New("login").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="color-scheme" content="light dark"><title>d.idle sign in</title><link rel="icon" href="data:,">
<link rel="stylesheet" href="/login.css"></head>
<body class="wz-body"><header class="top"><div class="brand"><span class="brand-mark" aria-hidden="true"></span><h1>d.idle</h1></div></header>
<div class="wz-wrap"><main class="wz-main"><section class="wz-card"><h2>Enter your passcode</h2>
<p class="wz-hint">This is the passcode you chose during setup.</p>
{{if .Msg}}<div class="wz-note bad" role="alert"><strong>{{.Msg}}</strong>{{.Fix}}</div>{{end}}
<form method="post" action="/login" class="wz-form">
<div class="wz-field"><label for="pc">Passcode</label><input type="password" id="pc" name="passcode" autocomplete="current-password" autofocus required></div>
<div class="wz-actions"><button class="btn primary big" type="submit">Open the watch screen</button></div></form>
<p class="wz-hint">Forgot it? Run <code>docker compose exec d2dbot d2dsetup reset-access</code> on the computer that runs d.idle.</p>
</section></main></div></body></html>`))

type loginData struct{ Msg, Fix string }

func (a *Access) login(w http.ResponseWriter, r *http.Request, rec access.Record) {
	h := w.Header()
	h.Set("Cache-Control", "no-store")
	h.Set("Content-Security-Policy", "default-src 'none'; style-src 'self'; img-src data:; form-action 'self'; base-uri 'none'; frame-ancestors 'none'")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "no-referrer")
	render := func(code int, d loginData) {
		h.Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(code)
		loginTmpl.Execute(w, d)
	}
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		render(http.StatusOK, loginData{})
	case http.MethodPost:
		if o := r.Header.Get("Origin"); o != "" {
			if u, err := url.Parse(o); err != nil || u.Host != r.Host {
				http.Error(w, "forbidden origin", http.StatusForbidden)
				return
			}
		}
		key := ClientKey(r)
		if ok, wait := a.Limiter.Allowed(key); !ok {
			h.Set("Retry-After", strconv.Itoa(int(wait.Seconds())+1))
			render(http.StatusTooManyRequests, loginData{"Too many wrong tries.", "Wait " + waitText(wait) + " and try again."})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 4<<10)
		if r.ParseForm() != nil || !rec.Verify(r.PostForm.Get("passcode")) {
			a.Limiter.Fail(key)
			render(http.StatusUnauthorized, loginData{"That passcode did not match.", "Check for typing mistakes and try again."})
			return
		}
		a.Limiter.Success(key)
		http.SetCookie(w, NewSessionCookie(rec))
		http.Redirect(w, r, "/watch", http.StatusSeeOther)
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func waitText(d time.Duration) string {
	if m := int(d.Round(time.Minute) / time.Minute); m >= 1 {
		return strconv.Itoa(m) + " minute(s)"
	}
	return strconv.Itoa(int(d.Seconds())+1) + " seconds"
}

// loginCSS serves the shared stylesheet of the setup page, so the sign-in page looks the same and needs no inline style.
func loginCSS(w http.ResponseWriter, r *http.Request) {
	b, err := web.FS.ReadFile("style.css")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Write(b)
}
