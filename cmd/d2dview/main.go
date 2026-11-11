// Command d2dview is a read-only viewport for a running (usually lockstep, sped-up) d2d: a top-down map of the level,
// the player's trail, monsters, life and mana, virtual time and the measured speed factor, plus sampled real game
// frames. Docs: docs/d2d-view.md.
//
// d2d's control socket takes one client, so the bot and the viewer cannot both dial it. d2dview therefore sits in
// between (see Mux): start d2d on UPSTREAM, point the bot at -bot-sock (--backend d2d --d2d-sock BOT --d2d-attach),
// and the viewer reads through its own filtered socket. Without -bot-sock it simply attaches to the d2d.
package main

import (
	"context"
	"crypto/subtle"
	"d2lod-bot/internal/watch"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

//go:embed web
var webFS embed.FS

func main() {
	addr := flag.String("addr", envOr("D2DVIEW_ADDR", "127.0.0.1:8893"), "listen address (env D2DVIEW_ADDR; default loopback 8893, 8891/8892 belong to the watch screen of d2bot --watch); a token is required unless it is loopback or -open")
	upstream := flag.String("upstream", os.Getenv("D2D_SOCK"), "the d2d control socket (default $D2D_SOCK)")
	botSock := flag.String("bot-sock", "", "socket to create for the bot (it dials this instead of the d2d); empty: no bot, the viewer attaches alone")
	tokenFile := flag.String("token-file", "", "file with the access token (or env D2DVIEW_TOKEN); default ~/.config/d2dview/token")
	open := flag.Bool("open", false, "serve without a token on a non-loopback address (LAN only)")
	hz := flag.Float64("hz", 15, "state polls per wall second while someone watches")
	fps := flag.Float64("fps", 6, "sampled game frames per wall second (the real rate backs off, see docs/d2d-view.md); 0: no frames")
	shots := flag.String("shots", fmt.Sprintf("/tmp/d2dview-%d", os.Getuid()), "private dir on the d2d host where screenshots are written")
	botAPI := flag.String("bot-api", "", "bot API base URL (read-only GET /api/v1/status), e.g. http://localhost:8080")
	watchSpeed := flag.Float64("watch-speed", 8, "watch mode: max virtual seconds per wall second (0: render every step, no limit)")
	jpegOut := flag.Bool("jpeg", false, "re-encode screenshot frames as JPEG (less bandwidth, CPU here); default passes d2d's PNG through (the -feed ring always sends JPEG)")
	jpegQ := flag.Int("jpeg-quality", 72, "JPEG quality 10-95 of re-encoded frames")
	half := flag.Bool("frame-half", false, "halve frame width and height before the JPEG encode (a quarter of the bytes and encode CPU; implies JPEG)")
	backoff := flag.Float64("frame-backoff", 4, "the next frame waits at least this many times the last frame's cost (screenshot: the verb's wall time inside d2d; -feed: our read and encode); 0 off")
	feedPath := flag.String("feed", "", "frame ring file for the d2d fork's framefeed verb (docs/viewer-fps.md), e.g. /dev/shm/d2dview/frames.ring (keep it on tmpfs); empty: screenshots only; a d2d without the verb falls back to screenshots")
	badgeFlag := flag.String("badge", envOr("D2DVIEW_BADGE", Backend), "header badge text of the stream page and the /api/pub backend field (env D2DVIEW_BADGE)")
	var wo watchOpts
	flag.BoolVar(&wo.on, "watch", false, "serve the player screen on /watch (read-only GET routes here; answers on -watch-answer-addr); without -upstream it runs on the simulated runner alone")
	flag.StringVar(&wo.answerAddr, "watch-answer-addr", "127.0.0.1:8892", "watch: listen address of the SEPARATE answer route (answer, confirm, revoke, orders)")
	flag.StringVar(&wo.origins, "watch-origin", "", "watch: comma list of page origins the answer route accepts (default: this viewer's port on localhost)")
	flag.StringVar(&wo.class, "watch-class", "sorceress", "watch: simulated build, sorceress or paladin")
	flag.Float64Var(&wo.speed, "watch-sim-speed", 1, "watch: simulated runner, virtual seconds per wall second")
	flag.Parse()
	badge = safeText(*badgeFlag, 40)
	if badge == "" {
		badge = Backend
	}
	if *upstream == "" && !wo.on {
		log.Fatal("d2dview: need -upstream or $D2D_SOCK")
	}
	token := os.Getenv("D2DVIEW_TOKEN")
	if token == "" {
		p := *tokenFile
		if p == "" {
			h, _ := os.UserHomeDir()
			p = filepath.Join(h, ".config", "d2dview", "token")
		}
		if b, err := os.ReadFile(p); err == nil {
			token = strings.TrimSpace(string(b))
		}
	}
	if token == "" && !*open && !loopback(*addr) {
		log.Fatal("d2dview: a token is required unless -addr is loopback or -open (D2DVIEW_TOKEN or -token-file)")
	}

	if *upstream == "" { // -watch alone: the screen on the simulated runner, no game
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()
		s := &server{token: token}
		wh, err := startWatch(ctx, wo, *addr, token, *open, false)
		if err != nil {
			log.Fatal(err)
		}
		s.watch = wh
		srv := &http.Server{Addr: *addr, Handler: s.watchOnlyRoutes()}
		go func() { <-ctx.Done(); srv.Close() }()
		log.Printf("d2dview: watch screen on %s (no game attached, token %v)", *addr, token != "")
		if err := srv.ListenAndServe(); err != nil && ctx.Err() == nil {
			log.Fatal(err)
		}
		return
	}
	mux := NewMux(*upstream)
	if *feedPath != "" {
		if !filepath.IsAbs(*feedPath) || strings.Contains(*feedPath, "..") || strings.ContainsAny(*feedPath, " \t") {
			log.Fatal("d2dview: -feed needs an absolute path without spaces or ..")
		}
		if err := os.MkdirAll(filepath.Dir(*feedPath), 0o700); err != nil {
			log.Fatal(err)
		}
		mux.FeedPath = *feedPath
	}
	viewSock := *upstream + ".view"
	if *botSock != "" {
		ln, err := mux.Serve(*botSock, true, "")
		if err != nil {
			log.Fatal(err)
		}
		defer ln.Close()
		log.Printf("d2dview: bot socket %s -> %s", *botSock, *upstream)
	}
	vln, err := mux.Serve(viewSock, false, *shots)
	if err != nil {
		log.Fatal(err)
	}
	defer vln.Close()
	defer os.Remove(viewSock)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	feed := NewFeed(viewSock, mux, *hz, *botAPI)
	go feed.Run(ctx)
	var frames *Frames
	if *fps > 0 {
		frames, err = NewFrames(viewSock, mux, *shots, *fps)
		if err != nil {
			log.Fatal(err)
		}
		frames.JPEG, frames.Half, frames.Backoff, frames.Feed = *jpegOut, *half, max(*backoff, 0), *feedPath
		frames.Quality = min(max(*jpegQ, 10), 95)
		feed.fr = frames
		go frames.Run(ctx)
	}
	s := &server{feed: feed, frames: frames, mux: mux, token: token, watchSpeed: *watchSpeed}
	if wo.on {
		if s.watch, err = startWatch(ctx, wo, *addr, token, *open, frames != nil); err != nil {
			log.Fatal(err)
		}
	}
	srv := &http.Server{Addr: *addr, Handler: s.routes()}
	go func() { <-ctx.Done(); srv.Close() }()
	log.Printf("d2dview: listening on %s (token %v)", *addr, token != "")
	if err := srv.ListenAndServe(); err != nil && ctx.Err() == nil {
		log.Fatal(err)
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func loopback(addr string) bool { return watch.Loopback(addr) }

type server struct {
	feed       *Feed
	frames     *Frames
	mux        *Mux
	token      string
	watchSpeed float64
	watch      http.Handler // the read-only player screen (internal/watch), nil unless -watch
}

func (s *server) routes() http.Handler {
	m := http.NewServeMux()
	sub, _ := fs.Sub(webFS, "web")
	m.Handle("/", http.FileServer(http.FS(sub)))
	m.HandleFunc("/stream", s.streamPage)
	m.HandleFunc("/live", s.streamPage)
	m.HandleFunc("/api/pub", s.pub)
	m.HandleFunc("/api/state", s.state)
	m.HandleFunc("/events", s.events)
	m.HandleFunc("/map.png", s.mapPNG)
	m.HandleFunc("/frame", s.frame)
	m.HandleFunc("/stream.mjpg", s.stream)
	m.HandleFunc("/api/frames", s.frameStats)
	m.HandleFunc("/api/mode", s.setMode)
	s.mountWatch(m)
	return s.auth(m)
}

func (s *server) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.token != "" {
			got := r.Header.Get("Authorization")
			got = strings.TrimPrefix(got, "Bearer ")
			if q := r.URL.Query().Get("t"); q != "" {
				got = q
				http.SetCookie(w, &http.Cookie{Name: "d2dview", Value: q, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode})
			} else if c, err := r.Cookie("d2dview"); err == nil && got == "" {
				got = c.Value
			}
			if subtle.ConstantTimeCompare([]byte(got), []byte(s.token)) != 1 {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *server) state(w http.ResponseWriter, r *http.Request) {
	s.feed.Touch()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(s.feed.Snap())
}

func (s *server) events(w http.ResponseWriter, r *http.Request) {
	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "no streaming", 500)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	var seq int64
	for r.Context().Err() == nil {
		s.feed.Touch()
		sn := s.feed.Wait(r.Context(), seq)
		if sn.Seq == seq {
			return
		}
		seq = sn.Seq
		b, _ := json.Marshal(sn)
		fmt.Fprintf(w, "data: %s\n\n", b)
		fl.Flush()
	}
}

// frameStats is GET /api/frames: the last completed 10 s window of the game video (FrameStats), 204 before the first.
func (s *server) frameStats(w http.ResponseWriter, r *http.Request) {
	if s.frames == nil {
		http.NotFound(w, r)
		return
	}
	st, ok := s.frames.Stats.lastWindow()
	if !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(st)
}

func (s *server) mapPNG(w http.ResponseWriter, r *http.Request) {
	b, v, mr := s.feed.Map()
	if b == nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Map-Version", strconv.FormatUint(v, 10))
	w.Header().Set("X-Map-Rect", fmt.Sprintf("%d,%d,%d,%d", mr[0], mr[1], mr[2], mr[3])) // x0,y0,w,h act-absolute subtiles: the page pairs it with this image
	w.Write(b)
}

func (s *server) frame(w http.ResponseWriter, r *http.Request) {
	if s.frames == nil {
		http.NotFound(w, r)
		return
	}
	s.frames.Touch()
	f := s.frames.Current()
	if f.Data == nil {
		http.Error(w, "no frame yet", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", f.Type)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Frame-Ticks", strconv.FormatInt(f.Ticks, 10))
	w.Write(f.Data)
}

// stream is multipart/x-mixed-replace (an <img> shows it, d2watch can embed it); every part carries the tick gap.
func (s *server) stream(w http.ResponseWriter, r *http.Request) {
	fl, ok := w.(http.Flusher)
	if !ok || s.frames == nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=f")
	w.Header().Set("Cache-Control", "no-store")
	fl.Flush() // headers now: a client sees the stream open before the first frame exists
	rc := http.NewResponseController(w)
	var seq int64
	for r.Context().Err() == nil {
		s.frames.Touch()
		f, newer := s.frames.Next()
		if f.Seq != seq && f.Data != nil {
			// Sent the moment it is published (no poll delay); a slow client blocks only its own handler and then
			// skips to the newest frame (latest wins). A client stuck for 10 s is dropped.
			seq = f.Seq
			rc.SetWriteDeadline(time.Now().Add(10 * time.Second))
			fmt.Fprintf(w, "--f\r\nContent-Type: %s\r\nContent-Length: %d\r\nX-Frame-Ticks: %d\r\n", f.Type, len(f.Data), f.Ticks)
			if f.GameMS >= 0 {
				fmt.Fprintf(w, "X-Frame-Game-Ms: %d\r\n", f.GameMS)
			}
			fmt.Fprint(w, "\r\n")
			w.Write(f.Data)
			fmt.Fprint(w, "\r\n")
			fl.Flush()
			s.frames.Stats.parts.Add(1)
		}
		select {
		case <-newer:
		case <-r.Context().Done():
			return
		case <-time.After(time.Second): // keeps the sampler awake (Touch) while no frame comes
		}
	}
}

// setMode switches fast/watch: POST /api/mode?m=fast|watch[&speed=8]. It is the only write; it only changes how the
// tap treats the bot's own step lines (see Mux).
func (s *server) setMode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	md := Mode{}
	switch r.URL.Query().Get("m") {
	case "fast":
	case "watch":
		md.Watch, md.Speed = true, s.watchSpeed
		if v, err := strconv.ParseFloat(r.URL.Query().Get("speed"), 64); err == nil && v >= 0 {
			if v > 0 {
				v = min(max(v, minWatchSpeed), maxWatchSpeed)
			}
			md.Speed = v
		}
	default:
		http.Error(w, "m=fast|watch", http.StatusBadRequest)
		return
	}
	s.mux.SetMode(md)
	json.NewEncoder(w).Encode(md)
}
