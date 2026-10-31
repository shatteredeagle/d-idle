package town_test

import (
	"context"
	"d2lod-bot/internal/gear/geartest"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/town"
)

func cubeRig(t *testing.T, stash map[string]int, pack ...string) *rig {
	r := newRig(t)
	r.Env.Tune.Bot.CubeJobs = true
	r.Env.Tune.Bot.CubeMaxJobs = 6
	r.Env.Tune.Bot.CubeRuneCap = 10
	r.Env.Tune.Bot.CubeGemCap = 2
	r.W.Add("box", d2.ContInventory, 0, 0)
	x := 0
	for _, c := range pack {
		r.W.Add(c, d2.ContInventory, 3+x%7, 1+x/7)
		x++
	}
	y := 0
	for c, n := range stash {
		for i := 0; i < n; i++ {
			r.W.Add(c, d2.ContStash, y%6, y/6)
			y++
		}
	}
	return r
}

func TestCubeJobsRunesFromStash(t *testing.T) {
	geartest.Need(t)
	r := cubeRig(t, map[string]int{"r07": 5}, "r07")
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Cube: true})
	if err != nil || rep.Errors != 0 || rep.Cubed != 2 {
		t.Fatalf("%v cubed %d %+v", err, rep.Cubed, rep.Steps)
	}
	if m := cubeCodes(r, d2.ContStash); m["r08"] != 2 || m["r07"] != 0 {
		t.Fatalf("stash %v", m)
	}
	if m := cubeCodes(r, d2.ContInventory); m["r08"] != 0 || m["r07"] != 0 || m["box"] != 1 {
		t.Fatalf("pack %v", m)
	}
}

func TestCubeJobsKnobOffAndKeep(t *testing.T) {
	geartest.Need(t)
	r := cubeRig(t, map[string]int{"r07": 6})
	r.Env.Tune.Bot.CubeJobs = false
	rep, _ := r.T.Chores(context.Background(), town.ChorePlan{Cube: true})
	if rep.Cubed != 0 || cubeCodes(r, d2.ContStash)["r07"] != 6 {
		t.Fatal("ran while off")
	}
	r.Env.Tune.Bot.CubeJobs = true
	rep, _ = r.T.Chores(context.Background(), town.ChorePlan{Cube: true, CubeKeep: map[string]int{"r07": 6}})
	if rep.Cubed != 0 {
		t.Fatal("consumed kept runes")
	}
}

func TestCubeJobsBoundedAndNoCube(t *testing.T) {
	geartest.Need(t)
	r := cubeRig(t, map[string]int{"gcv": 9})
	r.Env.Tune.Bot.CubeMaxJobs = 2
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Cube: true})
	if err != nil || rep.Cubed != 2 {
		t.Fatalf("%v cubed %d", err, rep.Cubed)
	}
	if m := cubeCodes(r, d2.ContStash); m["gcv"] != 3 || m["gfv"] != 2 {
		t.Fatalf("stash %v", m)
	}
	r2 := newRig(t)
	r2.Env.Tune.Bot.CubeJobs = true
	if rep, _ := r2.T.Chores(context.Background(), town.ChorePlan{Cube: true}); rep.Cubed != 0 {
		t.Fatal("no cube")
	}
}
