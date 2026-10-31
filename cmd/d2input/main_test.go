package main

import (
	"fmt"
	"strings"
	"testing"
)

type fakeSys struct {
	h    uintptr
	msgs []string
}

func (f *fakeSys) hung(uintptr) bool { return false }
func (f *fakeSys) list() []winInfo {
	if f.h == 0 {
		return nil
	}
	return []winInfo{{h: f.h, pid: 7, w: 800, ht: 600, visible: true}}
}
func (f *fakeSys) valid(h uintptr) bool { return h == f.h }
func (f *fakeSys) post(h uintptr, msg uint32, w, l uintptr) bool {
	f.msgs = append(f.msgs, fmt.Sprintf("%x/%x/%x", msg, w, l))
	return true
}
func (f *fakeSys) scan(uint32) uintptr { return 0 }
func (f *fakeSys) info(uintptr) string { return "hwnd=0x1" }

func TestHandle(t *testing.T) {
	f := &fakeSys{h: 1}
	v := &server{s: f}
	cases := map[string]string{
		"ping":           "ok pong",
		"info":           "ok hwnd=0x1",
		"bogus":          "err unknown unknown command bogus",
		"click x 1 2":    "err badbutton button must be l or r",
		"click l 1":      "err usage missing X Y",
		"key press nope": "err badkey unknown key nope",
		"move 10 20":     "ok",
		"seq move 1 2 ; sleep 1 ; key press a hold=1": "ok 3",
	}
	for in, want := range cases {
		if got := v.handle(in); got != want {
			t.Errorf("%q: got %q want %q", in, got, want)
		}
	}
	if got := v.handle("seq move 1 2 ; key press zzz"); !strings.HasPrefix(got, "err badkey step 2") {
		t.Errorf("seq error: %q", got)
	}
}

func TestShiftClickMessages(t *testing.T) {
	f := &fakeSys{h: 1}
	v := &server{s: f}
	if got := v.handle("click l 5 6 shift hold=1"); got != "ok" {
		t.Fatal(got)
	}
	want := []string{"100/10/1", "200/4/60005", "201/5/60005", "202/4/60005", "101/10/c0000001"}
	// keydown VK_SHIFT, move(MK_SHIFT), down(MK_LBUTTON|MK_SHIFT), up(MK_SHIFT), keyup
	if len(f.msgs) != len(want) {
		t.Fatalf("msgs %v", f.msgs)
	}
	for i := range want {
		if i == 0 || i == 4 {
			continue // scan code bits depend on the platform
		}
		if f.msgs[i] != want[i] {
			t.Errorf("msg %d: %s want %s", i, f.msgs[i], want[i])
		}
	}
}

func TestNoWindow(t *testing.T) {
	v := &server{s: &fakeSys{}}
	if got := v.handle("click l 1 2"); !strings.HasPrefix(got, "err nowindow") {
		t.Fatal(got)
	}
	if got := v.handle("ping"); got != "ok pong" {
		t.Fatal(got)
	}
}

func TestCharAndHung(t *testing.T) {
	f := &fakeSys{h: 1}
	v := &server{s: f}
	if got := v.handle("char a"); got != "ok" {
		t.Fatal(got)
	}
	if len(f.msgs) != 1 || f.msgs[0] != "102/61/1" {
		t.Fatalf("%v", f.msgs)
	}
	if got := v.handle("char ab"); !strings.HasPrefix(got, "err usage") {
		t.Fatal(got)
	}
	if got := v.handle("hung"); got != "ok alive" {
		t.Fatal(got)
	}
}

type multiSys struct {
	fakeSys
	hs   []uintptr
	last uintptr
}

func (m *multiSys) list() []winInfo {
	var out []winInfo
	for _, h := range m.hs {
		out = append(out, winInfo{h: h, pid: 100 + uint32(h), w: 800, ht: 600, visible: true})
	}
	return out
}
func (m *multiSys) valid(h uintptr) bool {
	for _, x := range m.hs {
		if x == h {
			return true
		}
	}
	return false
}
func (m *multiSys) post(h uintptr, msg uint32, w, l uintptr) bool { m.last = h; return true }

func TestTargeting(t *testing.T) {
	m := &multiSys{hs: []uintptr{0x10, 0x20}}
	v := &server{s: m}
	if got := v.handle("list"); got != "ok hwnd=0x10 pid=116 client=800x600 visible=1 ; hwnd=0x20 pid=132 client=800x600 visible=1" {
		t.Fatal(got)
	}
	if got := v.handle("move 1 2"); !strings.HasPrefix(got, "err ambiguous") {
		t.Fatal(got)
	}
	if got := v.handle("@0x20 move 1 2"); got != "ok" || m.last != 0x20 {
		t.Fatal(got, m.last)
	}
	if got := v.handle("@32 seq move 1 2 ; move 3 4"); got != "ok 2" || m.last != 0x20 {
		t.Fatal(got, m.last)
	}
	if got := v.handle("@0x99 move 1 2"); got != "err nowindow no such window" {
		t.Fatal(got)
	}
	if got := v.handle("@zz move 1 2"); !strings.HasPrefix(got, "err usage") {
		t.Fatal(got)
	}
	m.hs = m.hs[:1]
	if got := v.handle("move 1 2"); got != "ok" || m.last != 0x10 {
		t.Fatal(got)
	}
}

func TestBackspaceKey(t *testing.T) {
	if code, ok := vk("backspace"); !ok || code != 0x08 {
		t.Fatalf("backspace = %#x %v", code, ok)
	}
}
