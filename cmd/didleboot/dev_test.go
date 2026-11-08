package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"d2lod-bot/internal/selfupdate"
)

// A reload request stops the running release without blaming it and starts the selection again; the container (the
// shim) stays up. The second start picks up a release installed meanwhile.
func TestReloadRestartsOnNewSelection(t *testing.T) {
	r := newRig(t)
	r.b.Policy = DevPolicy()
	var reload bool
	r.b.Reload = func() bool { x := reload; reload = false; return x }
	n := 0
	r.b.Clock = clockFunc{r.clk, func() {
		n++
		if n == 5 { // while the base runs: a new release arrives and the shim is told to reload
			r.pendingUpdate("0.4.0", "")
			reload = true
		}
		if n == 80 {
			r.stop = true
		}
	}}
	code := r.b.Run()
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if got := strings.Join(r.run.started, ","); got != "0.3.0,0.4.0" {
		t.Fatalf("started %s", got)
	}
	st, _ := r.layout.LoadState()
	if len(st.Bad) != 0 {
		t.Fatalf("a reload must not mark anything bad: %+v", st)
	}
}

type clockFunc struct {
	*fakeClock
	on func()
}

func (c clockFunc) Sleep(d time.Duration) { c.fakeClock.Sleep(d); c.on() }

// Dev policy: no probes (there may be no hero), but 15 s of staying up still confirms, and a crash is rolled back.
func TestDevPolicyConfirmsWithoutProbesAndRollsBackCrash(t *testing.T) {
	r := newRig(t)
	r.b.Policy = DevPolicy()
	r.probe.apiAfter, r.probe.tickAfter, r.probe.watch = -1, -1, false // nothing answers
	r.pendingUpdate("0.4.0", "0.3.5")
	r.run.scripts["0.4.0"] = []script{{}}
	r.clk.Sleep(0)
	n := 0
	r.b.Clock = clockFunc{r.clk, func() {
		n++
		if n > 40 {
			r.stop = true
		}
	}}
	r.b.Run()
	st, _ := r.layout.LoadState()
	if st.Good != "0.4.0" || st.Pending != "" {
		t.Fatalf("not confirmed with dev policy: %+v", st)
	}

	// a build that dies after 3 s is rolled back to the previous one
	r2 := newRig(t)
	r2.b.Policy = DevPolicy()
	r2.pendingUpdate("0.4.0", "0.3.5")
	r2.run.scripts["0.4.0"] = []script{{exitAfter: 3 * time.Second, code: 1}}
	m := 0
	r2.b.Clock = clockFunc{r2.clk, func() {
		m++
		if m > 60 {
			r2.stop = true
		}
	}}
	r2.b.Run()
	if got := strings.Join(r2.run.started, ","); got != "0.4.0,0.3.5" {
		t.Fatalf("started %s", got)
	}
}

func TestDevCommandsRefuseOnNormalImage(t *testing.T) {
	t.Setenv("DIDLE_DEV", "")
	var out bytes.Buffer
	if code := devCommand("check-now", Image{}, selfupdate.Layout{Root: t.TempDir()}, &out); code != 1 || !strings.Contains(out.String(), "not a DEV image") {
		t.Fatalf("%d %q", code, out.String())
	}
}

func TestSignalShimRejectsNonShimPid(t *testing.T) {
	l := selfupdate.Layout{Root: t.TempDir()}
	if err := signalShim(l); err == nil {
		t.Error("missing pid file accepted")
	}
	os.WriteFile(pidFile(l), []byte("1\n"), 0o644)
	if err := signalShim(l); err == nil {
		t.Error("pid 1 accepted")
	}
	os.WriteFile(pidFile(l), []byte("0\n"), 0o644)
	if err := signalShim(l); err == nil {
		t.Error("pid 0 accepted")
	}
}

// check-now end to end against a static directory shaped like the dev server: signed manifest, one bin tarball.
func TestCheckNowAgainstDevDir(t *testing.T) {
	pub, priv, _ := selfupdate.GenerateKey(rand.Reader)
	files := map[string][]byte{}
	var tb bytes.Buffer
	gz := gzip.NewWriter(&tb)
	tw := tar.NewWriter(gz)
	tw.WriteHeader(&tar.Header{Name: "d2bot", Mode: 0o755, Size: 3, Typeflag: tar.TypeReg})
	tw.Write([]byte("bot"))
	tw.Close()
	gz.Close()
	files["bin.tar.gz"] = tb.Bytes()
	sum := sha256.Sum256(tb.Bytes())
	m := selfupdate.Manifest{Schema: 1, Version: "0.0.2611081530", Seq: 2611081530, Issued: "2026-11-08T15:30:00Z", Channel: "dev", MinImage: 1,
		Compat: selfupdate.Compat{BotAPI: 1, D2DCtl: 1, DataSchema: 1},
		Files:  []selfupdate.File{{Name: "bin.tar.gz", SHA256: hex.EncodeToString(sum[:]), Size: int64(tb.Len()), Dest: "bin"}}}
	raw, _ := m.Marshal()
	files["MANIFEST.json"], files["MANIFEST.json.sig"] = raw, selfupdate.Sign(priv, raw)
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/dl/") {
			w.Write(files[strings.TrimPrefix(r.URL.Path, "/dl/")])
			return
		}
		if !strings.HasSuffix(r.URL.Path, "/releases") {
			http.NotFound(w, r)
			return
		}
		var as []map[string]any
		for n := range files {
			as = append(as, map[string]any{"name": n, "browser_download_url": srv.URL + "/dl/" + n})
		}
		json.NewEncoder(w).Encode([]any{map[string]any{"tag_name": "app-v" + m.Version, "assets": as}})
	}))
	defer srv.Close()

	root := t.TempDir()
	t.Setenv("DIDLE_DEV", "1")
	t.Setenv("D2BOT_UPDATE_URL", srv.URL)
	t.Setenv("D2BOT_UPDATE_PUBKEY", string(selfupdate.EncodePublicKey(pub)))
	t.Setenv("D2BOT_UPDATE_CHECK", "manual")
	layout := selfupdate.Layout{Root: root}
	img := Image{ABI: 1, BaseVersion: "0.0.0", Local: selfupdate.BuildCompat}
	// the "shim" is this test process; it must survive the SIGHUP
	hup := make(chan os.Signal, 1)
	signal.Notify(hup, syscall.SIGHUP)
	defer signal.Stop(hup)
	os.WriteFile(pidFile(layout), []byte(strconv.Itoa(os.Getpid())), 0o644)

	var out bytes.Buffer
	if code := devCommand("check-now", img, layout, &out); code != 3 {
		t.Fatalf("untrusted key: code %d %q", code, out.String()) // the image does not trust the dev key (img.Keys empty)
	}
	if !strings.HasPrefix(out.String(), "REJECTED") {
		t.Fatalf("want REJECTED, got %q", out.String())
	}
	img.Keys = []ed25519.PublicKey{pub}
	out.Reset()
	if code := devCommand("check-now", img, layout, &out); code != 0 || strings.TrimSpace(out.String()) != "UPDATE 0.0.0 0.0.2611081530" {
		t.Fatalf("code %d %q", code, out.String())
	}
	select {
	case <-hup:
	case <-time.After(2 * time.Second):
		t.Fatal("the shim was not signalled")
	}
	if layout.CurrentVersion() != "0.0.2611081530" {
		t.Fatalf("current %q", layout.CurrentVersion())
	}
	out.Reset()
	img.BaseVersion = "0.0.0"
	if code := devCommand("check-now", img, layout, &out); code != 0 || !strings.HasPrefix(out.String(), "CURRENT") {
		t.Fatalf("second check: %d %q", code, out.String())
	}
	out.Reset()
	if code := devCommand("rollback-now", img, layout, &out); code != 0 || !strings.HasPrefix(out.String(), "ROLLBACK") {
		t.Fatalf("rollback: %d %q", code, out.String())
	}
	<-hup
	if layout.CurrentVersion() != "" {
		t.Fatalf("rollback to base left current %q", layout.CurrentVersion())
	}
}
