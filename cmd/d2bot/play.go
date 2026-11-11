package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"d2lod-bot/internal/buildinfo"
	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/paths"
	"d2lod-bot/internal/watch"
)

// `d2bot play [flags]` is the one-command play session (docs/play-session.md): the same bot as the tester container, with
// the ladder, the watch screen on the LAN (viewer 8891, answers 8892), the game video and a watch token file switched on,
// the phone URL printed once, and a session record written next to the instance state. Every other flag is the normal
// d2bot flag, so a flag given again overrides the default (the flag package keeps the last one).

// playMode is set by playArgs and read once in main.
var playMode bool

// playOut receives the phone URL; the token appears here and nowhere else (not in the log, not in the record).
var playOut io.Writer = os.Stderr

// playDefaults are the environment twins the command fills in when the person has not set them.
func playDefaults(dataDir, lanIP string) map[string]string {
	return map[string]string{
		"D2BOT_LADDER":            "1",
		"D2BOT_WATCH":             "1",
		"D2BOT_WATCH_VIDEO":       "1",
		"D2BOT_WATCH_ADDR":        "0.0.0.0:8891",
		"D2BOT_WATCH_ANSWER_ADDR": "0.0.0.0:8892",
		"D2BOT_WATCH_TOKEN_FILE":  filepath.Join(dataDir, "watch.token"),
		"D2BOT_WATCH_ORIGIN":      "http://" + lanIP + ":8891",
		"D2BOT_LADDER_STORE":      "shared", // what the person teaches carries on across sessions
	}
}

// lanIPv4 is the first private IPv4 address of this machine, "127.0.0.1" when there is none.
func lanIPv4() string {
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		if ipn, ok := a.(*net.IPNet); ok {
			if ip := ipn.IP.To4(); ip != nil && ip.IsPrivate() {
				return ip.String()
			}
		}
	}
	return "127.0.0.1"
}

// playArgs prepares a `play` run: it sets the defaults, makes sure the token file exists, prints the URL once and
// returns the flags for parseFlags. data is the -data value in args or the default data dir.
func playArgs(args []string) ([]string, error) {
	playMode = true
	dir := dataDirOf(args)
	if dir == "" {
		l, err := paths.FromEnv()
		if err != nil {
			return nil, err
		}
		dir = l.Root
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	lan := lanIPv4()
	for k, v := range playDefaults(dir, lan) {
		if _, set := os.LookupEnv(k); !set {
			os.Setenv(k, v)
		}
	}
	tf := os.Getenv("D2BOT_WATCH_TOKEN_FILE")
	if tf == "" || os.Getenv("D2BOT_WATCH_TOKEN") != "" || os.Getenv(superviseMarker) != "" { // a supervised child never prints again
		return args, nil
	}
	tok, _, err := loadOrCreateToken(tf)
	if err != nil {
		return nil, err
	}
	if tok != "" {
		addr := os.Getenv("D2BOT_WATCH_ADDR")
		_, port, _ := net.SplitHostPort(addr)
		fmt.Fprintf(playOut, "play: open on your phone (same network): http://%s:%s/watch?t=%s\n", lan, port, tok)
		fmt.Fprintln(playOut, "play: this link is printed once per start; it is not written to the session record or the log")
	}
	return args, nil
}

// dataDirOf finds -data/--data in args (both "-data X" and "-data=X").
func dataDirOf(args []string) string {
	for i, a := range args {
		a = strings.TrimLeft(a, "-")
		if v, ok := strings.CutPrefix(a, "data="); ok {
			return v
		}
		if a == "data" && i+1 < len(args) {
			return args[i+1]
		}
	}
	return os.Getenv("D2BOT_DATA")
}

// playSessionConfig is the recorder's wiring to the running bot (all reads; nothing here writes game state).
func playSessionConfig(l ladderOptions, stateDir string, st *decide.Store, cat *decide.Catalogue, clk decide.Clock, watchTok string, status func() goals.Status, hub *watch.Hub, rig *ladderRig) watch.SessionConfig {
	var credits int64
	var creditsAt time.Time
	sc := watch.SessionConfig{Root: filepath.Join(stateDir, "sessions"), Profile: l.storeProfile(), Commit: buildinfo.Current(),
		CatalogueVersion: cat.Version, Store: st, Cat: cat, Clock: clk,
		Secrets:   []string{watchTok, l.watchToken}, // the access token is never written; free text is scrubbed of it
		Orders:    func() []watch.Order { return hub.OrdersView().Orders },
		WhyCounts: hub.Whyline().Counts,
		Viewers: func() int {
			if rig.Video == nil {
				return -1 // no video: the viewer count is unknown (U: only video viewers are counted)
			}
			return rig.Video.Viewers()
		}}
	if status != nil {
		sc.Identity = func() (string, string) {
			b := ""
			if p := planBuild.Load(); p != nil {
				b = *p
			}
			return strings.ToLower(status().Character.Class), b
		}
		sc.Vitals = func() watch.Vitals {
			c := status().Character
			xp, _ := watch.AbsoluteXP(c.Level, c.XP, 0)
			v := watch.Vitals{XP: xp}
			if c.MaxLife > 0 {
				v.LifePct = float64(int(1000*float64(c.Life)/float64(c.MaxLife)+0.5)) / 10
			}
			if rig.Wallet != nil && rig.Wallet.Ledger != nil && time.Since(creditsAt) > 30*time.Second {
				if es, err := rig.Wallet.Ledger.Entries(); err == nil {
					credits = int64(len(es))
				}
				creditsAt = time.Now()
			}
			v.Credits = credits
			return v
		}
	}
	return sc
}

// runSession polls the record once a second and closes it when the run ends.
func runSession(ctx context.Context, s *watch.Session) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			_ = s.Stop()
			return
		case <-t.C:
			s.Poll()
		}
	}
}

// sessionsMain runs `d2bot sessions export DIR [-o FILE] [-name N]...`: the opt-in anonymised copy of a session's
// decision dataset. Off unless asked; writes to stdout without -o.
func sessionsMain(args []string, out io.Writer) error {
	if len(args) == 0 || args[0] != "export" {
		return errors.New("usage: d2bot sessions export [-o FILE] [-name N]... SESSION_DIR")
	}
	fs := flag.NewFlagSet("sessions export", flag.ContinueOnError)
	o := fs.String("o", "", "output file (default stdout)")
	var names multiFlag
	fs.Var(&names, "name", "a name that must not leave the machine (profile, account, character); repeatable. The user name, host name and the profile in the directory name are always scrubbed")
	// flags may follow the directory
	var dir string
	rest := args[1:]
	for len(rest) > 0 {
		if err := fs.Parse(rest); err != nil {
			return err
		}
		if fs.NArg() == 0 {
			break
		}
		if dir != "" {
			return errors.New("sessions export: one session directory")
		}
		dir, rest = fs.Arg(0), fs.Args()[1:]
	}
	if dir == "" {
		return errors.New("usage: d2bot sessions export [-o FILE] [-name N]... SESSION_DIR")
	}
	all := append([]string(nil), names...)
	if u := os.Getenv("USER"); u != "" {
		all = append(all, u)
	}
	if h, err := os.Hostname(); err == nil {
		all = append(all, h)
	}
	if _, prof, ok := strings.Cut(filepath.Base(dir), "Z-"); ok {
		all = append(all, prof)
	}
	w := out
	if *o != "" {
		f, err := os.OpenFile(*o, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}
	n, err := watch.ExportSession(dir, w, watch.ExportOptions{Names: all})
	if err == nil {
		fmt.Fprintf(os.Stderr, "sessions export: done, %d value(s) scrubbed; read the file before you send it\n", n)
	}
	return err
}

type multiFlag []string

func (m *multiFlag) String() string     { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error { *m = append(*m, v); return nil }
