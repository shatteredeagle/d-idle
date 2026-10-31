package input

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"d2lod-bot/internal/mem"
)

// ProcSource is a read-only mem.Source over /proc/<pid>/mem (needs CAP_SYS_PTRACE). It exists for
// the calibration tool and live tests; production code uses internal/mem's own Proc source.
type ProcSource struct{ f *os.File }

func OpenProc(pid int) (*ProcSource, error) {
	f, err := os.Open(fmt.Sprintf("/proc/%d/mem", pid))
	if err != nil {
		return nil, err
	}
	return &ProcSource{f}, nil
}

func (p *ProcSource) Close() error { return p.f.Close() }

func (p *ProcSource) ReadAt(b []byte, addr uint32) error {
	n, err := p.f.ReadAt(b, int64(addr))
	if n == len(b) {
		return nil
	}
	if err == nil {
		err = fmt.Errorf("short read")
	}
	return fmt.Errorf("read %d bytes at 0x%x: %w", len(b), addr, err)
}

// FindGamePID returns the Wine process running Game.exe.
func FindGamePID() (int, error) {
	ents, err := os.ReadDir("/proc")
	if err != nil {
		return 0, err
	}
	for _, e := range ents {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		exe, _ := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
		if !mem.IsWineExe(exe) {
			continue
		}
		cmd, _ := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
		if strings.Contains(strings.ToLower(string(cmd)), "game.exe") {
			return pid, nil
		}
	}
	return 0, fmt.Errorf("no Game.exe process")
}
