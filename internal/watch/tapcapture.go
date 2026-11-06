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
	"net/http"
	"net/url"
	"time"
)

// TapOptions describe the d2d view tap (cmd/d2dview) as a frame source for the game video, for the d2d backend where
// there is no X display. The tap samples screenshots only while somebody asks for /frame (VERIFIED by reading
// cmd/d2dview/main.go and frames.go: every /frame request Touches the sampler, which stays awake for 5 s after the last
// touch), so polling /frame is what keeps it awake and the on-demand start/stop of Video also starts/stops the sampling.
type TapOptions struct {
	URL     string // tap base URL, e.g. "http://127.0.0.1:8893"; the path /frame is added
	Token   string // sent as a Bearer token when the tap has one (d2dview -token-file), never logged
	FPS     int    // polls per second, 1 to MaxTapFPS; default DefaultTapFPS
	Quality int    // ffmpeg-style -q:v, 2 (best) to 31, mapped to a JPEG quality; default 5 (same flag as the x11 path)
	Client  *http.Client
	// Start is how long the first frame may take (the tap answers 503 until its sampler took one); default 3 s.
	Start time.Duration
}

// DefaultTapFPS and MaxTapFPS: the tap re-samples at most about 6 per second (d2dview -fps) and backs off to a quarter
// of the wall time, so polling faster only repeats frames.
const (
	DefaultTapFPS = 4
	MaxTapFPS     = 6
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
	if o.FPS == 0 {
		o.FPS = DefaultTapFPS
	}
	if o.FPS < 1 || o.FPS > MaxTapFPS {
		return nil, fmt.Errorf("video: tap frame rate must be 1 to %d", MaxTapFPS)
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
	u.Path = "/frame"
	u.RawQuery = ""
	return func(ctx context.Context) (<-chan []byte, error) { return o.start(ctx, u.String()) }, nil
}

// jpegQuality maps ffmpeg -q:v (2 best .. 31 smallest) to the image/jpeg 1..100 scale the way the import path does.
func (o TapOptions) jpegQuality() int { return 100 - 3*o.Quality }

var errTapNoFrame = errors.New("tap has no frame yet")

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
		t := time.NewTicker(time.Second / time.Duration(o.FPS))
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
