package watch

import (
	"context"
	"crypto/subtle"
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
	Origins              string // comma list; empty: the viewer's port on localhost
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
		if o.Token == "" && !Loopback(a) {
			return Served{}, fmt.Errorf("watch: %s is not loopback and needs a token", a)
		}
	}
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
		m.Handle(p, h.ViewHandler())
	}
	view := &http.Server{Handler: tokenGate(o.Token, m)}
	ans := &http.Server{Handler: h.AnswerHandler(AnswerOptions{Token: o.Token, Origins: Origins(o.Origins, viewURL)})}
	go view.Serve(vl)
	go ans.Serve(al)
	go func() { <-ctx.Done(); view.Close(); ans.Close() }()
	return Served{View: vl.Addr(), Answer: al.Addr()}, nil
}

func hostOf(addr string) string { h, _, _ := net.SplitHostPort(addr); return h }

// tokenGate checks the bearer header, the ?t= query (which sets the cookie) or the d2dview cookie, as d2dview does.
func tokenGate(token string, next http.Handler) http.Handler {
	if token == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if q := r.URL.Query().Get("t"); q != "" {
			got = q
			http.SetCookie(w, &http.Cookie{Name: "d2dview", Value: q, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode})
		} else if c, err := r.Cookie("d2dview"); err == nil && got == "" {
			got = c.Value
		}
		if subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
