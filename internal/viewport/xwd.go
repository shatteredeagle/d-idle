package viewport

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"os/exec"
	"time"
)

// XWDCapturer grabs the whole root window of a private X display with `xwd -root` (x11-apps, already in the
// container image) and encodes JPEG in Go, so it needs no ImageMagick or ffmpeg. It is meant for a display that
// holds one game only (a slot's Xvfb :10N); on a shared desktop use ImportCapturer, which captures one window.
// Spec.Window is not used; Spec.Display is required.
type XWDCapturer struct {
	Timeout time.Duration // per capture, default 4 s
	Bin     string        // default "xwd"
}

func (XWDCapturer) Persistent() bool { return false }

func (c XWDCapturer) Open(_ context.Context, s Spec) (Session, error) {
	if s.Display == "" {
		return nil, ErrNoTarget
	}
	return &xwdSession{c: c, s: s}, nil
}

type xwdSession struct {
	c XWDCapturer
	s Spec
}

func (*xwdSession) Close() error { return nil }

func (x *xwdSession) Next(ctx context.Context) (Frame, error) {
	to := x.c.Timeout
	if to <= 0 {
		to = 4 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, to)
	defer cancel()
	bin := x.c.Bin
	if bin == "" {
		bin = "xwd"
	}
	cmd := exec.CommandContext(ctx, bin, "-root", "-silent", "-display", x.s.Display)
	cmd.Env = xEnv(x.s)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &limitedWriter{w: &errb, n: 512}
	if err := cmd.Run(); err != nil {
		return Frame{}, fmt.Errorf("xwd: %w: %s", err, bytes.TrimSpace(errb.Bytes()))
	}
	img, err := DecodeXWD(out.Bytes())
	if err != nil {
		return Frame{}, err
	}
	var jb bytes.Buffer
	if err := jpeg.Encode(&jb, shrink(img, x.s.MaxW, x.s.MaxH), &jpeg.Options{Quality: x.s.quality()}); err != nil {
		return Frame{}, err
	}
	return Frame{Data: jb.Bytes(), Mime: "image/jpeg", At: time.Now()}, nil
}

// DecodeXWD decodes an X11 window dump (xwd) of a TrueColor ZPixmap with 24 or 32 bits per pixel, the format
// of Xvfb at depth 24. Layout: 25 big-endian uint32 header words (xwd.h, file version 7), the window name,
// the colormap entries, then the pixels in the dump's byte order (S: xwd.h and the Xlib XImage docs; the
// container's real xwd output has not been seen yet, U).
func DecodeXWD(b []byte) (*image.RGBA, error) {
	const hdr = 100
	if len(b) < hdr {
		return nil, errors.New("xwd: short header")
	}
	w := func(i int) uint32 { return binary.BigEndian.Uint32(b[4*i:]) }
	headerSize, version, format := int(w(0)), w(1), w(2)
	width, height := int(w(4)), int(w(5))
	byteOrder, bpp, bpl := w(7), int(w(11)), int(w(12))
	rm, gm, bm := w(14), w(15), w(16)
	ncolors := int(w(19))
	if version != 7 || format != 2 {
		return nil, fmt.Errorf("xwd: unsupported version %d format %d (need 7, ZPixmap)", version, format)
	}
	if bpp != 24 && bpp != 32 {
		return nil, fmt.Errorf("xwd: unsupported %d bits per pixel", bpp)
	}
	if width <= 0 || height <= 0 || width > 8192 || height > 8192 || bpl < width*bpp/8 || headerSize < hdr || ncolors < 0 || ncolors > 65536 {
		return nil, errors.New("xwd: bad geometry")
	}
	off := headerSize + 12*ncolors
	if len(b) < off+bpl*height {
		return nil, errors.New("xwd: short pixel data")
	}
	shift := func(m uint32) uint {
		n := uint(0)
		for m != 0 && m&1 == 0 {
			m >>= 1
			n++
		}
		return n
	}
	rs, gs, bs := shift(rm), shift(gm), shift(bm)
	bytesPP := bpp / 8
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		row := b[off+y*bpl:]
		dst := img.Pix[y*img.Stride:]
		for x := 0; x < width; x++ {
			p := row[x*bytesPP:]
			var v uint32
			switch {
			case bytesPP == 4 && byteOrder == 0:
				v = binary.LittleEndian.Uint32(p)
			case bytesPP == 4:
				v = binary.BigEndian.Uint32(p)
			case byteOrder == 0:
				v = uint32(p[0]) | uint32(p[1])<<8 | uint32(p[2])<<16
			default:
				v = uint32(p[0])<<16 | uint32(p[1])<<8 | uint32(p[2])
			}
			dst[4*x], dst[4*x+1], dst[4*x+2], dst[4*x+3] = uint8((v&rm)>>rs), uint8((v&gm)>>gs), uint8((v&bm)>>bs), 255
		}
	}
	return img, nil
}

// shrink downsamples (nearest neighbour, cheap) to fit maxW x maxH, never upscaling.
func shrink(src *image.RGBA, maxW, maxH int) image.Image {
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	nw, nh := w, h
	if maxW > 0 && nw > maxW {
		nw, nh = maxW, h*maxW/w
	}
	if maxH > 0 && nh > maxH {
		nw, nh = w*maxH/h, maxH
	}
	if nw < 1 || nh < 1 || (nw == w && nh == h) {
		return src
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	for y := 0; y < nh; y++ {
		sy := src.Pix[(y*h/nh)*src.Stride:]
		for x := 0; x < nw; x++ {
			copy(dst.Pix[y*dst.Stride+4*x:y*dst.Stride+4*x+4], sy[4*(x*w/nw):])
		}
	}
	return dst
}
