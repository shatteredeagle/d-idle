// Command d2idle-watch opens the d.idle watch page in its own app window and remembers the game address.
// Design and the reasons for this stack: docs/desktop-app.md.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

var goos = runtime.GOOS

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "d2idle-watch:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("d2idle-watch", flag.ContinueOnError)
	page := fs.String("page", "watch", "page to open: watch, inventory, policy, return")
	address := fs.String("address", "", "set and store the game address (host:port), then open")
	setup := fs.Bool("setup", false, "ask for the address again (change address)")
	dry := fs.Bool("print", false, "print the resolved URL and browser, open nothing")
	install := fs.Bool("install", false, "Linux: write the desktop entry and icon into ~/.local/share")
	if err := fs.Parse(args); err != nil {
		return err
	}
	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	if *install {
		return installDesktop(dir)
	}
	if _, ok := Pages[*page]; !ok {
		_, err := PageURL("", *page)
		return err
	}
	cfg, err := LoadConfig(dir)
	if err != nil {
		return fmt.Errorf("reading config in %s: %w", dir, err)
	}
	if *address != "" {
		a, err := NormalizeAddress(*address)
		if err != nil {
			return err
		}
		cfg.Address = a
		if err := SaveConfig(dir, cfg); err != nil {
			return err
		}
	}
	if *setup {
		cfg.Address = ""
	}
	profile := ProfileDir(dir)

	if cfg.Address != "" && Probe(context.Background(), NewProbeClient(), cfg.Address) == Up {
		u, _ := PageURL(cfg.Address, *page)
		if *dry {
			fmt.Println(u, "browser:", systemBrowser())
			return nil
		}
		return openWindow(u, profile)
	}

	s, ln, err := newShell(dir, cfg, *page)
	if err != nil {
		return err
	}
	if *dry {
		ln.Close()
		fmt.Println("shell:", s.URL(), "browser:", systemBrowser())
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(profile), 0o700); err != nil {
		return err
	}
	if err := openWindow(s.URL(), profile); err != nil {
		return err
	}
	// The window polls every 3 s; a hidden window may be throttled to about once a minute, hence the long idle.
	idle := 3 * time.Minute
	if d, err := time.ParseDuration(os.Getenv("D2IDLE_SHELL_IDLE")); err == nil {
		idle = d // tests only
	}
	s.Serve(context.Background(), ln, idle)
	return nil
}
