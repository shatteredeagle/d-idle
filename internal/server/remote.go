package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"d2lod-bot/internal/goals"
)

// Remote is a bot that runs as its own process with its own API on a loopback port (a fleet slot's d2bot). The
// server shows it as an instance: it lists it, proxies /api/v1/instances/{id}/... to the bot's CURRENT address
// (Config.Remotes is asked per request, so a restarted bot on a new port is followed at once) and builds the
// fleet view from the bot's /api/v1/status. Mounts (viewport) are served by the server itself.
type Remote struct {
	ID    string
	Title string
	State string // slot state: starting, running, restarting, draining, failed
	Addr  string // bot API host:port; empty while the bot is not up
	Error string // why the bot is not up, shown in the 503
	// Mounts are extra handlers under /api/v1/instances/{id}/<name>/, like Instance.Mounts.
	Mounts map[string]http.Handler
}

const (
	remoteStatusTTL     = time.Second
	remoteStatusTimeout = 2 * time.Second
)

var remoteTransport = &http.Transport{
	DialContext:           (&net.Dialer{Timeout: 2 * time.Second}).DialContext,
	ResponseHeaderTimeout: 15 * time.Second,
	MaxIdleConnsPerHost:   2,
	IdleConnTimeout:       30 * time.Second,
}

type remoteCache struct {
	mu sync.Mutex
	m  map[string]remoteStat // key: id + "@" + addr
}

type remoteStat struct {
	at  time.Time
	sum summary
	ok  bool
}

func (s *Server) remotes() []Remote {
	if s.cfg.Remotes == nil {
		return nil
	}
	out := s.cfg.Remotes()
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *Server) remote(id string) (Remote, bool) {
	for _, r := range s.remotes() {
		if r.ID == id {
			return r, true
		}
	}
	return Remote{}, false
}

// botPath maps an instance-relative path to the bot's own flat API.
func botPath(rest string) string {
	switch rest {
	case "health", "deaths", "state":
		return apiBase + "/bot/" + rest
	}
	return apiBase + "/" + rest
}

func notUp(w http.ResponseWriter, r Remote, why string) {
	msg := fmt.Sprintf("the bot of slot %s is not reachable yet (slot %s)", r.ID, r.State)
	if r.Error != "" {
		msg += ": " + r.Error
	} else if why != "" {
		msg += ": " + why
	}
	w.Header().Set("Retry-After", "3")
	writeJSON(w, http.StatusServiceUnavailable, errBody{msg, "bot_starting"})
}

// remoteRoute serves /api/v1/instances/{id}/{rest} of a remote.
func (s *Server) remoteRoute(w http.ResponseWriter, req *http.Request, rm Remote, rest string) {
	name, _, _ := strings.Cut(rest, "/")
	if h := rm.Mounts[name]; h != nil && name != "" {
		http.StripPrefix(instBase+"/"+rm.ID+"/"+name, h).ServeHTTP(w, req)
		return
	}
	if rest == "" {
		if getOnly(w, req) {
			sm, _ := s.remoteSummary(req.Context(), rm)
			writeJSON(w, 200, sm)
		}
		return
	}
	if rm.Addr == "" {
		notUp(w, rm, "")
		return
	}
	target := &url.URL{Scheme: "http", Host: rm.Addr}
	p := &httputil.ReverseProxy{
		Transport:     remoteTransport,
		FlushInterval: -1, // SSE
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)
			pr.Out.URL.Path, pr.Out.URL.RawPath = botPath(rest), ""
			pr.Out.Header.Del("Authorization") // the bot is loopback and has no token: do not hand ours on
			pr.Out.Header.Del("Cookie")
			pr.Out.Header.Del("X-Token")
			// The fleet server already checked Origin against its own host; the bot checks it against its loopback address.
			if pr.Out.Header.Get("Origin") != "" {
				pr.Out.Header.Set("Origin", "http://"+rm.Addr)
			}
			q := pr.In.URL.Query()
			q.Del("token")
			pr.Out.URL.RawQuery = q.Encode()
		},
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, err error) {
			notUp(w, rm, "bot API at "+rm.Addr+" did not answer: "+err.Error())
		},
	}
	p.ServeHTTP(w, req)
}

// remoteSummary builds the instance summary from the bot's /api/v1/status. ok is false when the bot did not
// answer: the summary then only carries the slot state (like a slot without an instance).
func (s *Server) remoteSummary(ctx context.Context, rm Remote) (summary, bool) {
	fallback := func() (summary, bool) {
		bad := rm.State == "failed"
		return summary{ID: rm.ID, Title: rm.Title, Slot: rm.State, OK: !bad, Ready: false, Now: time.Now(), Mounts: mountNames(rm.Mounts),
			Runner: goals.Status{State: goals.RunnerState(rm.State), Activity: rm.State, Message: rm.Error}}, false
	}
	if rm.Addr == "" {
		return fallback()
	}
	key := rm.ID + "@" + rm.Addr
	s.rc.mu.Lock()
	if s.rc.m == nil {
		s.rc.m = map[string]remoteStat{}
	}
	c, hit := s.rc.m[key]
	s.rc.mu.Unlock()
	if hit && time.Since(c.at) < remoteStatusTTL {
		return c.sum, c.ok
	}
	cctx, cancel := context.WithTimeout(ctx, remoteStatusTimeout)
	defer cancel()
	rq, _ := http.NewRequestWithContext(cctx, http.MethodGet, "http://"+rm.Addr+apiBase+"/status", nil)
	var sm summary
	ok := false
	if res, err := remoteTransport.RoundTrip(rq); err == nil {
		var sn goals.Snapshot
		if res.StatusCode == 200 && json.NewDecoder(res.Body).Decode(&sn) == nil {
			sm = summary{ID: rm.ID, Title: rm.Title, Mode: sn.Mode, Runner: sn.Runner, ActiveID: sn.ActiveGoalID, XPPerHour: sn.XPPerHour,
				Totals: sn.Totals, OK: sn.Runner.State != goals.RunnerError, Ready: true, LastSeq: sn.LastSeq, Now: sn.Now,
				Mounts: mountNames(rm.Mounts)}
			ok = true
		}
		res.Body.Close()
	}
	if !ok {
		sm, _ = fallback()
		sm.Slot = rm.State
	}
	s.rc.mu.Lock()
	s.rc.m[key] = remoteStat{at: time.Now(), sum: sm, ok: ok}
	s.rc.mu.Unlock()
	return sm, ok
}

func mountNames(m map[string]http.Handler) []string {
	var out []string
	for n := range m {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// remoteSummaries asks every remote concurrently.
func (s *Server) remoteSummaries(ctx context.Context) []summary {
	rs := s.remotes()
	out := make([]summary, len(rs))
	var wg sync.WaitGroup
	for i, r := range rs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			out[i], _ = s.remoteSummary(ctx, r)
		}()
	}
	wg.Wait()
	return out
}
