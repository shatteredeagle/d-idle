package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"d2lod-bot/internal/watch"
)

// The player screen (-watch, off by default; docs/research/watch-screen-design.md). The viewer side mounts only GET
// routes of internal/watch; every write (answer, confirm, revoke, orders) is served by a SEPARATE listener
// (-watch-answer-addr) so the viewer's routes stay read-only. Until the real bot is connected through internal/ladder,
// the thoughts come from the simulated runner (-watch-class).

type watchOpts struct {
	on         bool
	answerAddr string
	origins    string
	class      string
	speed      float64
}

// startWatch builds the simulated runner, starts it and the answer listener, and returns the read-only handler.
func startWatch(ctx context.Context, o watchOpts, viewAddr, token string, open, hasFrames bool) (http.Handler, error) {
	sim, err := watch.NewSim(o.class, o.speed)
	if err != nil {
		return nil, err
	}
	if token == "" && !open && !loopback(o.answerAddr) {
		return nil, fmt.Errorf("d2dview: the answer route on %s needs a token unless it is loopback or -open", o.answerAddr)
	}
	ln, err := net.Listen("tcp", o.answerAddr)
	if err != nil {
		return nil, err
	}
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	if p, err := strconv.Atoi(port); err == nil {
		sim.Hub.SetAnswerPort(p)
	}
	if hasFrames {
		sim.Hub.SetFrameURL("/stream.mjpg")
	}
	ans := &http.Server{Handler: sim.Hub.AnswerHandler(watch.AnswerOptions{Token: token, Origins: watchOrigins(o.origins, viewAddr)})}
	go ans.Serve(ln)
	go func() { <-ctx.Done(); ans.Close() }()
	go sim.Run(ctx, o.speed, time.Second)
	log.Printf("d2dview: watch screen on /watch (simulated %s runner), answer route on %s", o.class, ln.Addr())
	return sim.Hub.ViewHandler(), nil
}

// watchOrigins lists the page origins the answer route accepts (internal/watch.Origins).
func watchOrigins(flagVal, viewAddr string) []string { return watch.Origins(flagVal, viewAddr) }

// watchOnlyRoutes is the viewer without a game: the watch screen and nothing else.
func (s *server) watchOnlyRoutes() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/watch", http.StatusFound)
	})
	s.mountWatch(m)
	return s.auth(m)
}

func (s *server) mountWatch(m *http.ServeMux) {
	if s.watch == nil {
		return
	}
	for _, p := range watch.ViewRoutes {
		m.Handle(p, s.watch)
	}
}
