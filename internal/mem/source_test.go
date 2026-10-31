package mem

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestFakeReadsAndRejectsUnmapped(t *testing.T) {
	f := NewFake()
	f.PutU32(0x1000, 0xdeadbeef, 7)
	if v, err := U32(f, 0x1000); err != nil || v != 0xdeadbeef {
		t.Fatalf("got %x %v", v, err)
	}
	if v, err := U16(f, 0x1004); err != nil || v != 7 {
		t.Fatalf("got %d %v", v, err)
	}
	if _, err := U32(f, 0x2000); err == nil {
		t.Fatal("expected error for unmapped read")
	}
}

func TestSnapshotRoundTripAndRecording(t *testing.T) {
	f := NewFake()
	f.PutU32(0x401000, 1, 2, 3)
	f.PutU32(0x402ffc, 0xaabbccdd, 0x11223344) // crosses a page boundary
	rec := NewRecording(f)
	if _, err := U32(rec, 0x401004); err != nil {
		t.Fatal(err)
	}
	var b [8]byte
	if err := rec.ReadAt(b[:], 0x402ffc); err != nil {
		t.Fatal(err)
	}
	if _, err := U32(rec, 0x900000); err == nil {
		t.Fatal("expected failure")
	}
	if got := len(rec.Pages()); got != 3 {
		t.Fatalf("pages %v", rec.Pages())
	}
	snap := rec.Snapshot()
	var buf bytes.Buffer
	if _, err := snap.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}
	back, err := ReadSnapshot(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if v, err := U32(back, 0x401004); err != nil || v != 2 {
		t.Fatalf("got %d %v", v, err)
	}
	if v, err := U32(back, 0x403000); err != nil || v != 0x11223344 {
		t.Fatalf("got %x %v", v, err)
	}
	if _, err := U32(back, 0x404000); !errors.Is(err, ErrUnmapped) {
		t.Fatalf("want ErrUnmapped, got %v", err)
	}
}

func TestVerifyBuild(t *testing.T) {
	f := NewFake()
	hdr := make([]byte, 0x400)
	hdr[0], hdr[1] = 'M', 'Z'
	binary.LittleEndian.PutUint32(hdr[0x3c:], 0x80)
	copy(hdr[0x80:], "PE\x00\x00")
	binary.LittleEndian.PutUint16(hdr[0x80+24:], 0x10B)
	binary.LittleEndian.PutUint32(hdr[0x80+8:], ExpectTimestamp)
	binary.LittleEndian.PutUint32(hdr[0x80+24+28:], Base)
	binary.LittleEndian.PutUint32(hdr[0x80+24+56:], ExpectSizeOfImage)
	f.Put(Base, hdr)
	if err := VerifyBuild(f); err != nil {
		t.Fatal(err)
	}
	binary.LittleEndian.PutUint32(hdr[0x80+8:], 1)
	f.Put(Base, hdr)
	if err := VerifyBuild(f); !errors.Is(err, ErrBuild) {
		t.Fatalf("want ErrBuild, got %v", err)
	}
}

func TestProcUnmappedReadIsTyped(t *testing.T) {
	p, err := Open(os.Getpid())
	if err != nil {
		t.Skip(err)
	}
	defer p.Close()
	var b [4]byte
	err = p.ReadAt(b[:], 0x10) // page zero is never mapped
	var re *ReadError
	if !errors.As(err, &re) || !errors.Is(err, ErrUnmapped) {
		t.Fatalf("got %v", err)
	}
	if !p.Alive() {
		t.Fatal("alive")
	}
	if _, err := FindGame(); err != nil && !errors.Is(err, ErrNoGame) {
		t.Fatal(err)
	}
}

func TestProcInstance(t *testing.T) {
	cmd := exec.Command("sleep", "5")
	cmd.Env = []string{InstanceEnv + "=i7"}
	if err := cmd.Start(); err != nil {
		t.Skip(err)
	}
	defer cmd.Process.Kill()
	pid := cmd.Process.Pid
	// the child reads as empty until it has exec'd with the new environment
	got := ProcInstance(pid)
	for i := 0; i < 100 && got != "i7"; i++ {
		time.Sleep(10 * time.Millisecond)
		got = ProcInstance(pid)
	}
	if got != "i7" {
		t.Fatalf("got %q", got)
	}
	if !instanceMatches(pid, "i7") || instanceMatches(pid, "i2") || instanceMatches(pid, DefaultInstance) || !instanceMatches(pid, "") {
		t.Fatal("instanceMatches")
	}
}

func TestFakeWriteAt(t *testing.T) {
	f := NewFake()
	f.PutU32(0x1000, 1, 2)
	var w Writer = f
	if err := w.WriteAt([]byte{9, 0, 0, 0}, 0x1004); err != nil {
		t.Fatal(err)
	}
	if v, _ := U32(f, 0x1004); v != 9 {
		t.Fatalf("got %d", v)
	}
	if err := w.WriteAt([]byte{1}, 0x2000); err == nil {
		t.Fatal("write to unmapped address succeeded")
	}
}
