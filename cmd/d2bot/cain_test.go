package main

import (
	"context"
	"testing"

	"d2lod-bot/internal/task"
)

// The Act 1 chain (bark, stones, portal, cage) and the Rescue cages must name only things the adapter knows.
func TestAct1CampaignNamesResolve(t *testing.T) {
	r := &recorder{talks: map[[2]string]bool{}, objects: map[string]bool{}, bosses: map[string]bool{}}
	c := task.Template("normal_act1", true)
	for _, e := range c.Entries {
		for _, n := range task.Handler(e.Quest) {
			_ = n.Step.Do(context.Background(), r)
		}
	}
	for _, o := range []string{"Tree of Inifuss", CairnStones, "Cain's Cage"} {
		if !r.objects[o] {
			t.Errorf("the Act 1 campaign never asked for %q", o)
		}
	}
	for _, k := range [][2]string{{task.NPCAkara, "bark"}, {task.NPCCharsi, "imbue:best"}} {
		if !r.talks[k] {
			t.Errorf("the Act 1 campaign never asked for %v", k)
		}
	}
	for k := range r.talks {
		if _, ok := talkPaths[k]; !ok {
			t.Errorf("no talk path for %v", k)
		}
	}
	for o := range r.objects {
		if _, ok := objSpecs[o]; !ok && o != CairnStones {
			t.Errorf("object %q is not implemented", o)
		}
	}
	for _, o := range []string{"Tristram portal", "cage"} {
		if _, ok := objSpecs[o]; !ok {
			t.Errorf("object %q missing", o)
		}
	}
}

func TestOnceSetIsPerAreaAndPerObject(t *testing.T) {
	var s *onceSet
	s.add(1, 1) // nil is a no-op
	if s.has(1, 1) {
		t.Fatal("nil set remembers")
	}
	s = &onceSet{}
	s.add(25, 7)
	s.add(25, 8)
	if !s.has(25, 7) || !s.has(25, 8) || s.has(25, 9) || s.has(26, 7) {
		t.Fatal("per area and id")
	}
	s.add(38, 7) // new area forgets the old one
	if s.has(25, 7) || !s.has(38, 7) {
		t.Fatal("area change must reset")
	}
}

// fakeStones lights a stone when it is clicked at the right position of a secret order and resets all on a wrong one.
type fakeStones struct {
	secret []int
	lit    []int
	clicks int
	portal bool
}

func (f *fakeStones) Click(_ context.Context, i int) error {
	f.clicks++
	if i == f.secret[len(f.lit)] {
		f.lit = append(f.lit, i)
		f.portal = len(f.lit) == len(f.secret)
	} else {
		f.lit = nil
	}
	return nil
}
func (f *fakeStones) Lit(i int) bool {
	for _, l := range f.lit {
		if l == i {
			return true
		}
	}
	return false
}
func (f *fakeStones) Portal() bool { return f.portal }

func TestSolveCairnFindsAnyOrder(t *testing.T) {
	for _, secret := range [][]int{{0, 1, 2, 3, 4}, {4, 3, 2, 1, 0}, {2, 0, 4, 1, 3}, {1, 4, 0, 3, 2}} {
		f := &fakeStones{secret: secret}
		if err := solveCairn(context.Background(), f, 5, 30); err != nil || !f.portal {
			t.Fatalf("%v: %v portal %v", secret, err, f.portal)
		}
		if f.clicks > 30 {
			t.Fatalf("%v: %d clicks", secret, f.clicks)
		}
	}
}

// Stones that never read as lit end in an error, not an endless loop.
func TestSolveCairnGivesUpWhenLitIsUnreadable(t *testing.T) {
	f := &blindStones{}
	if err := solveCairn(context.Background(), f, 5, 30); err == nil {
		t.Fatal("expected an error")
	}
	if f.clicks > 30 {
		t.Fatalf("%d clicks", f.clicks)
	}
}

type blindStones struct{ clicks int }

func (b *blindStones) Click(context.Context, int) error { b.clicks++; return nil }
func (b *blindStones) Lit(int) bool                     { return false }
func (b *blindStones) Portal() bool                     { return false }
