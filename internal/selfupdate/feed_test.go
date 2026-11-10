package selfupdate

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"testing"
)

func TestFeedFromEnv(t *testing.T) {
	for _, bad := range []string{"http://feed.example", "ftp://x", "feed.example", "https://", "https://u:p" + "@feed.example", "https://feed.example/?a=1", "https://feed.example/#x"} {
		if f, err := FeedFromEnv(env(map[string]string{"D2BOT_UPDATE_FEED": bad})); err == nil || f != nil {
			t.Errorf("%q accepted", bad)
		}
	}
	if f, err := FeedFromEnv(env(nil)); f != nil || err != nil {
		t.Fatalf("empty: %v %v", f, err)
	}
	f, err := FeedFromEnv(env(map[string]string{"D2BOT_UPDATE_FEED": " https://feed.example:8443/d2/ "}))
	if err != nil || f.URL != "https://feed.example:8443/d2" {
		t.Fatalf("%+v %v", f, err)
	}
	c := f.Client("ua")
	if !c.AllowHost("feed.example") || c.AllowHost("api.github.com") || c.AllowHost("evil.example") || c.AllowHost("sub.feed.example") {
		t.Error("feed client must allow exactly the feed host")
	}
}

func feedEngine(t *testing.T, f *fakeGH, keys ...ed25519.PublicKey) *Engine {
	t.Helper()
	d := &FeedSource{URL: f.srv.URL}
	e, _ := f.engine(t, "0.3.0", func(c *Config) { c.Client = d.Client("test"); c.Keys = keys })
	return e
}

func TestFeedGoodManifest(t *testing.T) {
	f := newFake(t)
	f.publish("0.4.0", 5, nil)
	e := feedEngine(t, f, f.pub)
	if err := e.Check(ctx); err != nil {
		t.Fatal(err)
	}
	if a := e.Status().Available; a == nil || a.Version != "0.4.0" {
		t.Fatalf("available %+v", a)
	}
}

func TestFeedTamperedManifestRefused(t *testing.T) {
	f := newFake(t)
	f.publish("0.4.0", 5, nil)
	f.assets["MANIFEST.json"] = bytes.Replace(f.assets["MANIFEST.json"], []byte("notes"), []byte("NOTES"), 1)
	e := feedEngine(t, f, f.pub)
	if err := e.Check(ctx); !errors.Is(err, ErrBadSignature) || e.Status().Available != nil {
		t.Fatalf("tampered: %v", err)
	}
}

// A feed signed by a key the image does not bake (a friend pointing .env at their own feed) offers nothing.
func TestFeedWrongKeyRefused(t *testing.T) {
	f := newFake(t)
	f.publish("0.4.0", 5, nil)
	baked, _, _ := GenerateKey(rand.Reader)
	e := feedEngine(t, f, baked)
	if err := e.Check(ctx); !errors.Is(err, ErrBadSignature) || e.Status().Available != nil {
		t.Fatalf("wrong key: %v", err)
	}
}

func TestFeedUnsignedRefused(t *testing.T) {
	f := newFake(t)
	f.publish("0.4.0", 5, nil)
	delete(f.assets, "MANIFEST.json.sig")
	e := feedEngine(t, f, f.pub)
	if err := e.Check(ctx); err == nil || e.Status().Available != nil {
		t.Fatalf("unsigned accepted: %v", err)
	}
}

// Two pinned keys (beta for the friends' feed, current for GitHub releases): a manifest signed by either verifies,
// one signed by neither is refused. The fake signs with f.pub's private half.
func TestFeedTwoPinnedKeys(t *testing.T) {
	other, _, _ := GenerateKey(rand.Reader)
	unknown, _, _ := GenerateKey(rand.Reader)
	for _, tc := range []struct {
		name string
		keys func(signer ed25519.PublicKey) []ed25519.PublicKey
		ok   bool
	}{
		{"signed by beta (first pinned)", func(s ed25519.PublicKey) []ed25519.PublicKey { return []ed25519.PublicKey{s, other} }, true},
		{"signed by current (second pinned)", func(s ed25519.PublicKey) []ed25519.PublicKey { return []ed25519.PublicKey{other, s} }, true},
		{"signed by an unknown key", func(ed25519.PublicKey) []ed25519.PublicKey { return []ed25519.PublicKey{other, unknown} }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFake(t)
			f.publish("0.4.0", 5, nil)
			e := feedEngine(t, f, tc.keys(f.pub)...)
			err := e.Check(ctx)
			if tc.ok && (err != nil || e.Status().Available == nil) {
				t.Fatalf("refused: %v", err)
			}
			if !tc.ok && (!errors.Is(err, ErrBadSignature) || e.Status().Available != nil) {
				t.Fatalf("accepted: %v", err)
			}
		})
	}
}
