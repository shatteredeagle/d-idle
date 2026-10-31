// subreaperproof demonstrates (on a SCRATCH prefix, never the live game) that a process which is a
// child subreaper can read /proc/<pid>/mem of Wine processes with no capability and no root, even
// after the launcher exited, and that a non-descendant cannot.
//
//	subreaperproof run -winexe winprobe.exe -scratch DIR [-subreaper=false]
//	subreaperproof lifecycle -winexe winprobe.exe -scratch DIR   (Managed start/find/attach/health/stop)
//	subreaperproof probe PID          (read the MZ header of PID's image; used for the non-ancestor check)
//
// Build the Windows side with: GOOS=windows GOARCH=386 go build -o winprobe.exe ./tools/subreaperproof/winprobe
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"d2lod-bot/internal/launch"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: subreaperproof run|probe ...")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "probe":
		var pid int
		fmt.Sscan(os.Args[2], &pid)
		os.Exit(probe(pid))
	case "run":
		os.Exit(run(os.Args[2:]))
	case "lifecycle":
		os.Exit(lifecycle(os.Args[2:]))
	}
}

func probe(pid int) int {
	f, err := launch.OpenMem(pid)
	if err != nil {
		fmt.Printf("probe pid=%d ppid-chain-ok=false open /proc/%d/mem: %v\n", pid, pid, err)
		return 1
	}
	defer f.Close()
	var b [2]byte
	if _, err := f.ReadAt(b[:], 0x400000); err != nil {
		fmt.Printf("probe pid=%d read 0x400000: %v\n", pid, err)
		return 1
	}
	fmt.Printf("probe pid=%d read 0x400000 = %q\n", pid, b[:])
	return 0
}

func status(pid int, keys ...string) string {
	b, _ := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	var out []string
	for _, l := range strings.Split(string(b), "\n") {
		for _, k := range keys {
			if strings.HasPrefix(l, k+":") {
				out = append(out, strings.Join(strings.Fields(l), " "))
			}
		}
	}
	return strings.Join(out, "; ")
}

func run(args []string) int {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	winexe := fs.String("winexe", "winprobe.exe", "windows test program")
	scratch := fs.String("scratch", "", "scratch root (prefix is created under it)")
	sub := fs.Bool("subreaper", true, "set PR_SET_CHILD_SUBREAPER (false = control experiment)")
	keep := fs.Bool("keep", false, "do not wineserver -k at the end")
	fs.Parse(args)
	if *scratch == "" {
		fmt.Fprintln(os.Stderr, "-scratch required")
		return 2
	}
	self := os.Getpid()
	yama, _ := os.ReadFile("/proc/sys/kernel/yama/ptrace_scope")
	fmt.Printf("self pid=%d  %s  yama.ptrace_scope=%s", self, status(self, "CapEff", "Uid"), yama)
	if *sub {
		if err := launch.BecomeSubreaper(); err != nil {
			fmt.Println(err)
			return 1
		}
	}
	on, _ := launch.IsSubreaper()
	fmt.Println("child subreaper:", on)

	cfg := launch.Config{Root: *scratch}
	if err := cfg.Resolve(); err != nil {
		fmt.Println(err)
		return 1
	}
	os.MkdirAll(cfg.LogDir, 0o755)
	fmt.Println("runner:", cfg.ProtonScript)
	defer func() {
		if !*keep {
			cfg.WineserverKill()
		}
	}()

	cmd := cfg.ProtonCmd(context.Background(), "", "run", *winexe)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	t0 := time.Now()
	if err := cmd.Run(); err != nil {
		fmt.Println("proton run:", err)
	}
	fmt.Printf("launcher chain (proton run) exited after %s\n", time.Since(t0).Round(time.Second))

	procs, _ := launch.ListProcs()
	sl := launch.FindImage(procs, "winprobe.exe")
	if len(sl) == 0 {
		fmt.Println("FAIL: no winprobe.exe process found")
		return 1
	}
	// The sleeper is the highest-PID copy (the launcher already exited).
	p := sl[len(sl)-1]
	fmt.Printf("target: pid=%d ppid=%d comm=%q argv=%q exe=%s\n", p.PID, p.PPID, p.Comm, p.Argv, p.Exe)
	fmt.Printf("descendant of self: %v   (ppid==self: %v)\n", launch.IsDescendant(procs, p.PID, self), p.PPID == self)

	base, err := launch.ImageBase(p.PID, "winprobe.exe")
	fmt.Printf("image base from /proc/%d/maps: 0x%x err=%v\n", p.PID, base, err)
	f, err := launch.OpenMem(p.PID)
	rc := 0
	if err != nil {
		fmt.Printf("RESULT self-open: FAIL %v\n", err)
		rc = 1
	} else {
		var b [2]byte
		_, rerr := f.ReadAt(b[:], int64(base))
		fmt.Printf("RESULT self-open: ok, read at 0x%x = %q err=%v\n", base, b[:], rerr)
		f.Close()
	}

	// Non-descendant: run the probe from the systemd user manager, whose process tree is not ours.
	exe, _ := os.Executable()
	sd := exec.Command("systemd-run", "--user", "--pipe", "--wait", "--quiet", exe, "probe", fmt.Sprint(p.PID))
	out, err := sd.CombinedOutput()
	fmt.Printf("RESULT non-descendant probe (systemd-run --user): %s (err=%v)\n", strings.TrimSpace(string(out)), err)
	return rc
}

// lifecycle exercises Managed, Monitor, CPUTicks and Stop on the scratch prefix with the harmless program.
func lifecycle(args []string) int {
	fs := flag.NewFlagSet("lifecycle", flag.ExitOnError)
	winexe := fs.String("winexe", "winprobe.exe", "windows test program")
	scratch := fs.String("scratch", "", "scratch root")
	fs.Parse(args)
	if err := launch.BecomeSubreaper(); err != nil {
		fmt.Println(err)
		return 1
	}
	cfg := launch.Config{Root: *scratch}
	if err := cfg.Resolve(); err != nil {
		fmt.Println(err)
		return 1
	}
	defer cfg.WineserverKill()
	m := &launch.Managed{Cfg: cfg, Spawn: launch.NewSpawner(), Exe: *winexe, Args: []string{"sleeper"}, Image: "winprobe.exe", Log: "lifecycle.out"}
	if err := m.Start(context.Background()); err != nil {
		fmt.Println("start:", err)
		return 1
	}
	p, err := m.WaitUp(context.Background(), 90*time.Second)
	fmt.Printf("up: pid=%d err=%v\n", p.PID, err)
	if err != nil {
		return 1
	}
	time.Sleep(3 * time.Second) // let the launcher chain exit
	launch.Reap()
	f, err := launch.OpenMem(p.PID)
	fmt.Printf("mem open after launcher exit: err=%v\n", err)
	if f != nil {
		f.Close()
	}
	t1, _ := launch.CPUTicks(p.PID)
	mon := &launch.Monitor{P: launch.Probes{
		Alive:     func() (int, bool) { q, ok := m.Find(); return q.PID, ok },
		Heartbeat: launch.CPUTicks,
	}, HangAfter: 4 * time.Second}
	now := time.Now()
	fmt.Println("monitor t0:", mon.Step(now))
	fmt.Println("monitor t+5s (sleeper burns no CPU => expect hung):", mon.Step(now.Add(5*time.Second)), "ticks", t1)
	t0 := time.Now()
	err = m.Stop(10 * time.Second)
	_, still := m.Find()
	fmt.Printf("Stop(SIGTERM) err=%v took=%s still-running=%v\n", err, time.Since(t0).Round(100*time.Millisecond), still)
	fmt.Println("monitor after stop:", mon.Step(now.Add(6*time.Second)))
	return 0
}
