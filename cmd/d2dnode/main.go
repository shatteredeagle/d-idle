// d2dnode is one member of the leaderless d2d node mesh (docs/d2d-mesh.md): it gossips the
// node and job tables with its peers over mutual TLS, serves the same JSON API and status page
// as every other member on its one port, and runs allow-listed recipes locally.
//
//	d2dnode -name NAME -token-file FILE [-listen 127.0.0.1:8470] [-seed host:port ...] [-labels a,b] [-capacity N] [-dry-run]
//
// Nothing here names a real host: seeds come from flags or D2D_MESH_SEEDS, the token from a file
// (default $D2D_MESH_TOKEN_FILE or ~/.config/d2dnode/token). PROTOTYPE: loopback by default,
// never deployed; the real runner shells to tools/d2dslot.sh and is off unless -runner is given.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"d2lod-bot/internal/d2dmesh"
)

const version = "proto-1"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "d2dnode:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("d2dnode", flag.ContinueOnError)
	host, _ := os.Hostname()
	name := fs.String("name", envOr("D2D_MESH_NAME", host), "node name (letters, digits, . _ -)")
	listen := fs.String("listen", envOr("D2D_MESH_LISTEN", "127.0.0.1:8470"), "listen address (gossip, API and page; loopback unless you mean it)")
	seeds := fs.String("seed", envOr("D2D_MESH_SEEDS", ""), "comma-separated seed addresses host:port (an empty list starts a new mesh)")
	tokenFile := fs.String("token-file", envOr("D2D_MESH_TOKEN_FILE", ""), "file with the join token (default ~/.config/d2dnode/token)")
	labels := fs.String("labels", envOr("D2D_MESH_LABELS", ""), "comma-separated node labels (jobs may require some)")
	capacity := fs.Int("capacity", envInt("D2D_MESH_CAPACITY", 1), "max concurrent jobs on this node (= local d2d slots)")
	dry := fs.Bool("dry-run", false, "run recipes as sleeps instead of launching d2d")
	runner := fs.String("runner", envOr("D2D_MESH_RUNNER", ""), "script that runs a recipe (tools/d2dslot.sh-style; args: RECIPE OUTDIR then key=value); empty = refuse all work unless -dry-run")
	guard := fs.String("guard", envOr("D2D_MESH_GUARD", ""), "host admission script; exit 0 = may claim (for example tools/capacity.sh --add 1)")
	outRoot := fs.String("out", envOr("D2D_MESH_OUT", filepath.Join(os.TempDir(), "d2dnode")), "root for job output dirs (job id/attempt)")
	interval := fs.Duration("interval", 3*time.Second, "anti-entropy period")
	lease := fs.Duration("lease", 30*time.Second, "job lease length (heartbeat every third)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	token, err := readToken(*tokenFile)
	if err != nil {
		return err
	}
	cfg := d2dmesh.Config{Name: *name, Listen: *listen, Seeds: splitList(*seeds), Token: token, Labels: splitList(*labels),
		Capacity: *capacity, Version: version, Interval: *interval, LeaseTTL: *lease}
	switch {
	case *dry:
		cfg.Runner = d2dmesh.DryRunner{}
	case *runner != "":
		cfg.Runner = &scriptRunner{script: *runner, outRoot: *outRoot}
	default:
		cfg.Runner = refuseRunner{}
		cfg.Guard = func() bool { return false }
	}
	if *guard != "" {
		g := *guard
		cfg.Guard = func() bool { return exec.Command(g).Run() == nil }
	}
	n, err := d2dmesh.New(cfg)
	if err != nil {
		return err
	}
	if err := n.Start(); err != nil {
		return err
	}
	fmt.Printf("d2dnode %s listening on https://%s (seeds: %d, capacity %d, labels %v)\n", *name, n.Addr(), len(cfg.Seeds), *capacity, cfg.Labels)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	n.Leave()
	return nil
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func envInt(k string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(k)); err == nil {
		return v
	}
	return def
}

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func readToken(path string) (string, error) {
	if path == "" {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, ".config", "d2dnode", "token")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("join token: %w (write a random string to %s and share it with the other nodes)", err, path)
	}
	t := strings.TrimSpace(string(b))
	if len(t) < 16 {
		return "", errors.New("join token must be at least 16 characters")
	}
	return t, nil
}

// refuseRunner is the default when no runner is configured: the node is a pure manager.
type refuseRunner struct{}

func (refuseRunner) Run(context.Context, *d2dmesh.JobRecord, int) (d2dmesh.JobResult, error) {
	return d2dmesh.JobResult{Exit: 75}, errors.New("no runner configured on this node")
}

// scriptRunner runs one validated recipe through a local script (tools/d2dslot.sh-style: it
// takes a slot, launches a hidden d2d, records the pid). The arguments are the recipe name,
// the output dir and the validated key=value pairs; nothing from the job reaches a shell.
type scriptRunner struct {
	script  string
	outRoot string
}

func (r *scriptRunner) Run(ctx context.Context, j *d2dmesh.JobRecord, attempt int) (d2dmesh.JobResult, error) {
	out := filepath.Join(r.outRoot, d2dmesh.SafeName(j.ID), strconv.Itoa(attempt))
	if err := os.MkdirAll(out, 0o755); err != nil {
		return d2dmesh.JobResult{Exit: 1}, err
	}
	args := []string{j.Recipe, out}
	for k, v := range j.Args {
		args = append(args, k+"="+v)
	}
	cmd := exec.CommandContext(ctx, r.script, args...)
	cmd.Dir = out
	cmd.Env = append(os.Environ(), "D2D_JOB="+j.ID, "D2D_ATTEMPT="+strconv.Itoa(attempt))
	logf, err := os.Create(filepath.Join(out, "log"))
	if err != nil {
		return d2dmesh.JobResult{Exit: 1}, err
	}
	defer logf.Close()
	cmd.Stdout, cmd.Stderr = logf, logf
	cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) } // stop by the recorded pid only
	cmd.WaitDelay = 20 * time.Second
	if err := cmd.Start(); err != nil {
		return d2dmesh.JobResult{Exit: 1, OutDir: out}, err
	}
	res := d2dmesh.JobResult{OutDir: out, Pid: cmd.Process.Pid}
	_ = os.WriteFile(filepath.Join(out, "pid"), []byte(strconv.Itoa(cmd.Process.Pid)), 0o644)
	err = cmd.Wait()
	res.Exit = cmd.ProcessState.ExitCode()
	if err != nil {
		return res, err
	}
	res.Summary = "ok"
	return res, nil
}
