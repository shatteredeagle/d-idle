package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type docV3 struct {
	Name  string   `json:"name"`
	Count int      `json:"count"`
	Tags  []string `json:"tags"`
}

func chain() *Registry {
	r := &Registry{}
	// v1 {"n": "x"} -> v2 {"name": "x"} -> v3 adds count -> v4 (unused skip) ; here 1->2, 2->3
	r.Register(1, 2, func(in json.RawMessage) (json.RawMessage, error) {
		var m map[string]any
		if err := json.Unmarshal(in, &m); err != nil {
			return nil, err
		}
		m["name"] = m["n"]
		delete(m, "n")
		return json.Marshal(m)
	})
	r.Register(2, 3, func(in json.RawMessage) (json.RawMessage, error) {
		var m map[string]any
		if err := json.Unmarshal(in, &m); err != nil {
			return nil, err
		}
		m["count"] = float64(len(m["name"].(string)))
		return json.Marshal(m)
	})
	return r
}

func write(t *testing.T, p, s string) {
	t.Helper()
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRoundTripAndBak(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sub", "d.json")
	f := &File[docV3]{Path: p, Version: 3, Migrations: chain()}
	v, info, err := f.Load()
	if err != nil || info.Exists || v.Name != "" {
		t.Fatalf("missing: %v %+v %+v", err, info, v)
	}
	if err := f.Save(docV3{Name: "a", Count: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p + ".bak"); err == nil {
		t.Fatal("bak on first save")
	}
	if err := f.Save(docV3{Name: "b", Count: 2}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p + ".tmp"); err == nil {
		t.Fatal("tmp left behind")
	}
	v, info, _ = f.Load()
	if v.Name != "b" || !info.Exists || info.FromBackup {
		t.Fatalf("%+v %+v", v, info)
	}
	bak := &File[docV3]{Path: p + ".bak", Version: 3}
	if v, _, _ := bak.Load(); v.Name != "a" {
		t.Fatalf("bak holds %+v", v)
	}
}

func TestThreeStepChain(t *testing.T) {
	p := filepath.Join(t.TempDir(), "d.json")
	write(t, p, `{"version":1,"data":{"n":"hello"}}`)
	f := &File[docV3]{Path: p, Version: 3, Migrations: chain()}
	v, info, err := f.Load()
	if err != nil || v.Name != "hello" || v.Count != 5 || !info.Migrated || info.FromVersion != 1 {
		t.Fatalf("%v %+v %+v", err, v, info)
	}
	if got, _ := os.ReadFile(p); !strings.Contains(string(got), `"n"`) {
		t.Fatal("Load must not rewrite the file")
	}
	if err := f.Save(v); err != nil {
		t.Fatal(err)
	}
	// the pre-migration file is the .bak
	if got, _ := os.ReadFile(p + ".bak"); !strings.Contains(string(got), `"n"`) {
		t.Fatal("pre-migration backup missing")
	}
	if plan, _ := chain().Plan(1, 3); len(plan) != 3 {
		t.Fatalf("plan %v", plan)
	}
}

func TestNoMigrationAndFuture(t *testing.T) {
	p := filepath.Join(t.TempDir(), "d.json")
	write(t, p, `{"version":1,"data":{}}`)
	f := &File[docV3]{Path: p, Version: 2}
	if _, _, err := f.Load(); !errors.Is(err, ErrNoMigration) {
		t.Fatalf("got %v", err)
	}
	write(t, p, `{"version":9,"data":{"name":"x"}}`)
	f = &File[docV3]{Path: p, Version: 3}
	if _, _, err := f.Load(); !errors.Is(err, ErrFutureVersion) {
		t.Fatalf("got %v", err)
	}
	if err := f.Save(docV3{}); !errors.Is(err, ErrFutureVersion) {
		t.Fatalf("save over future: %v", err)
	}
	if b, _ := os.ReadFile(p); !strings.Contains(string(b), `"version":9`) {
		t.Fatal("future file was modified")
	}
	if m, _ := filepath.Glob(p + ".corrupt-*"); len(m) != 0 {
		t.Fatal("future file quarantined")
	}
}

func TestCorruptRecoversFromBak(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "d.json")
	f := &File[docV3]{Path: p, Version: 3, Migrations: chain()}
	f.Save(docV3{Name: "a"})
	f.Save(docV3{Name: "b"})
	write(t, p, `{not json`)
	v, info, err := f.Load()
	if err != nil || v.Name != "a" || !info.FromBackup || info.Quarantined == "" || info.Warning == "" {
		t.Fatalf("%v %+v %+v", err, v, info)
	}
	if _, err := os.Stat(p); err == nil {
		t.Fatal("corrupt file still in place")
	}
}

func TestCorruptEverythingStartsEmpty(t *testing.T) {
	p := filepath.Join(t.TempDir(), "d.json")
	write(t, p, `junk`)
	write(t, p+".bak", `junk2`)
	f := &File[docV3]{Path: p, Version: 1}
	v, info, err := f.Load()
	if err != nil || info.Exists || v.Name != "" || info.Warning == "" {
		t.Fatalf("%v %+v", err, info)
	}
}

func TestQuarantineKeepsNewest3(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "d.json")
	clock := time.Unix(1000, 0)
	f := &File[docV3]{Path: p, Version: 1, Now: func() time.Time { return clock }}
	for i := 0; i < 6; i++ {
		write(t, p, "junk")
		f.Load()
		if i%2 == 1 { // two quarantines in the same second exercise the -n suffix
			clock = clock.Add(time.Second)
		}
	}
	m, _ := filepath.Glob(p + ".corrupt-*")
	if len(m) != 3 {
		t.Fatalf("kept %v", m)
	}
	for _, x := range m {
		if strings.Contains(x, "corrupt-1000") {
			t.Fatalf("oldest kept: %v", m)
		}
	}
}

func TestLogSizeRotation(t *testing.T) {
	p := filepath.Join(t.TempDir(), "l", "x.jsonl")
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	l := &Log{Path: p, MaxBytes: 50, Keep: 2, Now: func() time.Time { now = now.Add(time.Millisecond); return now }}
	for i := 0; i < 20; i++ {
		if err := l.Append(map[string]int{"i": i}); err != nil {
			t.Fatal(err)
		}
	}
	m, _ := filepath.Glob(p + ".*")
	if len(m) != 2 {
		t.Fatalf("rotated %v", m)
	}
	if st, _ := os.Stat(p); st.Size() > 50 {
		t.Fatalf("active %d", st.Size())
	}
	b, _ := os.ReadFile(p)
	if !strings.Contains(string(b), `"i":19`) {
		t.Fatalf("newest line missing: %s", b)
	}
}

func TestLogAgeRotationAndAdopt(t *testing.T) {
	p := filepath.Join(t.TempDir(), "x.jsonl")
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	l := &Log{Path: p, MaxAge: time.Hour, Now: func() time.Time { return now }}
	l.Append(1)
	now = now.Add(30 * time.Minute)
	l.Append(2)
	now = now.Add(31 * time.Minute)
	l.Append(3)
	if m, _ := filepath.Glob(p + ".*"); len(m) != 1 {
		t.Fatalf("rotated %v", m)
	}
	if b, _ := os.ReadFile(p); strings.TrimSpace(string(b)) != "3" {
		t.Fatalf("active %q", b)
	}
	// a fresh Log adopts the existing file's size
	l2 := &Log{Path: p, MaxBytes: 3, Now: func() time.Time { return now }}
	l2.Append(4)
	if m, _ := filepath.Glob(p + ".*"); len(m) != 2 {
		t.Fatalf("adopt: %v", m)
	}
}

func TestLogConcurrent(t *testing.T) {
	p := filepath.Join(t.TempDir(), "x.jsonl")
	l := &Log{Path: p, MaxBytes: 1 << 20}
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				l.Append(i)
			}
		}()
	}
	wg.Wait()
	b, _ := os.ReadFile(p)
	if n := strings.Count(string(b), "\n"); n != 400 {
		t.Fatalf("lines %d", n)
	}
}
