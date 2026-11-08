// Package selfupdate holds the signed release manifest (this file), and the
// updater built on it. Design: docs/research/self-update-design.md; the
// field names below are the contract of docs/self-update-release.md.
package selfupdate

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
)

const (
	// SchemaVersion is the only manifest schema this code understands.
	SchemaVersion = 1
	// MaxManifestBytes caps what a parser or fetcher reads.
	MaxManifestBytes = 1 << 20
	// ManifestName is the file name of the manifest in a release set and in an installed release.
	ManifestName = "MANIFEST.json"
)

// Compat are the protocol pairs a release was built against (design 3.4).
type Compat struct {
	BotAPI     int `json:"bot_api"`
	D2DCtl     int `json:"d2d_ctl"`
	DataSchema int `json:"data_schema"`
}

// Source names the corresponding source of a release (GPL offer, design 3.11).
type Source struct {
	Bot   string `json:"bot"`
	Fork  string `json:"d2d"`
	Offer string `json:"offer"`
}

// File is one release asset. Dest is the directory below the release dir
// the tarball unpacks into ("bin", "d2d", "data").
type File struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
	Dest   string `json:"dest"`
}

// Revoke is the signed revocation carried in a manifest. MinVersion is a
// monotonic floor: every version below it is refused.
type Revoke struct {
	Versions   []string `json:"versions"`
	MinVersion string   `json:"min_version,omitempty"`
}

// Manifest is MANIFEST.json. The signature covers its exact bytes.
type Manifest struct {
	Schema   int    `json:"schema"`
	Version  string `json:"version"`
	Seq      int64  `json:"seq"`
	Issued   string `json:"issued"`
	Channel  string `json:"channel"`
	Notes    string `json:"notes"`
	MinImage int    `json:"min_image"`
	Compat   Compat `json:"compat"`
	Source   Source `json:"source"`
	Files    []File `json:"files"`
	Revoke   Revoke `json:"revoke"`
}

// Revoked is the standalone REVOKED document, published and signed next to
// the latest manifest (REVOKED + REVOKED.sig).
type Revoked struct {
	Schema     int      `json:"schema"`
	Seq        int64    `json:"seq"`
	Versions   []string `json:"versions"`
	MinVersion string   `json:"min_version,omitempty"`
}

// Marshal renders the canonical bytes: files sorted by name, two-space
// indent, trailing newline. The same input always gives the same bytes, so
// the signature is reproducible.
func (m Manifest) Marshal() ([]byte, error) {
	c := m
	c.Files = append([]File(nil), m.Files...)
	sort.Slice(c.Files, func(i, j int) bool { return c.Files[i].Name < c.Files[j].Name })
	if c.Revoke.Versions == nil {
		c.Revoke.Versions = []string{}
	} else {
		c.Revoke.Versions = append([]string(nil), c.Revoke.Versions...)
		sort.Strings(c.Revoke.Versions)
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return marshalIndent(c)
}

// Marshal renders the canonical REVOKED bytes.
func (r Revoked) Marshal() ([]byte, error) {
	c := r
	c.Versions = append([]string{}, r.Versions...)
	sort.Strings(c.Versions)
	if c.Schema != SchemaVersion {
		return nil, fmt.Errorf("revoked: schema %d, want %d", c.Schema, SchemaVersion)
	}
	for _, v := range c.Versions {
		if _, err := ParseVersion(v); err != nil {
			return nil, fmt.Errorf("revoked: %w", err)
		}
	}
	if c.MinVersion != "" {
		if _, err := ParseVersion(c.MinVersion); err != nil {
			return nil, fmt.Errorf("revoked: min_version: %w", err)
		}
	}
	return marshalIndent(c)
}

func marshalIndent(v any) ([]byte, error) {
	var b bytes.Buffer
	e := json.NewEncoder(&b)
	e.SetEscapeHTML(false)
	e.SetIndent("", "  ")
	if err := e.Encode(v); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// ParseManifest decodes and validates manifest bytes. Unknown fields are an
// error: a field this code does not know may carry a rule it would skip.
func ParseManifest(b []byte) (Manifest, error) {
	var m Manifest
	if len(b) > MaxManifestBytes {
		return m, errors.New("manifest: too large")
	}
	if err := strictDecode(b, &m); err != nil {
		return Manifest{}, fmt.Errorf("manifest: %w", err)
	}
	if err := m.Validate(); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

// ParseRevoked decodes the REVOKED document.
func ParseRevoked(b []byte) (Revoked, error) {
	var r Revoked
	if len(b) > MaxManifestBytes {
		return r, errors.New("revoked: too large")
	}
	if err := strictDecode(b, &r); err != nil {
		return Revoked{}, fmt.Errorf("revoked: %w", err)
	}
	if _, err := r.Marshal(); err != nil {
		return Revoked{}, err
	}
	return r, nil
}

func strictDecode(b []byte, v any) error {
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return err
	}
	if d.More() {
		return errors.New("trailing data")
	}
	return nil
}

// Validate checks the schema and every field a consumer relies on.
func (m Manifest) Validate() error {
	if m.Schema != SchemaVersion {
		return fmt.Errorf("manifest: schema %d, want %d", m.Schema, SchemaVersion)
	}
	if _, err := ParseVersion(m.Version); err != nil {
		return fmt.Errorf("manifest: %w", err)
	}
	if m.Seq < 1 {
		return errors.New("manifest: seq must be >= 1")
	}
	if m.Issued == "" || m.Channel == "" {
		return errors.New("manifest: issued and channel are required")
	}
	if m.MinImage < 0 {
		return errors.New("manifest: min_image < 0")
	}
	if len(m.Files) == 0 {
		return errors.New("manifest: no files")
	}
	seen := map[string]bool{}
	for _, f := range m.Files {
		if err := f.validate(); err != nil {
			return fmt.Errorf("manifest: file %q: %w", f.Name, err)
		}
		if seen[f.Name] {
			return fmt.Errorf("manifest: duplicate file %q", f.Name)
		}
		seen[f.Name] = true
	}
	for _, v := range m.Revoke.Versions {
		if _, err := ParseVersion(v); err != nil {
			return fmt.Errorf("manifest: revoke: %w", err)
		}
	}
	if m.Revoke.MinVersion != "" {
		if _, err := ParseVersion(m.Revoke.MinVersion); err != nil {
			return fmt.Errorf("manifest: revoke.min_version: %w", err)
		}
	}
	return nil
}

func (f File) validate() error {
	if f.Name == "" || f.Name != path.Base(f.Name) || strings.ContainsAny(f.Name, `\`) || f.Name == "." || f.Name == ".." {
		return errors.New("name must be a plain file name")
	}
	if f.Dest == "" || path.IsAbs(f.Dest) || path.Clean(f.Dest) != f.Dest || f.Dest == ".." || strings.HasPrefix(f.Dest, "../") {
		return errors.New("dest must be a clean relative path")
	}
	if len(f.SHA256) != 64 {
		return errors.New("sha256 must be 64 hex characters")
	}
	if _, err := hex.DecodeString(f.SHA256); err != nil || f.SHA256 != strings.ToLower(f.SHA256) {
		return errors.New("sha256 must be lower-case hex")
	}
	if f.Size < 0 {
		return errors.New("negative size")
	}
	return nil
}

// Bans reports whether this revoke block bans v (listed, or below the floor).
func (r Revoke) Bans(v string) bool {
	for _, x := range r.Versions {
		if CompareVersions(x, v) == 0 {
			return true
		}
	}
	return r.MinVersion != "" && CompareVersions(v, r.MinVersion) < 0
}

// Bans reports whether the REVOKED document bans v.
func (r Revoked) Bans(v string) bool {
	return Revoke{Versions: r.Versions, MinVersion: r.MinVersion}.Bans(v)
}

// HashReader returns the sha256 hex and byte count of r.
func HashReader(r io.Reader) (string, int64, error) {
	h := sha256.New()
	n, err := io.Copy(h, r)
	if err != nil {
		return "", n, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

// VerifyFile checks one file on disk against its manifest entry.
func (f File) VerifyFile(dir string) error {
	fh, err := os.Open(path.Join(dir, f.Name))
	if err != nil {
		return err
	}
	defer fh.Close()
	sum, n, err := HashReader(fh)
	if err != nil {
		return err
	}
	if n != f.Size {
		return fmt.Errorf("%s: size %d, manifest says %d", f.Name, n, f.Size)
	}
	if sum != f.SHA256 {
		return fmt.Errorf("%s: sha256 mismatch", f.Name)
	}
	return nil
}

// VerifyFiles checks every asset of the manifest found in dir.
func (m Manifest) VerifyFiles(dir string) error {
	for _, f := range m.Files {
		if err := f.VerifyFile(dir); err != nil {
			return err
		}
	}
	return nil
}

// Version is a parsed x.y.z release number (no pre-release part).
type Version [3]int

// ParseVersion parses "1.2.3" strictly.
func ParseVersion(s string) (Version, error) {
	var v Version
	p := strings.Split(s, ".")
	if len(p) != 3 {
		return v, fmt.Errorf("version %q: want x.y.z", s)
	}
	for i, x := range p {
		n, err := strconv.Atoi(x)
		if err != nil || n < 0 || strconv.Itoa(n) != x {
			return v, fmt.Errorf("version %q: bad part %q", s, x)
		}
		v[i] = n
	}
	return v, nil
}

// CompareVersions returns -1, 0 or 1. An unparsable version sorts lowest
// (callers validate first; this never panics).
func CompareVersions(a, b string) int {
	va, ea := ParseVersion(a)
	vb, eb := ParseVersion(b)
	switch {
	case ea != nil && eb != nil:
		return 0
	case ea != nil:
		return -1
	case eb != nil:
		return 1
	}
	for i := range va {
		if va[i] != vb[i] {
			if va[i] < vb[i] {
				return -1
			}
			return 1
		}
	}
	return 0
}

// Key file format: a one-line header, then the base64 key. The private key
// is the 32-byte seed so a file cannot be mistaken for a public key.
const (
	privHeader = "d2sign-ed25519-private-v1"
	pubHeader  = "d2sign-ed25519-public-v1"
)

// GenerateKey makes a keypair from random (crypto/rand.Reader in production).
func GenerateKey(random io.Reader) (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return ed25519.GenerateKey(random)
}

// EncodePrivateKey renders the private key file body.
func EncodePrivateKey(k ed25519.PrivateKey) []byte {
	return []byte(privHeader + "\n" + base64.StdEncoding.EncodeToString(k.Seed()) + "\n")
}

// EncodePublicKey renders the public key file body.
func EncodePublicKey(k ed25519.PublicKey) []byte {
	return []byte(pubHeader + "\n" + base64.StdEncoding.EncodeToString(k) + "\n")
}

func decodeKey(b []byte, header string, size int) ([]byte, error) {
	lines := strings.Fields(string(b))
	if len(lines) != 2 || lines[0] != header {
		return nil, fmt.Errorf("key file: want header %q and one base64 line", header)
	}
	raw, err := base64.StdEncoding.DecodeString(lines[1])
	if err != nil || len(raw) != size {
		return nil, errors.New("key file: bad key bytes")
	}
	return raw, nil
}

// DecodePrivateKey parses EncodePrivateKey output. The error never echoes key bytes.
func DecodePrivateKey(b []byte) (ed25519.PrivateKey, error) {
	raw, err := decodeKey(b, privHeader, ed25519.SeedSize)
	if err != nil {
		return nil, err
	}
	return ed25519.NewKeyFromSeed(raw), nil
}

// DecodePublicKey parses EncodePublicKey output.
func DecodePublicKey(b []byte) (ed25519.PublicKey, error) {
	raw, err := decodeKey(b, pubHeader, ed25519.PublicKeySize)
	if err != nil {
		return nil, err
	}
	return ed25519.PublicKey(raw), nil
}

// Fingerprint is the first 16 hex characters of sha256(public key).
func Fingerprint(k ed25519.PublicKey) string {
	s := sha256.Sum256(k)
	return hex.EncodeToString(s[:8])
}

// Sign signs the exact bytes of doc and returns the base64 signature line.
func Sign(k ed25519.PrivateKey, doc []byte) []byte {
	return []byte(base64.StdEncoding.EncodeToString(ed25519.Sign(k, doc)) + "\n")
}

// ErrBadSignature is returned for every signature failure.
var ErrBadSignature = errors.New("signature does not verify")

// Verify checks sig (as written by Sign) over the exact bytes of doc against
// any of the trusted keys (current and next, design 3.3).
func Verify(keys []ed25519.PublicKey, doc, sig []byte) error {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(sig)))
	if err != nil || len(raw) != ed25519.SignatureSize {
		return ErrBadSignature
	}
	for _, k := range keys {
		if len(k) == ed25519.PublicKeySize && ed25519.Verify(k, doc, raw) {
			return nil
		}
	}
	return ErrBadSignature
}
