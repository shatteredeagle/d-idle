package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func do(t *testing.T, want int, args ...string) string {
	t.Helper()
	var out, errb bytes.Buffer
	if got := run(args, &out, &errb); got != want {
		t.Fatalf("d2sign %v: exit %d, want %d: %s %s", args, got, want, out.String(), errb.String())
	}
	return out.String() + errb.String()
}

func release(t *testing.T) (dir, keys string) {
	t.Helper()
	dir, keys = t.TempDir(), t.TempDir()
	os.WriteFile(filepath.Join(dir, "bin.tar.gz"), []byte("bin"), 0o644)
	os.WriteFile(filepath.Join(dir, "d2d.tar.gz"), []byte("d2d"), 0o644)
	do(t, 0, "keygen", "-dir", keys)
	do(t, 0, "manifest", "-dir", dir, "-version", "0.4.0", "-seq", "3", "-issued", "2026-11-20T00:00:00Z",
		"-min-image", "1", "-bot-api", "4", "-d2d-ctl", "2", "-data-schema", "5", "-src-bot", "app-v0.4.0", "-src-d2d", "tag",
		"-file", "bin.tar.gz=bin", "-file", "d2d.tar.gz=d2d", "-out", filepath.Join(dir, "MANIFEST.json"))
	return dir, keys
}

func TestKeygenModesAndNoPrivateOutput(t *testing.T) {
	keys := t.TempDir()
	out := do(t, 0, "keygen", "-dir", keys)
	st, _ := os.Stat(filepath.Join(keys, "d2sign.key"))
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("private key mode %v", st.Mode().Perm())
	}
	priv, _ := os.ReadFile(filepath.Join(keys, "d2sign.key"))
	body := strings.Fields(string(priv))[1]
	if strings.Contains(out, body) {
		t.Fatal("private key printed")
	}
	do(t, 1, "keygen", "-dir", keys) // never overwrites
}

func TestSignVerifyEndToEnd(t *testing.T) {
	dir, keys := release(t)
	m := filepath.Join(dir, "MANIFEST.json")
	do(t, 0, "sign", "-key", filepath.Join(keys, "d2sign.key"), "-in", m)
	out := do(t, 0, "verify", "-pub", filepath.Join(keys, "d2sign.pub"), "-in", m, "-dir", dir)
	if !strings.Contains(out, "2 file hashes") {
		t.Fatal(out)
	}
	// signing twice gives the same bytes
	first, _ := os.ReadFile(m + ".sig")
	do(t, 0, "sign", "-key", filepath.Join(keys, "d2sign.key"), "-in", m)
	second, _ := os.ReadFile(m + ".sig")
	if !bytes.Equal(first, second) {
		t.Fatal("signature not reproducible")
	}
	// tampered asset
	os.WriteFile(filepath.Join(dir, "bin.tar.gz"), []byte("BIN"), 0o644)
	do(t, 1, "verify", "-pub", filepath.Join(keys, "d2sign.pub"), "-in", m, "-dir", dir)
	// tampered manifest
	b, _ := os.ReadFile(m)
	os.WriteFile(m, bytes.Replace(b, []byte("0.4.0"), []byte("0.4.1"), 1), 0o644)
	do(t, 1, "verify", "-pub", filepath.Join(keys, "d2sign.pub"), "-in", m)
}

func TestSignRefusals(t *testing.T) {
	dir, keys := release(t)
	key := filepath.Join(keys, "d2sign.key")
	os.Chmod(key, 0o644)
	do(t, 1, "sign", "-key", key, "-in", filepath.Join(dir, "MANIFEST.json"))
	os.Chmod(key, 0o600)
	do(t, 1, "sign", "-key", key, "-in", filepath.Join(dir, "bin.tar.gz")) // not a manifest
	do(t, 1, "sign", "-in", filepath.Join(dir, "MANIFEST.json"))           // no key
}

func TestSignKeyEnvAndRevoked(t *testing.T) {
	dir, keys := release(t)
	b, _ := os.ReadFile(filepath.Join(keys, "d2sign.key"))
	t.Setenv("D2SIGN_TEST_KEY", string(b))
	rev := filepath.Join(dir, "REVOKED")
	do(t, 0, "revoked", "-seq", "2", "-revoke", "0.3.1", "-min-version", "0.3.0", "-out", rev)
	do(t, 0, "sign", "-key-env", "D2SIGN_TEST_KEY", "-in", rev)
	do(t, 0, "verify", "-pub", filepath.Join(keys, "d2sign.pub"), "-in", rev)
	if s := do(t, 0, "summary", "-in", rev); !strings.Contains(s, "0.3.1") {
		t.Fatal(s)
	}
	if s := do(t, 0, "summary", "-in", filepath.Join(dir, "MANIFEST.json")); !strings.Contains(s, "bin.tar.gz") || !strings.Contains(s, "sha256") {
		t.Fatal(s)
	}
}

func TestUsage(t *testing.T) {
	do(t, 2)
	do(t, 2, "nonsense")
}
