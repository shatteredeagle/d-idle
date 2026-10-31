// Package memfd passes an already-open /proc/<pid>/mem descriptor between processes.
//
// Why: under Yama ptrace_scope=1 only the subreaper (or an ancestor of the game) may open
// /proc/<pid>/mem (docs/launch.md section 1). A separate process (sentinel, watcher) therefore cannot open
// it itself. The permission check happens at open(2), not at read(2), so the owner opens it once and sends
// the descriptor over a unix socket with SCM_RIGHTS; the receiver reads through it with no capability.
// Result and evidence: docs/memfd.md (VERIFIED by the tests in this package, Linux only).
//
// Protocol (unixpacket socket, one message per request/reply):
//
//	client -> "GET <name>"
//	server -> "OK <info>" + SCM_RIGHTS(fd)   or   "ERR <message>"
package memfd

import (
	"errors"
	"fmt"
	"os"
	"sync"
)

// Source implements mem.Source over a descriptor opened on /proc/<pid>/mem (or any file where the file
// offset is the game's virtual address).
type Source struct {
	mu sync.Mutex
	f  *os.File
}

// FromFile wraps f. The Source owns f and closes it in Close.
func FromFile(f *os.File) *Source { return &Source{f: f} }

// OpenPID opens /proc/<pid>/mem read-only. Only works for the subreaper or an ancestor under Yama scope 1.
func OpenPID(pid int) (*Source, error) {
	f, err := os.Open(fmt.Sprintf("/proc/%d/mem", pid))
	if err != nil {
		return nil, err
	}
	return FromFile(f), nil
}

// File returns the underlying descriptor (for the Server to send). Do not close it.
func (s *Source) File() *os.File {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.f
}

// ReadAt fills p from addr, all or nothing (mem.Source).
func (s *Source) ReadAt(p []byte, addr uint32) error {
	f := s.File()
	if f == nil {
		return errors.New("memfd: closed")
	}
	n, err := f.ReadAt(p, int64(addr))
	if n == len(p) {
		return nil
	}
	if err == nil {
		err = errors.New("short read")
	}
	return fmt.Errorf("memfd: read %d bytes at 0x%x: %w", len(p), addr, err)
}

// Close closes the descriptor; later reads fail.
func (s *Source) Close() error {
	s.mu.Lock()
	f := s.f
	s.f = nil
	s.mu.Unlock()
	if f == nil {
		return nil
	}
	return f.Close()
}
