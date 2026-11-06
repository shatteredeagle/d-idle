// Command d2dsetup is the first-run wizard of the d2d tester container: it checks the mounted game data, asks for
// a character name, class and goal preset, warms the game table cache and then writes /data/wizard.json (and
// wizard.env) for the entrypoint. No Wine, no prefix, no patch step, no CD key. Design and hook: README.md.
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"d2lod-bot/internal/data"
	"d2lod-bot/internal/paths"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "d2dsetup:", err)
		os.Exit(1)
	}
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func run() error {
	dataDir := flag.String("data", env("D2SETUP_DATA", "/data"), "state directory (wizard.json, wizard.env, watch.token)")
	game := flag.String("game", env("D2D_DATA", "/game"), "mounted game data folder")
	addr := flag.String("addr", env("D2SETUP_ADDR", "0.0.0.0:8891"), "listen address (the port the watch screen takes over afterwards)")
	tokenFile := flag.String("token-file", env("D2BOT_WATCH_TOKEN_FILE", ""), "watch token file (default <data>/watch.token); created 0600 when missing")
	tables := flag.String("tables", "", "game table cache dir (default <d2bot data root>/cache/tables, where d2bot looks)")
	origin := flag.String("origin", env("D2BOT_WATCH_ORIGIN", ""), "page origin used in the printed URL (first of a comma list)")
	linger := flag.Duration("linger", 20*time.Second, "how long the page stays up after Start finished")
	flag.Parse()

	if *tokenFile == "" {
		*tokenFile = filepath.Join(*dataDir, "watch.token")
	}
	if *tables == "" {
		l, err := paths.FromEnv()
		if err != nil {
			return err
		}
		*tables = filepath.Join(l.Root, "cache", "tables")
	}
	w, err := NewWizard(*dataDir, *game, DefaultTables(), tableExtractor(*tables))
	if err != nil {
		return err
	}
	if w.Done() {
		fmt.Println("d2dsetup: setup already done (", filepath.Join(*dataDir, stateFile), ")")
		return nil
	}
	tok, fresh, err := loadOrCreateToken(*tokenFile)
	if err != nil {
		return err
	}
	base := "http://" + *addr
	if o, _, _ := strings.Cut(*origin, ","); o != "" {
		base = strings.TrimRight(o, "/")
	}
	srv, err := NewServer(w, tok)
	if err != nil {
		return err
	}
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		return err
	}
	if fresh {
		fmt.Printf("d2dsetup: token created in %s\n", *tokenFile)
		fmt.Printf("d2dsetup: open %s/setup.html?t=%s\n", base, tok)
	} else {
		fmt.Printf("d2dsetup: open %s/setup.html?t=<the token in %s>\n", base, *tokenFile)
	}
	hs := &http.Server{Handler: srv, ReadHeaderTimeout: 10 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		t := time.NewTicker(500 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if w.Done() {
					time.Sleep(*linger)
					stop()
					return
				}
			}
		}
	}()
	go func() {
		<-ctx.Done()
		sc, c := context.WithTimeout(context.Background(), 3*time.Second)
		defer c()
		hs.Shutdown(sc)
	}()
	if err := hs.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	if !w.Done() {
		return errors.New("stopped before the wizard finished")
	}
	log.Print("d2dsetup: done")
	return nil
}

// tableExtractor warms the cache d2bot reads (cmd/d2bot/d2dbackend.go openTables), so the bot's first start finds it current.
func tableExtractor(cacheDir string) Extractor {
	return func(game string) error {
		_, err := data.Open(game, cacheDir)
		return err
	}
}

// loadOrCreateToken is the watch token file contract of cmd/d2bot/ladder.go (32 characters base64url, 0600), so the
// bot that starts afterwards reads the same file.
func loadOrCreateToken(path string) (tok string, created bool, err error) {
	if b, err := os.ReadFile(path); err == nil {
		if st, e := os.Stat(path); e == nil && st.Mode().Perm()&0o077 != 0 {
			if e := os.Chmod(path, 0o600); e != nil {
				return "", false, fmt.Errorf("token file %s is readable by others and cannot be fixed: %w", path, e)
			}
		}
		tok = strings.TrimSpace(string(b))
		if len(tok) < 16 {
			return "", false, fmt.Errorf("token file %s is too short: delete it to have a new token made", path)
		}
		return tok, false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", false, err
	}
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", false, err
	}
	tok = base64.RawURLEncoding.EncodeToString(raw)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", false, err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", false, err
	}
	if _, err := f.WriteString(tok + "\n"); err != nil {
		f.Close()
		return "", false, err
	}
	return tok, true, f.Close()
}
