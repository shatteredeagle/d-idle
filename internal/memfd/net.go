package memfd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Provider returns the current descriptor for name (for example an instance id) and a short info string
// (for example "pid=123 base=0x400000"). The Server only sends the descriptor; it keeps ownership.
type Provider func(name string) (*os.File, string, error)

// Server hands descriptors to clients that connect to a unix socket.
type Server struct {
	Path string // socket path; created mode 0600, replaced if stale
	Prov Provider

	wg sync.WaitGroup
}

// Serve listens and answers until ctx is done. It returns nil on a clean stop.
func (s *Server) Serve(ctx context.Context) error {
	_ = os.Remove(s.Path)
	ln, err := net.ListenUnix("unixpacket", &net.UnixAddr{Name: s.Path, Net: "unixpacket"})
	if err != nil {
		return err
	}
	_ = os.Chmod(s.Path, 0o600)
	conns := map[*net.UnixConn]bool{}
	var cmu sync.Mutex
	go func() {
		<-ctx.Done()
		ln.Close()
		cmu.Lock()
		for c := range conns {
			c.Close()
		}
		cmu.Unlock()
	}()
	defer os.Remove(s.Path)
	defer s.wg.Wait()
	for {
		c, err := ln.AcceptUnix()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		cmu.Lock()
		conns[c] = true
		cmu.Unlock()
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.handle(c)
			cmu.Lock()
			delete(conns, c)
			cmu.Unlock()
		}()
	}
}

func (s *Server) handle(c *net.UnixConn) {
	defer c.Close()
	buf := make([]byte, 512)
	for {
		n, err := c.Read(buf)
		if err != nil {
			return
		}
		req := strings.TrimSpace(string(buf[:n]))
		name, ok := strings.CutPrefix(req, "GET ")
		if !ok {
			_, _ = c.Write([]byte("ERR bad request"))
			continue
		}
		f, info, err := s.Prov(name)
		if err != nil || f == nil {
			if err == nil {
				err = errors.New("no descriptor")
			}
			_, _ = c.Write([]byte("ERR " + err.Error()))
			continue
		}
		// SyscallConn keeps the fd valid during the send and, unlike Fd(), leaves the file's mode alone.
		rc, err := f.SyscallConn()
		if err != nil {
			_, _ = c.Write([]byte("ERR " + err.Error()))
			continue
		}
		var werr error
		if err := rc.Control(func(fd uintptr) {
			_, _, werr = c.WriteMsgUnix([]byte("OK "+info), syscall.UnixRights(int(fd)), nil)
		}); err != nil || werr != nil {
			return
		}
	}
}

// Client fetches descriptors from a Server and reconnects when the server restarts.
type Client struct {
	Path    string
	Timeout time.Duration // per request, default 2s

	mu sync.Mutex
	c  *net.UnixConn
}

func (c *Client) timeout() time.Duration {
	if c.Timeout > 0 {
		return c.Timeout
	}
	return 2 * time.Second
}

// Close drops the connection (a later Get redials).
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.c == nil {
		return nil
	}
	err := c.c.Close()
	c.c = nil
	return err
}

// Get asks for the descriptor of name. A broken connection is redialed once.
func (c *Client) Get(name string) (*Source, string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var last error
	for attempt := 0; attempt < 2; attempt++ {
		if c.c == nil {
			conn, err := net.DialUnix("unixpacket", nil, &net.UnixAddr{Name: c.Path, Net: "unixpacket"})
			if err != nil {
				return nil, "", err // server not there: caller decides when to retry
			}
			c.c = conn
		}
		s, info, err := c.roundTrip(name)
		if err == nil {
			return s, info, nil
		}
		var re remoteError
		if errors.As(err, &re) {
			return nil, "", err
		}
		last = err
		c.c.Close()
		c.c = nil
	}
	return nil, "", last
}

type remoteError string

func (e remoteError) Error() string { return "memfd server: " + string(e) }

func (c *Client) roundTrip(name string) (*Source, string, error) {
	_ = c.c.SetDeadline(time.Now().Add(c.timeout()))
	if _, err := c.c.Write([]byte("GET " + name)); err != nil {
		return nil, "", err
	}
	buf := make([]byte, 512)
	oob := make([]byte, syscall.CmsgSpace(4*4))
	n, oobn, _, _, err := c.c.ReadMsgUnix(buf, oob)
	if err != nil {
		return nil, "", err
	}
	msg := string(buf[:n])
	if e, ok := strings.CutPrefix(msg, "ERR "); ok {
		return nil, "", remoteError(e)
	}
	info, ok := strings.CutPrefix(msg, "OK ")
	if !ok {
		return nil, "", fmt.Errorf("memfd: bad reply %q", msg)
	}
	msgs, err := syscall.ParseSocketControlMessage(oob[:oobn])
	if err != nil {
		return nil, "", err
	}
	for _, m := range msgs {
		fds, err := syscall.ParseUnixRights(&m)
		if err != nil || len(fds) == 0 {
			continue
		}
		for _, extra := range fds[1:] {
			syscall.Close(extra)
		}
		syscall.CloseOnExec(fds[0])
		return FromFile(os.NewFile(uintptr(fds[0]), "memfd:"+name)), info, nil
	}
	return nil, "", errors.New("memfd: reply carried no descriptor")
}

// Remote is a mem.Source that survives a game restart: when a read fails it asks the server for a fresh
// descriptor (the game has a new pid after a restart) and retries once.
type Remote struct {
	C    *Client
	Name string

	mu  sync.Mutex
	src *Source
}

// ReadAt implements mem.Source.
func (r *Remote) ReadAt(p []byte, addr uint32) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.src != nil {
		if err := r.src.ReadAt(p, addr); err == nil {
			return nil
		}
		r.src.Close()
		r.src = nil
	}
	s, _, err := r.C.Get(r.Name)
	if err != nil {
		return err
	}
	r.src = s
	return s.ReadAt(p, addr)
}

// Close releases the current descriptor.
func (r *Remote) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.src == nil {
		return nil
	}
	err := r.src.Close()
	r.src = nil
	return err
}
