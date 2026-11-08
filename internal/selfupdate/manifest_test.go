package selfupdate

import (
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sample() Manifest {
	return Manifest{
		Schema: SchemaVersion, Version: "0.4.0", Seq: 17, Issued: "2026-11-20T00:00:00Z",
		Channel: "stable", Notes: "n", MinImage: 3,
		Compat: Compat{BotAPI: 4, D2DCtl: 2, DataSchema: 5},
		Source: Source{Bot: "app-v0.4.0", Fork: "didle-gate2-lab3", Offer: "SOURCE.txt"},
		Files: []File{
			{Name: "b.tar.gz", SHA256: strings.Repeat("ab", 32), Size: 2, Dest: "d2d"},
			{Name: "a.tar.gz", SHA256: strings.Repeat("cd", 32), Size: 1, Dest: "bin"},
		},
	}
}

func TestMarshalDeterministicAndSorted(t *testing.T) {
	a, err := sample().Marshal()
	if err != nil {
		t.Fatal(err)
	}
	m2 := sample()
	m2.Files[0], m2.Files[1] = m2.Files[1], m2.Files[0]
	b, _ := m2.Marshal()
	if string(a) != string(b) {
		t.Fatal("file order changed the bytes")
	}
	if strings.Index(string(a), "a.tar.gz") > strings.Index(string(a), "b.tar.gz") {
		t.Fatal("files not sorted")
	}
	got, err := ParseManifest(a)
	if err != nil || got.Version != "0.4.0" || len(got.Files) != 2 {
		t.Fatalf("round trip: %v %+v", err, got)
	}
}

func TestValidateRejects(t *testing.T) {
	cases := map[string]func(*Manifest){
		"schema":   func(m *Manifest) { m.Schema = 2 },
		"version":  func(m *Manifest) { m.Version = "1.2" },
		"leading0": func(m *Manifest) { m.Version = "1.02.3" },
		"seq":      func(m *Manifest) { m.Seq = 0 },
		"nofiles":  func(m *Manifest) { m.Files = nil },
		"traverse": func(m *Manifest) { m.Files[0].Name = "../x" },
		"dest":     func(m *Manifest) { m.Files[0].Dest = "../x" },
		"absdest":  func(m *Manifest) { m.Files[0].Dest = "/etc" },
		"hash":     func(m *Manifest) { m.Files[0].SHA256 = "zz" },
		"upper":    func(m *Manifest) { m.Files[0].SHA256 = strings.Repeat("AB", 32) },
		"dup":      func(m *Manifest) { m.Files[1].Name = m.Files[0].Name },
		"revoke":   func(m *Manifest) { m.Revoke.Versions = []string{"x"} },
	}
	for name, mut := range cases {
		m := sample()
		mut(&m)
		if _, err := m.Marshal(); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestParseStrict(t *testing.T) {
	b, _ := sample().Marshal()
	if _, err := ParseManifest([]byte(strings.Replace(string(b), `"seq"`, `"extra": 1, "seq"`, 1))); err == nil {
		t.Error("unknown field accepted")
	}
	if _, err := ParseManifest(append(b, []byte(`{}`)...)); err == nil {
		t.Error("trailing data accepted")
	}
}

func TestSignVerify(t *testing.T) {
	pub, priv, _ := GenerateKey(rand.Reader)
	pub2, _, _ := GenerateKey(rand.Reader)
	doc, _ := sample().Marshal()
	sig := Sign(priv, doc)
	if err := Verify([]ed25519.PublicKey{pub2, pub}, doc, sig); err != nil {
		t.Fatal(err) // second trusted key (rotation) must work
	}
	if err := Verify([]ed25519.PublicKey{pub2}, doc, sig); err == nil {
		t.Error("wrong key accepted")
	}
	if err := Verify([]ed25519.PublicKey{pub}, append(doc, ' '), sig); err == nil {
		t.Error("changed bytes accepted")
	}
	if err := Verify([]ed25519.PublicKey{pub}, doc, []byte("not base64!")); err == nil {
		t.Error("garbage signature accepted")
	}
	if string(Sign(priv, doc)) != string(sig) {
		t.Error("signing is not deterministic")
	}
}

func TestKeyFiles(t *testing.T) {
	pub, priv, _ := GenerateKey(rand.Reader)
	p2, err := DecodePrivateKey(EncodePrivateKey(priv))
	if err != nil || !p2.Equal(priv) {
		t.Fatal("private round trip")
	}
	q, err := DecodePublicKey(EncodePublicKey(pub))
	if err != nil || !q.Equal(pub) {
		t.Fatal("public round trip")
	}
	if _, err := DecodePublicKey(EncodePrivateKey(priv)); err == nil {
		t.Error("private file read as public")
	}
	if _, err := DecodePrivateKey([]byte(privHeader + "\nAAAA\n")); err == nil || strings.Contains(err.Error(), "AAAA") {
		t.Errorf("bad key error: %v", err)
	}
}

func TestVersionsAndRevoke(t *testing.T) {
	if CompareVersions("0.10.0", "0.9.9") != 1 || CompareVersions("1.0.0", "1.0.0") != 0 || CompareVersions("0.3.1", "0.4.0") != -1 {
		t.Error("compare")
	}
	if CompareVersions("junk", "0.0.1") != -1 {
		t.Error("junk should sort lowest")
	}
	r := Revoke{Versions: []string{"0.3.1"}, MinVersion: "0.3.0"}
	if !r.Bans("0.3.1") || !r.Bans("0.2.9") || r.Bans("0.3.2") {
		t.Error("bans")
	}
	b, err := Revoked{Schema: 1, Seq: 2, Versions: []string{"0.3.1"}}.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if got, err := ParseRevoked(b); err != nil || !got.Bans("0.3.1") {
		t.Fatal("revoked round trip")
	}
}

func TestVerifyFiles(t *testing.T) {
	dir := t.TempDir()
	content := []byte("hello")
	os.WriteFile(filepath.Join(dir, "a.tar.gz"), content, 0o644)
	h, n, _ := HashReader(strings.NewReader("hello"))
	f := File{Name: "a.tar.gz", SHA256: h, Size: n, Dest: "bin"}
	if err := f.VerifyFile(dir); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "a.tar.gz"), []byte("hellO"), 0o644)
	if err := f.VerifyFile(dir); err == nil {
		t.Error("tampered file accepted")
	}
	os.WriteFile(filepath.Join(dir, "a.tar.gz"), []byte("hello!"), 0o644)
	if err := f.VerifyFile(dir); err == nil {
		t.Error("wrong size accepted")
	}
}
