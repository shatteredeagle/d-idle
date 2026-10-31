package launch

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFetchResumeAndVerify(t *testing.T) {
	data := bytes.Repeat([]byte("diablo"), 5000)
	sum := sha256.Sum256(data)
	want := hex.EncodeToString(sum[:])
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "p.exe", time.Time{}, bytes.NewReader(data))
	}))
	defer srv.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "p.exe")
	// a truncated earlier attempt
	os.WriteFile(dest+".part", data[:1234], 0o644)
	d := Download{URL: srv.URL, Dest: dest, SHA256: want, Size: int64(len(data))}
	if err := Fetch(context.Background(), nil, d); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(dest)
	if !bytes.Equal(got, data) {
		t.Fatal("content differs after resume")
	}
	if _, err := os.Stat(dest + ".part"); err == nil {
		t.Fatal("part file left behind")
	}
	// second call is a no-op even with the server gone
	srv.Close()
	if err := Fetch(context.Background(), nil, d); err != nil {
		t.Fatal(err)
	}
}

func TestFetchRefusesMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("tampered"))
	}))
	defer srv.Close()
	dest := filepath.Join(t.TempDir(), "p.exe")
	err := Fetch(context.Background(), nil, Download{URL: srv.URL, Dest: dest, SHA256: PatchSHA256})
	if !errors.Is(err, ErrChecksum) {
		t.Fatalf("err = %v", err)
	}
	for _, f := range []string{dest, dest + ".part"} {
		if _, err := os.Stat(f); err == nil {
			t.Fatalf("%s exists after mismatch", f)
		}
	}
	if err := Fetch(context.Background(), nil, Download{URL: srv.URL, Dest: dest}); err == nil {
		t.Fatal("must refuse a download with no hash")
	}
}
