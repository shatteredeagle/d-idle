package d2

import (
	"errors"
	"testing"
	"time"

	"d2lod-bot/internal/mem"
)

func reader(s mem.Source) *Reader {
	r := New(s)
	r.Sleep = func(time.Duration) {}
	return r
}

func TestStateMachine(t *testing.T) {
	fx := buildInGame()
	r := reader(fx.f)
	if st := r.State(); st != StateInGame {
		t.Fatalf("want ingame, got %v", st)
	}
	fx.set(absFirst, 0, 0x1234) // both set: ambiguous
	if st := r.State(); st != StateLoading {
		t.Fatalf("both set: got %v", st)
	}
	fx.set(absPlayer, 0, 0) // menu
	if st := r.State(); st != StateMenu {
		t.Fatalf("menu: got %v", st)
	}
	fx.set(absFirst, 0, 0) // neither
	if st := r.State(); st != StateLoading {
		t.Fatalf("neither: got %v", st)
	}
	fx.set(absPlayer, 0, fx.player)
	fx.set(fx.player, uoUpdateUnit, 0x5000000)
	if st := r.State(); st != StateLoading {
		t.Fatalf("pUpdateUnit: got %v", st)
	}
	fx.set(fx.player, uoUpdateUnit, 0)
	fx.set(fx.player, uoPath, 0) // chain broken
	if st := r.State(); st != StateLoading {
		t.Fatalf("broken chain: got %v", st)
	}
}

func TestPlayer(t *testing.T) {
	r := reader(buildInGame().f)
	p, err := r.Player()
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "Testy" || p.Class != Sorceress || p.X != 6053 || p.Y != 4963 || p.Area != 1 || !p.InTown() {
		t.Fatalf("%+v", p)
	}
	if p.Life != 40 || p.MaxMana != 35 || p.Stamina != 74 || p.Level != 1 || p.Strength != 10 {
		t.Fatalf("stats %+v", p)
	}
	if p.LifePct() != 100 {
		t.Fatal("life pct")
	}
	if p.LeftSkill != SkillAttack || p.RightSkill != SkillFireBolt || len(p.Skills) != 3 || p.Skills[2].Level != 0 {
		t.Fatalf("skills %+v L%d R%d", p.Skills, p.LeftSkill, p.RightSkill)
	}
	if p.Skills[2].Bonus != 1 || p.Skills[2].Effective() != 1 || p.Skills[0].Bonus != 0 {
		t.Fatalf("bonus %+v", p.Skills)
	}
	if !p.Stats.Has(67) {
		t.Fatal("effective vector (+0x48) must be used for Ex lists")
	}
}

func TestEffectiveVsBaseStats(t *testing.T) {
	fx := buildInGame()
	// equipment raises max life only in the effective vector
	sl, _ := mem.U32(fx.f, fx.player+uoStats)
	r := reader(fx.f)
	base, _ := r.ReadBaseStatList(sl)
	eff, _ := r.ReadStatList(sl)
	if base.Has(67) || !eff.Has(67) {
		t.Fatalf("base %v eff %v", base, eff)
	}
}

func TestUnits(t *testing.T) {
	r := reader(buildInGame().f)
	mons, err := r.Units(TypeMonster)
	if err != nil || len(mons) != 2 {
		t.Fatalf("%v %v", mons, err)
	}
	for _, m := range mons {
		if m.Monster == nil || m.Monster.LifePct != 100 || !m.Alive() {
			t.Fatalf("%+v", m)
		}
	}
	n := Nearest(mons, 6050, 4940, nil)
	if n == nil || n.Class != 150 || n.X != 6048 {
		t.Fatalf("nearest %+v", n)
	}
	objs, _ := r.Units(TypeObject)
	if len(objs) != 1 || objs[0].X != 6052 || objs[0].Y != 4964 {
		t.Fatalf("%+v", objs)
	}
	if _, err := r.Units(9); err == nil {
		t.Fatal("bad type")
	}
}

func TestUnitsRejectsTornType(t *testing.T) {
	fx := buildInGame()
	mons, _ := reader(fx.f).Units(TypeMonster)
	fx.set(mons[0].Addr, uoType, 4) // slot reused by an item mid-read
	got, err := reader(fx.f).Units(TypeMonster)
	if err != nil || len(got) != 1 {
		t.Fatalf("%v %v", got, err)
	}
}

func TestUnitChainCycleAndBadPointer(t *testing.T) {
	fx := buildInGame()
	mons, _ := reader(fx.f).Units(TypeMonster)
	fx.set(mons[0].Addr, uoListNext, mons[0].Addr) // self loop
	if _, err := reader(fx.f).Units(TypeMonster); err != nil {
		t.Fatal(err)
	}
	fx.set(mons[0].Addr, uoListNext, 0xdeadbeef)
	if _, err := reader(fx.f).Units(TypeMonster); err != nil {
		t.Fatal(err)
	}
}

func TestInventory(t *testing.T) {
	r := reader(buildInGame().f)
	inv, err := r.Inventory()
	if err != nil {
		t.Fatal(err)
	}
	if len(inv.Items) != 4 || inv.ItemCount != 4 {
		t.Fatalf("%d items", len(inv.Items))
	}
	if st := inv.Equipped(BodyRHand); st == nil || st.Class != 63 || !st.Item.Identified() {
		t.Fatalf("staff %+v", st)
	}
	belt := inv.Container(ContBelt)
	if len(belt) != 2 || belt[1].Item.GX != 1 || belt[1].Item.GY != 0 {
		t.Fatalf("belt %+v", belt)
	}
	pack := inv.Container(ContInventory)
	if len(pack) != 1 || pack[0].Item.GX != 9 || pack[0].Item.GY != 3 || pack[0].Class != 529 {
		t.Fatalf("pack %+v", pack)
	}
	if g, _ := r.GroundItems(); len(g) != 0 {
		t.Fatalf("ground %v", g)
	}
}

func TestInventoryOwnerMismatch(t *testing.T) {
	fx := buildInGame()
	fx.set(fx.inv, 0x08, 0x1234567)
	if _, err := reader(fx.f).Inventory(); err == nil {
		t.Fatal("expected error")
	}
}

func TestUIAndScreen(t *testing.T) {
	fx := buildInGame()
	r := reader(fx.f)
	ui, _ := r.UI()
	if ui.AnyPanelOpen() || !ui.Open(UIMiniPanel) {
		t.Fatalf("%v", ui)
	}
	fx.set(absUI, UIInventory*4, 1)
	ui, _ = r.UI()
	if !ui.AnyPanelOpen() || !ui.Open(UIInventory) {
		t.Fatal("inventory not seen")
	}
	m, _ := r.Mouse()
	sz, _ := r.ScreenSize()
	vp, _ := r.Viewport()
	if m != (Point{304, 164}) || sz != (Point{800, 600}) {
		t.Fatalf("%v %v", m, sz)
	}
	sp := WorldToScreen(6053, 4963, vp)
	if sp != (Point{(6053-4963)*16 - 17216, (6053+4963)*8 - 88004}) {
		t.Fatal(sp)
	}
}

func TestHover(t *testing.T) {
	fx := buildInGame()
	r := reader(fx.f)
	fx.set(mem.Base+rvaSelID, 0, 9)
	fx.set(mem.Base+rvaSelType, 0, 1)
	h, _ := r.Hovered()
	if h.Active {
		t.Fatal("flag 0 must be inactive")
	}
	fx.set(mem.Base+rvaSelActive, 0, 1)
	if h, _ = r.Hovered(); !h.Active || h.ID != 9 || h.Type != 1 {
		t.Fatalf("%+v", h)
	}
}

func TestSnapshotInGame(t *testing.T) {
	s, err := reader(buildInGame().f).Snapshot()
	if err != nil || s.State != StateInGame || s.Player == nil || len(s.Monsters) != 2 || len(s.Objects) != 1 || s.Inventory == nil {
		t.Fatalf("%+v %v", s, err)
	}
}

func TestSnapshotMenu(t *testing.T) {
	fx := buildInGame()
	fx.set(absPlayer, 0, 0)
	fx.set(absFirst, 0, 0x777)
	s, err := reader(fx.f).Snapshot()
	if err != nil || s.State != StateMenu || s.Player != nil {
		t.Fatalf("%+v %v", s, err)
	}
}

// tornSource flips the player's area on the Nth read of the level number to model a level change
// between the two player reads.
type tornSource struct {
	mem.Source
	lvl   uint32
	reads int
	fx    *fixture
}

func (s *tornSource) ReadAt(p []byte, addr uint32) error {
	if addr == s.lvl+0x1D0 {
		s.reads++
		if s.reads == 3 { // the second Player read of the first attempt
			s.fx.set(s.lvl, 0x1D0, 2)
		}
	}
	return s.Source.ReadAt(p, addr)
}

func TestSnapshotRetriesTornRead(t *testing.T) {
	fx := buildInGame()
	path, _ := mem.U32(fx.f, fx.player+uoPath)
	r1, _ := mem.U32(fx.f, path+0x1C)
	r2, _ := mem.U32(fx.f, r1+0x10)
	lvl, _ := mem.U32(fx.f, r2+0x58)
	ts := &tornSource{Source: fx.f, lvl: lvl, fx: fx}
	s, err := reader(ts).Snapshot()
	if err != nil || s.State != StateInGame {
		t.Fatalf("%+v %v", s, err)
	}
	if s.Player.Area != 2 {
		t.Fatalf("retry should observe the settled area, got %d (reads %d)", s.Player.Area, ts.reads)
	}
}

func TestSnapshotUnreadable(t *testing.T) {
	r := reader(mem.NewFake())
	s, err := r.Snapshot()
	if err == nil {
		t.Fatalf("want error, got %+v", s)
	}
	if !errors.Is(err, mem.ErrUnmapped) {
		t.Fatal(err)
	}
}

func TestUIKindsOneTable(t *testing.T) {
	var u UIState
	for _, i := range []int{0, 6, 7, 0x11, UIMiniPanel, 0x21, 0x22, 0x23, 0x1E} { // HUD flags and an unknown index
		u[i] = 1
	}
	if u.BlockingPanelOpen() || u.EscClosableOpen() || u.AnyPanelOpen() {
		t.Fatal("HUD and unknown flags must not count as a panel")
	}
	if got := u.UnknownOpen(); len(got) != 1 || got[0] != 0x1E {
		t.Fatalf("unknown %v", got)
	}
	u[UIStash] = 1
	if !u.BlockingPanelOpen() || len(u.Indexes(UIPanel)) != 1 {
		t.Fatal("stash is a blocking panel")
	}
	if UIKindOf(-1) != UIUnknown || UIKindOf(uiCount) != UIUnknown || UIKindOf(UIEscMenu) != UIPanel {
		t.Fatal("kind lookup")
	}
}

func TestUnitStatesBitArray(t *testing.T) {
	fx := buildInGame()
	r := reader(fx.f)
	sl := fx.alloc(0x100)
	fx.set(sl, slFlags, slFlagEx)
	arr := fx.alloc(0x20)
	fx.set(arr, 0, 1<<StatePoison, 0, 1<<StateFrozenArmor)
	fx.set(sl, slStateFlags, arr)
	fx.set(fx.player, uoStats, sl)
	ss, err := r.UnitStates(fx.player)
	if err != nil {
		t.Fatal(err)
	}
	if !ss.Has(StatePoison) || ss.Has(3) || ss.Has(-1) || ss.Has(1000) || !ss.Has(64+StateFrozenArmor) {
		t.Fatalf("states %v", ss)
	}
	// a plain (non-extended) list has no state array
	sl2 := fx.alloc(0x100)
	fx.set(fx.player, uoStats, sl2)
	if ss, _ := r.UnitStates(fx.player); ss.Has(StatePoison) {
		t.Fatal("non-extended list must read empty")
	}
}

// The boss name is a wide string behind the pointer at MonsterData+0x2C (VERIFIED LIVE: read inline it was garbage).
func TestMonsterNameIsReadThroughItsPointer(t *testing.T) {
	fx := buildInGame()
	r := reader(fx.f)
	mons, _ := r.Units(TypeMonster)
	md, _ := mem.U32(fx.f, mons[0].Addr+uoData)
	name := fx.alloc(0x40)
	for i, c := range "Blood Raven" {
		fx.set16(name, uint32(i*2), uint16(c))
	}
	fx.setBytes(md, 0x16, []byte{monFlagBoss})
	fx.set(md, 0x2C, name)
	fx.set16(md, 0x26, 0)
	mons, _ = r.Units(TypeMonster)
	var got *MonsterInfo
	for i := range mons {
		if mons[i].Addr == mons[0].Addr {
			got = mons[i].Monster
		}
	}
	if got == nil || got.Name != "Blood Raven" || !got.IsUnique() {
		t.Fatalf("%+v", got)
	}
	fx.set(md, 0x2C, 0xFFFF0000) // a garbage pointer leaves the name empty instead of failing the read
	mons, _ = r.Units(TypeMonster)
	for _, u := range mons {
		if u.Addr == mons[0].Addr && (u.Monster == nil || u.Monster.Name != "") {
			t.Fatalf("bad pointer %+v", u.Monster)
		}
	}
}
