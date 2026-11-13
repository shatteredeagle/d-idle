package main

// Production-wiring matrix (docs/wiring.md). bot.Runner reaches many features through optional-interface type
// assertions on its Deps (r.d.Mover.(interface{ NotePosition(...) }) and so on). A wrapper that embeds an interface
// or forgets a method turns the feature off WITHOUT any error: the loot memory never ran in production for weeks
// because looterAdapter hid its methods. This test reads every such assertion out of internal/bot, resolves the
// interface to its method list and checks each concrete type that production puts in that Deps slot has all of them.
// A new assertion in internal/bot is covered automatically; a new wrapper type goes into prodTypes.

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"reflect"
	"sort"
	"strings"
	"testing"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2denv"
	"d2lod-bot/internal/d2dtown"
	"d2lod-bot/internal/d2dworld"
	"d2lod-bot/internal/invdesk"
	"d2lod-bot/internal/leveler"
	"d2lod-bot/internal/loot"
	"d2lod-bot/internal/nav"
	"d2lod-bot/internal/senses"
	"d2lod-bot/internal/town"
	"d2lod-bot/internal/watch"
)

// prodTypes lists, per Deps slot, every concrete type cmd/d2bot can hand the runner (live path and d2d path,
// with and without the merc wrapper). V: read from buildBehaviors (behaviors.go), runD2D (d2dbackend.go),
// d2dStack.seams (d2dwire.go) and installMercSeam (mercwire.go).
var prodTypes = map[string][]any{
	"Mover":      {moverAdapter{}},
	"Fighter":    {fighterAdapter{}},
	"Town":       {townAdapter{}, d2dTown{}, &mercTown{}},
	"Loot":       {looterAdapter{}},
	"Leveler":    {&leveler.Leveler{}},
	"Gear":       {gearAdapter{}},
	"Quester":    {questAdapter{}},
	"PlayerGear": {&invdesk.Desk{}},
}

// absent records assertions that production types legitimately do not satisfy (feature is optional by design).
// key: Slot/Type/InterfaceSource -> reason.
var absent = map[string]string{
	"Town/mercTown/MercTown": "the merc policy wiring (merc.enabled=1, mercwire.go) replaces the legacy hire/revive chore",
}

type method struct {
	name    string
	in, out int
}

type assertion struct {
	slot, iface string
	methods     []method
	pos         string
}

// botAssertions parses internal/bot (non-test files) and returns every x.(T) whose operand is r.d.<Slot> or
// a.r.d.<Slot>, with T resolved to its method list (named interfaces declared in package bot, embedded ones
// included, and inline interface literals).
func botAssertions(t *testing.T) []assertion {
	t.Helper()
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, "../../internal/bot", func(fi fs.FileInfo) bool { return !strings.HasSuffix(fi.Name(), "_test.go") }, 0)
	if err != nil {
		t.Fatal(err)
	}
	ifaces := map[string]*ast.InterfaceType{}
	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
			for _, d := range f.Decls {
				gd, ok := d.(*ast.GenDecl)
				if !ok {
					continue
				}
				for _, sp := range gd.Specs {
					if ts, ok := sp.(*ast.TypeSpec); ok {
						if it, ok := ts.Type.(*ast.InterfaceType); ok {
							ifaces[ts.Name.Name] = it
						}
					}
				}
			}
		}
	}
	var resolve func(it *ast.InterfaceType) []method
	resolve = func(it *ast.InterfaceType) []method {
		var ms []method
		for _, fl := range it.Methods.List {
			switch ft := fl.Type.(type) {
			case *ast.FuncType:
				m := method{out: 0}
				for _, p := range ft.Params.List {
					n := len(p.Names)
					if n == 0 {
						n = 1
					}
					m.in += n
				}
				if ft.Results != nil {
					for _, r := range ft.Results.List {
						n := len(r.Names)
						if n == 0 {
							n = 1
						}
						m.out += n
					}
				}
				for _, nm := range fl.Names {
					mm := m
					mm.name = nm.Name
					ms = append(ms, mm)
				}
			case *ast.Ident: // embedded interface
				if emb := ifaces[ft.Name]; emb != nil {
					ms = append(ms, resolve(emb)...)
				}
			}
		}
		return ms
	}
	var out []assertion
	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
			ast.Inspect(f, func(n ast.Node) bool {
				ta, ok := n.(*ast.TypeAssertExpr)
				if !ok || ta.Type == nil {
					return true
				}
				sel, ok := ta.X.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				base, ok := sel.X.(*ast.SelectorExpr) // ....d.<Slot>
				if !ok || base.Sel.Name != "d" {
					return true
				}
				a := assertion{slot: sel.Sel.Name, pos: fset.Position(ta.Pos()).String()}
				switch ty := ta.Type.(type) {
				case *ast.Ident:
					it := ifaces[ty.Name]
					if it == nil {
						return true // a concrete type, not an optional interface
					}
					a.iface, a.methods = ty.Name, resolve(it)
				case *ast.InterfaceType:
					a.methods = resolve(ty)
					names := make([]string, len(a.methods))
					for i, m := range a.methods {
						names[i] = m.name
					}
					a.iface = "interface{" + strings.Join(names, ",") + "}"
				default:
					return true
				}
				out = append(out, a)
				return true
			})
		}
	}
	return out
}

func TestProductionTypesSatisfyTheRunnersOptionalInterfaces(t *testing.T) {
	as := botAssertions(t)
	if len(as) < 20 {
		t.Fatalf("found only %d optional-interface assertions on Deps in internal/bot: the scanner is broken", len(as))
	}
	seen := map[string]bool{}
	for _, a := range as {
		types, ok := prodTypes[a.slot]
		if !ok {
			t.Errorf("%s: assertion on Deps.%s (%s) but the slot has no entry in prodTypes", a.pos, a.slot, a.iface)
			continue
		}
		for _, v := range types {
			rt := reflect.TypeOf(v)
			key := a.slot + "/" + strings.TrimPrefix(rt.String(), "*main.") + "/" + a.iface
			key = strings.Replace(key, "main.", "", 1)
			if seen[key] {
				continue
			}
			seen[key] = true
			for _, m := range a.methods {
				got, ok := rt.MethodByName(m.name)
				switch {
				case !ok:
					if why, okAbs := absent[key]; okAbs {
						t.Logf("absent by design: %s (%s)", key, why)
					} else {
						t.Errorf("Deps.%s %s hides %s.%s (asserted at %s): the feature is silently off", a.slot, rt, a.iface, m.name, a.pos)
					}
				case got.Type.NumIn()-1 != m.in || got.Type.NumOut() != m.out:
					t.Errorf("Deps.%s %s: %s has the wrong shape for %s (asserted at %s): the assertion never matches", a.slot, rt, m.name, a.iface, a.pos)
				}
			}
		}
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	t.Logf("checked %d slot/type/interface combinations", len(keys))
}

// otherSeams are the optional-interface assertions below the Deps level (inside combat, nav, town, boss, goals):
// holder = who asserts, prod = the concrete type production hands it. V: types read from buildBehaviors,
// d2dcombat.Attach (the fighter's navigator is the same *nav.Navigator), d2dtown.Input and d2denv.Build.
func TestProductionTypesSatisfyTheInnerOptionalInterfaces(t *testing.T) {
	rt := func(v any) reflect.Type { return reflect.TypeOf(v) }
	cases := []struct {
		holder, what string
		prod         reflect.Type
		methods      []string
	}{
		{"combat.Fighter.mv", "Syncer", rt(&nav.Navigator{}), []string{"Sync"}},
		{"combat.Fighter.mv", "mouth gate", rt(&nav.Navigator{}), []string{"NearMouth", "MouthClickOK"}},
		{"town.Town.mv", "ToArea", rt(&nav.Navigator{}), []string{"ToArea"}},
		{"nav.Navigator.w (d2d)", "fullyKnown", rt(&d2dworld.World{}), []string{"FullyKnown"}},
		{"town.Town.judge", "Valuables", rt(&loot.Looter{}), []string{"Valuables"}},
		{"town.Town.e.Read (d2d)", "affixNamer", rt(&d2denv.Reader{}), []string{"ItemName"}},
		{"town.Town.e.In (d2d)", "beltPutter", rt(d2dtown.Input(nil, nil)), []string{"BeltPut"}},
		{"town.Town.e.In (d2d)", "goldMover", rt(d2dtown.Input(nil, nil)), []string{"MoveGold"}},
		{"town.Town.e.In (d2d)", "WalkingInteractor", rt(d2dtown.Input(nil, nil)), []string{"InteractWalks"}},
		{"bot.Runner.beltFillStep (BELTFIX)", "BeltFiller (fill and layout repair)", rt(townAdapter{}), []string{"NeedsBeltFill", "FillBelt"}},
		{"boss.Run.a", "Teleporter", rt(bossActs{}), []string{"TeleportTo"}},
		{"goals.Manager.Runner", "Configurer + ProgressReporter", rt(&bot.Runner{}), []string{"Configure", "Progress"}},
	}
	for _, c := range cases {
		for _, m := range c.methods {
			if _, ok := c.prod.MethodByName(m); !ok {
				t.Errorf("%s asserts %s but production type %s has no %s: the feature is silently off", c.holder, c.what, c.prod, m)
			}
		}
	}
}

type fakeTown struct {
	bot.Townsman
	forgot, reconciled, belt int
}

func (f *fakeTown) ForgetBelt()                           { f.forgot++ }
func (f *fakeTown) PlannedSpend(*d2.Snapshot) int         { return 77 }
func (f *fakeTown) ReconcileMarks(*d2.Snapshot)           { f.reconciled++ }
func (f *fakeTown) MarkOf(d2.Unit) string                 { return "sell" }
func (f *fakeTown) NeedsBeltFill(*d2.Snapshot) bool       { return true }
func (f *fakeTown) FillBelt(context.Context) (int, error) { f.belt++; return 3, nil }
func (f *fakeTown) UseWeapon(ctx context.Context, item string, fn func(context.Context) error) error {
	return fn(ctx)
}

// A smoke call through the merc wrapper reaches the wrapped town; a wrapped town without the parts answers "off".
func TestMercTownForwardsTheOptionalTownParts(t *testing.T) {
	in := &fakeTown{}
	var tn bot.Townsman = &mercTown{Townsman: in}
	tn.(interface{ ForgetBelt() }).ForgetBelt()
	tn.(interface{ ReconcileMarks(*d2.Snapshot) }).ReconcileMarks(nil)
	if in.forgot != 1 || in.reconciled != 1 {
		t.Errorf("forwards missed: %+v", in)
	}
	if n := tn.(interface{ PlannedSpend(*d2.Snapshot) int }).PlannedSpend(nil); n != 77 {
		t.Errorf("PlannedSpend = %d", n)
	}
	if m := tn.(interface{ MarkOf(d2.Unit) string }).MarkOf(d2.Unit{}); m != "sell" {
		t.Errorf("MarkOf = %q", m)
	}
	bf := tn.(bot.BeltFiller)
	if n, err := bf.FillBelt(context.Background()); n != 3 || err != nil || !bf.NeedsBeltFill(nil) || in.belt != 1 {
		t.Errorf("belt fill: %d %v", n, err)
	}
	ran := false
	if err := tn.(interface {
		UseWeapon(context.Context, string, func(context.Context) error) error
	}).UseWeapon(context.Background(), "hax", func(context.Context) error { ran = true; return nil }); err != nil || !ran {
		t.Errorf("UseWeapon did not reach the inner town: %v", err)
	}
	bare := &mercTown{Townsman: stubTown{}}
	if bare.NeedsBeltFill(nil) || bare.PlannedSpend(nil) != 0 || bare.MarkOf(d2.Unit{}) != "" {
		t.Error("a wrapped town without the parts must answer off")
	}
	if err := bare.UseWeapon(context.Background(), "hax", nil); err == nil {
		t.Error("no weapon swap on the inner town must be an error")
	}
}

// Compile-time guards for optional parts that only a type assertion elsewhere would notice (watch, invdesk).
var (
	_ watch.SensesSource    = (*senses.Publisher)(nil)
	_ watch.TrailSource     = (*senses.Publisher)(nil)
	_ watch.CoverageSource  = (*senses.Publisher)(nil)
	_ watch.EventSource     = (*senses.Publisher)(nil)
	_ watch.HistorySource   = (*senses.Publisher)(nil)
	_ watch.HeatSource      = (*senses.Publisher)(nil)
	_ watch.Teaser          = watch.LadderUnlocker{}
	_ watch.UnitGate        = watch.LadderUnlocker{}
	_ invdesk.StashServices = (*town.Town)(nil)
)
