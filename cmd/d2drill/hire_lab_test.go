package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/drill"
)

// scriptMenu is a fake dialog: Menu answers menus[step]; clicking "talk" advances to the next menu.
type scriptMenu struct {
	menus  [][]d2dctl.MenuLine
	step   int
	picked []string
}

func (s *scriptMenu) Menu(context.Context) (d2dctl.List[d2dctl.MenuLine], error) {
	return d2dctl.List[d2dctl.MenuLine]{Items: s.menus[s.step]}, nil
}

func (s *scriptMenu) PickMenuLine(_ context.Context, text string) error {
	s.picked = append(s.picked, text)
	if text == "talk" && s.step+1 < len(s.menus) {
		s.step++
	}
	return nil
}

func ml(t string) d2dctl.MenuLine { return d2dctl.MenuLine{Text: t} }

var (
	first  = []d2dctl.MenuLine{ml("Kashya"), ml("talk"), ml("cancel")}
	window = []d2dctl.MenuLine{ml("Kashya"), ml("Scout - Lvl: 3 Life 80 Cost: 450"), ml("cancel")}
)

func TestHireFlowTalkThenRow(t *testing.T) {
	s := &scriptMenu{menus: [][]d2dctl.MenuLine{first, window}}
	info, err := hireFlow(context.Background(), s, first, 5000, time.Millisecond)
	if err != nil || info.Quote != 450 || info.WantCode != "" {
		t.Fatalf("%+v %v", info, err)
	}
	if len(s.picked) != 2 || s.picked[0] != "talk" || s.picked[1] != window[1].Text {
		t.Fatalf("picked %v", s.picked)
	}
}

func TestHireFlowDirectLine(t *testing.T) {
	direct := []d2dctl.MenuLine{ml("Kashya"), ml("hire: 450"), ml("cancel")}
	s := &scriptMenu{menus: [][]d2dctl.MenuLine{direct}}
	info, err := hireFlow(context.Background(), s, direct, 5000, time.Millisecond)
	if err != nil || info.Quote != 450 || len(s.picked) != 1 {
		t.Fatalf("%+v %v %v", info, err, s.picked)
	}
}

func TestHireFlowCantAfford(t *testing.T) {
	s := &scriptMenu{menus: [][]d2dctl.MenuLine{first, window}}
	info, err := hireFlow(context.Background(), s, first, 1, time.Millisecond)
	if !errors.Is(err, drill.ErrHireCantAfford) || info.Quote != 450 {
		t.Fatalf("%+v %v", info, err)
	}
	if last := s.picked[len(s.picked)-1]; last != "cancel" {
		t.Fatalf("picked %v", s.picked)
	}
}

func TestHireFlowNoLineAfterTalk(t *testing.T) {
	s := &scriptMenu{menus: [][]d2dctl.MenuLine{first, first}}
	if _, err := hireFlow(context.Background(), s, first, 5000, time.Millisecond); err == nil {
		t.Fatal("want an error naming the menu")
	}
	if _, err := hireFlow(context.Background(), s, []d2dctl.MenuLine{ml("Kashya"), ml("cancel")}, 5000, time.Millisecond); err == nil {
		t.Fatal("no talk line must fail")
	}
}
