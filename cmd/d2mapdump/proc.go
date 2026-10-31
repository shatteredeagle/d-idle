package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// procSource is a minimal read-only mem.Source over /proc/<pid>/mem (MEM's Proc replaces it later).
type procSource struct{ f *os.File }

func (p procSource) ReadAt(b []byte, addr uint32) error {
	n, err := p.f.ReadAt(b, int64(addr))
	if err != nil || n != len(b) {
		return fmt.Errorf("read %d at 0x%x: n=%d err=%v", len(b), addr, n, err)
	}
	return nil
}

func findGamePID() (int, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0, err
	}
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		exe, _ := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
		if !strings.Contains(exe, "preloader") {
			continue
		}
		cmd, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
		if err == nil && strings.Contains(strings.ToLower(string(cmd)), "game.exe") {
			return pid, nil
		}
	}
	return 0, fmt.Errorf("Game.exe not running")
}

func openGame() (procSource, error) {
	pid, err := findGamePID()
	if err != nil {
		return procSource{}, err
	}
	f, err := os.Open(fmt.Sprintf("/proc/%d/mem", pid))
	if err != nil {
		return procSource{}, err
	}
	return procSource{f}, nil
}
