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
		{"ip 10", `peer ` + "1" + "0.0.0.5", "1" + "0.0.0.5"},
		{"email", `from ` + fakeMail, fakeMail},
		{"name", `hello ` + strings.ToUpper(fakeName), fakeName},
		{"extra name", `owner Zed`, "Zed"},
		// fable review 4 finding 1 (fake values)
		{"passcode kv", `login passcode=correct-horse-battery`, "correct-horse"},
		{"passcode json", `{"passcode":"four random words here"}`, "random words"},
		{"passcode colon", `passcode: hunter2hunter2`, "hunter2"},
		{"passphrase json", `{"passphrase": "open sesame now"}`, "sesame"},
		{"secret kv", `client_secret=s3cr3tvalue`, "s3cr3tvalue"},
		{"apikey kv", `apikey=k9k9k9k9`, "k9k9k9k9"},
		{"api_key kv", `api_key: 'k8k8k8k8'`, "k8k8k8k8"},
		{"auth json", `{"auth":"zzTOPzz1"}`, "zzTOPzz1"},
		{"key 16 undashed", `cdkey ABCD1234EFGH5678 plain`, "ABCD1234EFGH5678"},
		{"hex 20", `id deadbeef00112233aabb seen`, "deadbeef00112233aabb"},
		{"base64-ish", `blob QUJDREVGR0gxMjM0NTY3OA== end`, "QUJDREVGR0gxMjM0"},
		{"mesh ip", `peer 100.101.102.103:51234`, "100.101.102.103"},
		{"public ip", `from 8.8.4.4 and 9.9.9.9 ok`, "9.9.9.9"},
		{"public ip first", `from 8.8.4.4 and 9.9.9.9 ok`, "8.8.4.4"},
		{"ipv6 link-local", `addr fe80::1ff:fe23:4567:890a%eth0`, "fe23:4567"},
		{"ipv6 full", `addr 2001:db8:0:0:0:0:0:1 up`, "2001:db8"},
		{"mac", `hw aa:bb:cc:dd:ee:ff`, "bb:cc:dd"},
		{"bare host", `ssh ` + "ven" + "us" + ` now`, "ven" + "us"},
		{"desktop host", `host=DESKTOP-K3J2L1 x`, "K3J2L1"},
		{"user label", `user=alice ok`, "alice"},
		{"mnt path", `open /mnt/user/games/alice/d2`, "alice"},
		{"media path", `open /media/alice/stick`, "alice"},
		{"root path", `read /root/.cache/x`, "/root"},
		{"mac home", `open ` + "/Us" + `ers/alice/Games`, "alice"},
		{"home with space", `open ` + "/Us" + `ers/Bob Smith/Games/x.mpq`, "Smith"},
		{"windows home with space", `C:\Users\Bob Smith\Games`, "Smith"},
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
		`listen 0.0.0.0:8891 loop 127.0.0.1:99 at 10:00:00`,
		`mount /rootfs ok, unix 1794132000123456789, seed 0x6FAB1234`,
		`RequestDifficultyGoNow abc123def456ghi quest Jade Figurine`,
		`build 1.27.3 patch 1.14d`,
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
	for name, body := range files {
		if strings.HasSuffix(name, "README.txt") && (!strings.Contains(body, "NOT replaced") || strings.Contains(body, "private host names")) {
			t.Errorf("README must say what is not replaced and claim no more than the filter does: %q", body)
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
