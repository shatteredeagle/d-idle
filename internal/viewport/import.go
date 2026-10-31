package viewport

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"time"
)

// ImportCapturer runs ImageMagick `import -window <id>` once per frame (the original d2watch method, V live:
// about 22 ms CPU per 800x587 frame). It only ever captures the given window, never the root.
type ImportCapturer struct {
	Timeout time.Duration // per capture, default 4 s
	Bin     string        // default "import"
}

func (ImportCapturer) Persistent() bool { return false }

func (c ImportCapturer) Open(_ context.Context, s Spec) (Session, error) {
	if s.Window == "" {
		return nil, ErrNoTarget
	}
	return &importSession{c: c, s: s}, nil
}

type importSession struct {
	c ImportCapturer
	s Spec
}

func (*importSession) Close() error { return nil }

func (is *importSession) Next(ctx context.Context) (Frame, error) {
	to := is.c.Timeout
	if to <= 0 {
		to = 4 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, to)
	defer cancel()
	bin := is.c.Bin
	if bin == "" {
		bin = "import"
	}
	args := []string{"-window", is.s.Window, "-quality", strconv.Itoa(is.s.quality())}
	if is.s.MaxW > 0 || is.s.MaxH > 0 {
		args = append(args, "-resize", geom(is.s.MaxW, is.s.MaxH)+">")
	}
	args = append(args, "jpg:-")
	cmd := exec.CommandContext(ctx, bin, args...)
	// One thread: ImageMagick's OpenMP workers spin under load and multiplied the CPU cost many times over (V).
	cmd.Env = append(xEnv(is.s), "OMP_NUM_THREADS=1", "MAGICK_THREAD_LIMIT=1", "OMP_WAIT_POLICY=passive")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		return Frame{}, err
	}
	b := buf.Bytes()
	if len(b) < 4 || b[0] != 0xFF || b[1] != 0xD8 {
		return Frame{}, errors.New("not a jpeg")
	}
	return Frame{Data: b, Mime: "image/jpeg", At: time.Now()}, nil
}

// xEnv is the environment for X client tools: s.Env, or the process environment, plus DISPLAY.
func xEnv(s Spec) []string {
	env := s.Env
	if env == nil {
		env = os.Environ()
	}
	if s.Display != "" {
		env = append(env[:len(env):len(env)], "DISPLAY="+s.Display)
	}
	return env
}

// geom is a geometry string where a missing limit is left open ("800x", "x600").
func geom(w, h int) string {
	s := ""
	if w > 0 {
		s = strconv.Itoa(w)
	}
	s += "x"
	if h > 0 {
		s += strconv.Itoa(h)
	}
	return s
}
