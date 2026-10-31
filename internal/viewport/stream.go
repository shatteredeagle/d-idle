package viewport

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Config configures a Stream.
type Config struct {
	Capturer Capturer
	// Target returns what to capture right now; called before every session open and every pull capture, so a
	// restarted game (new window) is picked up. ErrNoTarget (or any error) fails that attempt.
	Target func() (Spec, error)
	FPS    float64 // frame-rate cap; default 4
	MaxW   int     // size limit (0 native)
	MaxH   int
	// Quality is the JPEG quality (default 65).
	Quality int
	// Limiter is the global capture concurrency cap; nil: none.
	Limiter *Limiter
	// Idle is how long a capture continues after the last frame request when nobody streams; default 3 s.
	Idle time.Duration
	// OnError is told about every failed attempt (d2watch uses it to rediscover the window).
	OnError func(error)
}

// Stream is one shared capture. Viewers (Frame callers and subscribers) never capture themselves: they ask
// for frames and a single loop produces them, only while somebody watches.
type Stream struct {
	cfg    Config
	period time.Duration

	mu      sync.Mutex
	running bool
	closed  bool
	wantAt  time.Time
	viewers int
	frame   Frame
	have    bool
	err     error
	errSeq  uint64
	notify  chan struct{} // closed and replaced on every new frame or error
	stop    context.CancelFunc
}

func New(cfg Config) *Stream {
	if cfg.FPS <= 0 {
		cfg.FPS = 4
	}
	if cfg.Idle <= 0 {
		cfg.Idle = 3 * time.Second
	}
	return &Stream{cfg: cfg, period: time.Duration(float64(time.Second) / cfg.FPS), notify: make(chan struct{})}
}

// Close stops the capture and makes later requests fail.
func (s *Stream) Close() {
	s.mu.Lock()
	s.closed = true
	stop := s.stop
	close(s.notify) // wake Next/Frame waiters so they see closed
	s.notify = make(chan struct{})
	s.mu.Unlock()
	if stop != nil {
		stop()
	}
}

// Closed reports whether Close was called; Next and Frame then fail at once, so a push loop must stop.
func (s *Stream) Closed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

// Status is a snapshot for APIs.
type Status struct {
	Running bool
	Viewers int
	FrameAt time.Time // zero: no frame yet
	Seq     uint64
	Err     string
}

func (s *Stream) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := Status{Running: s.running, Viewers: s.viewers}
	if s.have {
		st.FrameAt, st.Seq = s.frame.At, s.frame.Seq
	}
	if s.err != nil {
		st.Err = s.err.Error()
	}
	return st
}

// Latest returns the cached frame without asking for a capture.
func (s *Stream) Latest() (Frame, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.frame, s.have
}

// Frame returns a frame no older than maxAge, starting the capture if needed and waiting for it (at most
// until ctx is done). A capture error that happens while waiting is returned.
func (s *Stream) Frame(ctx context.Context, maxAge time.Duration) (Frame, error) {
	s.touch()
	s.mu.Lock()
	errSeq := s.errSeq
	for {
		if s.closed {
			s.mu.Unlock()
			return Frame{}, errors.New("viewport: closed")
		}
		if s.have && time.Since(s.frame.At) <= maxAge {
			f := s.frame
			s.mu.Unlock()
			return f, nil
		}
		if s.errSeq != errSeq {
			err := s.err
			s.mu.Unlock()
			return Frame{}, err
		}
		ch := s.notify
		s.mu.Unlock()
		select {
		case <-ch:
		case <-ctx.Done():
			return Frame{}, ctx.Err()
		}
		s.mu.Lock()
	}
}

// Watch registers a viewer (capture runs while at least one watches) and returns a function that removes it.
func (s *Stream) Watch() func() {
	s.mu.Lock()
	s.viewers++
	s.mu.Unlock()
	s.touch()
	var once sync.Once
	return func() {
		once.Do(func() {
			s.mu.Lock()
			s.viewers--
			s.wantAt = time.Now()
			s.mu.Unlock()
		})
	}
}

// Next waits for a frame newer than seq (a new frame or an error); the basis of a push stream.
func (s *Stream) Next(ctx context.Context, seq uint64) (Frame, error) {
	s.touch()
	s.mu.Lock()
	errSeq := s.errSeq
	for {
		if s.closed {
			s.mu.Unlock()
			return Frame{}, errors.New("viewport: closed")
		}
		if s.have && s.frame.Seq > seq {
			f := s.frame
			s.mu.Unlock()
			return f, nil
		}
		if s.errSeq != errSeq {
			err := s.err
			s.mu.Unlock()
			return Frame{}, err
		}
		ch := s.notify
		s.mu.Unlock()
		select {
		case <-ch:
		case <-ctx.Done():
			return Frame{}, ctx.Err()
		}
		s.mu.Lock()
	}
}

func (s *Stream) touch() {
	s.mu.Lock()
	s.wantAt = time.Now()
	if s.running || s.closed {
		s.mu.Unlock()
		return
	}
	s.running = true
	ctx, cancel := context.WithCancel(context.Background())
	s.stop = cancel
	s.mu.Unlock()
	go s.loop(ctx)
}

func (s *Stream) wanted() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.closed && (s.viewers > 0 || time.Since(s.wantAt) < s.cfg.Idle)
}

func (s *Stream) publish(f Frame, err error) {
	s.mu.Lock()
	if err != nil {
		s.err = err
		s.errSeq++
	} else {
		f.Seq = s.frame.Seq + 1
		s.frame, s.have, s.err = f, true, nil
	}
	close(s.notify)
	s.notify = make(chan struct{})
	s.mu.Unlock()
	if err != nil && s.cfg.OnError != nil {
		s.cfg.OnError(err)
	}
}

func (s *Stream) spec() (Spec, error) {
	sp, err := s.cfg.Target()
	if err != nil {
		return sp, err
	}
	if sp.FPS <= 0 {
		sp.FPS = s.cfg.FPS
	}
	if sp.MaxW == 0 {
		sp.MaxW = s.cfg.MaxW
	}
	if sp.MaxH == 0 {
		sp.MaxH = s.cfg.MaxH
	}
	if sp.Quality == 0 {
		sp.Quality = s.cfg.Quality
	}
	return sp, nil
}

// loop is the only capturer of the stream: it runs while wanted, with one capture in flight, and exits (closing
// the persistent process) when nobody has watched for Idle.
func (s *Stream) loop(ctx context.Context) {
	var sess Session
	var held bool // a persistent session holds a limiter slot
	closeSess := func() {
		if sess != nil {
			sess.Close()
			sess = nil
		}
		if held {
			s.cfg.Limiter.Release()
			held = false
		}
	}
	defer func() {
		closeSess()
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		// a request that arrived while exiting would otherwise wait forever
		if s.wanted() {
			s.touch()
		}
	}()
	persistent := s.cfg.Capturer.Persistent()
	for ctx.Err() == nil && s.wanted() {
		start := time.Now()
		f, err := s.capture(ctx, &sess, &held, persistent)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			closeSess()
			s.publish(Frame{}, err)
			if !sleep(ctx, max(s.period, 500*time.Millisecond)) {
				return
			}
			continue
		}
		s.publish(f, nil)
		if !persistent {
			if !sleep(ctx, s.period-time.Since(start)) {
				return
			}
		}
	}
}

func (s *Stream) capture(ctx context.Context, sess *Session, held *bool, persistent bool) (Frame, error) {
	if persistent {
		if *sess == nil {
			sp, err := s.spec()
			if err != nil {
				return Frame{}, err
			}
			if err := s.cfg.Limiter.Acquire(ctx); err != nil {
				return Frame{}, err
			}
			*held = true
			ns, err := s.cfg.Capturer.Open(ctx, sp)
			if err != nil {
				return Frame{}, err
			}
			*sess = ns
		}
		return (*sess).Next(ctx)
	}
	sp, err := s.spec()
	if err != nil {
		return Frame{}, err
	}
	if err := s.cfg.Limiter.Acquire(ctx); err != nil {
		return Frame{}, err
	}
	defer s.cfg.Limiter.Release()
	ns, err := s.cfg.Capturer.Open(ctx, sp)
	if err != nil {
		return Frame{}, err
	}
	defer ns.Close()
	return ns.Next(ctx)
}

func sleep(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() == nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-ctx.Done():
		return false
	}
}
