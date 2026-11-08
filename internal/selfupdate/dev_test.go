package selfupdate

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestParseCheckMode(t *testing.T) {
	for in, want := range map[string]CheckMode{"": CheckOn, "on": CheckOn, "ON": CheckOn, "off": CheckOff, "0": CheckOff, "false": CheckOff, "no": CheckOff, "manual": CheckManual, " Manual ": CheckManual, "bogus": CheckOn} {
		if got := ParseCheckMode(in); got != want {
			t.Errorf("%q: %v, want %v", in, got, want)
		}
	}
}

func TestDevFromEnvGatedByImage(t *testing.T) {
	// not a dev image: the URL is ignored, production behaviour
	if d, err := DevFromEnv(env(map[string]string{"D2BOT_UPDATE_URL": "http://x:1"})); d != nil || err != nil {
		t.Fatalf("non-dev image must ignore the URL: %v %v", d, err)
	}
	if d, err := DevFromEnv(env(map[string]string{"DIDLE_DEV": "1"})); d != nil || err != nil {
		t.Fatalf("dev image without URL: %v %v", d, err)
	}
}

func TestDevFromEnvKeyForms(t *testing.T) {
	pub, _, _ := GenerateKey(rand.Reader)
	txt := string(EncodePublicKey(pub))
	dir := t.TempDir()
	f := filepath.Join(dir, "dev.pub")
	os.WriteFile(f, []byte(txt), 0o644)
	escaped := ""
	for _, c := range txt {
		if c == '\n' {
			escaped += `\n`
		} else {
			escaped += string(c)
		}
	}
	for name, v := range map[string]string{"text": txt, "escaped": escaped, "file": f} {
		d, err := DevFromEnv(env(map[string]string{"DIDLE_DEV": "1", "D2BOT_UPDATE_URL": "http://h:8899/", "D2BOT_UPDATE_PUBKEY": v}))
		if err != nil || d == nil || len(d.Keys) != 1 || !d.Keys[0].Equal(pub) || d.URL != "http://h:8899" {
			t.Errorf("%s: %+v %v", name, d, err)
		}
	}
	for _, bad := range []string{"ftp://h/", "nonsense", "http://"} {
		if _, err := DevFromEnv(env(map[string]string{"DIDLE_DEV": "1", "D2BOT_UPDATE_URL": bad})); err == nil {
			t.Errorf("url %q accepted", bad)
		}
	}
	if _, err := DevFromEnv(env(map[string]string{"DIDLE_DEV": "1", "D2BOT_UPDATE_URL": "http://h", "D2BOT_UPDATE_PUBKEY": "/nonexistent.pub"})); err == nil {
		t.Error("unreadable key accepted")
	}
}

func TestDevClientAllowsOnlyItsHost(t *testing.T) {
	d := &DevSource{URL: "http://lan.example:8899"}
	c := d.Client("ua")
	if !c.AllowHost("lan.example") || c.AllowHost("api.github.com") || c.AllowHost("evil.example") {
		t.Error("dev client must allow exactly the dev host")
	}
}

// The dev loop end to end through the dev client: two successive dev builds apply, a key the install does not trust
// is refused, and an older manifest (rollback attack) is refused after a newer one was applied.
func TestDevLoopSequence(t *testing.T) {
	f := newFake(t)
	d := &DevSource{URL: f.srv.URL}
	mk := func(running string) (*Engine, Layout) {
		return f.engine(t, running, func(c *Config) { c.Client = d.Client("test") })
	}
	f.publish("0.0.1001", 1001, nil)
	e, l := mk("0.0.0")
	if err := e.Check(ctx); err != nil {
		t.Fatal(err)
	}
	if a := e.Status().Available; a == nil || a.Version != "0.0.1001" {
		t.Fatalf("available %+v", a)
	}
	if err := e.Download(ctx); err != nil {
		t.Fatal(err)
	}
	if err := e.Apply("0.0.1001", true); err != nil {
		t.Fatal(err)
	}
	if l.CurrentVersion() != "0.0.1001" {
		t.Fatalf("current %q", l.CurrentVersion())
	}

	// second build, same layout
	f.etag = `"v2"`
	f.publish("0.0.1002", 1002, nil)
	e2 := New(Config{Layout: l, Keys: []ed25519.PublicKey{f.pub}, Running: "0.0.1001", ImageABI: 1, Local: Compat{BotAPI: 1, D2DCtl: 1, DataSchema: 1},
		CheckEnabled: true, Client: d.Client("test")})
	if err := e2.Check(ctx); err != nil {
		t.Fatal(err)
	}
	if err := e2.Download(ctx); err != nil {
		t.Fatal(err)
	}
	if err := e2.Apply("0.0.1002", true); err != nil {
		t.Fatal(err)
	}
	st, _ := l.LoadState()
	if st.Current != "0.0.1002" || st.Previous != "0.0.1001" || st.SeqFloor != 1002 {
		t.Fatalf("state %+v", st)
	}

	// rollback attack: a validly signed but older build is refused, nothing swapped
	f.etag = `"v3"`
	f.publish("0.0.1001", 1001, nil)
	e3 := New(Config{Layout: l, Keys: []ed25519.PublicKey{f.pub}, Running: "0.0.1002", ImageABI: 1, Local: Compat{BotAPI: 1, D2DCtl: 1, DataSchema: 1},
		CheckEnabled: true, Client: d.Client("test")})
	if err := e3.Check(ctx); !errors.Is(err, ErrDowngrade) {
		t.Fatalf("older build: %v", err)
	}
	if l.CurrentVersion() != "0.0.1002" {
		t.Fatalf("current moved to %q", l.CurrentVersion())
	}

	// a key the install does not trust
	other, _, _ := GenerateKey(rand.Reader)
	f.etag = `"v4"`
	f.publish("0.0.1003", 1003, nil)
	e4 := New(Config{Layout: l, Keys: []ed25519.PublicKey{other}, Running: "0.0.1002", ImageABI: 1, Local: Compat{BotAPI: 1, D2DCtl: 1, DataSchema: 1},
		CheckEnabled: true, Client: d.Client("test")})
	if err := e4.Check(ctx); err == nil {
		t.Fatal("manifest signed by an untrusted key accepted")
	}
	if e4.Status().Available != nil {
		t.Error("untrusted manifest offered")
	}
}
