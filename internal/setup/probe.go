package setup

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"d2lod-bot/internal/launch"
)

// ProbeEnv makes a binary act as the probe child (see MaybeProbe).
const ProbeEnv = "D2BOT_SETUP_PROBE"

// ProbeResult is what the probe child reports.
type ProbeResult struct {
	Subreaper  bool   `json:"subreaper"`  // prctl(PR_SET_CHILD_SUBREAPER) worked
	Reparented bool   `json:"reparented"` // an orphaned grandchild became our child
	MemRead    bool   `json:"memRead"`    // /proc/<grandchild>/mem opened and read with no capability
	Error      string `json:"error,omitempty"`
}

// MaybeProbe must be called first in main of any binary that wants the subreaper probe (d2doctor, d2bot).
// When ProbeEnv is set it runs the probe, prints one JSON line and exits; otherwise it returns.
func MaybeProbe() {
	if os.Getenv(ProbeEnv) == "" {
		return
	}
	_ = json.NewEncoder(os.Stdout).Encode(runProbe())
	os.Exit(0)
}

// RunProbeChild re-executes this binary as the probe, so the parent's own process state is untouched.
func RunProbeChild(ctx context.Context) (ProbeResult, error) {
	exe, err := os.Executable()
	if err != nil {
		return ProbeResult{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe)
	cmd.Env = append(os.Environ(), ProbeEnv+"=1")
	out, err := cmd.Output()
	if err != nil {
		return ProbeResult{}, fmt.Errorf("probe child: %w", err)
	}
	var r ProbeResult
	if err := json.Unmarshal(out, &r); err != nil {
		return ProbeResult{}, fmt.Errorf("probe output %q: %w", strings.TrimSpace(string(out)), err)
	}
	return r, nil
}

// runProbe mirrors what the app does with the game: become subreaper, let a launcher start a long-lived
// child and exit (sh backgrounds sleep, like Wine's launchers), then read the orphan's memory directly.
// VERIFIED in principle by docs/launch.md 1 with a real Wine process; this probe needs only /bin/sh and sleep.
func runProbe() (r ProbeResult) {
	if err := launch.BecomeSubreaper(); err != nil {
		r.Error = err.Error()
		return
	}
	r.Subreaper = true
	cmd := exec.Command("/bin/sh", "-c", "sleep 20 >/dev/null 2>&1 & echo $!")
	out, err := cmd.Output()
	if err != nil {
		r.Error = "sh: " + err.Error()
		return
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		r.Error = "no pid from sh"
		return
	}
	defer func() { // the sleeper is our child now
		p, _ := os.FindProcess(pid)
		_ = p.Kill()
		_, _ = p.Wait()
	}()
	for i := 0; i < 40 && !r.Reparented; i++ {
		if ppid(pid) == os.Getpid() {
			r.Reparented = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !r.Reparented {
		r.Error = fmt.Sprintf("orphan %d has ppid %d, not us", pid, ppid(pid))
		return
	}
	addr, err := firstReadable(pid)
	if err != nil {
		r.Error = err.Error()
		return
	}
	f, err := os.Open("/proc/" + strconv.Itoa(pid) + "/mem")
	if err != nil {
		r.Error = "open mem: " + err.Error()
		return
	}
	defer f.Close()
	var b [4]byte
	if _, err := f.ReadAt(b[:], int64(addr)); err != nil {
		r.Error = "read mem: " + err.Error()
		return
	}
	r.MemRead = true
	return
}

func ppid(pid int) int {
	b, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return -1
	}
	s := string(b)
	i := strings.LastIndexByte(s, ')')
	f := strings.Fields(s[i+1:]) // state ppid ...
	if len(f) < 2 {
		return -1
	}
	n, _ := strconv.Atoi(f[1])
	return n
}

func firstReadable(pid int) (uint64, error) {
	f, err := os.Open("/proc/" + strconv.Itoa(pid) + "/maps")
	if err != nil {
		return 0, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fl := strings.Fields(sc.Text())
		if len(fl) < 2 || !strings.HasPrefix(fl[1], "r") || strings.Contains(sc.Text(), "[vvar") || strings.Contains(sc.Text(), "[vsyscall") {
			continue
		}
		start, _, _ := strings.Cut(fl[0], "-")
		return strconv.ParseUint(start, 16, 64)
	}
	return 0, fmt.Errorf("no readable mapping")
}

func sha256File(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
