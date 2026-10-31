// Package cubejobs plans Horadric Cube upgrades that cost no gold: gem grade upgrades and rune upgrades.
// Pure: it sees item counts and returns an ordered list of transmutes; internal/town runs them.
//
// Recipes are from docs/items.md section 4 (V: copied from this install's cubemain.txt rows). The item codes
// (gcv chipped amethyst ... r01 El .. r33 Zod) are V for r01/r07/r10/gcv/gfv (used by the data tests and the
// pickit tests) and S (standard D2 tables) for the other gem codes. Whether Dol->Hel and above work in offline
// single-player is U (docs/items.md Q1): RuneCap defaults to Thul, which has no ladder flag.
package cubejobs

import "fmt"

// Job is one transmute: put Inputs in the cube, expect Out.
type Job struct {
	Inputs []string
	Out    string
}

func (j Job) String() string { return fmt.Sprintf("%v -> %s", j.Inputs, j.Out) }

// Opts bounds the planner.
type Opts struct {
	MaxJobs int            // transmutes per visit (0: none)
	RuneCap int            // highest rune number r<N> the planner may produce (10 = Thul)
	GemCap  int            // highest gem grade it may produce, 0 chipped .. 4 perfect
	Keep    map[string]int // codes never consumed (runes a wanted runeword needs, gems for a recipe)
}

var gemTypes = []string{"v", "y", "b", "g", "r", "w"} // amethyst, topaz, sapphire, emerald, ruby, diamond
var gemGrade = []string{"gc", "gf", "gs", "gl", "gp"}

func gem(typ, grade int) string { return gemGrade[grade] + gemTypes[typ] }

func rune_(n int) string { return fmt.Sprintf("r%02d", n) }

// GemCodes lists every code the planner handles (for the town to know what to pull from the stash).
func GemCodes() []string {
	var out []string
	for g := range gemGrade {
		for t := range gemTypes {
			out = append(out, gem(t, g))
		}
	}
	return out
}

// RuneCodes lists r01..r33.
func RuneCodes() []string {
	var out []string
	for n := 1; n <= 33; n++ {
		out = append(out, rune_(n))
	}
	return out
}

// runeNeed returns how many runes of number n-1 make rune n, and the gem it also needs ("" for none).
// V cubemain, docs/items.md 4.1/4.2: r02..r10 3 of the lower rune; r11..r21 3 + gem; r22..r33 2 + gem.
func runeNeed(n int) (qty int, gemCode string) {
	switch {
	case n <= 10:
		return 3, ""
	case n <= 21:
		return 3, gemFor(n)
	default:
		return 2, gemFor(n)
	}
}

// gemFor: from Amn (r11) the gem type cycles topaz, amethyst, sapphire, ruby, emerald, diamond and the grade
// steps up every 6 runes (chipped r11..r16, flawed r17..r22, standard r23..r28, flawless r29..r33). V cubemain list.
func gemFor(n int) string {
	order := []int{1, 0, 2, 4, 3, 5} // indexes into gemTypes
	k := n - 11
	return gem(order[k%6], k/6)
}

// Plan returns at most MaxJobs transmutes that turn spare items in have into higher ones. Rune upgrades come
// first (lowest rune first so a chain cascades), then gem upgrades from three of a kind. have is not modified.
func Plan(have map[string]int, o Opts) []Job {
	n := map[string]int{}
	for k, v := range have {
		n[k] = v
	}
	spare := func(c string) int { return n[c] - o.Keep[c] }
	var jobs []Job
	add := func(in []string, out string) {
		for _, c := range in {
			n[c]--
		}
		n[out]++
		jobs = append(jobs, Job{in, out})
	}
	for len(jobs) < o.MaxJobs {
		done := false
		for r := 2; r <= o.RuneCap && r <= 33 && len(jobs) < o.MaxJobs; r++ {
			qty, g := runeNeed(r)
			lo := rune_(r - 1)
			if spare(lo) < qty || g != "" && spare(g) < 1 {
				continue
			}
			var in []string
			for i := 0; i < qty; i++ {
				in = append(in, lo)
			}
			if g != "" {
				in = append(in, g)
			}
			add(in, rune_(r))
			done = true
			break
		}
		if done {
			continue
		}
		for gr := 0; gr < o.GemCap && gr < 4 && !done; gr++ {
			for t := range gemTypes {
				lo := gem(t, gr)
				if spare(lo) >= 3 {
					add([]string{lo, lo, lo}, gem(t, gr+1))
					done = true
					break
				}
			}
		}
		if !done {
			break
		}
	}
	return jobs
}
