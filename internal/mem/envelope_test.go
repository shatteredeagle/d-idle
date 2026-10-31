package mem

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestEnvelopeSnapshotRoundTrip(t *testing.T) {
	s := NewSnapshot()
	s.Put(0x403000, []byte{1, 2, 3, 4})
	p := filepath.Join(t.TempDir(), "a.snap")
	h := NewHeader(KindPages)
	h.Char, h.Verified = "Testy", []string{"2026-10-28 by hand"}
	if err := s.SaveEnvelope(p, h); err != nil {
		t.Fatal(err)
	}
	gh, back, err := LoadSnapshotHeader(p)
	if err != nil || gh.Schema != EnvSchema || gh.Char != "Testy" || gh.Build != ExpectedBuild() || len(gh.Verified) != 1 {
		t.Fatalf("%+v %v", gh, err)
	}
	if v, err := U32(back, 0x403000); err != nil || v != 0x04030201 {
		t.Fatalf("%x %v", v, err)
	}
	// The plain loaders unwrap envelopes too.
	if _, err := LoadSnapshot(p); err != nil {
		t.Fatal(err)
	}
}

func TestEnvelopeLegacyAndErrors(t *testing.T) {
	s := NewSnapshot()
	s.Put(0x403000, []byte{9})
	var buf bytes.Buffer
	s.WriteTo(&buf)
	h, _, err := OpenEnvelope(bytes.NewReader(buf.Bytes()))
	if err != nil || h.Schema != 0 || h.Kind != KindPages {
		t.Fatalf("%+v %v", h, err)
	}
	h, _, err = OpenEnvelope(bytes.NewReader([]byte(`{"T":1}`)))
	if err != nil || h.Kind != KindTrace {
		t.Fatalf("%+v %v", h, err)
	}
	if _, _, err := OpenEnvelope(bytes.NewReader([]byte("garbage!"))); err == nil {
		t.Fatal("want error")
	}
	// newer schema is rejected
	var nb bytes.Buffer
	WriteEnvelope(&nb, NewHeader(KindTrace), GzipJSONL([]byte("{}\n")))
	b := nb.Bytes()
	b[4] = 9
	if _, _, err := OpenEnvelope(bytes.NewReader(b)); err == nil {
		t.Fatal("want schema error")
	}
	// kind mismatch
	nb.Reset()
	WriteEnvelope(&nb, NewHeader(KindTrace), GzipJSONL([]byte("{}\n")))
	if _, err := ReadSnapshot(&nb); err == nil {
		t.Fatal("want kind error")
	}
}
