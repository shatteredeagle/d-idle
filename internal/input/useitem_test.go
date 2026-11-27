package input

import "testing"

type useCtl struct{ Fake }

func (u *useCtl) UseItem(id uint32) error { return nil }

type useWorld struct {
	*FakeWorld
	used []uint32
}

func (u *useWorld) UseItem(id uint32) error { u.used = append(u.used, id); return nil }

type holder struct {
	Fake
	v WorldVerbs
}

func (h *holder) WorldVerbs() (WorldVerbs, bool) { return h.v, h.v != nil }

func TestAsItemUser(t *testing.T) {
	if _, ok := AsItemUser(&useCtl{}); !ok {
		t.Fatal("direct ItemUser not found")
	}
	if _, ok := AsItemUser(&Fake{}); ok {
		t.Fatal("plain controller reported as ItemUser")
	}
	w := &useWorld{FakeWorld: &FakeWorld{}}
	u, ok := AsItemUser(&holder{v: w})
	if !ok {
		t.Fatal("wrapper's world verbs not passed through")
	}
	_ = u.UseItem(7)
	if len(w.used) != 1 || w.used[0] != 7 {
		t.Fatalf("used %v", w.used)
	}
	if _, ok := AsItemUser(&holder{v: &FakeWorld{}}); ok {
		t.Fatal("world verbs without UseItem reported as ItemUser")
	}
}
