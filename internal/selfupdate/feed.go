package selfupdate

import (
	"fmt"
	"net/url"
	"strings"
)

// A user-controlled release feed for a non-dev image (friend build, docs/research/friend-build.md section 5.4).
//
//	D2BOT_UPDATE_FEED   https://host[:port][/path] of a static directory in the shape of the GitHub releases API
//	                    (<feed>/repos/<repo>/releases, asset URLs on the same host). Empty = GitHub, as before.
//
// The feed only chooses WHERE the manifest comes from. It adds no trust: the keys stay the baked ones (the DEV key
// and D2BOT_UPDATE_PUBKEY are honoured only with DIDLE_DEV=1), so a feed that cannot sign with a baked key cannot
// make the client install anything. The allow-list is exactly the feed host (also on redirects), never a wildcard.

// FeedSource is a validated D2BOT_UPDATE_FEED.
type FeedSource struct{ URL string }

// FeedFromEnv returns nil when D2BOT_UPDATE_FEED is empty. A set but unusable value is an error: it must be https,
// have a host, and carry no credentials, query or fragment (a mistyped feed must not fall back to GitHub silently).
func FeedFromEnv(getenv func(string) string) (*FeedSource, error) {
	raw := strings.TrimRight(strings.TrimSpace(getenv("D2BOT_UPDATE_FEED")), "/")
	if raw == "" {
		return nil, nil
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" {
		return nil, fmt.Errorf("D2BOT_UPDATE_FEED %q: want https://host[:port][/path]", raw)
	}
	if u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("D2BOT_UPDATE_FEED %q: no credentials, query or fragment allowed", raw)
	}
	return &FeedSource{URL: raw}, nil
}

// Client builds a releases client that talks only to the feed host.
func (f *FeedSource) Client(userAgent string) *Client {
	host := ""
	if u, err := url.Parse(f.URL); err == nil {
		host = u.Hostname()
	}
	return &Client{BaseURL: f.URL, UserAgent: userAgent, AllowHost: func(h string) bool { return h == host }}
}
