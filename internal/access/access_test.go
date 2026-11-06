package access

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func init() { Iterations = 1000 }

var t0 = time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)

func TestPasscodeRules(t *testing.T) {
	for p, ok := range map[string]bool{"": false, "12345": false, "123456": true, "correct horse battery": true, "aaaaaa": true, "ünïcö": false, "ünïcöd": true, strings.Repeat("x", 129): false} {
		if (ValidatePasscode(p) == nil) != ok {
			t.Errorf("ValidatePasscode(%q) ok=%v, want %v", p, !ok, ok)
		}
	}
}

func TestHashVerifyAndSession(t *testing.T) {
	r, err := NewPasscode("four plain words", t0)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(r.Hash+r.Salt, "four") || r.Iter != 1000 || r.Kind != KindPasscode || r.SetAt != "2026-11-06T12:00:00Z" {
		t.Errorf("record %+v", r)
	}
	if !r.Verify("four plain words") || r.Verify("four plain word") || r.Verify("") || r.Verify(strings.Repeat("x", 500)) {
		t.Error("verify wrong")
	}
	r2, _ := NewPasscode("four plain words", t0)
	if r2.Salt == r.Salt || r2.Hash == r.Hash {
		t.Error("salt is not per record")
	}
	if !r.SessionOK(r.Session()) || r.SessionOK("") || r.SessionOK(r2.Session()) {
		t.Error("session cookie check wrong")
	}
	if TokenRecord(t0).Verify("anything") || TokenRecord(t0).Session() != "" {
		t.Error("token record must never verify a passcode")
	}
}

func TestStoreLifecycle(t *testing.T) {
	p := filepath.Join(t.TempDir(), "d", "access.json")
	s := NewStore(p)
	if s.Current().Kind != "" {
		t.Fatal("missing file must be empty")
	}
	r, _ := NewPasscode("secret one", t0)
	if err := s.Save(r); err != nil {
		t.Fatal(err)
	}
	if st, _ := os.Stat(p); st.Mode().Perm() != 0o600 {
		t.Errorf("mode %v", st.Mode().Perm())
	}
	if raw, _ := os.ReadFile(p); strings.Contains(string(raw), "secret one") {
		t.Error("passcode in file")
	}
	if !s.Current().Verify("secret one") {
		t.Error("not read back")
	}
	// another writer (the reset command) is seen without a restart
	other := NewStore(p)
	if err := other.Save(TokenRecord(t0)); err != nil {
		t.Fatal(err)
	}
	if s.Current().Kind != KindToken {
		t.Errorf("reset not seen: %+v", s.Current())
	}
	os.WriteFile(p, []byte("{broken"), 0o600)
	if s.Current().Kind != "" {
		t.Error("damaged file must fail closed")
	}
	if err := other.Remove(); err != nil || other.Remove() != nil {
		t.Error("remove twice")
	}
}

func TestLimiter(t *testing.T) {
	now := t0
	l := NewLimiter()
	l.Now = func() time.Time { return now }
	for i := 0; i < 4; i++ {
		l.Fail("a")
	}
	if ok, _ := l.Allowed("a"); !ok {
		t.Fatal("locked after 4")
	}
	l.Fail("a")
	if ok, wait := l.Allowed("a"); ok || wait != 5*time.Minute {
		t.Fatalf("not locked after 5: %v %v", ok, wait)
	}
	if ok, _ := l.Allowed("b"); !ok {
		t.Error("other client locked")
	}
	now = now.Add(5*time.Minute + time.Second)
	if ok, _ := l.Allowed("a"); !ok {
		t.Error("lockout did not end")
	}
	// failures spread over more than the window never lock
	for i := 0; i < 20; i++ {
		now = now.Add(30 * time.Second)
		l.Fail("c")
	}
	if ok, _ := l.Allowed("c"); !ok {
		t.Error("slow failures locked")
	}
	l.Fail("d")
	l.Success("d")
	for i := 0; i < 4; i++ {
		l.Fail("d")
	}
	if ok, _ := l.Allowed("d"); !ok {
		t.Error("success did not clear")
	}
}
