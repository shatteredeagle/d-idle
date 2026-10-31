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
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
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
	addr := flag.String("addr", ":8891", "listen address; a token is required unless it is loopback or -open")
	upstream := flag.String("upstream", os.Getenv("D2D_SOCK"), "the d2d control socket (default $D2D_SOCK)")
	botSock := flag.String("bot-sock", "", "socket to create for the bot (it dials this instead of the d2d); empty: no bot, the viewer attaches alone")
	tokenFile := flag.String("token-file", "", "file with the access token (or env D2DVIEW_TOKEN); default ~/.config/d2dview/token")
	open := flag.Bool("open", false, "serve without a token on a non-loopback address (LAN only)")
	hz := flag.Float64("hz", 15, "state polls per wall second while someone watches")
	fps := flag.Float64("fps", 6, "sampled game frames per wall second (the real rate backs off, see docs/d2d-view.md); 0: no frames")
	shots := flag.String("shots", fmt.Sprintf("/tmp/d2dview-%d", os.Getuid()), "private dir on the d2d host where screenshots are written")
	botAPI := flag.String("bot-api", "", "bot API base URL (read-only GET /api/v1/status), e.g. http://localhost:8080")
	watchSpeed := flag.Float64("watch-speed", 8, "watch mode: max virtual seconds per wall second (0: render every step, no limit)")
	jpegOut := flag.Bool("jpeg", false, "re-encode frames as JPEG (less bandwidth, about 25 ms CPU per frame); default passes d2d's PNG through")
	badgeFlag := flag.String("badge", envOr("D2DVIEW_BADGE", Backend), "header badge text of the stream page and the /api/pub backend field (env D2DVIEW_BADGE)")
	flag.Parse()
	badge = safeText(*badgeFlag, 40)
	if badge == "" {
		badge = Backend
	}
	if *upstream == "" {
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

	mux := NewMux(*upstream)
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
		frames.JPEG = *jpegOut
		feed.fr = frames
		go frames.Run(ctx)
	}
	s := &server{feed: feed, frames: frames, mux: mux, token: token, watchSpeed: *watchSpeed}
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

func loopback(addr string) bool {
	h, _, err := net.SplitHostPort(addr)
	if err != nil || h == "" {
		return false
	}
	ip := net.ParseIP(h)
	return h == "localhost" || (ip != nil && ip.IsLoopback())
}

type server struct {
	feed       *Feed
	frames     *Frames
	mux        *Mux
	token      string
	watchSpeed float64
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
	m.HandleFunc("/api/mode", s.setMode)
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

func (s *server) mapPNG(w http.ResponseWriter, r *http.Request) {
	b, v := s.feed.PNG()
	if b == nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Map-Version", strconv.FormatUint(v, 10))
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
	var seq int64
	for r.Context().Err() == nil {
		s.frames.Touch()
		f := s.frames.Current()
		if f.Seq == seq || f.Data == nil {
			time.Sleep(20 * time.Millisecond)
			continue
		}
		seq = f.Seq
		fmt.Fprintf(w, "--f\r\nContent-Type: %s\r\nContent-Length: %d\r\nX-Frame-Ticks: %d\r\n\r\n", f.Type, len(f.Data), f.Ticks)
		w.Write(f.Data)
		fmt.Fprint(w, "\r\n")
		fl.Flush()
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
