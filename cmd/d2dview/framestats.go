package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// FrameStats is what the game video costs and delivers over the last completed window (GET /api/frames, and one log
// line per window while someone watches). Delivered fps is what the sampler produced; partsPerSec counts the parts
// written to all /stream.mjpg clients together.
type FrameStats struct {
	Source      string  `json:"source"` // framefeed (the fork's ring) or screenshot
	WindowSec   float64 `json:"windowSec"`
	Frames      int     `json:"frames"`
	FPS         float64 `json:"fps"`
	PartsPerSec float64 `json:"partsPerSec"`
	GrabMSAvg   float64 `json:"grabMsAvg"` // screenshot: the verb's wall time (d2d encodes the PNG inside it); framefeed: the ring read
	GrabMSMax   float64 `json:"grabMsMax"`
	EncodeMSAvg float64 `json:"encodeMsAvg"` // our decode plus JPEG encode (0 when d2d's PNG passes through)
	BytesAvg    int     `json:"bytesAvg"`
	CPUPct      float64 `json:"cpuPct"` // this process, percent of one core over the window (all of d2dview, the tap included)
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	Quality     int     `json:"quality"`
	Half        bool    `json:"half"`
	Backoff     float64 `json:"backoff"`
	TargetFPS   float64 `json:"targetFps"`
	Every       int     `json:"every"`    // ring: one frame published per this many d2d draws
	SourceHz    float64 `json:"sourceHz"` // ring: d2d's measured draw rate
	FeedWanted  bool    `json:"feedWanted"`
	// Degraded is set when the ring was asked for but the window ran on screenshots or under half the game's 25 fps;
	// the log line says "DEGRADED" so it shows without a client.
	Degraded bool `json:"degraded"`
}

func (s FrameStats) String() string {
	d := ""
	if s.Degraded {
		d = "DEGRADED "
	}
	return d + fmt.Sprintf("frames %s (every %d of %.0f Hz): %.1f fps over %.0fs (%d frames, %.1f parts/s), grab %.1f ms avg %.1f max, encode %.1f ms, %d bytes, %dx%d, d2dview cpu %.1f%%",
		s.Source, s.Every, s.SourceHz, s.FPS, s.WindowSec, s.Frames, s.PartsPerSec, s.GrabMSAvg, s.GrabMSMax, s.EncodeMSAvg, s.BytesAvg, s.Width, s.Height, s.CPUPct)
}

type frameStats struct {
	window time.Duration
	parts  atomic.Int64
	mu     sync.Mutex
	start  time.Time
	cpu0   time.Duration
	n      int
	grab   time.Duration
	grabMx time.Duration
	enc    time.Duration
	bytes  int
	w, h   int
	src    string
	last   FrameStats
	ready  bool
}

func newFrameStats(window time.Duration) *frameStats {
	return &frameStats{window: window, start: time.Now(), cpu0: procCPU()}
}

func (s *frameStats) add(src string, grab, enc time.Duration, bytes, w, h int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.src, s.n, s.grab, s.enc, s.bytes, s.w, s.h = src, s.n+1, s.grab+grab, s.enc+enc, s.bytes+bytes, w, h
	s.grabMx = max(s.grabMx, grab)
}

// roll closes the window when it is due and returns it (ok false otherwise). base fills the knob fields.
func (s *frameStats) roll(now time.Time, base FrameStats) (FrameStats, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	el := now.Sub(s.start)
	if el < s.window {
		return FrameStats{}, false
	}
	cpu := procCPU()
	out := base
	out.Source, out.WindowSec, out.Frames, out.Width, out.Height = s.src, el.Seconds(), s.n, s.w, s.h
	out.FPS = float64(s.n) / el.Seconds()
	out.PartsPerSec = float64(s.parts.Swap(0)) / el.Seconds()
	out.CPUPct = 100 * float64(cpu-s.cpu0) / float64(el)
	if s.n > 0 {
		out.GrabMSAvg = ms(s.grab) / float64(s.n)
		out.EncodeMSAvg = ms(s.enc) / float64(s.n)
		out.BytesAvg = s.bytes / s.n
	}
	out.GrabMSMax = ms(s.grabMx)
	out.Degraded = out.FeedWanted && (out.Source != "framefeed" || out.FPS < 12.5)
	s.last, s.ready = out, true
	s.start, s.cpu0, s.n, s.grab, s.grabMx, s.enc, s.bytes = now, cpu, 0, 0, 0, 0, 0
	return out, true
}

func (s *frameStats) lastWindow() (FrameStats, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.last, s.ready
}

func ms(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }
