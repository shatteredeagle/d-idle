package mem

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

// Errors returned by Source implementations. Use errors.Is.
var (
	ErrNoGame   = errors.New("mem: game process not found")
	ErrUnmapped = errors.New("mem: address not readable")
	ErrShort    = errors.New("mem: short read")
	ErrBuild    = errors.New("mem: Game.exe build mismatch")
)

// ReadError describes a failed read; it wraps ErrUnmapped, ErrShort or the OS error.
type ReadError struct {
	Addr uint32
	Len  int
	Err  error
}

func (e *ReadError) Error() string {
	return fmt.Sprintf("mem: read %d bytes at 0x%x: %v", e.Len, e.Addr, e.Err)
}
func (e *ReadError) Unwrap() error { return e.Err }

// Build fingerprint of Game.exe 1.14.3.71 (1.14d). VERIFIED LIVE (PE header read on the worker).
const (
	ExpectSizeOfImage uint32 = 0x5ba000
	ExpectTimestamp   uint32 = 0x574ddfbc
)

// Proc reads /proc/<pid>/mem of the Wine process running Game.exe. It never writes or attaches
// (plain pread); it needs CAP_SYS_PTRACE, or the game must be a child/descendant of the reader.
type Proc struct {
	PID int
	f   *os.File
	w   *os.File // opened lazily by WriteAt (cheat tooling only)
}

// InstanceEnv is the environment variable that marks which bot instance a game process belongs to
// (set by internal/launch when it starts a game; children inherit it). Empty/unset means "any".
const InstanceEnv = "D2BOT_INSTANCE"

// DefaultInstance is the instance of the original game, started before markers existed: it matches
// processes with no marker as well as an explicit "i0".
const DefaultInstance = "i0"

// FindGames returns the PIDs of Wine processes running Game.exe, filtered to the instance named by
// $D2BOT_INSTANCE when that is set (all games otherwise). See FindGamesFor.
func FindGames() []int { return FindGamesFor(os.Getenv(InstanceEnv)) }

// ProcInstance returns the D2BOT_INSTANCE marker of a process ("" when it has none or the environment
// is unreadable). The environment is readable for processes of the same user.
func ProcInstance(pid int) string {
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/environ", pid))
	if err != nil {
		return ""
	}
	for _, kv := range strings.Split(string(b), "\x00") {
		if v, ok := strings.CutPrefix(kv, InstanceEnv+"="); ok {
			return v
		}
	}
	return ""
}

// IsWineExe reports whether a /proc/pid/exe link is a Wine Windows-process host: the wine-preloader of
// Proton and older Wine builds, or plain "wine" (Debian trixie wine 10, where Game.exe's exe link is
// /usr/lib/wine/wine; VERIFIED LIVE in the container first run, docs/container.md). wineserver does not match.
func IsWineExe(exe string) bool {
	if strings.Contains(exe, "preloader") {
		return true
	}
	switch filepath.Base(exe) {
	case "wine", "wine64", "wine32":
		return true
	}
	return false
}

func instanceMatches(pid int, instance string) bool {
	if instance == "" {
		return true
	}
	got := ProcInstance(pid)
	return got == instance || (instance == DefaultInstance && got == "")
}

// FindGamesFor returns the PIDs of Wine processes running Game.exe that belong to instance ("" = any;
// "i0" also matches unmarked games): the exe link contains "preloader" and the cmdline contains
// Game.exe (the comm is "Main"; not required). PIDs are in ascending order.
func FindGamesFor(instance string) []int {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	var out []int
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		exe, _ := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
		if !IsWineExe(exe) {
			continue
		}
		cmd, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
		if err != nil {
			continue
		}
		if strings.Contains(strings.ToLower(string(cmd)), "game.exe") && instanceMatches(pid, instance) {
			out = append(out, pid)
		}
	}
	sort.Ints(out)
	return out
}

// FindGame returns the first game PID (of $D2BOT_INSTANCE when set), or ErrNoGame.
func FindGame() (int, error) { return FindGameFor(os.Getenv(InstanceEnv)) }

// FindGameFor returns the first game PID of the instance, or ErrNoGame.
func FindGameFor(instance string) (int, error) {
	pids := FindGamesFor(instance)
	if len(pids) == 0 {
		return 0, ErrNoGame
	}
	return pids[0], nil
}

// Open opens /proc/<pid>/mem read-only.
func Open(pid int) (*Proc, error) {
	f, err := os.Open(fmt.Sprintf("/proc/%d/mem", pid))
	if err != nil {
		return nil, fmt.Errorf("mem: open process %d: %w", pid, err)
	}
	return &Proc{PID: pid, f: f}, nil
}

// OpenGame is FindGame + Open + VerifyBuild.
func OpenGame() (*Proc, error) { return OpenInstance(os.Getenv(InstanceEnv)) }

// OpenInstance is OpenGame for one named instance ("" = any, like FindGame without a marker).
func OpenInstance(instance string) (*Proc, error) {
	pid, err := FindGameFor(instance)
	if err != nil {
		return nil, err
	}
	p, err := Open(pid)
	if err != nil {
		return nil, err
	}
	if err := VerifyBuild(p); err != nil {
		p.Close()
		return nil, err
	}
	return p, nil
}

func (p *Proc) Close() error {
	if p.w != nil {
		p.w.Close()
	}
	return p.f.Close()
}

// WriteAt writes into the game's memory (/proc/pid/mem, needs the same ptrace right as reading). The
// read-only bot never calls it; only internal/cheat does, and always through its ledger.
func (p *Proc) WriteAt(b []byte, addr uint32) error {
	if len(b) == 0 {
		return nil
	}
	if p.w == nil {
		w, err := os.OpenFile(fmt.Sprintf("/proc/%d/mem", p.PID), os.O_RDWR, 0)
		if err != nil {
			return fmt.Errorf("mem: open process %d for writing: %w", p.PID, err)
		}
		p.w = w
	}
	n, err := p.w.WriteAt(b, int64(addr))
	if n == len(b) {
		return nil
	}
	if err == nil {
		err = ErrShort
	}
	return &ReadError{Addr: addr, Len: len(b), Err: err}
}

// Alive reports whether the process still exists.
func (p *Proc) Alive() bool {
	_, err := os.Stat(fmt.Sprintf("/proc/%d", p.PID))
	return err == nil
}

// ReadAt is all-or-nothing: any failure returns a *ReadError and p content is unspecified.
func (p *Proc) ReadAt(b []byte, addr uint32) error {
	if len(b) == 0 {
		return nil
	}
	n, err := p.f.ReadAt(b, int64(addr))
	if n == len(b) {
		return nil
	}
	switch {
	case err == nil || errors.Is(err, io.EOF):
		err = ErrShort
	case errors.Is(err, syscall.EIO), errors.Is(err, syscall.EFAULT), errors.Is(err, syscall.EINVAL):
		err = ErrUnmapped
	}
	return &ReadError{Addr: addr, Len: len(b), Err: err}
}

// VerifyBuild checks the PE32 header at Base against the 1.14d fingerprint.
func VerifyBuild(s Source) error {
	hdr := make([]byte, 0x400)
	if err := s.ReadAt(hdr, Base); err != nil {
		return fmt.Errorf("%w: %v", ErrBuild, err)
	}
	if hdr[0] != 'M' || hdr[1] != 'Z' {
		return fmt.Errorf("%w: no MZ header at 0x%x", ErrBuild, Base)
	}
	pe := int(binary.LittleEndian.Uint32(hdr[0x3c:]))
	if pe < 0x40 || pe+0x60 > len(hdr) || string(hdr[pe:pe+4]) != "PE\x00\x00" {
		return fmt.Errorf("%w: bad PE header", ErrBuild)
	}
	if binary.LittleEndian.Uint16(hdr[pe+24:]) != 0x10B {
		return fmt.Errorf("%w: not PE32", ErrBuild)
	}
	stamp := binary.LittleEndian.Uint32(hdr[pe+8:])
	imageBase := binary.LittleEndian.Uint32(hdr[pe+24+28:])
	size := binary.LittleEndian.Uint32(hdr[pe+24+56:])
	if imageBase != Base || size != ExpectSizeOfImage || stamp != ExpectTimestamp {
		return fmt.Errorf("%w: imageBase=0x%x size=0x%x stamp=0x%x", ErrBuild, imageBase, size, stamp)
	}
	return nil
}
