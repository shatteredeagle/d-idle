package launch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// Blizzard's official 1.14d patch. VERIFIED: served over plain HTTP only (HTTPS does not answer),
// supports Range requests, 6343200 bytes. The URL and hash are constants here, not downloaded data.
const (
	PatchURL    = "http://ftp.blizzard.com/pub/diablo2exp/patches/PC/LODPatch_114d.exe"
	PatchSHA256 = "5ee5b6f09c338324db14ea3ec03a4b8a9b36e4eea249bb12f1fb592558d6434d"
	PatchSize   = 6343200
)

// ErrChecksum is returned when a finished download does not match the expected hash. The bad file is
// deleted and never left under the final name.
var ErrChecksum = errors.New("checksum mismatch")

// Download describes one file to fetch.
type Download struct {
	URL    string
	Dest   string
	SHA256 string // lower-case hex, required
	Size   int64  // expected size, 0 = unknown
}

// Fetch downloads d into d.Dest. Existing Dest that verifies is kept untouched. A partial Dest+".part"
// is resumed with a Range request (if the server ignores Range the file restarts). The hash is checked
// on the complete file before the atomic rename; on mismatch the part file is removed and ErrChecksum
// is returned. Pass nil client for http.DefaultClient.
func Fetch(ctx context.Context, client *http.Client, d Download) error {
	if d.SHA256 == "" {
		return errors.New("refusing to download without an expected sha256")
	}
	if client == nil {
		client = http.DefaultClient
	}
	if ok, _ := fileMatches(d.Dest, d.SHA256); ok {
		return nil
	}
	part := d.Dest + ".part"
	var have int64
	if st, err := os.Stat(part); err == nil {
		have = st.Size()
	}
	if d.Size > 0 && have > d.Size {
		os.Remove(part)
		have = 0
	}
	if d.Size == 0 || have < d.Size {
		if err := fetchRange(ctx, client, d.URL, part, have); err != nil {
			return err
		}
	}
	ok, err := fileMatches(part, d.SHA256)
	if err != nil {
		return err
	}
	if !ok {
		os.Remove(part)
		return fmt.Errorf("%s: %w", d.URL, ErrChecksum)
	}
	return os.Rename(part, d.Dest)
}

func fetchRange(ctx context.Context, client *http.Client, url, part string, have int64) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	if have > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", have))
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	flags := os.O_CREATE | os.O_WRONLY
	switch resp.StatusCode {
	case http.StatusPartialContent:
		flags |= os.O_APPEND
	case http.StatusOK:
		flags |= os.O_TRUNC // server ignored Range: start over
	case http.StatusRequestedRangeNotSatisfiable:
		return nil // part is already complete; the hash decides
	default:
		return fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	f, err := os.OpenFile(part, flags, 0o644)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, resp.Body)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err // a cut connection leaves the part file for the next resume
}

func fileMatches(path, wantHex string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return false, err
	}
	return strings.EqualFold(hex.EncodeToString(h.Sum(nil)), wantHex), nil
}
