package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"d2lod-bot/internal/selfupdate"
)

const base = "https://feed.example/d2"

type env struct {
	t    *testing.T
	pub  ed25519.PublicKey
	priv ed25519.PrivateKey
	pubF string
}

func newEnv(t *testing.T) *env {
	pub, priv, _ := selfupdate.GenerateKey(rand.Reader)
	d := t.TempDir()
	f := filepath.Join(d, "beta.pub")
	os.WriteFile(f, selfupdate.EncodePublicKey(pub), 0o644)
	return &env{t, pub, priv, f}
}

// release writes a signed release set (throwaway key, tiny assets) and returns its directory.
func (e *env) release(version string, seq int64, sign bool, priv ed25519.PrivateKey) string {
	dir := e.t.TempDir()
	m := selfupdate.Manifest{Schema: 1, Version: version, Seq: seq, Issued: "2026-11-20T00:00:00Z", Channel: "beta", Notes: "n " + version,
		MinImage: 1, Compat: selfupdate.Compat{BotAPI: 1, D2DCtl: 1, DataSchema: 1}}
	for _, a := range []struct{ name, dest string }{{"bin.tar.gz", "bin"}, {"d2d.tar.gz", "d2d"}} {
		b := []byte(a.name + " " + version)
		os.WriteFile(filepath.Join(dir, a.name), b, 0o644)
		s := sha256.Sum256(b)
		m.Files = append(m.Files, selfupdate.File{Name: a.name, SHA256: hex.EncodeToString(s[:]), Size: int64(len(b)), Dest: a.dest})
	}
	raw, err := m.Marshal()
	if err != nil {
		e.t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "MANIFEST.json"), raw, 0o644)
	if sign {
		os.WriteFile(filepath.Join(dir, "MANIFEST.json.sig"), selfupdate.Sign(priv, raw), 0o644)
	}
	return dir
}

func do(t *testing.T, want int, args ...string) string {
	t.Helper()
	var o, er bytes.Buffer
	if got := run(args, &o, &er); got != want {
		t.Fatalf("d2feed %v: exit %d, want %d: %s %s", args, got, want, o.String(), er.String())
	}
	return o.String() + er.String()
}

func (e *env) build(want int, rel, out string) string {
	return do(e.t, want, "build", "-release", rel, "-pub", e.pubF, "-base-url", base, "-out", out)
}

func (e *env) verify(want int, out string) string {
	return do(e.t, want, "verify", "-out", out, "-pub", e.pubF, "-base-url", base)
}

func TestPublishVerifyIdempotentAndKeepsOld(t *testing.T) {
	e := newEnv(t)
	out := t.TempDir()
	r1 := e.release("0.4.0", 1, true, e.priv)
	e.build(0, r1, out)
	e.verify(0, out)
	lp := listPath(out, selfupdate.DefaultRepo)
	first, _ := os.ReadFile(lp)
	e.build(0, r1, out) // again: no change
	if again, _ := os.ReadFile(lp); !bytes.Equal(first, again) {
		t.Fatal("republishing the same release changed the list")
	}
	if !strings.Contains(string(first), base+"/dl/app-v0.4.0/bin.tar.gz") {
		t.Fatalf("asset URLs are not under the base URL:\n%s", first)
	}
	e.build(0, e.release("0.5.0", 2, true, e.priv), out)
	e.verify(0, out)
	l, _ := loadList(lp)
	if len(l) != 2 || l[0].Tag != "app-v0.5.0" || l[1].Tag != "app-v0.4.0" {
		t.Fatalf("list %+v", l)
	}
	if _, err := os.Stat(filepath.Join(out, "dl", "app-v0.4.0", "bin.tar.gz")); err != nil {
		t.Fatal("old release files were removed")
	}
}

func TestBuildRefusals(t *testing.T) {
	e := newEnv(t)
	_, other, _ := selfupdate.GenerateKey(rand.Reader)
	out := t.TempDir()
	e.build(1, e.release("0.4.0", 1, false, nil), out)  // unsigned
	e.build(1, e.release("0.4.0", 1, true, other), out) // wrong key
	tam := e.release("0.4.0", 1, true, e.priv)          // asset changed after signing
	os.WriteFile(filepath.Join(tam, "bin.tar.gz"), []byte("evil"), 0o644)
	e.build(1, tam, out)
	man := e.release("0.4.0", 1, true, e.priv) // manifest changed after signing
	b, _ := os.ReadFile(filepath.Join(man, "MANIFEST.json"))
	os.WriteFile(filepath.Join(man, "MANIFEST.json"), bytes.Replace(b, []byte("n 0.4.0"), []byte("N 0.4.0"), 1), 0o644)
	e.build(1, man, out)
	if _, err := os.Stat(listPath(out, selfupdate.DefaultRepo)); err == nil {
		t.Fatal("a refused release left a list behind")
	}
	do(t, 1, "build", "-release", e.release("0.4.0", 1, true, e.priv), "-pub", e.pubF, "-base-url", "http://feed.example", "-out", out)
	good := e.release("0.5.0", 2, true, e.priv)
	e.build(0, good, out)
	e.build(1, e.release("0.4.0", 1, true, e.priv), out) // older than the published newest
	diff := e.release("0.5.0", 2, true, e.priv)          // same version, other bytes (other issue time)
	m, _ := os.ReadFile(filepath.Join(diff, "MANIFEST.json"))
	m2 := bytes.Replace(m, []byte("n 0.5.0"), []byte("x 0.5.0"), 1)
	os.WriteFile(filepath.Join(diff, "MANIFEST.json"), m2, 0o644)
	os.WriteFile(filepath.Join(diff, "MANIFEST.json.sig"), selfupdate.Sign(e.priv, m2), 0o644)
	e.build(1, diff, out)
}

func TestVerifyCatchesTamperedTreeAndWrongKey(t *testing.T) {
	e := newEnv(t)
	out := t.TempDir()
	e.build(0, e.release("0.4.0", 1, true, e.priv), out)
	other := newEnv(t)
	do(t, 1, "verify", "-out", out, "-pub", other.pubF, "-base-url", base) // a different pinned key
	os.WriteFile(filepath.Join(out, "dl", "app-v0.4.0", "bin.tar.gz"), []byte("tampered on the host"), 0o644)
	e.verify(1, out)
}

func TestWithdrawFallsBackToPreviousRelease(t *testing.T) {
	e := newEnv(t)
	out := t.TempDir()
	e.build(0, e.release("0.4.0", 1, true, e.priv), out)
	e.build(0, e.release("0.5.0", 2, true, e.priv), out)
	do(t, 1, "withdraw", "-out", out, "-version", "9.9.9")
	do(t, 0, "withdraw", "-out", out, "-version", "0.5.0")
	if o := e.verify(0, out); !strings.Contains(o, "app-v0.4.0") {
		t.Fatalf("verify after withdraw: %s", o)
	}
	do(t, 0, "withdraw", "-out", out, "-version", "0.4.0")
	e.verify(1, out) // nothing left to offer
}

// The wrapper: refuses unsigned input without a key, signs with a throwaway key path, verifies, then hands the tree
// to the user's upload hook.
func TestFeedPublishScript(t *testing.T) {
	for _, n := range []string{"bash", "go"} {
		if _, err := exec.LookPath(n); err != nil {
			t.Skip(n + " missing")
		}
	}
	e := newEnv(t)
	keyF := filepath.Join(t.TempDir(), "k.key")
	os.WriteFile(keyF, selfupdate.EncodePrivateKey(e.priv), 0o600)
	rel := e.release("0.4.0", 1, false, nil)
	out, mark := t.TempDir(), filepath.Join(t.TempDir(), "uploaded")
	script, _ := filepath.Abs("../../tools/release/feed-publish.sh")
	run := func(extra ...string) (string, error) {
		c := exec.Command("bash", append([]string{script, "--release", rel, "--base-url", base, "--out", out, "--pub", e.pubF}, extra...)...)
		c.Env = append(os.Environ(), "D2_FEED_UPLOAD_CMD=cp -r \"$1\" "+mark)
		b, err := c.CombinedOutput()
		return string(b), err
	}
	if o, err := run("--upload"); err == nil || !strings.Contains(o, "sign first") {
		t.Fatalf("unsigned: %v\n%s", err, o)
	}
	if _, err := os.Stat(mark); err == nil {
		t.Fatal("uploaded unsigned input")
	}
	if o, err := run("--sign-key", keyF); err != nil || !strings.Contains(o, "not uploaded") {
		t.Fatalf("sign and build: %v\n%s", err, o)
	}
	if _, err := os.Stat(mark); err == nil {
		t.Fatal("uploaded without --upload")
	}
	if o, err := run("--upload"); err != nil || !strings.Contains(o, "uploaded") {
		t.Fatalf("upload: %v\n%s", err, o)
	}
	if _, err := os.Stat(filepath.Join(mark, "repos", "shatteredeagle", "d-idle", "releases")); err != nil {
		t.Fatalf("hook did not get the tree: %v", err)
	}
}
