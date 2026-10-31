// Command d2watch is a view-only web viewer for the running Diablo II instances: live tiles of each game
// window plus metadata read from game memory. It runs on the worker and never sends input to a game.
// Docs: docs/watch.md.
package main

import (
	"crypto/subtle"
	"d2lod-bot/internal/buildinfo"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"strings"
	"time"

	"d2lod-bot/internal/paths"
	"d2lod-bot/internal/viewport"
)

//go:embed web
var webFS embed.FS

func main() {
	buildinfo.Exit("d2watch")
	addr := flag.String("addr", ":8890", "listen address; a token is required unless it is a loopback address")
	tokenFile := flag.String("token-file", "", "file holding the access token (or env D2WATCH_TOKEN); default <data>/tokens/d2watch, the old ~/.config/d2watch/token is moved there")
	conf := flag.String("conf", "~/code/d2lod-bot/tools/instances.conf", "agent -> instance map (tools/instances.conf)")
	fps := flag.Float64("fps", 4, "capture frames per second per watched instance")
	conc := flag.Int("capture-conc", 3, "max concurrent window captures (a persistent ffmpeg capturer holds one slot per watched instance)")
	capt := flag.String("capturer", "import", "capture method: import (one ImageMagick import per frame) or ffmpeg (persistent x11grab)")
	pprofAddr := flag.String("pprof", "", "serve net/http/pprof on this loopback address (empty: off)")
	flag.Parse()
	if *pprofAddr != "" {
		go func() { log.Println(http.ListenAndServe(*pprofAddr, nil)) }()
	}

	token := os.Getenv("D2WATCH_TOKEN")
	if token == "" {
		if b, err := os.ReadFile(resolveTokenFile(*tokenFile)); err == nil {
			token = strings.TrimSpace(string(b))
		}
	}
	if token == "" && !loopback(*addr) {
		log.Fatal("d2watch: a token is required unless -addr is loopback (D2WATCH_TOKEN or -token-file)")
	}

	var c viewport.Capturer
	switch *capt {
	case "import":
		c = viewport.ImportCapturer{}
	case "ffmpeg":
		c = viewport.FFmpegCapturer{}
	default:
		log.Fatalf("d2watch: unknown -capturer %q (import or ffmpeg)", *capt)
	}
	reg := newRegistry(expand(*conf), *fps, *conc, c)
	go reg.run()

	sub, _ := fs.Sub(webFS, "web")
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServerFS(sub))
	mux.HandleFunc("/api/instances", reg.handleAPI)
	mux.HandleFunc("/frame/", reg.handleFrame)
	mux.HandleFunc("/stream/", reg.handleStream)

	srv := &http.Server{Addr: *addr, Handler: guard(token, mux), ReadHeaderTimeout: 10 * time.Second}
	log.Printf("d2watch listening on %s (token required: %v)", *addr, token != "")
	log.Fatal(srv.ListenAndServe())
}

// resolveTokenFile returns the token file to read: the -token-file flag, else paths.Layout.Token("d2watch").
// With the default, a token at the old ~/.config/d2watch/token is moved there first (never overwriting).
func resolveTokenFile(flagVal string) string {
	if flagVal != "" {
		return expand(flagVal)
	}
	l, err := paths.FromEnv()
	if err != nil {
		return ""
	}
	os.MkdirAll(filepath.Dir(l.Token("d2watch")), 0o700)
	if _, err := paths.MoveIfAbsent(l.LegacyToken("d2watch"), l.Token("d2watch")); err != nil {
		log.Printf("d2watch: legacy token not moved: %v", err)
	}
	return l.Token("d2watch")
}

func expand(p string) string {
	if strings.HasPrefix(p, "~/") {
		if h, err := os.UserHomeDir(); err == nil {
			return filepath.Join(h, p[2:])
		}
	}
	return p
}

func loopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil || host == "" {
		return false
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// guard makes the server read-only (GET/HEAD only) and checks the token (query, X-Token header or cookie).
// A valid query token is turned into a cookie so <img>/stream requests need no token in their URL.
func guard(token string, next http.Handler) http.Handler {
	eq := func(a string) bool { return subtle.ConstantTimeCompare([]byte(a), []byte(token)) == 1 }
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "view only", http.StatusMethodNotAllowed)
			return
		}
		if token != "" {
			q := r.URL.Query().Get("token")
			ok := eq(q) || eq(r.Header.Get("X-Token"))
			if !ok {
				if c, err := r.Cookie("d2watch"); err == nil && eq(c.Value) {
					ok = true
				}
			}
			if !ok {
				http.Error(w, "token required", http.StatusUnauthorized)
				return
			}
			if q != "" && eq(q) {
				http.SetCookie(w, &http.Cookie{Name: "d2watch", Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: 30 * 86400})
			}
		}
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func fail(w http.ResponseWriter, code int, msg string) { http.Error(w, fmt.Sprint(msg), code) }
