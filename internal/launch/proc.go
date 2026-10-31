package launch

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Proc is a snapshot of one /proc/<pid> entry.
type Proc struct {
	PID, PPID int
	// State is the one-letter state from /proc/pid/stat (R, S, D, Z, ...).
	State byte
	// Comm is the kernel thread name, at most 15 bytes. Under Wine it is only a hint (the game shows as
	// "Game.exe" but wineserver as "wineserver"); never match on it alone.
	Comm string
	// Argv is /proc/pid/cmdline split on NUL. Wine rewrites it to the Windows command line, so for the
	// game Argv[0] is `C:\Program Files (x86)\Diablo II\Game.exe`.
	Argv []string
	// Exe is the readlink of /proc/pid/exe: wine-preloader for every Windows process (not useful to tell
	// them apart), "" if unreadable.
	Exe string
}

// ListProcs reads every numeric entry of /proc. Processes that vanish mid-scan are skipped.
func ListProcs() ([]Proc, error) { return listProcs("/proc") }

func listProcs(root string) ([]Proc, error) {
	ents, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var out []Proc
	for _, e := range ents {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		p, err := readProc(root, pid)
		if err != nil {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

func readProc(root string, pid int) (Proc, error) {
	dir := filepath.Join(root, strconv.Itoa(pid))
	stat, err := os.ReadFile(filepath.Join(dir, "stat"))
	if err != nil {
		return Proc{}, err
	}
	// "pid (comm) S ppid ..." ; comm may contain spaces and parentheses, so cut at the LAST ')'.
	i, j := bytes.IndexByte(stat, '('), bytes.LastIndexByte(stat, ')')
	if i < 0 || j < i {
		return Proc{}, fmt.Errorf("bad stat for %d", pid)
	}
	rest := strings.Fields(string(stat[j+1:]))
	if len(rest) < 2 {
		return Proc{}, fmt.Errorf("short stat for %d", pid)
	}
	ppid, _ := strconv.Atoi(rest[1])
	p := Proc{PID: pid, PPID: ppid, State: rest[0][0], Comm: string(stat[i+1 : j])}
	if cl, err := os.ReadFile(filepath.Join(dir, "cmdline")); err == nil {
		cl = bytes.TrimRight(cl, "\x00 ")
		if len(cl) > 0 {
			p.Argv = strings.Split(string(cl), "\x00")
		}
	}
	p.Exe, _ = os.Readlink(filepath.Join(dir, "exe"))
	return p, nil
}

// ImageName is the lower-cased base name of the Windows image of a Wine process: the last element of
// Argv[0] split on both '\' and '/'. "" for non-Wine processes without argv.
func (p Proc) ImageName() string {
	if len(p.Argv) == 0 {
		return ""
	}
	a := p.Argv[0]
	// Wine rewrites the whole command line into argv[0] with spaces; cut at ".exe" if present.
	if k := strings.Index(strings.ToLower(a), ".exe"); k >= 0 {
		a = a[:k+4]
	}
	if k := strings.LastIndexAny(a, `\/`); k >= 0 {
		a = a[k+1:]
	}
	return strings.ToLower(a)
}

// FindImage returns the processes whose Windows image name equals name (case-insensitive), lowest PID
// first. This is how to find the game: FindImage(procs, "game.exe"). The same name can appear twice
// (Wine launches a process and re-execs it); callers wanting the real one should use GameProc.
func FindImage(procs []Proc, name string) []Proc {
	name = strings.ToLower(name)
	var out []Proc
	for _, p := range procs {
		if p.ImageName() == name {
			out = append(out, p)
		}
	}
	return out
}

// IsDescendant reports whether pid has ancestor somewhere above it in the real-parent chain, which is
// exactly the relation Yama's ptrace_scope=1 checks.
func IsDescendant(procs []Proc, pid, ancestor int) bool {
	by := make(map[int]int, len(procs))
	for _, p := range procs {
		by[p.PID] = p.PPID
	}
	for n := 0; n < 4096; n++ {
		pp, ok := by[pid]
		if !ok || pp <= 0 {
			return false
		}
		if pp == ancestor {
			return true
		}
		pid = pp
	}
	return false
}

// OpenMem opens /proc/<pid>/mem read-only. Reads use pread at the target's virtual address. With Yama
// ptrace_scope=1 this succeeds for descendants of the calling process (see BecomeSubreaper) and fails
// with EPERM otherwise.
func OpenMem(pid int) (*os.File, error) {
	return os.Open("/proc/" + strconv.Itoa(pid) + "/mem")
}

// MemReader adapts an open /proc/<pid>/mem to the ReadAt(p, addr uint32) shape used by internal/mem.
type MemReader struct{ F *os.File }

// ReadAt fills p from the 32-bit game address addr (all-or-nothing).
func (m MemReader) ReadAt(p []byte, addr uint32) error {
	n, err := m.F.ReadAt(p, int64(addr))
	if n == len(p) {
		return nil
	}
	if err == nil {
		err = fmt.Errorf("short read")
	}
	return fmt.Errorf("read %d bytes at 0x%x: %w", len(p), addr, err)
}

// ImageBase returns the lowest mapping address of the file-backed image whose path ends in
// imageSuffix (e.g. "Game.exe"), from /proc/<pid>/maps. VERIFIED on the live game: Game.exe is a
// file-backed mapping starting at 0x400000.
func ImageBase(pid int, imageSuffix string) (uint64, error) {
	f, err := os.Open("/proc/" + strconv.Itoa(pid) + "/maps")
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return imageBaseFrom(f, imageSuffix)
}

func imageBaseFrom(r interface{ Read([]byte) (int, error) }, suffix string) (uint64, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	best := uint64(0)
	found := false
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) < 6 || !strings.HasSuffix(strings.ToLower(f[len(f)-1]), strings.ToLower(suffix)) {
			continue
		}
		lo, _, _ := strings.Cut(f[0], "-")
		a, err := strconv.ParseUint(lo, 16, 64)
		if err != nil {
			continue
		}
		if !found || a < best {
			best, found = a, true
		}
	}
	if !found {
		return 0, fmt.Errorf("no mapping ending in %q", suffix)
	}
	return best, sc.Err()
}
