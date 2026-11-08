package selfupdate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DefaultRepo owns the app releases (design decision 2).
const DefaultRepo = "shatteredeagle/d-idle"

// TagPrefix marks app releases among the repo's other releases (tester image tags use a different prefix).
const TagPrefix = "app-v"

var (
	ErrOffline     = errors.New("update check is switched off (D2BOT_UPDATE_CHECK=off)")
	ErrRateLimited = errors.New("GitHub rate limit reached; will retry later")
	ErrNoRelease   = errors.New("no app release found")
)

// Client talks to the GitHub releases API. Outbound calls go only to hosts AllowHost accepts (design 3.5); the
// tests point BaseURL at an httptest server and widen AllowHost.
type Client struct {
	Repo      string
	BaseURL   string // default https://api.github.com
	HTTP      *http.Client
	UserAgent string // didle-updater/<version>
	Offline   bool
	AllowHost func(host string) bool
}

// Remote is what a check found.
type Remote struct {
	Tag                    string
	Manifest               Manifest
	Raw, Sig               []byte            // the exact manifest bytes and its signature
	Revoked                *Revoked          // set (empty) when the release carries a REVOKED file; Engine.Check fills it after verifying
	assets                 map[string]string // name -> download URL
	revokedRaw, revokedSig []byte
	localDir               string // set for an inbox release: files are copied, not fetched
	NotModified            bool
}

type ghRelease struct {
	Tag        string `json:"tag_name"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
	Assets     []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
		Size int64  `json:"size"`
	} `json:"assets"`
}

func defaultAllow(host string) bool {
	return host == "api.github.com" || host == "github.com" || strings.HasSuffix(host, ".githubusercontent.com")
}

func (c *Client) httpc() *http.Client {
	hc := c.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 5 * time.Minute}
	}
	cp := *hc
	allow := c.AllowHost
	if allow == nil {
		allow = defaultAllow
	}
	cp.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) > 5 {
			return errors.New("too many redirects")
		}
		if !allow(req.URL.Hostname()) {
			return fmt.Errorf("redirect to %s refused", req.URL.Hostname())
		}
		return nil
	}
	return &cp
}

func (c *Client) get(ctx context.Context, raw string, hdr map[string]string) (*http.Response, error) {
	if c.Offline {
		return nil, ErrOffline
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	allow := c.AllowHost
	if allow == nil {
		allow = defaultAllow
	}
	if !allow(u.Hostname()) {
		return nil, fmt.Errorf("host %s is not allowed", u.Hostname())
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return nil, err
	}
	ua := c.UserAgent
	if ua == "" {
		ua = "didle-updater/unknown"
	}
	req.Header.Set("User-Agent", ua)
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	resp, err := c.httpc().Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusTooManyRequests || (resp.StatusCode == http.StatusForbidden && resp.Header.Get("X-RateLimit-Remaining") == "0") {
		resp.Body.Close()
		return nil, ErrRateLimited
	}
	return resp, nil
}

func readCapped(r io.Reader, n int64) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, n+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > n {
		return nil, errors.New("response too large")
	}
	return b, nil
}

// Latest asks the releases API (If-None-Match etag) for the highest app release and downloads its manifest, the
// signature and REVOKED if present. It does NOT verify anything: Engine.Check does, with the pinned keys. A 304
// returns Remote{NotModified: true}. newETag is the tag to store.
func (c *Client) Latest(ctx context.Context, etag string) (r Remote, newETag string, err error) {
	base := c.BaseURL
	if base == "" {
		base = "https://api.github.com"
	}
	repo := c.Repo
	if repo == "" {
		repo = DefaultRepo
	}
	hdr := map[string]string{"Accept": "application/vnd.github+json"}
	if etag != "" {
		hdr["If-None-Match"] = etag
	}
	resp, err := c.get(ctx, base+"/repos/"+repo+"/releases?per_page=10", hdr)
	if err != nil {
		return r, etag, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusNotModified:
		return Remote{NotModified: true}, etag, nil
	case http.StatusOK:
	default:
		return r, etag, fmt.Errorf("releases API: HTTP %d", resp.StatusCode)
	}
	newETag = resp.Header.Get("ETag")
	body, err := readCapped(resp.Body, 4<<20)
	if err != nil {
		return r, etag, err
	}
	var rels []ghRelease
	if err := json.Unmarshal(body, &rels); err != nil {
		return r, etag, fmt.Errorf("releases API: %w", err)
	}
	var best *ghRelease
	for i := range rels {
		x := &rels[i]
		if x.Draft || x.Prerelease || !strings.HasPrefix(x.Tag, TagPrefix) {
			continue
		}
		if _, err := ParseVersion(strings.TrimPrefix(x.Tag, TagPrefix)); err != nil {
			continue
		}
		if best == nil || CompareVersions(strings.TrimPrefix(x.Tag, TagPrefix), strings.TrimPrefix(best.Tag, TagPrefix)) > 0 {
			best = x
		}
	}
	if best == nil {
		return r, newETag, ErrNoRelease
	}
	r.Tag = best.Tag
	r.assets = map[string]string{}
	for _, a := range best.Assets {
		r.assets[a.Name] = a.URL
	}
	if r.Raw, err = c.asset(ctx, r, "MANIFEST.json", MaxManifestBytes); err != nil {
		return r, etag, err
	}
	if r.Sig, err = c.asset(ctx, r, "MANIFEST.json.sig", 4096); err != nil {
		return r, etag, err
	}
	if _, ok := r.assets["REVOKED"]; ok {
		raw, err := c.asset(ctx, r, "REVOKED", MaxManifestBytes)
		if err != nil {
			return r, etag, err
		}
		sig, err := c.asset(ctx, r, "REVOKED.sig", 4096)
		if err != nil {
			return r, etag, err
		}
		r.Revoked = &Revoked{}
		r.revokedRaw, r.revokedSig = raw, sig
	}
	return r, newETag, nil
}

func (c *Client) asset(ctx context.Context, r Remote, name string, limit int64) ([]byte, error) {
	u, ok := r.assets[name]
	if !ok {
		return nil, fmt.Errorf("release %s has no asset %s", r.Tag, name)
	}
	resp, err := c.get(ctx, u, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: HTTP %d", name, resp.StatusCode)
	}
	return readCapped(resp.Body, limit)
}

// Download fetches the manifest file f of r into dir (stream, capped at f.Size so a lying server cannot fill the
// disk). Hash and size are checked by the caller with File.VerifyFile.
func (c *Client) Download(ctx context.Context, r Remote, f File, dir string, progress func(n int64)) error {
	if r.localDir != "" {
		return copyLocal(filepath.Join(r.localDir, f.Name), dir, f, progress)
	}
	u, ok := r.assets[f.Name]
	if !ok {
		return fmt.Errorf("release %s has no asset %s", r.Tag, f.Name)
	}
	resp, err := c.get(ctx, u, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: HTTP %d", f.Name, resp.StatusCode)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(filepath.Join(dir, f.Name), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	n, err := io.Copy(out, &countReader{r: io.LimitReader(resp.Body, f.Size+1), on: progress})
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	if n > f.Size {
		return fmt.Errorf("%s: larger than the manifest says", f.Name)
	}
	return nil
}

type countReader struct {
	r  io.Reader
	n  int64
	on func(int64)
}

func (c *countReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	if c.on != nil && n > 0 {
		c.on(c.n)
	}
	return n, err
}

func copyLocal(src, dir string, f File, progress func(int64)) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(filepath.Join(dir, f.Name), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	n, err := io.Copy(out, &countReader{r: io.LimitReader(in, f.Size+1), on: progress})
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err == nil && n > f.Size {
		err = fmt.Errorf("%s: larger than the manifest says", f.Name)
	}
	return err
}

// LoadInbox reads a release set dropped into dir (MANIFEST.json, MANIFEST.json.sig, optional REVOKED and
// REVOKED.sig, and the assets). Nothing is verified here.
func LoadInbox(dir string) (Remote, error) {
	var r Remote
	var err error
	if r.Raw, err = os.ReadFile(filepath.Join(dir, "MANIFEST.json")); err != nil {
		return r, fmt.Errorf("inbox: %w", err)
	}
	if r.Sig, err = os.ReadFile(filepath.Join(dir, "MANIFEST.json.sig")); err != nil {
		return r, fmt.Errorf("inbox: %w", err)
	}
	if len(r.Raw) > MaxManifestBytes {
		return r, errors.New("inbox: manifest too large")
	}
	if raw, err := os.ReadFile(filepath.Join(dir, "REVOKED")); err == nil {
		sig, err := os.ReadFile(filepath.Join(dir, "REVOKED.sig"))
		if err != nil {
			return r, fmt.Errorf("inbox: %w", err)
		}
		r.revokedRaw, r.revokedSig = raw, sig
	}
	r.localDir, r.Tag = dir, "inbox"
	return r, nil
}
