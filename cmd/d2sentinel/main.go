// d2sentinel: out-of-process failsafe watcher for the game instances (layer 2, PLAN.md "Failsafes").
// Observe-only unless --act (all rules) or --act-rules a,b. Run on the worker via tools/sentinel.sh.
//
//	d2sentinel [-config FILE] [-instances i1,i2] [-act] [-act-rules r1,r2] [-keep i5] [-log FILE] [-addr 127.0.0.1:8895] [-token T] [-once]
package main

import (
	"context"
	"d2lod-bot/internal/buildinfo"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"d2lod-bot/internal/paths"
	"d2lod-bot/internal/sentinel"
	"d2lod-bot/internal/status"
)

func main() {
	buildinfo.Exit("d2sentinel")
	conf := flag.String("config", "", "JSON config (thresholds, instances, rules); defaults when empty")
	insts := flag.String("instances", "i1,i2,i3,i4,i5", "comma list of instances to watch (config entries with enabled:false stay off)")
	act := flag.Bool("act", false, "let every rule act (default: observe-only, log what it would do)")
	actRules := flag.String("act-rules", "", "comma list of rules allowed to act (implies observe-only for the others)")
	actInst := flag.String("act-instances", "", "comma list: acting is allowed only on these instances (others observe-only)")
	keep := flag.String("keep", "", "comma list of instances the idle reaper never stops")
	logPath := flag.String("log", "", "JSONL log file (default <data>/logs/sentinel.jsonl; the old ~/.local/state/d2lod/sentinel.jsonl is moved there on start)")
	addr := flag.String("addr", "127.0.0.1:8895", "HTTP listen address for /api/v1/sentinel and /live/ (non-loopback requires -token)")
	token := flag.String("token", os.Getenv("D2BOT_TOKEN"), "require this token (Bearer, X-Token, cookie, ?token=); default $D2BOT_TOKEN")
	allow := flag.String("allow-host", "", "extra Host name accepted on a loopback listener")
	gameSh := flag.String("game-sh", "", "DEPRECATED, ignored (restarts use internal/launch)")
	muteSh := flag.String("mute-sh", "", "path of tools/mute-d2.sh (run after launches)")
	once := flag.Bool("once", false, "print one observation per instance as JSON and exit")
	flag.Parse()

	cfg, err := sentinel.LoadConfig(*conf)
	if err != nil {
		log.Fatal(err)
	}
	cfg.WithInstances(strings.Split(*insts, ","))
	if *keep != "" {
		cfg.Keep = append(cfg.Keep, strings.Split(*keep, ",")...)
	}
	if *gameSh != "" {
		cfg.GameSh = *gameSh
	}
	if *muteSh != "" {
		cfg.MuteSh = *muteSh
	}
	enabled := map[string]bool{}
	for _, id := range strings.Split(*insts, ",") {
		if id = strings.TrimSpace(id); id != "" {
			enabled[id] = true
		}
	}

	if *once {
		for _, in := range cfg.Instances {
			if !enabled[in.ID] {
				continue
			}
			o := sentinel.NewObserver(in, cfg).Observe(time.Now())
			b, _ := json.Marshal(o)
			fmt.Println(string(b))
		}
		return
	}

	if *logPath == "" {
		l, err := paths.FromEnv()
		if err != nil {
			log.Fatal(err)
		}
		*logPath = l.SentinelLog()
		if moved, err := paths.MoveIfAbsent(l.LegacySentinelLog(), *logPath); err != nil {
			log.Printf("legacy sentinel log not moved: %v", err)
		} else if moved {
			log.Printf("moved %s to %s", l.LegacySentinelLog(), *logPath)
		}
	}
	os.MkdirAll(filepath.Dir(*logPath), 0o755)
	lf, err := os.OpenFile(*logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		log.Fatal(err)
	}
	mode := sentinel.Mode{Act: *act}
	if *actRules != "" {
		mode.ActRules = map[string]bool{}
		for _, r := range strings.Split(*actRules, ",") {
			if r = strings.TrimSpace(r); r != "" {
				mode.ActRules[r] = true
			}
		}
	}
	if *actInst != "" {
		mode.ActInst = map[string]bool{}
		for _, r := range strings.Split(*actInst, ",") {
			mode.ActInst[strings.TrimSpace(r)] = true
		}
	}
	actor := &sentinel.LiveActor{Cfg: cfg, Log: func(s string) { log.Print(s) }}
	r := sentinel.NewRunner(cfg, mode, actor, lf)
	r.Enabled = enabled
	r.Observe = func(in sentinel.InstanceConf) func(time.Time) sentinel.Observation {
		return sentinel.NewObserver(in, cfg).Observe
	}
	hub := status.NewHub()
	var hosts []string
	if *allow != "" {
		hosts = []string{*allow}
	}
	srv, handler, err := r.NewServer(hub, *addr, *token, hosts)
	if err != nil {
		log.Fatal(err)
	}
	ln, err := srv.Listen()
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go hub.Run(ctx)
	go func() {
		hs := &http.Server{Handler: handler, ReadHeaderTimeout: 10 * time.Second}
		go func() {
			<-ctx.Done()
			hs.Close()
			srv.Close()
		}()
		if err := hs.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Print(err)
		}
	}()
	log.Printf("d2sentinel %s on %s, log %s", mode, *addr, *logPath)
	r.Run(ctx)
}
