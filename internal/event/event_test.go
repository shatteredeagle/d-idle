package event

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"d2lod-bot/internal/goals"
)

func samples() []Event {
	return []Event{
		New(KindKill, SrcObserver, Kill{Monster: "Andariel", ClassID: 156, Boss: true, XP: 1200, Area: 37, AreaName: "Catacombs 4"}),
		New(KindDrop, SrcLoot, Drop{Item: Item{Name: "Shako", Quality: "unique", Qty: 1}, Area: 3}),
		New(KindDeath, SrcObserver, Death{Area: 5, Pos: [2]int{10, 20}, Killer: "Fire Enchanted", Affixes: []string{"fire"}, LifeCurve: []int{100, 20}, CharLevel: 12}),
		New(KindLevelUp, SrcObserver, LevelUp{Level: 9}),
		New(KindAreaChange, SrcObserver, AreaChange{Area: 2, AreaName: "Blood Moor"}),
		New(KindPotion, SrcObserver, Potion{Name: "Minor Healing"}),
		New(KindError, SrcManager, Error{Message: "boom", Fatal: true}),
		New(KindFailsafe, SrcSentinel, Failsafe{Layer: "L2", Rule: "no-progress", Step: 3, Action: "restart-game", Result: "ok"}),
		New(KindLifecycle, SrcLaunch, Lifecycle{State: "exited", Reason: "crash"}),
		New(KindCheat, SrcCheat, Cheat{Cheat: "set-level", Reason: "test"}),
		New(KindActivity, SrcObserver, Activity{State: "fighting"}),
		New(KindGoalDone, SrcManager, GoalEvent{GoalID: "g1", Message: "done"}),
		New(KindQuestStep, SrcObserver, QuestStep{Quest: "den", Step: "enter", State: "done"}),
		New(KindQuestDone, SrcObserver, QuestDone{Quest: "den", Difficulty: 1}),
		New(KindWaypointActivated, SrcObserver, WaypointActivated{Area: 3}),
		New(KindDifficultyChange, SrcObserver, DifficultyChange{Difficulty: 1}),
		New(KindActChange, SrcObserver, ActChange{Act: 2}),
		New(KindItemAcquired, SrcLoot, ItemAcquired{Item: Item{Name: "Horadric Cube"}, Where: "ground"}),
		New(KindItemEquipped, SrcTown, ItemEquipped{Item: Item{Name: "Buckler"}, Slot: "lhand"}),
		New(KindPointsSpent, SrcTown, PointsSpent{Kind: "skill", Name: "Fire Bolt", Level: 2}),
		New(KindGold, SrcTown, Gold{Delta: -250, Reason: "repair"}),
		New(KindTownTrip, SrcTown, TownTrip{Reason: "potions"}),
		New(KindChicken, SrcCombat, Empty{}),
		New(KindCorpseRun, SrcNav, Empty{}),
		New(KindInfo, SrcManager, Message{Message: "hi"}),
	}
}

func TestPayloadForEveryKind(t *testing.T) {
	for _, e := range samples() {
		p := payloadType(e.Kind)
		if p == nil || reflect.TypeOf(p).Elem() != reflect.TypeOf(e.Data) {
			t.Errorf("%s: payloadType %T does not match sample %T", e.Kind, p, e.Data)
		}
	}
}

func TestJSONRoundTrip(t *testing.T) {
	at := time.Date(2026, 10, 28, 1, 2, 3, 0, time.UTC)
	for i, e := range samples() {
		e.Seq, e.Time, e.Instance, e.Run, e.Game, e.Goal = uint64(i+1), at, "i2", "run-1", 4, "g9"
		b, err := json.Marshal(e)
		if err != nil {
			t.Fatal(err)
		}
		var got Event
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatalf("%s: %v (%s)", e.Kind, err, b)
		}
		if !reflect.DeepEqual(got.Payload(), e.Data) {
			t.Errorf("%s: payload %#v != %#v", e.Kind, got.Payload(), e.Data)
		}
		got.Data, e.Data = nil, nil
		if !reflect.DeepEqual(got, e) {
			t.Errorf("%s: envelope %#v != %#v", e.Kind, got, e)
		}
	}
}

func TestAs(t *testing.T) {
	b, _ := json.Marshal(New(KindKill, SrcObserver, Kill{Monster: "x", XP: 5}))
	var e Event
	if err := json.Unmarshal(b, &e); err != nil {
		t.Fatal(err)
	}
	if k, ok := As[Kill](e); !ok || k.XP != 5 {
		t.Fatalf("As[Kill] = %v %v", k, ok)
	}
	if _, ok := As[Death](e); ok {
		t.Fatal("As[Death] matched a kill")
	}
	if k, ok := As[Kill](New(KindKill, "", &Kill{Monster: "p"})); !ok || k.Monster != "p" {
		t.Fatal("pointer payload")
	}
}

func TestVersion(t *testing.T) {
	var e Event
	err := json.Unmarshal([]byte(`{"v":2,"seq":1,"kind":"kill","data":{}}`), &e)
	if !errors.Is(err, ErrFutureVersion) {
		t.Fatalf("want ErrFutureVersion, got %v", err)
	}
	if err := json.Unmarshal([]byte(`{"seq":1,"kind":"level_up","data":{"level":3}}`), &e); err != nil || e.V != 1 {
		t.Fatalf("v0 accepted as 1: %v %d", err, e.V)
	}
	b, _ := json.Marshal(New(KindInfo, "", Message{Message: "x"}))
	if !strings.Contains(string(b), `"v":1`) {
		t.Fatalf("version missing: %s", b)
	}
}

func TestUnknownKindPreserved(t *testing.T) {
	in := `{"v":1,"seq":7,"time":"0001-01-01T00:00:00Z","kind":"future_thing","data":{"a":1}}`
	var e Event
	if err := json.Unmarshal([]byte(in), &e); err != nil {
		t.Fatal(err)
	}
	if _, ok := e.Data.(json.RawMessage); !ok || e.Payload() != nil {
		t.Fatalf("data = %#v", e.Data)
	}
	b, _ := json.Marshal(e)
	if !strings.Contains(string(b), `"data":{"a":1}`) {
		t.Fatalf("lost data: %s", b)
	}
}

func TestBusStampsAndFansOut(t *testing.T) {
	at := time.Unix(100, 0)
	b := NewBus("i3", func() time.Time { return at })
	s1, s2 := b.Subscribe("a", 4), b.Subscribe("b", 4)
	b.Publish(Event{Kind: KindInfo})
	b.Publish(Event{Kind: KindInfo, Instance: "other", Time: at.Add(time.Hour)})
	for _, s := range []*Sub{s1, s2} {
		e1, e2 := <-s.C(), <-s.C()
		if e1.Seq != 1 || e2.Seq != 2 || e1.Instance != "i3" || e2.Instance != "other" || !e1.Time.Equal(at) || e1.V != Version {
			t.Fatalf("%+v %+v", e1, e2)
		}
	}
	s1.Close()
	s1.Close()
	if _, ok := <-s1.C(); ok {
		t.Fatal("closed channel delivered")
	}
	b.Publish(Event{Kind: KindInfo})
	if (<-s2.C()).Seq != 3 {
		t.Fatal("seq")
	}
}

func TestBusDropOldest(t *testing.T) {
	b := NewBus("", nil)
	s := b.Subscribe("slow", 3)
	for i := 0; i < 10; i++ {
		b.Publish(Event{Kind: KindInfo})
	}
	if s.Dropped() != 7 {
		t.Fatalf("dropped %d", s.Dropped())
	}
	for want := uint64(8); want <= 10; want++ {
		if e := <-s.C(); e.Seq != want {
			t.Fatalf("got seq %d want %d", e.Seq, want)
		}
	}
}

func TestBusNeverBlocksRace(t *testing.T) {
	b := NewBus("", nil)
	stuck := b.Subscribe("stuck", 1) // never read
	busy := b.Subscribe("busy", 8)
	var got int
	done := make(chan struct{})
	go func() {
		defer close(done)
		for range busy.C() {
			got++
		}
	}()
	var wg sync.WaitGroup
	for p := 0; p < 8; p++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 2000; i++ {
				b.Publish(Event{Kind: KindInfo})
			}
		}()
	}
	extra := make(chan struct{})
	go func() { // subscribers coming and going while publishing
		defer close(extra)
		for i := 0; i < 200; i++ {
			b.Subscribe("tmp", 2).Close()
		}
	}()
	finished := make(chan struct{})
	go func() { wg.Wait(); <-extra; close(finished) }()
	select {
	case <-finished:
	case <-time.After(10 * time.Second):
		t.Fatal("Publish blocked")
	}
	busy.Close()
	<-done
	if b.Seq() != 16000 || stuck.Dropped() != 16000-1 {
		t.Fatalf("seq %d dropped %d", b.Seq(), stuck.Dropped())
	}
	if got == 0 {
		t.Fatal("busy subscriber got nothing")
	}
}

func TestGoalsRoundTrip(t *testing.T) {
	cases := []goals.Event{
		{Type: goals.EvKill, Monster: "Andariel", Boss: true, XP: 99, Area: "Catacombs", AreaID: 37, GoalID: "g"},
		{Type: goals.EvDrop, Item: &goals.Item{Name: "Shako", Base: "Cap", Type: "helm", Quality: "unique", Qty: 2}, Area: "Moor", AreaID: 2},
		{Type: goals.EvDeath, Area: "Moor", AreaID: 2},
		{Type: goals.EvLevelUp, Level: 5},
		{Type: goals.EvAreaChange, Area: "Moor", AreaID: 2},
		{Type: goals.EvPotion, Potion: "Rejuv"},
		{Type: goals.EvError, Message: "x", Fatal: true},
		{Type: goals.EvControl, Message: "start"},
		{Type: goals.EvInfo, Message: "hello"},
		{Type: goals.EvGoalAdded, GoalID: "g1"},
		{Type: goals.EvGoalFailed, GoalID: "g1", Message: "stuck"},
		{Type: goals.EvGoalPreempted, GoalID: "g1"},
		{Type: goals.EvGoalEdited, GoalID: "g1"},
		{Type: goals.EvGoalSkipped, GoalID: "g1"},
		{Type: goals.EvGoalRemoved, GoalID: "g1"},
		{Type: goals.EvGoalStarted, GoalID: "g1"},
		{Type: goals.EvGoalPaused, GoalID: "g1"},
		{Type: goals.EvGoalResumed, GoalID: "g1"},
		{Type: goals.EvGoalDone, GoalID: "g1"},
	}
	for _, g := range cases {
		g.Seq, g.Time = 5, time.Unix(50, 0).UTC() // UTC: a JSON round trip yields UTC, and DeepEqual tells time.Local from UTC even when TZ=UTC
		e := FromGoals(g, SrcManager)
		b, _ := json.Marshal(e) // through the wire too
		var d Event
		if err := json.Unmarshal(b, &d); err != nil {
			t.Fatal(err)
		}
		back, ok := ToGoals(d)
		if !ok || !reflect.DeepEqual(back, g) {
			t.Errorf("%s: %#v != %#v (ok=%v)", g.Type, back, g, ok)
		}
	}
	if _, ok := ToGoals(New(KindFailsafe, "", Failsafe{})); ok {
		t.Fatal("failsafe should not map to goals")
	}
	if e := FromGoals(goals.Event{Type: "weird", Message: "m"}, "x"); e.Kind != KindInfo {
		t.Fatalf("unknown goals type -> %s", e.Kind)
	}
}
