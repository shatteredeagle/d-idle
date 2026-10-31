package memfd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"d2lod-bot/internal/launch"
	"d2lod-bot/internal/mem"
)

var _ mem.Source = (*Source)(nil)
var _ mem.Source = (*Remote)(nil)

const (
	markerAddr = 0x30000000
	marker     = "MEMFD-MARKER-d2lod"
)

// TestMain doubles as the child programs of the proof: D2_MEMFD_ROLE selects forker, sleeper or helper.
func TestMain(m *testing.M) {
	switch os.Getenv("D2_MEMFD_ROLE") {
	case "forker":
		roleForker()
	case "sleeper":
		roleSleeper()
	case "helper":
		roleHelper()
	}
	os.Exit(m.Run())
}

// forker starts the sleeper without waiting and exits at once, so the sleeper is reparented (to the
// subreaper test process): the same shape as Wine's short-lived launchers.
func roleForker() {
	c := exec.Command(os.Args[0])
	c.Env = append(os.Environ(), "D2_MEMFD_ROLE=sleeper")
	if err := c.Start(); err != nil {
		os.Exit(2)
	}
	os.Exit(0)
}

// sleeper maps a page at a fixed low address (like a 32-bit game) holding the marker, writes its pid, sleeps.
func roleSleeper() {
	const mapFixedNoreplace = 0x100000
	b, _, e := syscall.Syscall6(syscall.SYS_MMAP, markerAddr, 4096, syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_PRIVATE|syscall.MAP_ANON|mapFixedNoreplace, ^uintptr(0), 0)
	if e != 0 || b != markerAddr {
		os.Exit(3)
	}
	copy(unsafeBytes(b, 4096), marker)
	_ = os.WriteFile(os.Getenv("D2_MEMFD_PIDFILE"), []byte(strconv.Itoa(os.Getpid())), 0o600)
	time.Sleep(time.Minute)
	os.Exit(0)
}

// helper is the sibling: it gets the fd from the server and reads, then tries a direct open and reports.
func roleHelper() {
	type result struct {
		Got       string `json:"got"`
		Err       string `json:"err,omitempty"`
		DirectErr string `json:"direct_err"`
		Euid      int    `json:"euid"`
	}
	var r result
	r.Euid = os.Geteuid()
	cl := &Client{Path: os.Getenv("D2_MEMFD_SOCK")}
	s, info, err := cl.Get("i0")
	if err != nil {
		r.Err = err.Error()
	} else {
		pid, _ := strconv.Atoi(strings.TrimPrefix(strings.Fields(info)[0], "pid="))
		buf := make([]byte, len(marker))
		if err := s.ReadAt(buf, markerAddr); err != nil {
			r.Err = err.Error()
		} else {
			r.Got = string(buf)
		}
		_, derr := os.Open(fmt.Sprintf("/proc/%d/mem", pid))
		if derr != nil {
			r.DirectErr = derr.Error()
		}
	}
	_ = json.NewEncoder(os.Stdout).Encode(r)
	os.Exit(0)
}

func yamaScope() int {
	b, err := os.ReadFile("/proc/sys/kernel/yama/ptrace_scope")
	if err != nil {
		return -1
	}
	n, _ := strconv.Atoi(strings.TrimSpace(string(b)))
	return n
}

func kernel() string {
	b, _ := os.ReadFile("/proc/sys/kernel/osrelease")
	return strings.TrimSpace(string(b))
}

func startSleeper(t *testing.T) (pid int) {
	t.Helper()
	pidfile := filepath.Join(t.TempDir(), "pid")
	c := exec.Command(os.Args[0])
	c.Env = append(os.Environ(), "D2_MEMFD_ROLE=forker", "D2_MEMFD_PIDFILE="+pidfile)
	if err := c.Run(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if b, err := os.ReadFile(pidfile); err == nil {
			pid, _ = strconv.Atoi(string(b))
			// wait until reparented to us
			if st, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid)); err == nil {
				f := strings.Fields(string(st[bytes.LastIndexByte(st, ')')+2:]))
				if pp, _ := strconv.Atoi(f[1]); pp == os.Getpid() {
					t.Cleanup(func() { syscall.Kill(pid, syscall.SIGKILL); launch.Reap() })
					return pid
				}
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("sleeper never reparented to the test process")
	return 0
}

func TestFDPassingSiblingHelper(t *testing.T) {
	runtime.LockOSThread()
	if err := launch.BecomeSubreaper(); err != nil {
		t.Fatal(err)
	}
	scope := yamaScope()
	t.Logf("kernel %s, yama ptrace_scope=%d, euid=%d", kernel(), scope, os.Geteuid())
	pid := startSleeper(t)

	own, err := OpenPID(pid) // subreaper opens once
	if err != nil {
		t.Fatalf("subreaper cannot open sleeper mem: %v", err)
	}
	defer own.Close()
	got := make([]byte, len(marker))
	if err := own.ReadAt(got, markerAddr); err != nil || string(got) != marker {
		t.Fatalf("owner read: %q %v", got, err)
	}

	sock := filepath.Join(t.TempDir(), "m.sock")
	srv := &Server{Path: sock, Prov: func(name string) (*os.File, string, error) {
		return own.File(), fmt.Sprintf("pid=%d", pid), nil
	}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go srv.Serve(ctx)
	waitSock(t, sock)

	h := exec.Command(os.Args[0])
	h.Env = append(os.Environ(), "D2_MEMFD_ROLE=helper", "D2_MEMFD_SOCK="+sock)
	h.Stderr = os.Stderr
	out, err := h.Output()
	if err != nil {
		t.Fatalf("helper: %v", err)
	}
	var r struct {
		Got       string `json:"got"`
		Err       string `json:"err"`
		DirectErr string `json:"direct_err"`
		Euid      int    `json:"euid"`
	}
	if err := json.Unmarshal(out, &r); err != nil {
		t.Fatalf("helper output %q: %v", out, err)
	}
	if r.Err != "" || r.Got != marker {
		t.Fatalf("helper via passed fd: got=%q err=%q", r.Got, r.Err)
	}
	t.Logf("sibling helper read marker through the passed fd: OK")

	switch {
	case scope != 1 && scope != 2 && scope != 3:
		t.Skipf("Yama ptrace_scope=%d (not 1+): a direct open by a sibling would also succeed, EACCES check skipped; direct_err=%q", scope, r.DirectErr)
	case r.Euid == 0:
		t.Skipf("helper runs as root (CAP_SYS_PTRACE), direct open not denied; direct_err=%q", r.DirectErr)
	}
	if !strings.Contains(r.DirectErr, "permission denied") {
		t.Fatalf("expected direct open by the sibling to fail with EACCES, got %q", r.DirectErr)
	}
	t.Logf("sibling direct open of the same /proc/%d/mem: %s", pid, r.DirectErr)
}

func waitSock(t *testing.T, p string) {
	t.Helper()
	for i := 0; i < 100; i++ {
		if _, err := os.Stat(p); err == nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("socket never appeared")
}

func TestRemoteReconnect(t *testing.T) {
	f1 := tmpWith(t, "AAAA")
	f2 := tmpWith(t, "BBBB")
	var curMu sync.Mutex
	cur := f1
	setCur := func(f *os.File) { curMu.Lock(); cur = f; curMu.Unlock() }
	sock := filepath.Join(t.TempDir(), "r.sock")
	prov := func(name string) (*os.File, string, error) {
		if name != "i0" {
			return nil, "", errors.New("unknown instance")
		}
		curMu.Lock()
		defer curMu.Unlock()
		return cur, "pid=1", nil
	}
	start := func() (stop func()) {
		srv := &Server{Path: sock, Prov: prov}
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() { srv.Serve(ctx); close(done) }()
		waitSock(t, sock)
		return func() { cancel(); <-done }
	}
	stop := start()
	cl := &Client{Path: sock}
	r := &Remote{C: cl, Name: "i0"}
	defer r.Close()
	b := make([]byte, 4)
	if err := r.ReadAt(b, 0); err != nil || string(b) != "AAAA" {
		t.Fatalf("first read %q %v", b, err)
	}
	if _, _, err := cl.Get("nope"); err == nil || !strings.Contains(err.Error(), "unknown instance") {
		t.Fatalf("want server error, got %v", err)
	}
	// Server restart plus a new game (new descriptor): the Remote must recover on its own.
	stop()
	f1.Truncate(0) // the old game is gone: reads through the old (duplicated) descriptor now fail
	setCur(f2)
	stop = start()
	defer stop()
	if err := r.ReadAt(b, 0); err != nil || string(b) != "BBBB" {
		t.Fatalf("after restart %q %v", b, err)
	}
	// Server down: read fails cleanly, then works again when it is back.
	stop()
	f2.Truncate(0)
	if err := r.ReadAt(b, 0); err == nil {
		t.Fatal("expected error with server down")
	}
	setCur(tmpWith(t, "CCCC"))
	stop = start()
	if err := r.ReadAt(b, 0); err != nil || string(b) != "CCCC" {
		t.Fatalf("after second restart %q %v", b, err)
	}
}

func tmpWith(t *testing.T, s string) *os.File {
	t.Helper()
	f, err := os.Create(filepath.Join(t.TempDir(), "mem"))
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(s)
	return f
}
