package friendpack

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const d2dCommit = "0123456789abcdef0123456789abcdef01234567"

func need(t *testing.T, tools ...string) {
	t.Helper()
	// the builder ships the bot source with git archive; a synced copy whose .git link is broken cannot do that
	if b, err := exec.Command("git", "rev-parse", "--verify", "HEAD").CombinedOutput(); err != nil {
		t.Skipf("no usable git repository here: %s", b)
	}
	for _, n := range append(tools, "bash", "tar", "gzip", "git", "sha256sum") {
		if _, err := exec.LookPath(n); err != nil {
			t.Skipf("%s not available", n)
		}
	}
}

func tgz(t *testing.T, dir, name string, files map[string]string) string {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(zw)
	for n, c := range files {
		if err := tw.WriteHeader(&tar.Header{Name: n, Mode: 0o644, Size: int64(len(c))}); err != nil {
			t.Fatal(err)
		}
		tw.Write([]byte(c))
	}
	tw.Close()
	zw.Close()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func build(t *testing.T, out string, d2dFiles map[string]string, extra ...string) (string, error) {
	t.Helper()
	in := t.TempDir()
	d2d := tgz(t, in, "d2d.tar.gz", d2dFiles)
	img := tgz(t, in, "image.tar.gz", map[string]string{"manifest.json": "[]"})
	args := append([]string{"build.sh", "--version", "0.1.0", "--image", "didle-friends:0.1.0", "--image-tar", img,
		"--d2d-src", d2d, "--d2d-commit", d2dCommit, "--out", out, "--bot-ref", "HEAD"}, extra...)
	b, err := exec.Command("bash", args...).CombinedOutput()
	return string(b), err
}

func unpack(t *testing.T, pack string) string {
	t.Helper()
	dir := t.TempDir()
	if b, err := exec.Command("tar", "xzf", pack, "-C", dir).CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, b)
	}
	return filepath.Join(dir, "didle-friends-0.1.0")
}

func TestPackFileListAndChecksums(t *testing.T) {
	need(t)
	out := t.TempDir()
	if o, err := build(t, out, map[string]string{"d2d/README": "x"}); err != nil {
		t.Fatalf("%v\n%s", err, o)
	}
	root := unpack(t, filepath.Join(out, "didle-friends-0.1.0.tar.gz"))
	var got []string
	filepath.Walk(root, func(p string, i os.FileInfo, err error) error {
		if err == nil && !i.IsDir() {
			r, _ := filepath.Rel(root, p)
			if strings.HasPrefix(r, "src/d-idle-") {
				r = "src/d-idle-<commit>.tar.gz"
			}
			got = append(got, r)
		}
		return nil
	})
	sort.Strings(got)
	want := []string{"README.txt", "RELEASE_NOTES.md", "SHA256SUMS", "UPDATE_FEED", "VERSION", "compose.yml", "didle", "guide/friend-guide.md",
		"image/IMAGE_TAG", "image/didle-image.tar.gz", "install.sh", "licenses/GPL-3.0.txt", "licenses/NOTICE", "licenses/SOURCE.txt",
		"licenses/THIRD_PARTY.md", "licenses/licence-note.txt", "pubkeys/beta.pub", "pubkeys/current.pub", "src/BUILD.txt", "src/d-idle-<commit>.tar.gz",
		"src/d2d-didle-0123456789ab.tar.gz"}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("file list\n got %v\nwant %v", got, want)
	}
	for _, n := range got {
		l := strings.ToLower(n)
		for _, bad := range []string{".mpq", ".dc6", ".dcc", ".dt1", ".d2s", ".key", ".png", "token"} {
			if strings.HasSuffix(l, bad) {
				t.Errorf("forbidden file in pack: %s", n)
			}
		}
	}
	c := exec.Command("sha256sum", "-c", "--quiet", "SHA256SUMS")
	c.Dir = root
	if b, err := c.CombinedOutput(); err != nil {
		t.Fatalf("SHA256SUMS: %v %s", err, b)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "licenses", "SOURCE.txt")); !bytes.Contains(b, []byte(d2dCommit)) {
		t.Error("SOURCE.txt lacks the d2d commit")
	}
	// reproducible: a second build is byte-identical
	out2 := t.TempDir()
	if o, err := build(t, out2, map[string]string{"d2d/README": "x"}); err != nil {
		t.Fatalf("%v\n%s", err, o)
	}
	a, _ := os.ReadFile(filepath.Join(out, "didle-friends-0.1.0.tar.gz.sha256"))
	b, _ := os.ReadFile(filepath.Join(out2, "didle-friends-0.1.0.tar.gz.sha256"))
	if !bytes.Equal(a, b) {
		t.Errorf("pack is not reproducible: %s vs %s", a, b)
	}
}

func TestPackRefusesGameArtAndKeyFiles(t *testing.T) {
	need(t)
	for _, f := range []string{"d2/d2data.mpq", "d2/web/art/x.dc6", "d2/leak.key", "d2/watch.token", "d2/game.exe", "d2/D2Client.dll", "d2/internal/watch/web/a.png"} {
		out := t.TempDir()
		o, err := build(t, out, map[string]string{f: "x"})
		if err == nil || !strings.Contains(o, "STOP") {
			t.Errorf("%s accepted:\n%s", f, o)
		}
		if _, e := os.Stat(filepath.Join(out, "didle-friends-0.1.0.tar.gz")); e == nil {
			t.Errorf("%s: a pack was written", f)
		}
	}
}

// install.sh against a shim docker: the refusals, then one happy path that checks .env and the printed address.
func TestInstallScript(t *testing.T) {
	need(t, "curl", "df", "awk")
	out := t.TempDir()
	if o, err := build(t, out, map[string]string{"d2d/README": "x"}, "--feed", "https://feed.example/d2"); err != nil {
		t.Fatalf("%v\n%s", err, o)
	}
	root := unpack(t, filepath.Join(out, "didle-friends-0.1.0.tar.gz"))
	shim := t.TempDir()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(shim, name), []byte("#!/bin/sh\n"+body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write("docker", `echo "$@" >>"$SHIMLOG"
case "$1 $2" in
"image inspect") echo 1 ;;
"compose exec") echo TOK123 ;;
"compose ps") ;;
esac
exit 0`)
	write("curl", "exit 0")
	log := filepath.Join(t.TempDir(), "log")
	install := func(args ...string) (string, error) {
		c := exec.Command("bash", append([]string{filepath.Join(root, "install.sh")}, args...)...)
		c.Env = append(os.Environ(), "PATH="+shim+":"+os.Getenv("PATH"), "SHIMLOG="+log)
		c.Stdin = strings.NewReader("y\n")
		b, err := c.CombinedOutput()
		return string(b), err
	}
	// the licence needs a terminal answer, so without a tty it stops; --yes before any acceptance is refused
	if o, err := install("--yes", "--game-dir", t.TempDir()); err == nil || !strings.Contains(o, "--yes is only allowed") {
		t.Fatalf("--yes without ack: %v\n%s", err, o)
	}
	if err := os.WriteFile(filepath.Join(root, ".licence-ack"), []byte("0.1.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	game := t.TempDir()
	if o, err := install("--yes", "--game-dir", game); err == nil || !strings.Contains(o, "missing in") || !strings.Contains(o, "d2data.mpq") {
		t.Fatalf("empty game dir: %v\n%s", err, o)
	}
	for _, n := range []string{"d2data.mpq", "D2EXP.MPQ", "d2char.mpq", "Patch_D2.mpq"} {
		os.WriteFile(filepath.Join(game, n), []byte("x"), 0o644)
	}
	if o, err := install("--yes", "--game-dir", game, "--port", "80"); err == nil || !strings.Contains(o, "--port") {
		t.Fatalf("low port: %v\n%s", err, o)
	}
	o, err := install("--yes", "--game-dir", game, "--port", "19391", "--project", "didle-test")
	if err != nil {
		t.Fatalf("happy path: %v\n%s", err, o)
	}
	if !strings.Contains(o, "http://localhost:19391/setup.html?t=TOK123") {
		t.Errorf("exact URL not printed:\n%s", o)
	}
	env, _ := os.ReadFile(filepath.Join(root, ".env"))
	for _, w := range []string{"COMPOSE_PROJECT_NAME=didle-test", "D2_IMAGE=didle-friends:0.1.0", "D2_ANSWER_PORT=19392", "D2_PUBLISH=127.0.0.1",
		"D2_WATCH_ORIGIN=http://localhost:19391,http://127.0.0.1:19391", "D2BOT_UPDATE_FEED=https://feed.example/d2", "D2_GAME_DIR=" + game} {
		if !strings.Contains(string(env), w+"\n") {
			t.Errorf(".env lacks %q:\n%s", w, env)
		}
	}
	if fi, _ := os.Stat(filepath.Join(root, ".env")); fi.Mode().Perm() != 0o600 {
		t.Errorf(".env mode %v", fi.Mode().Perm())
	}
	// a damaged download is refused before anything is loaded
	os.WriteFile(filepath.Join(root, "README.txt"), []byte("changed"), 0o644)
	if o, err := install("--yes", "--game-dir", game); err == nil || !strings.Contains(o, "damaged") {
		t.Fatalf("tampered pack: %v\n%s", err, o)
	}
	// --host publishes on all interfaces and prints that name
	os.WriteFile(filepath.Join(root, "README.txt"), mustRead(t, filepath.Join("README.txt")), 0o644)
	if o, err := install("--yes", "--game-dir", game, "--port", "19391", "--host", "box.example"); err != nil || !strings.Contains(o, "http://box.example:19391/setup.html?t=TOK123") {
		t.Fatalf("--host: %v\n%s", err, o)
	}
	env, _ = os.ReadFile(filepath.Join(root, ".env"))
	if !strings.Contains(string(env), "D2_PUBLISH=0.0.0.0\n") || !strings.Contains(string(env), "D2_WATCH_ORIGIN=http://box.example:19391,") {
		t.Errorf("--host .env:\n%s", env)
	}
}

func mustRead(t *testing.T, p string) []byte {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
