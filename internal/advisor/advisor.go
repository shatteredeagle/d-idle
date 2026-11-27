// Package advisor suggests runewords and cube upgrades from what the hero owns (roadmap R14, TASK-991). It only
// returns advice lines; it never moves an item (owner rule: inventory beyond starter gear and potions is
// player-driven). Pure: the caller gathers the stock from the pack, stash and vault and shows the lines.
package advisor

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"d2lod-bot/internal/cubejobs"
)

//go:embed advisor.json
var raw []byte

// Recipe is one runeword row of advisor.json.
type Recipe struct {
	Name    string   `json:"name"`
	Runes   []string `json:"runes"` // names in socket order
	Types   []string `json:"types"` // base item type codes, any of
	Classes []string `json:"classes"`
	Rank    int      `json:"rank"`
	Ladder  bool     `json:"ladder"`
	Merc    bool     `json:"merc"`
	Why     string   `json:"why"`
}

// runeNames is r01..r33 in order (V docs/items.md section 2).
var runeNames = strings.Fields("El Eld Tir Nef Eth Ith Tal Ral Ort Thul Amn Sol Shael Dol Hel Io Lum Ko Fal Lem Pul Um Mal Ist Gul Vex Ohm Lo Sur Ber Jah Cham Zod")

// RuneCode maps "Tal" to "r07"; false for an unknown name.
func RuneCode(name string) (string, bool) {
	for i, n := range runeNames {
		if strings.EqualFold(n, name) {
			return fmt.Sprintf("r%02d", i+1), true
		}
	}
	return "", false
}

var recipes = func() []Recipe {
	var t struct {
		Runewords []Recipe `json:"runewords"`
	}
	if err := json.Unmarshal(raw, &t); err != nil {
		panic("advisor: bad advisor.json: " + err.Error())
	}
	for _, r := range t.Runewords {
		for _, n := range r.Runes {
			if _, ok := RuneCode(n); !ok {
				panic("advisor: " + r.Name + ": unknown rune " + n)
			}
		}
	}
	return t.Runewords
}()

// Recipes returns the table (a copy).
func Recipes() []Recipe { return append([]Recipe(nil), recipes...) }

// Base is one item that may become a runeword base.
type Base struct {
	Name    string   // display name, may be empty
	Code    string   // item code
	Chain   []string // type chain, most specific first (swor, weap)
	Quality string   // "normal", "superior", "low", "magic", ... (status.Item words)
	Sockets int
	Filled  int
	Where   string // "pack", "stash", "vault"
}

// Stock is what the hero owns, gathered by the caller.
type Stock struct {
	Runes map[string]int // item code r01..r33 -> count, all places together
	Gems  map[string]int // gem codes for cube upgrades (gcv .. gpw)
	Bases []Base
}

// Options select what is advised.
type Options struct {
	Class  string // required; the class the advice must help (lower case, as plandata names it)
	Ladder bool   // items.ladderRunewords: ladder rows and rune upgrades above Sol count
	Max    int    // lines returned, default 3
}

// Kind of an advice.
type Kind string

const (
	Make     Kind = "make" // runes and a base are in hand: socket it
	NeedBase Kind = "base" // runes in hand, no base yet
	Cube     Kind = "cube" // the runes can be had by cube upgrades from spare runes
	Near     Kind = "near" // one rune short
)

// Advice is one suggestion.
type Advice struct {
	Kind     Kind
	Runeword string
	Line     string
	Rank     int
	Class    bool // the recipe names the class (not a general recipe)
	Merc     bool
}

func relevant(r Recipe, class string) (ok, named bool) {
	if len(r.Classes) == 0 {
		return true, false
	}
	for _, c := range r.Classes {
		if strings.EqualFold(c, class) {
			return true, true
		}
	}
	return false, false
}

func whiteBase(q string) bool { return q == "normal" || q == "superior" }

func pickBase(r Recipe, bases []Base) *Base {
	for i := range bases {
		b := &bases[i]
		if !whiteBase(b.Quality) || b.Sockets != len(r.Runes) || b.Filled != 0 {
			continue
		}
		for _, t := range r.Types {
			for _, c := range append([]string{b.Code}, b.Chain...) {
				if c == t {
					return b
				}
			}
		}
	}
	return nil
}

func runeList(r Recipe) string { return strings.Join(r.Runes, " ") }

func needs(r Recipe) map[string]int {
	n := map[string]int{}
	for _, name := range r.Runes {
		c, _ := RuneCode(name)
		n[c]++
	}
	return n
}

// Advise returns the suggestions for the class, best first. An empty class is an error: no fallback to a default class.
func Advise(s Stock, o Options) ([]Advice, error) {
	if o.Class == "" {
		return nil, fmt.Errorf("advisor: no class given")
	}
	if o.Max <= 0 {
		o.Max = 3
	}
	var out []Advice
	for _, r := range recipes {
		if ok, _ := relevant(r, o.Class); !ok || (r.Ladder && !o.Ladder) {
			continue
		}
		_, named := relevant(r, o.Class)
		a := Advice{Runeword: r.Name, Rank: r.Rank, Class: named, Merc: r.Merc}
		need := needs(r)
		missing, short := 0, ""
		for c, n := range need {
			if d := n - s.Runes[c]; d > 0 {
				missing += d
				short = c
			}
		}
		who := "for " + r.Name
		if r.Merc {
			who += " (best on the mercenary)"
		}
		switch {
		case missing == 0:
			if b := pickBase(r, s.Bases); b != nil {
				a.Kind = Make
				a.Line = fmt.Sprintf("You have the runes %s (%s); socket base: %s, %d sockets, in the %s.", who, runeList(r), baseName(b), b.Sockets, b.Where)
			} else {
				a.Kind = NeedBase
				a.Line = fmt.Sprintf("You have the runes %s (%s); you need a plain %d-socket %s base (Larzuk can socket a plain one).", who, runeList(r), len(r.Runes), typeWords(r.Types))
			}
		default:
			steps, ok := cubePlan(s, need, o.Ladder)
			switch {
			case ok:
				a.Kind = Cube
				a.Line = fmt.Sprintf("Cube the spare runes to get the runes %s (%s): %s.", who, runeList(r), strings.Join(steps, ", then "))
			case missing == 1 && short != "":
				a.Kind = Near
				a.Line = fmt.Sprintf("One rune short %s (%s): %s.", who, runeList(r), runeName(short))
			default:
				continue
			}
		}
		out = append(out, a)
	}
	order := map[Kind]int{Make: 0, NeedBase: 1, Cube: 2, Near: 3}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if order[a.Kind] != order[b.Kind] {
			return order[a.Kind] < order[b.Kind]
		}
		if a.Class != b.Class {
			return a.Class
		}
		return a.Rank > b.Rank
	})
	if len(out) > o.Max {
		out = out[:o.Max]
	}
	return out, nil
}

// Lines is Advise reduced to the text lines the Return screen and the Vault show.
func Lines(s Stock, o Options) ([]string, error) {
	adv, err := Advise(s, o)
	if err != nil {
		return nil, err
	}
	var l []string
	for _, a := range adv {
		l = append(l, a.Line)
	}
	return l, nil
}

func baseName(b *Base) string {
	if b.Name != "" {
		return b.Name
	}
	return b.Code
}

func typeWords(t []string) string { return strings.Join(t, "/") }

func runeName(code string) string {
	var n int
	fmt.Sscanf(code, "r%d", &n)
	if n < 1 || n > len(runeNames) {
		return code
	}
	return runeNames[n-1]
}

var gemTypes = map[byte]string{'v': "Amethyst", 'y': "Topaz", 'b': "Sapphire", 'g': "Emerald", 'r': "Ruby", 'w': "Diamond"}
var gemGrades = map[string]string{"gc": "Chipped", "gf": "Flawed", "gs": "", "gl": "Flawless", "gp": "Perfect"}

func gemName(code string) string {
	if len(code) != 3 {
		return code
	}
	g, ok := gemGrades[code[:2]]
	t, ok2 := gemTypes[code[2]]
	if !ok || !ok2 {
		return code
	}
	return strings.TrimSpace(g + " " + t)
}

// cubePlan returns the transmute steps that turn spare runes (never the ones the target needs) into the missing
// ones, or false when the stock does not allow it. Rune upgrades above Sol (r12) need the ladder switch (docs/items.md Q1, U).
func cubePlan(s Stock, need map[string]int, ladder bool) ([]string, bool) {
	runeCap := 13
	if ladder {
		runeCap = 33
	}
	have := map[string]int{}
	for k, v := range s.Runes {
		have[k] = v
	}
	gems := map[string]int{}
	for k, v := range s.Gems {
		gems[k] = v
	}
	var steps []string
	var make1 func(n, depth int) bool
	spare := func(n int) int { return have[fmt.Sprintf("r%02d", n)] - need[fmt.Sprintf("r%02d", n)] }
	make1 = func(n, depth int) bool {
		if n < 2 || n > runeCap || depth > 6 {
			return false
		}
		qty, gem := cubejobs.RuneNeed(n)
		if gem != "" && gems[gem] < 1 {
			return false
		}
		for spare(n-1) < qty {
			if !make1(n-1, depth+1) {
				return false
			}
		}
		lo := fmt.Sprintf("r%02d", n-1)
		have[lo] -= qty
		have[fmt.Sprintf("r%02d", n)]++
		step := fmt.Sprintf("%d %s", qty, runeName(lo))
		if gem != "" {
			gems[gem]--
			step += " and a " + gemName(gem)
		}
		steps = append(steps, step+" make "+runeName(fmt.Sprintf("r%02d", n)))
		return true
	}
	codes := make([]string, 0, len(need))
	for c := range need {
		codes = append(codes, c)
	}
	sort.Strings(codes)
	for _, c := range codes {
		var n int
		fmt.Sscanf(c, "r%d", &n)
		for have[c] < need[c] {
			if !make1(n, 0) {
				return nil, false
			}
		}
	}
	return steps, len(steps) > 0
}
