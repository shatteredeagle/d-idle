package cubejobs

import "testing"

func TestGemFor(t *testing.T) {
	want := map[int]string{11: "gcy", 12: "gcv", 16: "gcw", 17: "gfy", 21: "gfg", 22: "gfw", 23: "gsy", 28: "gsw", 29: "gly", 33: "glg"}
	for n, g := range want {
		if got := gemFor(n); got != g {
			t.Errorf("r%d: %s want %s", n, got, g)
		}
	}
}

func TestRuneChain(t *testing.T) {
	jobs := Plan(map[string]int{"r07": 27}, Opts{MaxJobs: 20, RuneCap: 10})
	if len(jobs) != 13 || jobs[len(jobs)-1].Out != "r10" {
		t.Fatalf("%v", jobs)
	}
}

func TestRuneCapAndKeep(t *testing.T) {
	if j := Plan(map[string]int{"r10": 9}, Opts{MaxJobs: 5, RuneCap: 10}); len(j) != 0 {
		t.Fatalf("cap: %v", j)
	}
	if j := Plan(map[string]int{"r07": 5}, Opts{MaxJobs: 5, RuneCap: 10, Keep: map[string]int{"r07": 3}}); len(j) != 0 {
		t.Fatalf("keep: %v", j)
	}
	if j := Plan(map[string]int{"r07": 6}, Opts{MaxJobs: 1, RuneCap: 10}); len(j) != 1 {
		t.Fatalf("max: %v", j)
	}
}

func TestRuneWithGem(t *testing.T) {
	have := map[string]int{"r10": 3}
	if j := Plan(have, Opts{MaxJobs: 3, RuneCap: 11}); len(j) != 0 {
		t.Fatalf("no gem: %v", j)
	}
	have["gcy"] = 1
	j := Plan(have, Opts{MaxJobs: 3, RuneCap: 11})
	if len(j) != 1 || j[0].Out != "r11" || len(j[0].Inputs) != 4 {
		t.Fatalf("%v", j)
	}
}

func TestGemChain(t *testing.T) {
	j := Plan(map[string]int{"gcv": 9}, Opts{MaxJobs: 9, GemCap: 2})
	if len(j) != 4 || j[3].Out != "gsv" {
		t.Fatalf("%v", j)
	}
	if j := Plan(map[string]int{"gcv": 9}, Opts{MaxJobs: 9, GemCap: 0}); len(j) != 0 {
		t.Fatalf("cap %v", j)
	}
}
