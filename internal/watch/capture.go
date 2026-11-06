package watch

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
	"strconv"
	"time"
)

// CaptureOptions describe how to grab the instance's private X display.
type CaptureOptions struct {
	Display string // ":105", from the instance mapping, never the desktop
	FPS     int    // default 8
	Quality int    // ffmpeg -q:v, 2 (best) to 31; default 5
	// LookPath and Command are seams for tests; they default to exec.LookPath and exec.CommandContext.
	LookPath func(string) (string, error)
	Command  func(ctx context.Context, name string, arg ...string) *exec.Cmd
}

// DefaultVideoFPS is the frame rate when -watch-video-fps is not given.
const DefaultVideoFPS = 8

// MaxVideoFPS caps the rate: the capture is shared by all viewers and costs CPU on the worker host.
const MaxVideoFPS = 15

// ExecCapture picks a capture tool at start time: ffmpeg x11grab (piped mjpeg) when present, else ImageMagick import
// polled at a lower rate. It returns an error naming what to install when neither exists.
func ExecCapture(o CaptureOptions) (CaptureFunc, error) {
	if o.Display == "" {
		return nil, errors.New("video: no display for this instance")
	}
	if o.FPS <= 0 {
		o.FPS = DefaultVideoFPS
	}
	if o.FPS > MaxVideoFPS {
		o.FPS = MaxVideoFPS
	}
	if o.Quality < 2 || o.Quality > 31 {
		o.Quality = 5
	}
	if o.LookPath == nil {
		o.LookPath = exec.LookPath
	}
	if o.Command == nil {
		o.Command = exec.CommandContext
	}
	if _, err := o.LookPath("ffmpeg"); err == nil {
		return func(ctx context.Context) (<-chan []byte, error) { return o.ffmpeg(ctx) }, nil
	}
	if _, err := o.LookPath("import"); err == nil {
		return func(ctx context.Context) (<-chan []byte, error) { return o.poll(ctx), nil }, nil
	}
	return nil, errors.New("video: neither ffmpeg nor ImageMagick (import) is installed on this host")
}

// ffmpegArgs is the command line of the shared capture (VERIFIED LIVE 2026-11-05 on the worker host against :105).
func (o CaptureOptions) ffmpegArgs() []string {
	return []string{"-hide_banner", "-loglevel", "error", "-nostdin",
		"-f", "x11grab", "-draw_mouse", "0", "-framerate", strconv.Itoa(o.FPS), "-i", o.Display,
		"-c:v", "mjpeg", "-q:v", strconv.Itoa(o.Quality), "-f", "image2pipe", "-"}
}

func (o CaptureOptions) ffmpeg(ctx context.Context) (<-chan []byte, error) {
	cmd := o.Command(ctx, "ffmpeg", o.ffmpegArgs()...)
	cmd.WaitDelay = 2 * time.Second
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	ch := make(chan []byte, 2)
	go func() {
		defer close(ch)
		defer cmd.Wait()
		splitJPEG(out, func(f []byte) bool {
			select {
			case ch <- f:
				return true
			case <-ctx.Done():
				return false
			}
		})
	}()
	return ch, nil
}

// poll is the no-ffmpeg path: one `import -window root` per tick, at most 2 frames a second.
func (o CaptureOptions) poll(ctx context.Context) <-chan []byte {
	fps := o.FPS
	if fps > 2 {
		fps = 2
	}
	ch := make(chan []byte, 1)
	go func() {
		defer close(ch)
		t := time.NewTicker(time.Second / time.Duration(fps))
		defer t.Stop()
		fails := 0
		for {
			var buf bytes.Buffer
			cmd := o.Command(ctx, "import", "-display", o.Display, "-window", "root", "-quality", strconv.Itoa(100-3*o.Quality), "jpeg:-")
			cmd.Stdout = &buf
			if err := cmd.Run(); err != nil || buf.Len() == 0 {
				if fails++; fails >= 3 || ctx.Err() != nil {
					return
				}
			} else {
				fails = 0
				select {
				case ch <- buf.Bytes():
				case <-ctx.Done():
					return
				}
			}
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
		}
	}()
	return ch
}

// splitJPEG cuts a concatenation of JPEG files (FFD8 .. FFD9; inside entropy data every FF is followed by 00 or an RST
// marker, so FFD9 only ends a picture) and calls f for each. It stops when r ends or f returns false.
func splitJPEG(r io.Reader, f func([]byte) bool) {
	br := bufio.NewReaderSize(r, 64<<10)
	var cur []byte
	in := false
	var prev byte
	for {
		b, err := br.ReadByte()
		if err != nil {
			return
		}
		if !in {
			if prev == 0xFF && b == 0xD8 {
				in = true
				cur = append(cur[:0], 0xFF, 0xD8)
			}
			prev = b
			continue
		}
		cur = append(cur, b)
		if prev == 0xFF && b == 0xD9 {
			in = false
			prev = 0
			if !f(append([]byte(nil), cur...)) {
				return
			}
			continue
		}
		prev = b
		if len(cur) > 8<<20 {
			in, prev = false, 0 // runaway garbage: resync on the next SOI
		}
	}
}
