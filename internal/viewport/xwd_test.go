package viewport

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"image/jpeg"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// xwdDump builds a version-7 ZPixmap dump the way Xvfb depth 24 writes it (32 bpp little endian, BGRX).
func xwdDump(w, h int, px func(x, y int) (r, g, b byte)) []byte {
	name := []byte("root\x00")
	hdr := make([]uint32, 25)
	hdr[0], hdr[1], hdr[2], hdr[3] = uint32(100+len(name)), 7, 2, 24
	hdr[4], hdr[5] = uint32(w), uint32(h)
	hdr[7], hdr[8], hdr[9], hdr[10], hdr[11], hdr[12] = 0, 32, 0, 32, 32, uint32(4*w)
	hdr[13], hdr[14], hdr[15], hdr[16] = 4, 0xff0000, 0xff00, 0xff
	var b bytes.Buffer
	binary.Write(&b, binary.BigEndian, hdr)
	b.Write(name)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, bl := px(x, y)
			b.Write([]byte{bl, g, r, 0})
		}
	}
	return b.Bytes()
}

func TestDecodeXWD(t *testing.T) {
	d := xwdDump(4, 2, func(x, y int) (byte, byte, byte) { return byte(10 * x), byte(20 * y), 200 })
	img, err := DecodeXWD(d)
	if err != nil {
		t.Fatal(err)
	}
	if got := img.RGBAAt(3, 1); got.R != 30 || got.G != 20 || got.B != 200 || got.A != 255 {
		t.Fatalf("pixel = %+v", got)
	}
	if _, err := DecodeXWD(d[:120]); err == nil {
		t.Fatal("short dump accepted")
	}
	if _, err := DecodeXWD(d[:20]); err == nil {
		t.Fatal("short header accepted")
	}
}

// A fake xwd on PATH-less Bin: the capturer decodes its output, downsizes and encodes a real JPEG.
func TestXWDCapturerWithFakeBinary(t *testing.T) {
	dir := t.TempDir()
	dump := filepath.Join(dir, "d.xwd")
	os.WriteFile(dump, xwdDump(64, 48, func(x, y int) (byte, byte, byte) { return 200, 10, 10 }), 0o644)
	bin := filepath.Join(dir, "xwd")
	os.WriteFile(bin, []byte("#!/bin/sh\n[ \"$3\" = -display ] || exit 3\ncat "+dump+"\n"), 0o755)
	c := XWDCapturer{Bin: bin}
	sess, err := c.Open(context.Background(), Spec{Display: ":101", MaxW: 32})
	if err != nil {
		t.Fatal(err)
	}
	f, err := sess.Next(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	img, err := jpeg.Decode(bytes.NewReader(f.Data))
	if err != nil || img.Bounds().Dx() != 32 || img.Bounds().Dy() != 24 {
		t.Fatalf("jpeg %v %v", img, err)
	}
	if _, err := c.Open(context.Background(), Spec{}); err != ErrNoTarget {
		t.Fatalf("no display: %v", err)
	}
	bad := XWDCapturer{Bin: "/nonexistent/xwd"}
	s2, _ := bad.Open(context.Background(), Spec{Display: ":1"})
	if _, err := s2.Next(context.Background()); err == nil {
		t.Fatal("missing xwd must fail")
	}
}

type bigCap struct{}

func (bigCap) Persistent() bool { return false }
func (bigCap) Open(context.Context, Spec) (Session, error) {
	return bigSess{}, nil
}

type bigSess struct{}

func (bigSess) Close() error { return nil }
func (bigSess) Next(context.Context) (Frame, error) {
	return Frame{Data: make([]byte, 256<<10), Mime: "image/jpeg", At: time.Now()}, nil
}

// A client that never reads is dropped by the write deadline while a healthy one keeps streaming; the viewer
// cap answers 503 and the capture stops once everybody is gone.
func TestStreamSlowClientAndViewerCap(t *testing.T) {
	st := New(Config{Capturer: bigCap{}, FPS: 50, Idle: 100 * time.Millisecond, Target: func() (Spec, error) { return Spec{}, nil }})
	defer st.Close()
	h := &handler{c: HandlerConfig{Stream: st, MaxViewers: 2}}
	srv := httptest.NewServer(http.HandlerFunc(h.stream))
	defer srv.Close()

	slow, err := net.Dial("tcp", srv.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer slow.Close()
	slow.(*net.TCPConn).SetReadBuffer(4096)
	slow.Write([]byte("GET / HTTP/1.1\r\nHost: x\r\n\r\n"))
	rr, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer rr.Body.Close()
	br := bufio.NewReader(rr.Body)
	go func() {
		for {
			if _, err := br.ReadByte(); err != nil {
				return
			}
		}
	}()
	waitFor := func(cond func() bool, why string) {
		t.Helper()
		for end := time.Now().Add(StreamWriteTimeout + 4*time.Second); !cond(); time.Sleep(20 * time.Millisecond) {
			if time.Now().After(end) {
				t.Fatal(why)
			}
		}
	}
	waitFor(func() bool { return h.viewers.Load() == 2 }, "two viewers expected")
	if r3, err := http.Get(srv.URL); err != nil || r3.StatusCode != 503 {
		t.Fatalf("third viewer: %v %v", r3, err)
	}
	waitFor(func() bool { return h.viewers.Load() == 1 }, "slow client not dropped")
	rr.Body.Close()
	waitFor(func() bool { return h.viewers.Load() == 0 }, "healthy client not released")
	waitFor(func() bool { return !st.Status().Running }, "capture still running with no viewers")
}

type failCap struct{}

func (failCap) Persistent() bool                            { return false }
func (failCap) Open(context.Context, Spec) (Session, error) { return nil, ErrNoTarget }

// With nothing captured yet a stream answers 503 with the reason instead of hanging.
func TestStreamNoCapture503(t *testing.T) {
	st := New(Config{Capturer: failCap{}, FPS: 50, Idle: 50 * time.Millisecond, Target: func() (Spec, error) { return Spec{}, ErrNoTarget }})
	defer st.Close()
	srv := httptest.NewServer(Handler(HandlerConfig{Stream: st}))
	defer srv.Close()
	r, err := http.Get(srv.URL + "/stream")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	if r.StatusCode != 503 {
		t.Fatalf("status %d", r.StatusCode)
	}
}
