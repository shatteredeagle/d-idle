package selfupdate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func tgz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	tw := tar.NewWriter(gz)
	for n, c := range files {
		tw.WriteHeader(&tar.Header{Name: n, Mode: 0o755, Size: int64(len(c)), Typeflag: tar.TypeReg})
		tw.Write([]byte(c))
	}
	tw.Close()
	gz.Close()
	return b.Bytes()
}

// fakeGH is a stand-in for the GitHub releases API plus the asset host.
type fakeGH struct {
	t        *testing.T
	srv      *httptest.Server
	priv     ed25519.PrivateKey
	pub      ed25519.PublicKey
	assets   map[string][]byte // asset name -> bytes (served under /dl/)
	tag      string
	etag     string
	status   int // forced status for the releases list (0: normal)
	calls    atomic.Int32
	manifest Manifest
}

func newFake(t *testing.T) *fakeGH {
	f := &fakeGH{t: t, assets: map[string][]byte{}, etag: `"v1"`}
	f.pub, f.priv, _ = GenerateKey(rand.Reader)
	f.srv = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeGH) serve(w http.ResponseWriter, r *http.Request) {
	f.calls.Add(1)
	if strings.HasPrefix(r.URL.Path, "/dl/") {
		b, ok := f.assets[strings.TrimPrefix(r.URL.Path, "/dl/")]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Write(b)
		return
	}
	if f.status != 0 {
		if f.status == 403 {
			w.Header().Set("X-RateLimit-Remaining", "0")
		}
		w.WriteHeader(f.status)
		return
	}
	if r.Header.Get("If-None-Match") == f.etag {
		w.WriteHeader(304)
		return
	}
	w.Header().Set("ETag", f.etag)
	type asset struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	}
	var as []asset
	for n := range f.assets {
		as = append(as, asset{n, f.srv.URL + "/dl/" + n})
	}
	json.NewEncoder(w).Encode([]any{
		map[string]any{"tag_name": "tester-v9.9.9", "assets": []any{}},
		map[string]any{"tag_name": f.tag, "assets": as},
	})
}

// publish builds a signed release set with the given pieces.
func (f *fakeGH) publish(version string, seq int64, mut func(*Manifest)) {
	f.tag = TagPrefix + version
	f.assets = map[string][]byte{}
	bin := tgz(f.t, map[string]string{"d2bot": "bot " + version})
	d2d := tgz(f.t, map[string]string{"d2d": "d2d " + version})
	add := func(name string, b []byte) File {
		f.assets[name] = b
		s := sha256.Sum256(b)
		return File{Name: name, SHA256: hex.EncodeToString(s[:]), Size: int64(len(b))}
	}
	fb, fd := add("bin.tar.gz", bin), add("d2d.tar.gz", d2d)
	fb.Dest, fd.Dest = "bin", "d2d"
	m := Manifest{Schema: 1, Version: version, Seq: seq, Issued: "2026-11-20T00:00:00Z", Channel: "stable", Notes: "notes " + version,
		MinImage: 1, Compat: Compat{BotAPI: 1, D2DCtl: 1, DataSchema: 1}, Files: []File{fb, fd}}
	if mut != nil {
		mut(&m)
	}
	raw, err := m.Marshal()
	if err != nil {
		f.t.Fatal(err)
	}
	f.assets["MANIFEST.json"] = raw
	f.assets["MANIFEST.json.sig"] = Sign(f.priv, raw)
	f.manifest = m
}

func (f *fakeGH) engine(t *testing.T, running string, mut func(*Config)) (*Engine, Layout) {
	l := Layout{Root: t.TempDir()}
	cfg := Config{
		Layout: l, Keys: []ed25519.PublicKey{f.pub}, Running: running, ImageABI: 1,
		Local: Compat{BotAPI: 1, D2DCtl: 1, DataSchema: 1}, CheckEnabled: true,
		Client: &Client{BaseURL: f.srv.URL, UserAgent: "didle-updater/test", AllowHost: func(string) bool { return true }},
	}
	if mut != nil {
		mut(&cfg)
	}
	return New(cfg), l
}

var ctx = context.Background()

func TestGoodUpdate(t *testing.T) {
	f := newFake(t)
	f.publish("0.4.0", 5, nil)
	var shut int
	e, l := f.engine(t, "0.3.0", func(c *Config) { c.Shutdown = func() error { shut++; return nil } })
	// the old release is on disk and current
	os.MkdirAll(l.Release("0.3.0"), 0o755)
	l.SwapCurrent("0.3.0")

	if err := e.Check(ctx); err != nil {
		t.Fatal(err)
	}
	s := e.Status()
	if s.Available == nil || s.Available.Version != "0.4.0" || s.State != StateIdle {
		t.Fatalf("status after check: %+v", s)
	}
	if _, err := os.Stat(l.Staging()); err == nil {
		t.Fatal("check downloaded assets")
	}
	if err := e.Apply("0.4.0", false); !errors.Is(err, ErrNotReady) {
		t.Fatalf("apply before download: %v", err)
	}
	if err := e.Download(ctx); err != nil {
		t.Fatal(err)
	}
	if e.Status().State != StateReady {
		t.Fatal("not ready")
	}
	if err := e.Apply("0.3.9", false); !errors.Is(err, ErrConfirm) {
		t.Fatalf("wrong confirm: %v", err)
	}
	if err := e.Apply("0.4.0", false); err != nil {
		t.Fatal(err)
	}
	if shut != 1 {
		t.Fatalf("shutdown called %d times", shut)
	}
	if l.CurrentVersion() != "0.4.0" {
		t.Fatalf("current = %q", l.CurrentVersion())
	}
	if b, _ := os.ReadFile(filepath.Join(l.Current(), "bin", "d2bot")); string(b) != "bot 0.4.0" {
		t.Fatalf("unpacked content %q", b)
	}
	if b, _ := os.ReadFile(filepath.Join(l.Current(), "d2d", "d2d")); string(b) != "d2d 0.4.0" {
		t.Fatalf("d2d content %q", b)
	}
	st, _ := l.LoadState()
	if st.Current != "0.4.0" || st.Previous != "0.3.0" || st.Pending != "0.4.0" || st.SeqFloor != 5 || st.VersionFloor != "0.4.0" {
		t.Fatalf("state %+v", st)
	}
	if _, err := os.Stat(filepath.Join(l.Staging(), "0.4.0")); err == nil {
		t.Fatal("staging not cleaned")
	}
}

func TestKeepsLastTwoReleases(t *testing.T) {
	f := newFake(t)
	e, l := f.engine(t, "0.1.0", nil)
	for i, v := range []string{"0.2.0", "0.3.0", "0.4.0"} {
		f.etag = fmt.Sprintf(`"e%d"`, i)
		f.publish(v, int64(i+2), nil)
		if err := e.Check(ctx); err != nil {
			t.Fatal(err)
		}
		if err := e.Download(ctx); err != nil {
			t.Fatal(err)
		}
		if err := e.Apply(v, true); err != nil {
			t.Fatal(err)
		}
		e.cfg.Running = v
		e.set(StateIdle, "")
	}
	ents, _ := os.ReadDir(l.Releases())
	var got []string
	for _, x := range ents {
		got = append(got, x.Name())
	}
	if strings.Join(got, ",") != "0.3.0,0.4.0" {
		t.Fatalf("releases kept: %v", got)
	}
}

func TestHashMismatchRefused(t *testing.T) {
	f := newFake(t)
	f.publish("0.4.0", 5, nil)
	f.assets["bin.tar.gz"] = tgz(t, map[string]string{"d2bot": "EVIL"}) // manifest still lists the real hash
	e, l := f.engine(t, "0.3.0", nil)
	if err := e.Check(ctx); err != nil {
		t.Fatal(err)
	}
	err := e.Download(ctx)
	if !errors.Is(err, ErrHashMismatch) {
		t.Fatalf("download: %v", err)
	}
	if e.Status().State != StateError {
		t.Fatalf("state %s", e.Status().State)
	}
	if _, err := os.Stat(filepath.Join(l.Staging(), "0.4.0")); err == nil {
		t.Fatal("staging kept after a mismatch")
	}
	if l.CurrentVersion() != "" {
		t.Fatal("something was swapped")
	}
	if err := e.Apply("0.4.0", true); !errors.Is(err, ErrNotReady) {
		t.Fatalf("apply after mismatch: %v", err)
	}
}

func TestBadSignatureRefused(t *testing.T) {
	f := newFake(t)
	f.publish("0.4.0", 5, nil)
	_, other, _ := GenerateKey(rand.Reader)
	f.assets["MANIFEST.json.sig"] = Sign(other, f.assets["MANIFEST.json"])
	e, l := f.engine(t, "0.3.0", nil)
	err := e.Check(ctx)
	if !errors.Is(err, ErrBadSignature) {
		t.Fatalf("check: %v", err)
	}
	if e.Status().Available != nil {
		t.Fatal("unsigned release offered")
	}
	if st, _ := l.LoadState(); st.ETag != "" {
		t.Fatal("etag stored for a refused release")
	}
}

func TestTamperedManifestRefused(t *testing.T) {
	f := newFake(t)
	f.publish("0.4.0", 5, nil)
	f.assets["MANIFEST.json"] = bytes.Replace(f.assets["MANIFEST.json"], []byte("notes"), []byte("NOTES"), 1)
	e, _ := f.engine(t, "0.3.0", nil)
	if err := e.Check(ctx); !errors.Is(err, ErrBadSignature) {
		t.Fatalf("check: %v", err)
	}
}

func TestOfflineMakesNoCall(t *testing.T) {
	f := newFake(t)
	f.publish("0.4.0", 5, nil)
	e, _ := f.engine(t, "0.3.0", func(c *Config) { c.CheckEnabled = false })
	if err := e.Check(ctx); !errors.Is(err, ErrOffline) {
		t.Fatalf("check: %v", err)
	}
	e.Run(ctx, 0) // returns at once when off
	if n := f.calls.Load(); n != 0 {
		t.Fatalf("%d outbound calls while off", n)
	}
	if s := e.Status(); s.CheckEnabled || s.Running != "0.3.0" {
		t.Fatalf("status %+v", s)
	}
}

func TestDowngradeRefused(t *testing.T) {
	f := newFake(t)
	f.publish("0.4.0", 5, nil)
	e, l := f.engine(t, "0.4.0", nil)
	st := State{SeqFloor: 9, VersionFloor: "0.5.0"}
	l.SaveState(st)
	// older version, lower seq, correctly signed (a replay): refused
	if err := e.Check(ctx); !errors.Is(err, ErrDowngrade) {
		t.Fatalf("replay: %v", err)
	}
	// same version as running: not an error, just nothing available
	f.etag = `"v2"`
	f.publish("0.4.0", 12, nil)
	l.SaveState(State{SeqFloor: 9, VersionFloor: "0.3.0"})
	if err := e.Check(ctx); err != nil || e.Status().Available != nil {
		t.Fatalf("same version: %v %+v", err, e.Status())
	}
}

func TestRevokedRefused(t *testing.T) {
	f := newFake(t)
	f.publish("0.4.0", 5, func(m *Manifest) { m.Revoke.Versions = []string{"0.4.0"} })
	e, _ := f.engine(t, "0.3.0", nil)
	if err := e.Check(ctx); !errors.Is(err, ErrRevoked) {
		t.Fatalf("manifest self-revoke: %v", err)
	}

	// a separately signed REVOKED file bans the version and raises the floor
	f2 := newFake(t)
	f2.publish("0.4.0", 5, nil)
	rv, _ := Revoked{Schema: 1, Seq: 2, Versions: []string{"0.4.0"}, MinVersion: "0.3.5"}.Marshal()
	f2.assets["REVOKED"], f2.assets["REVOKED.sig"] = rv, Sign(f2.priv, rv)
	e2, l2 := f2.engine(t, "0.3.0", nil)
	if err := e2.Check(ctx); !errors.Is(err, ErrRevoked) {
		t.Fatalf("REVOKED file: %v", err)
	}
	if st, _ := l2.LoadState(); st.VersionFloor != "0.3.5" {
		t.Fatalf("floor not raised: %+v", st)
	}

	// an unsigned REVOKED (wrong key) is itself refused
	f3 := newFake(t)
	f3.publish("0.4.0", 5, nil)
	_, other, _ := GenerateKey(rand.Reader)
	f3.assets["REVOKED"], f3.assets["REVOKED.sig"] = rv, Sign(other, rv)
	e3, _ := f3.engine(t, "0.3.0", nil)
	if err := e3.Check(ctx); !errors.Is(err, ErrBadSignature) {
		t.Fatalf("forged REVOKED: %v", err)
	}
}

func TestNeedsImageAndIncompatible(t *testing.T) {
	f := newFake(t)
	f.publish("0.4.0", 5, func(m *Manifest) { m.MinImage = 2 })
	e, _ := f.engine(t, "0.3.0", nil)
	if err := e.Check(ctx); err != nil {
		t.Fatal(err)
	}
	if a := e.Status().Available; a == nil || !a.NeedsImage {
		t.Fatalf("available %+v", a)
	}
	if err := e.Download(ctx); !errors.Is(err, ErrNeedsImage) {
		t.Fatalf("download: %v", err)
	}
	f.etag = `"v2"`
	f.publish("0.5.0", 6, func(m *Manifest) { m.Compat.DataSchema = 2 })
	if err := e.Check(ctx); !errors.Is(err, ErrIncompatible) {
		t.Fatalf("data schema: %v", err)
	}
}

func TestETagAndRateLimit(t *testing.T) {
	f := newFake(t)
	f.publish("0.4.0", 5, nil)
	e, l := f.engine(t, "0.3.0", nil)
	if err := e.Check(ctx); err != nil {
		t.Fatal(err)
	}
	if st, _ := l.LoadState(); st.ETag != `"v1"` || st.LastCheck.IsZero() {
		t.Fatalf("state %+v", st)
	}
	before := f.calls.Load()
	if err := e.Check(ctx); err != nil { // 304
		t.Fatal(err)
	}
	if f.calls.Load()-before != 1 {
		t.Fatalf("a 304 check made %d calls", f.calls.Load()-before)
	}
	if e.Status().Available == nil {
		t.Fatal("304 dropped the known update")
	}
	f.status = 403
	if err := e.Check(ctx); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("rate limit: %v", err)
	}
	if s := e.Status(); s.LastError == "" || s.State == StateError {
		t.Fatalf("rate limit should be a quiet last error: %+v", s)
	}
}

func TestHostAllowList(t *testing.T) {
	f := newFake(t)
	f.publish("0.4.0", 5, nil)
	e, _ := f.engine(t, "0.3.0", func(c *Config) { c.Client.AllowHost = nil }) // default list: the test server is not GitHub
	if err := e.Check(ctx); err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("check against a foreign host: %v", err)
	}
}

func TestNotSafeWaits(t *testing.T) {
	f := newFake(t)
	f.publish("0.4.0", 5, nil)
	safe := false
	e, l := f.engine(t, "0.3.0", func(c *Config) { c.IdleSafe = func() bool { return safe } })
	e.Check(ctx)
	if err := e.Download(ctx); err != nil {
		t.Fatal(err)
	}
	if err := e.Apply("0.4.0", false); !errors.Is(err, ErrNotSafe) {
		t.Fatalf("apply unsafe: %v", err)
	}
	if e.Status().State != StateReady || l.CurrentVersion() != "" {
		t.Fatal("unsafe apply changed things")
	}
	safe = true
	if err := e.Apply("0.4.0", false); err != nil {
		t.Fatal(err)
	}
}

func TestShutdownFailureRollsBackAndWritesMarker(t *testing.T) {
	f := newFake(t)
	f.publish("0.4.0", 5, nil)
	e, l := f.engine(t, "0.3.0", func(c *Config) { c.Shutdown = func() error { return errors.New("hero not saved") } })
	os.MkdirAll(l.Release("0.3.0"), 0o755)
	l.SwapCurrent("0.3.0")
	e.Check(ctx)
	e.Download(ctx)
	if err := e.Apply("0.4.0", true); err == nil {
		t.Fatal("expected the shutdown error")
	}
	if l.CurrentVersion() != "0.3.0" {
		t.Fatalf("current = %q after failed shutdown", l.CurrentVersion())
	}
	m, _ := l.ReadMarker()
	if m == nil || m.From != "0.4.0" || m.To != "0.3.0" || !strings.Contains(m.Reason, "hero not saved") {
		t.Fatalf("marker %+v", m)
	}
	if st, _ := l.LoadState(); !st.IsBad("0.4.0") || st.Current != "0.3.0" {
		t.Fatalf("state %+v", st)
	}
	if s := e.Status(); s.Rollback == nil {
		t.Fatal("status hides the marker")
	}
	e.MarkSeen()
	if e.Status().Rollback != nil {
		t.Fatal("marker still shown after MarkSeen")
	}
}

func TestManualRollback(t *testing.T) {
	f := newFake(t)
	e, l := f.engine(t, "0.4.0", func(c *Config) { c.Shutdown = func() error { return nil } })
	os.MkdirAll(l.Release("0.3.0"), 0o755)
	os.MkdirAll(l.Release("0.4.0"), 0o755)
	l.SwapCurrent("0.4.0")
	l.SaveState(State{Current: "0.4.0", Previous: "0.3.0"})
	if err := e.Rollback(); err != nil {
		t.Fatal(err)
	}
	if l.CurrentVersion() != "0.3.0" {
		t.Fatalf("current %q", l.CurrentVersion())
	}
	if m, _ := l.ReadMarker(); m == nil || m.Reason != "manual rollback" {
		t.Fatalf("marker %+v", m)
	}
	// no previous: falls back to the base by removing the link
	e.set(StateIdle, "") // the real process exits after a rollback
	os.MkdirAll(l.Release("0.5.0"), 0o755)
	l.SwapCurrent("0.5.0")
	l.SaveState(State{Current: "0.5.0"})
	if err := e.Rollback(); err != nil || l.CurrentVersion() != "" {
		t.Fatalf("to base: %v %q", err, l.CurrentVersion())
	}
	e.set(StateIdle, "")
	if err := e.Rollback(); err == nil {
		t.Fatal("rollback from the base should fail")
	}
}

func TestCancelClearsStaging(t *testing.T) {
	f := newFake(t)
	f.publish("0.4.0", 5, nil)
	e, l := f.engine(t, "0.3.0", nil)
	e.Check(ctx)
	e.Download(ctx)
	e.Cancel()
	if _, err := os.Stat(filepath.Join(l.Staging(), "0.4.0")); err == nil {
		t.Fatal("staging kept")
	}
	if e.Status().State != StateIdle {
		t.Fatalf("state %s", e.Status().State)
	}
}

func TestUnpackRefusesUnsafeTar(t *testing.T) {
	mk := func(h tar.Header, body string) []byte {
		var b bytes.Buffer
		gz := gzip.NewWriter(&b)
		tw := tar.NewWriter(gz)
		h.Size = int64(len(body))
		tw.WriteHeader(&h)
		tw.Write([]byte(body))
		tw.Close()
		gz.Close()
		return b.Bytes()
	}
	bad := map[string]tar.Header{
		"dotdot":   {Name: "../evil", Typeflag: tar.TypeReg, Mode: 0o644},
		"nested":   {Name: "a/../../evil", Typeflag: tar.TypeReg, Mode: 0o644},
		"absolute": {Name: "/etc/evil", Typeflag: tar.TypeReg, Mode: 0o644},
		"symlink":  {Name: "l", Linkname: "/etc", Typeflag: tar.TypeSymlink},
		"hardlink": {Name: "h", Linkname: "x", Typeflag: tar.TypeLink},
		"device":   {Name: "d", Typeflag: tar.TypeChar},
		"fifo":     {Name: "f", Typeflag: tar.TypeFifo},
	}
	for name, h := range bad {
		dir := t.TempDir()
		if err := Unpack(bytes.NewReader(mk(h, "x")), filepath.Join(dir, "out")); err == nil {
			t.Errorf("%s: accepted", name)
		}
		if _, err := os.Stat(filepath.Join(dir, "evil")); err == nil {
			t.Errorf("%s: wrote outside", name)
		}
	}
	dir := t.TempDir()
	if err := Unpack(bytes.NewReader(tgz(t, map[string]string{"sub/a": "1", "b": "2"})), dir); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "sub", "a")); string(b) != "1" {
		t.Fatal("good tar not unpacked")
	}
	if st, _ := os.Stat(filepath.Join(dir, "b")); st.Mode().Perm() != 0o755 {
		t.Fatalf("mode %v", st.Mode().Perm())
	}
}

func TestInstallFailureLeavesNoRelease(t *testing.T) {
	f := newFake(t)
	f.publish("0.4.0", 5, nil)
	e, l := f.engine(t, "0.3.0", func(c *Config) { c.SelfTest = func(string) error { return errors.New("binary does not start") } })
	e.Check(ctx)
	e.Download(ctx)
	if err := e.Apply("0.4.0", true); err == nil {
		t.Fatal("selftest failure ignored")
	}
	ents, _ := os.ReadDir(l.Releases())
	if len(ents) != 0 || l.CurrentVersion() != "" {
		t.Fatalf("leftovers %v", ents)
	}
}

func TestRunChecksOnce(t *testing.T) {
	f := newFake(t)
	f.publish("0.4.0", 5, nil)
	e, _ := f.engine(t, "0.3.0", nil)
	c, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() { e.Run(c, time.Millisecond); close(done) }()
	for i := 0; i < 200 && e.Status().Available == nil; i++ {
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	<-done
	if e.Status().Available == nil {
		t.Fatal("Run did not check")
	}
	if e.Status().State == StateDownloading {
		t.Fatal("Run downloaded")
	}
}
