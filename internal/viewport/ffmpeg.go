package viewport

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// FFmpegCapturer runs one persistent `ffmpeg -f x11grab -window_id <id>` per session and splits its MJPEG
// output into frames. It grabs the given window only. Pacing is done by ffmpeg (-framerate); a slow consumer
// just backpressures the pipe.
type FFmpegCapturer struct {
	Bin string // default "ffmpeg"
}

func (FFmpegCapturer) Persistent() bool { return true }

func (c FFmpegCapturer) Open(ctx context.Context, s Spec) (Session, error) {
	if s.Window == "" {
		return nil, ErrNoTarget
	}
	bin := c.Bin
	if bin == "" {
		bin = "ffmpeg"
	}
	cctx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(cctx, bin, c.args(s)...)
	cmd.Env = xEnv(s)
	var stderr bytes.Buffer
	cmd.Stderr = &limitedWriter{w: &stderr, n: 4096}
	out, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, err
	}
	return &ffmpegSession{cmd: cmd, cancel: cancel, sp: &jpegSplitter{r: out}, stderr: &stderr}, nil
}

func (FFmpegCapturer) args(s Spec) []string {
	a := []string{"-hide_banner", "-loglevel", "error", "-nostdin",
		"-f", "x11grab", "-draw_mouse", "0", "-framerate", strconv.FormatFloat(s.fps(), 'f', -1, 64),
		"-window_id", s.Window, "-i", s.Display}
	if s.MaxW > 0 || s.MaxH > 0 {
		w, h := "iw", "ih"
		if s.MaxW > 0 {
			w = fmt.Sprintf("min(%d,iw)", s.MaxW)
		}
		if s.MaxH > 0 {
			h = fmt.Sprintf("min(%d,ih)", s.MaxH)
		}
		a = append(a, "-vf", fmt.Sprintf("scale='%s':'%s':force_original_aspect_ratio=decrease:force_divisible_by=2", w, h))
	}
	// mjpeg q:v 2 (best) .. 31 (worst); map JPEG quality 100..1 onto it.
	qv := 2 + (100-s.quality())*29/99
	return append(a, "-q:v", strconv.Itoa(qv), "-c:v", "mjpeg", "-f", "mjpeg", "-")
}

type ffmpegSession struct {
	cmd    *exec.Cmd
	cancel context.CancelFunc
	sp     *jpegSplitter
	stderr *bytes.Buffer
}

func (f *ffmpegSession) Next(ctx context.Context) (Frame, error) {
	type res struct {
		b   []byte
		err error
	}
	ch := make(chan res, 1)
	go func() { b, err := f.sp.Next(); ch <- res{b, err} }()
	select {
	case r := <-ch:
		if r.err != nil {
			msg := strings.TrimSpace(f.stderr.String())
			if msg != "" {
				return Frame{}, fmt.Errorf("ffmpeg: %s", msg)
			}
			return Frame{}, fmt.Errorf("ffmpeg: %w", r.err)
		}
		return Frame{Data: r.b, Mime: "image/jpeg", At: time.Now()}, nil
	case <-ctx.Done():
		return Frame{}, ctx.Err()
	}
}

func (f *ffmpegSession) Close() error {
	f.cancel()
	return f.cmd.Wait()
}

type limitedWriter struct {
	w io.Writer
	n int
}

func (l *limitedWriter) Write(p []byte) (int, error) {
	if l.n > 0 {
		q := p
		if len(q) > l.n {
			q = q[:l.n]
		}
		l.w.Write(q)
		l.n -= len(q)
	}
	return len(p), nil
}

// jpegSplitter cuts a concatenation of JPEG files (SOI FFD8 ... EOI FFD9) into frames. FFD9 cannot occur inside
// entropy-coded data (0xFF is stuffed as FF00) and ffmpeg's mjpeg encoder writes no embedded thumbnails.
type jpegSplitter struct {
	r   io.Reader
	buf []byte
}

var errNoJPEG = errors.New("stream ended")

func (j *jpegSplitter) Next() ([]byte, error) {
	var tmp [64 << 10]byte
	for {
		if i := bytes.Index(j.buf, []byte{0xFF, 0xD8}); i >= 0 {
			if e := bytes.Index(j.buf[i+2:], []byte{0xFF, 0xD9}); e >= 0 {
				end := i + 2 + e + 2
				out := append([]byte(nil), j.buf[i:end]...)
				j.buf = j.buf[end:]
				return out, nil
			}
			j.buf = j.buf[i:]
		} else if len(j.buf) > 1 {
			j.buf = j.buf[len(j.buf)-1:]
		}
		n, err := j.r.Read(tmp[:])
		j.buf = append(j.buf, tmp[:n]...)
		if err != nil && n == 0 {
			if err == io.EOF {
				return nil, errNoJPEG
			}
			return nil, err
		}
	}
}
