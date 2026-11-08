package selfupdate

import (
	"crypto/ed25519"
	"fmt"
	"net/url"
	"os"
	"strings"
)

// The dev-loop update source (TASK-192, docs/dev-update.md). A DEV image (build arg DEV=1, which sets DIDLE_DEV=1)
// may be pointed at a plain HTTP directory on the LAN that mimics the GitHub releases API, and may trust one extra
// public key. Without DIDLE_DEV=1 none of this exists: the variables are ignored and the client talks to GitHub
// with the baked keys only, as before.
//
//	DIDLE_DEV=1            switches the dev source on (set by the image, not by the user)
//	D2BOT_UPDATE_URL       base URL of the directory; the client asks <URL>/repos/<repo>/releases like the GitHub API
//	D2BOT_UPDATE_PUBKEY    a d2sign public key: the key text itself, or the path of a .pub file
//	D2BOT_UPDATE_CHECK     on (default) | off | manual (no daily check; only an explicit check-now asks)

// CheckMode is D2BOT_UPDATE_CHECK.
type CheckMode int

const (
	CheckOn     CheckMode = iota // daily check (the default)
	CheckOff                     // no outbound call at all
	CheckManual                  // no daily check; an explicit request still asks
)

func (m CheckMode) String() string {
	return [...]string{"on", "off", "manual"}[m]
}

// ParseCheckMode reads D2BOT_UPDATE_CHECK. The empty string is on; the old off spellings (0, false, no) stay off.
func ParseCheckMode(s string) CheckMode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "off", "0", "false", "no":
		return CheckOff
	case "manual":
		return CheckManual
	}
	return CheckOn
}

// DevSource is the optional LAN source of a DEV image.
type DevSource struct {
	URL  string
	Keys []ed25519.PublicKey
}

// DevFromEnv returns nil unless DIDLE_DEV=1 and D2BOT_UPDATE_URL is set. A set but unusable URL or key is an error
// (a dev who typed it wrongly must be told, not silently sent to GitHub).
func DevFromEnv(getenv func(string) string) (*DevSource, error) {
	if getenv("DIDLE_DEV") != "1" {
		return nil, nil
	}
	raw := strings.TrimRight(strings.TrimSpace(getenv("D2BOT_UPDATE_URL")), "/")
	if raw == "" {
		return nil, nil
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return nil, fmt.Errorf("D2BOT_UPDATE_URL %q: want http(s)://host[:port]", raw)
	}
	d := &DevSource{URL: raw}
	if pk := strings.TrimSpace(getenv("D2BOT_UPDATE_PUBKEY")); pk != "" {
		k, err := ParseDevKey(pk)
		if err != nil {
			return nil, fmt.Errorf("D2BOT_UPDATE_PUBKEY: %w", err)
		}
		d.Keys = []ed25519.PublicKey{k}
	}
	return d, nil
}

// ParseDevKey accepts the key text (header line plus base64, newlines may be written as \n) or a file path.
func ParseDevKey(v string) (ed25519.PublicKey, error) {
	if strings.HasPrefix(v, pubHeader) {
		v = strings.ReplaceAll(v, `\n`, "\n")
		return DecodePublicKey([]byte(v))
	}
	b, err := os.ReadFile(v)
	if err != nil {
		return nil, fmt.Errorf("neither key text nor a readable file: %w", err)
	}
	return DecodePublicKey(b)
}

// Client builds a releases client that talks only to the dev host (also on redirects).
func (d *DevSource) Client(userAgent string) *Client {
	host := ""
	if u, err := url.Parse(d.URL); err == nil {
		host = u.Hostname()
	}
	return &Client{BaseURL: d.URL, UserAgent: userAgent, AllowHost: func(h string) bool { return h == host }}
}
