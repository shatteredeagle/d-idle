package watch

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"time"
)

// TapOptions describe the d2d view tap (cmd/d2dview) as a frame source for the game video, for the d2d backend where
// there is no X display. The capture reads the tap's own /stream.mjpg and relays every part (the tap samples only while
// a stream client or a /frame poll touches it, so the on-demand start and stop of Video also start and stop the
// sampling). A tap without /stream.mjpg (404) is polled on /frame instead (docs/viewer-fps.md section 7).
type TapOptions struct {
	URL     string // tap base URL, e.g. "http://127.0.0.1:8893"; the paths /stream.mjpg and /frame are added
	Token   string // sent as a Bearer token when the tap has one (d2dview -token-file), never logged
	FPS     int    // cap in frames per second, 0 to MaxTapFPS; 0: the tap's own rate (polling: DefaultTapFPS)
	Quality int    // ffmpeg-style -q:v, 2 (best) to 31, mapped to a JPEG quality for PNG frames; default 5
	Client  *http.Client
	// Start is how long the first frame may take (the tap answers 503 until its sampler took one); default 3 s.
	Start time.Duration
	// Idle ends a stream that sent nothing for this long (a stalled tap; the page reconnects); default 15 s.
	Idle time.Duration
}

// DefaultTapFPS is the poll rate of the /frame fallback; MaxTapFPS caps the option (the tap follows the game, which
// d2d draws at 60 Hz in real time; the tap itself samples at most -fps).
const (
	DefaultTapFPS = 4
	MaxTapFPS     = 30
)

const (
	maxTapBody  = 8 << 20 // a raw PNG screenshot is far below this; protects against a runaway tap
	maxTapFails = 3       // consecutive failures after the stream started before the capture ends (the page reconnects)
)

// TapCapture returns a CaptureFunc polling the tap's /frame. JPEG frames pass through; PNG frames are re-encoded as JPEG
// with the stdlib. Capture start fails when the tap is down, refuses the token or has no frame source (-fps 0), so the
// viewer gets an error instead of a black picture; once running, three failures in a row end the capture and the channel
// closes. A frame identical to the previous one (the tap has not re-sampled yet) is not sent again.
func TapCapture(o TapOptions) (CaptureFunc, error) {
	u, err := url.Parse(o.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("video: tap URL %q is not an http(s) URL", o.URL)
	}
	if o.FPS < 0 || o.FPS > MaxTapFPS {
		return nil, fmt.Errorf("video: tap frame rate must be 0 (the tap's own) to %d", MaxTapFPS)
	}
	if o.Quality < 2 || o.Quality > 31 {
		o.Quality = 5
	}
	if o.Client == nil {
		o.Client = &http.Client{Timeout: 5 * time.Second}
	}
	if o.Start <= 0 {
		o.Start = 3 * time.Second
	}
	if o.Idle <= 0 {
		o.Idle = 15 * time.Second
	}
	u.RawQuery = ""
	u.Path = "/stream.mjpg"
	stream := u.String()
	u.Path = "/frame"
	frame := u.String()
	return func(ctx context.Context) (<-chan []byte, error) {
		ch, err := o.stream(ctx, stream)
		if errors.Is(err, errTapNoStream) {
			return o.start(ctx, frame)
		}
		return ch, err
	}, nil
}

// jpegQuality maps ffmpeg -q:v (2 best .. 31 smallest) to the image/jpeg 1..100 scale the way the import path does.
func (o TapOptions) jpegQuality() int { return 100 - 3*o.Quality }

var (
	errTapNoFrame  = errors.New("tap has no frame yet")
	errTapNoStream = errors.New("tap has no stream")
)

// stream relays the tap's multipart stream: every part as it arrives (latest wins when the reader lags), no polling,
// so the video keeps the tap's cadence. A tap without the route answers errTapNoStream (the caller polls instead).
func (o TapOptions) stream(ctx context.Context, streamURL string) (<-chan []byte, error) {
	sctx, cancel := context.WithCancel(ctx)
	req, err := http.NewRequestWithContext(sctx, http.MethodGet, streamURL, nil)
	if err != nil {
		cancel()
		return nil, err
	}
	if o.Token != "" {
		req.Header.Set("Authorization", "Bearer "+o.Token)
	}
	cl := *o.Client
	cl.Timeout = 0 // a stream has no end; Start and Idle guard it instead
	idle := time.AfterFunc(o.Start, cancel)
	resp, err := cl.Do(req)
	if err != nil {
		idle.Stop()
		cancel()
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, errors.New("video: tap: tap not reachable")
	}
	fail := func(err error) (<-chan []byte, error) {
		idle.Stop()
		resp.Body.Close()
		cancel()
		return nil, err
	}
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return fail(errTapNoStream)
	case http.StatusUnauthorized:
		return fail(errors.New("video: tap: tap refused the token"))
	default:
		return fail(fmt.Errorf("video: tap: tap answered %s", resp.Status))
	}
	_, params, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil || params["boundary"] == "" {
		return fail(errors.New("video: tap: stream is not multipart"))
	}
	parts := multipart.NewReader(resp.Body, params["boundary"])
	next := func() ([]byte, error) {
		p, err := parts.NextPart()
		if err != nil {
			return nil, err
		}
		b, err := io.ReadAll(io.LimitReader(p, maxTapBody+1))
		if err != nil {
			return nil, err
		}
		if len(b) > maxTapBody {
			return nil, errors.New("tap frame too large")
		}
		return o.toJPEG(b)
	}
	first, err := next()
	if err != nil {
		return fail(fmt.Errorf("video: tap: %w", err))
	}
	ch := make(chan []byte, 1)
	ch <- first
	go func() {
		defer close(ch)
		defer cancel()
		defer resp.Body.Close()
		defer idle.Stop()
		var gap time.Duration
		if o.FPS > 0 {
			gap = time.Second / time.Duration(o.FPS) * 9 / 10 // a cap, with room for arrival jitter
		}
		last := time.Now()
		for {
			idle.Reset(o.Idle)
			f, err := next()
			if err != nil {
				return
			}
			if gap > 0 && time.Since(last) < gap {
				continue
			}
			last = time.Now()
			select { // latest wins: a frame not yet taken is replaced
			case <-ch:
			default:
			}
			select {
			case ch <- f:
			case <-sctx.Done():
				return
			}
		}
	}()
	return ch, nil
}

func (o TapOptions) start(ctx context.Context, frameURL string) (<-chan []byte, error) {
	var raw, first []byte
	deadline := time.Now().Add(o.Start)
	for {
		b, err := o.fetch(ctx, frameURL)
		if err == nil {
			if first, err = o.toJPEG(b); err != nil {
				return nil, fmt.Errorf("video: tap: %w", err)
			}
			raw = b
			break
		}
		// Only "no frame yet" is worth waiting for: a refused connection, a bad token or a tap without frames will not heal.
		if !errors.Is(err, errTapNoFrame) || time.Now().After(deadline) {
			return nil, fmt.Errorf("video: tap: %w", err)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(150 * time.Millisecond):
		}
	}
	ch := make(chan []byte, 2)
	ch <- first
	go func() {
		defer close(ch)
		fps := o.FPS
		if fps == 0 {
			fps = DefaultTapFPS
		}
		t := time.NewTicker(time.Second / time.Duration(fps))
		defer t.Stop()
		fails := 0
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
			b, err := o.fetch(ctx, frameURL)
			if err != nil {
				if fails++; fails >= maxTapFails || ctx.Err() != nil {
					return
				}
				continue
			}
			fails = 0
			if bytes.Equal(b, raw) {
				continue
			}
			f, err := o.toJPEG(b)
			if err != nil {
				if fails++; fails >= maxTapFails {
					return
				}
				continue
			}
			raw = b
			select {
			case ch <- f:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch, nil
}

func (o TapOptions) fetch(ctx context.Context, frameURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, frameURL, nil)
	if err != nil {
		return nil, err
	}
	if o.Token != "" {
		req.Header.Set("Authorization", "Bearer "+o.Token)
	}
	resp, err := o.Client.Do(req)
	if err != nil {
		return nil, errors.New("tap not reachable") // the URL may carry a host; the cause is in the tap log
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusServiceUnavailable:
		return nil, errTapNoFrame
	case http.StatusUnauthorized:
		return nil, errors.New("tap refused the token")
	case http.StatusNotFound:
		return nil, errors.New("tap has no frames (started with -fps 0?)")
	default:
		return nil, fmt.Errorf("tap answered %s", resp.Status)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxTapBody+1))
	if err != nil {
		return nil, err
	}
	if len(b) > maxTapBody {
		return nil, errors.New("tap frame too large")
	}
	return b, nil
}

// toJPEG passes JPEG through and converts PNG (the tap's default) with the stdlib.
func (o TapOptions) toJPEG(b []byte) ([]byte, error) {
	if len(b) > 2 && b[0] == 0xFF && b[1] == 0xD8 {
		return b, nil
	}
	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("frame is neither JPEG nor PNG: %w", err)
	}
	var out bytes.Buffer
	if err := jpeg.Encode(&out, img, &jpeg.Options{Quality: o.jpegQuality()}); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
