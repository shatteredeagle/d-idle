package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/selfupdate"
)

// The offline drill: the updater (inbox, so no network) installs a signed release, then the shim confirms or
// rolls it back. Same code paths as the container, fake clock and probes; the worker-host drills in
// docs/self-update-drill.md repeat it with a real image.
func dropRelease(t *testing.T, r *rig, version string) {
	t.Helper()
	inbox := r.layout.Inbox()
	os.MkdirAll(inbox, 0o755)
	var tb bytes.Buffer
	gz := gzip.NewWriter(&tb)
	tw := tar.NewWriter(gz)
	for n, c := range map[string]string{"d2bot": "bot " + version, "entrypoint.sh": "#!/bin/sh\n"} {
		tw.WriteHeader(&tar.Header{Name: n, Mode: 0o755, Size: int64(len(c)), Typeflag: tar.TypeReg})
		tw.Write([]byte(c))
	}
	tw.Close()
	gz.Close()
	sum := sha256.Sum256(tb.Bytes())
	os.WriteFile(filepath.Join(inbox, "bin.tar.gz"), tb.Bytes(), 0o644)
	m := selfupdate.Manifest{Schema: 1, Version: version, Seq: 20, Issued: "2026-11-20T00:00:00Z", Channel: "stable", MinImage: 1,
		Compat: r.b.Image.Local, Files: []selfupdate.File{{Name: "bin.tar.gz", SHA256: hex.EncodeToString(sum[:]), Size: int64(tb.Len()), Dest: "bin"}}}
	raw, err := m.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(inbox, "MANIFEST.json"), raw, 0o644)
	os.WriteFile(filepath.Join(inbox, "MANIFEST.json.sig"), selfupdate.Sign(r.priv, raw), 0o644)
}

func applyViaEngine(t *testing.T, r *rig, version string) {
	t.Helper()
	dropRelease(t, r, version)
	e := selfupdate.New(selfupdate.Config{Layout: r.layout, Keys: r.b.Image.Keys, Running: "0.3.0", ImageABI: r.b.Image.ABI,
		Local: r.b.Image.Local, CheckEnabled: false, Shutdown: func() error { return nil }})
	if err := e.Inbox(); err != nil {
		t.Fatal(err)
	}
	if err := e.Download(ctx()); err != nil {
		t.Fatal(err)
	}
	if err := e.Apply(version, true); err != nil {
		t.Fatal(err)
	}
}

func TestDrillUpdateThenHealthy(t *testing.T) {
	r := newRig(t)
	applyViaEngine(t, r, "0.4.0")
	r.run.scripts["0.4.0"] = []script{{exitAfter: 20 * time.Minute, code: 0}}
	if code := r.b.Run(); code != 0 || strings.Join(r.run.started, ",") != "0.4.0" {
		t.Fatalf("code %d started %v", code, r.run.started)
	}
	st, _ := r.layout.LoadState()
	if st.Good != "0.4.0" || st.Pending != "" || r.layout.CurrentVersion() != "0.4.0" {
		t.Fatalf("state %+v current %q", st, r.layout.CurrentVersion())
	}
}

func TestDrillBadUpdateRollsBackToBase(t *testing.T) {
	r := newRig(t)
	applyViaEngine(t, r, "0.4.0")
	r.run.scripts["0.4.0"] = []script{{exitAfter: 2 * time.Second, code: 1}}
	r.run.scripts["0.3.0"] = []script{{exitAfter: time.Hour}}
	if code := r.b.Run(); code != 0 || strings.Join(r.run.started, ",") != "0.4.0,0.3.0" {
		t.Fatalf("code %d started %v", code, r.run.started)
	}
	m, _ := r.layout.ReadMarker()
	if m == nil || m.From != "0.4.0" || r.layout.CurrentVersion() != "" {
		t.Fatalf("marker %+v current %q", m, r.layout.CurrentVersion())
	}
	// the updater reads the same marker and refuses to offer the version again
	e := selfupdate.New(selfupdate.Config{Layout: r.layout, Keys: r.b.Image.Keys, Running: "0.3.0", ImageABI: 2, Local: r.b.Image.Local})
	if s := e.Status(); s.Rollback == nil || s.Rollback.From != "0.4.0" {
		t.Fatalf("status %+v", s)
	}
	dropRelease(t, r, "0.4.0")
	if err := e.Inbox(); err == nil || !strings.Contains(err.Error(), "rolled back before") {
		t.Fatalf("a rolled-back version was offered again: %v", err)
	}
}
