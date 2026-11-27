package d2dcontainer_test

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Fixtures are fake and built from parts, so this file itself carries no private-looking text for leakcheck.
var (
	fakeHome  = "/ho" + "me/alice/code"
	fakeHost  = "box1.lan.bir" + "bhub.com"
	fakeIP    = "192.168." + "1.20"
	fakeTok   = "0123456789abcdef" + "0123456789abcdef" + "0123456789abcdef"
	fakeKey   = "ABCD1-EF2GH-IJ3KL-MN4OP-QR5ST"
	fakeMail  = "alice@" + "mail.example.org"
	fakeName  = "des" + "mond"
	fakeBearr = "Bearer " + "abc123DEF456ghi789"
)

func scrub(t *testing.T, in string, env ...string) string {
	t.Helper()
	cmd := exec.Command("sh", "tester-report.sh")
	cmd.Env = append(os.Environ(), append([]string{"REPORT_SCRUB_ONLY=1", "D2BOT_DATA=" + t.TempDir()}, env...)...)
	cmd.Stdin = strings.NewReader(in)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("scrub: %v\n%s", err, out)
	}
	return string(out)
}

func TestScrubRemovesPrivateText(t *testing.T) {
	cases := []struct{ name, in, secret string }{
		{"token labelled", `watch_token=` + fakeTok, fakeTok},
		{"json secret", `{"api_key": "` + fakeTok + `"}`, fakeTok},
		{"bearer", `Authorization: ` + fakeBearr, "abc123DEF456ghi789"},
		{"url token", `GET /watch?t=` + fakeTok[:20] + `&x=1`, fakeTok[:20]},
		{"unlabelled long token", `got ` + fakeTok + ` from ui`, fakeTok},
		{"cd key dashed", `key ` + fakeKey, fakeKey},
		{"cd key 26", `ABCDEFGH1JKLMNOPQRSTUVWX2Y`, "ABCDEFGH1JKLMNOPQRSTUVWX2Y"},
		{"home path", `loading ` + fakeHome + `/d2d/save`, "alice"},
		{"windows home", `C:\Users\alice\Documents`, "alice"},
		{"private host", `dial ` + fakeHost + `:8891`, "bir" + "bhub"},
		{"url host", `origin http://mybox:8891/watch`, "mybox"},
		{"ip", `peer ` + fakeIP + `:8420 up`, fakeIP},
		{"ip 10", `peer 10.0.0.5`, "10.0.0.5"},
		{"email", `from ` + fakeMail, fakeMail},
		{"name", `hello ` + strings.ToUpper(fakeName), fakeName},
		{"extra name", `owner Zed`, "Zed"},
	}
	for _, c := range cases {
		out := scrub(t, c.in+"\n", "REPORT_NAMES=Zed")
		if strings.Contains(strings.ToLower(out), strings.ToLower(c.secret)) {
			t.Errorf("%s: %q still contains %q", c.name, out, c.secret)
		}
		if !strings.Contains(out, "<") {
			t.Errorf("%s: nothing replaced: %q", c.name, out)
		}
	}
}

func TestScrubKeepsUsefulText(t *testing.T) {
	keep := []string{
		`{"time":"2026-11-08T10:00:00Z","level":"INFO","msg":"minute","area":"Blood Moor","kills":62,"xp":120,"deaths":0}`,
		`screen=ingame class=1 name=Csmoke`,
		`t0.area.toohard: Forgotten Tower is too hard for you`,
		`http://localhost:8891/watch`,
		`http://127.0.0.1:8420/api/status`,
		`/opt/d2lod/entrypoint.sh /data/instances/main/state/ladder/journal.jsonl`,
		`session 20261108T100000Z-human gold 13 clvl 3 key press Return`,
		`Paladin level 3 Cold Plains waypoint`,
	}
	for _, k := range keep {
		if out := strings.TrimSpace(scrub(t, k+"\n")); out != k {
			t.Errorf("changed:\n in  %q\n out %q", k, out)
		}
	}
}

func TestBundle(t *testing.T) {
	data := t.TempDir()
	w := func(rel, s string) {
		p := filepath.Join(data, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(s), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	w("watch.token", fakeTok+"\n")
	w("access.json", `{"hash":"`+fakeTok+`"}`)
	w("wizard.json", `{"char_name":"Hero","class":"Paladin","game_dir":"`+fakeHome+`/game","patch_sha256":"x","phase":"done"}`)
	w("wizard.env", "D2D_NAME=Hero\nD2BOT_GOALS=campaign:normal_act1\n")
	w("instances/main/logs/d2bot.log", `{"msg":"start","url":"http://`+fakeHost+`:8891/watch?t=`+fakeTok+`"}`+"\n"+
		`{"msg":"minute","area":"Blood Moor","deaths":0}`+"\n"+`path `+fakeHome+"\n")
	w("instances/main/d2d.out", "d2d started "+fakeIP+" "+fakeTok+"\n")
	w("instances/main/state/deaths.jsonl", `{"area":"Blood Moor","by":"Fallen"}`+"\n")
	w("instances/main/state/ladder/journal.jsonl", `{"q":"t0.first","who":"`+fakeMail+`"}`+"\n")
	out := filepath.Join(t.TempDir(), "reports")
	cmd := exec.Command("sh", "tester-report.sh")
	cmd.Env = append(os.Environ(), "D2BOT_DATA="+data, "REPORT_OUT="+out, "D2D_SOCK="+filepath.Join(data, "none.sock"))
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run: %v\n%s", err, b)
	}
	tgz := strings.TrimSpace(string(b))
	if !strings.HasSuffix(tgz, ".tar.gz") {
		t.Fatalf("last line is not the tarball path: %q", b)
	}
	files := readTgz(t, tgz)
	for _, want := range []string{"bot.log", "d2d.log", "minutes.txt", "deaths.jsonl", "wizard.json", "wizard.env", "d2dping-state.txt", "README.txt", "ladder/"} {
		found := false
		for name := range files {
			if strings.Contains(name, want) {
				found = true
			}
		}
		if !found {
			t.Errorf("bundle lacks %s (has %v)", want, keys(files))
		}
	}
	all := ""
	for name, body := range files {
		all += body
		if strings.Contains(name, "watch.token") || strings.Contains(name, "access.json") {
			t.Errorf("bundle contains %s", name)
		}
	}
	for _, bad := range []string{fakeTok, fakeHome, "alice", fakeHost, fakeIP, fakeMail, "game_dir", "patch_sha"} {
		if strings.Contains(all, bad) {
			t.Errorf("bundle still contains %q", bad)
		}
	}
	for name, body := range files {
		if strings.HasSuffix(name, "minutes.txt") && !strings.Contains(body, `"msg":"minute"`) {
			t.Errorf("minutes.txt lacks the minute line: %q", body)
		}
		if strings.HasSuffix(name, "wizard.json") && !strings.Contains(body, "Paladin") {
			t.Errorf("wizard.json lost the class: %q", body)
		}
	}
}

func keys(m map[string]string) []string {
	var k []string
	for n := range m {
		k = append(k, n)
	}
	return k
}

func readTgz(t *testing.T, path string) map[string]string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(zr)
	files := map[string]string{}
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return files
		}
		if err != nil {
			t.Fatal(err)
		}
		if h.Typeflag != tar.TypeReg {
			continue
		}
		b, _ := io.ReadAll(tr)
		files[h.Name] = string(b)
	}
}
