package main

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"d2lod-bot/internal/selfupdate"
)

// ghAsset and ghRelease are the parts of the GitHub releases API the client reads (internal/selfupdate/fetch.go).
type ghAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	Size int64  `json:"size"`
}

type ghRelease struct {
	Tag    string    `json:"tag_name"`
	Draft  bool      `json:"draft"`
	Pre    bool      `json:"prerelease"`
	Assets []ghAsset `json:"assets"`
}

func newFlags(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func loadKeys(list string) ([]ed25519.PublicKey, error) {
	if list == "" {
		return nil, errors.New("-pub is required (the pinned public key file)")
	}
	var keys []ed25519.PublicKey
	for _, p := range strings.Split(list, ",") {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		k, err := selfupdate.DecodePublicKey(b)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		keys = append(keys, k)
	}
	return keys, nil
}

func listPath(out, repo string) string {
	return filepath.Join(out, "repos", filepath.FromSlash(repo), "releases")
}

func loadList(path string) ([]ghRelease, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var l []ghRelease
	if err := json.Unmarshal(b, &l); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return l, nil
}

// saveList writes the list atomically (clients never see a half-written file) and newest first.
func saveList(path string, l []ghRelease) error {
	sort.SliceStable(l, func(i, j int) bool {
		return selfupdate.CompareVersions(strings.TrimPrefix(l[i].Tag, selfupdate.TagPrefix), strings.TrimPrefix(l[j].Tag, selfupdate.TagPrefix)) > 0
	})
	b, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func checkBaseURL(u string) (string, error) {
	u = strings.TrimRight(u, "/")
	if !strings.HasPrefix(u, "https://") || strings.ContainsAny(u, "?#@ ") {
		return "", fmt.Errorf("-base-url %q: want https://host[:port][/path] (the address clients use; D2BOT_UPDATE_FEED)", u)
	}
	return u, nil
}

// copyIdem copies src to dst unless dst already has the same bytes. A different existing file is an error: a
// published asset is immutable (clients may hold its hash).
func copyIdem(src, dst string) error {
	sb, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if db, err := os.ReadFile(dst); err == nil {
		if bytes.Equal(sb, db) {
			return nil
		}
		return fmt.Errorf("%s already published with different bytes", dst)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmp := dst + ".tmp"
	if err := os.WriteFile(tmp, sb, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, dst)
}

func cmdBuild(args []string, out io.Writer) error {
	fs := newFlags("build")
	rel := fs.String("release", "", "directory with MANIFEST.json, MANIFEST.json.sig and the assets")
	pubs := fs.String("pub", "", "pinned public key file(s), comma separated")
	base := fs.String("base-url", "", "https URL the feed is served at")
	outDir := fs.String("out", "", "feed tree directory (created or extended)")
	repo := fs.String("repo", "", "owner/name in the API path (default "+selfupdate.DefaultRepo+")")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *rel == "" || *outDir == "" || *base == "" {
		return errors.New("-release, -base-url and -out are required")
	}
	keys, err := loadKeys(*pubs)
	if err != nil {
		return err
	}
	baseURL, err := checkBaseURL(*base)
	if err != nil {
		return err
	}
	rp := repoOrDefault(*repo)

	raw, err := os.ReadFile(filepath.Join(*rel, "MANIFEST.json"))
	if err != nil {
		return err
	}
	sig, err := os.ReadFile(filepath.Join(*rel, "MANIFEST.json.sig"))
	if err != nil {
		return fmt.Errorf("unsigned release refused: %w (sign with d2sign first)", err)
	}
	if err := selfupdate.Verify(keys, raw, sig); err != nil {
		return fmt.Errorf("manifest: %w", err)
	}
	m, err := selfupdate.ParseManifest(raw)
	if err != nil {
		return err
	}
	if err := m.Validate(); err != nil {
		return err
	}
	if err := m.VerifyFiles(*rel); err != nil {
		return err
	}
	names := []string{"MANIFEST.json", "MANIFEST.json.sig"}
	for _, f := range m.Files {
		names = append(names, f.Name)
	}
	if rv, err := os.ReadFile(filepath.Join(*rel, "REVOKED")); err == nil {
		rsig, err := os.ReadFile(filepath.Join(*rel, "REVOKED.sig"))
		if err != nil {
			return fmt.Errorf("REVOKED without REVOKED.sig refused: %w", err)
		}
		if err := selfupdate.Verify(keys, rv, rsig); err != nil {
			return fmt.Errorf("REVOKED: %w", err)
		}
		if _, err := selfupdate.ParseRevoked(rv); err != nil {
			return err
		}
		names = append(names, "REVOKED", "REVOKED.sig")
	}
	if _, err := os.Stat(filepath.Join(*rel, "SOURCE.txt")); err == nil {
		names = append(names, "SOURCE.txt")
	}

	tag := selfupdate.TagPrefix + m.Version
	list, err := loadList(listPath(*outDir, rp))
	if err != nil {
		return err
	}
	for _, r := range list {
		if r.Tag == tag {
			continue
		}
		if selfupdate.CompareVersions(strings.TrimPrefix(r.Tag, selfupdate.TagPrefix), m.Version) > 0 {
			return fmt.Errorf("%s is older than the published %s; clients ignore it. Publish a higher version (withdraw a bad one instead)", tag, r.Tag)
		}
	}
	// assets first, the list last: a client never sees a list that points at missing files
	r := ghRelease{Tag: tag}
	for _, n := range names {
		if err := copyIdem(filepath.Join(*rel, n), filepath.Join(*outDir, "dl", tag, n)); err != nil {
			return err
		}
		st, _ := os.Stat(filepath.Join(*rel, n))
		r.Assets = append(r.Assets, ghAsset{Name: n, URL: baseURL + "/dl/" + tag + "/" + n, Size: st.Size()})
	}
	replaced := false
	for i := range list {
		if list[i].Tag == tag {
			// same version: only an identical manifest is idempotent; the copy above already refused a different one
			r.Draft = list[i].Draft
			list[i], replaced = r, true
		}
	}
	if !replaced {
		list = append(list, r)
	}
	if err := saveList(listPath(*outDir, rp), list); err != nil {
		return err
	}
	fmt.Fprintf(out, "published %s (seq %d, channel %s, %d files); %d release(s) listed\n", tag, m.Seq, m.Channel, len(names), len(list))
	return nil
}

func cmdWithdraw(args []string, out io.Writer) error {
	fs := newFlags("withdraw")
	outDir := fs.String("out", "", "feed tree directory")
	ver := fs.String("version", "", "version to hide, x.y.z")
	repo := fs.String("repo", "", "owner/name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *outDir == "" || *ver == "" {
		return errors.New("-out and -version are required")
	}
	p := listPath(*outDir, repoOrDefault(*repo))
	list, err := loadList(p)
	if err != nil {
		return err
	}
	tag := selfupdate.TagPrefix + *ver
	found := false
	for i := range list {
		if list[i].Tag == tag {
			list[i].Draft, found = true, true
		}
	}
	if !found {
		return fmt.Errorf("%s is not in the feed", tag)
	}
	if err := saveList(p, list); err != nil {
		return err
	}
	fmt.Fprintf(out, "withdrew %s: clients now see the next highest listed release (installs that already took it need a signed REVOKED or a higher version)\n", tag)
	return nil
}
