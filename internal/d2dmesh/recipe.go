package d2dmesh

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
)

// A Recipe is one allow-listed kind of work. Jobs name a recipe and give string arguments;
// Validate rejects anything outside the recipe's ranges. A node never runs arbitrary shell:
// the Runner maps a validated recipe to a fixed local command (docs/d2d-mesh.md section 8).
type Recipe struct {
	Name string
	Args map[string]ArgSpec
}

// ArgSpec validates one argument. Exactly one of Enum, Range or Pattern is set.
type ArgSpec struct {
	Required bool
	Enum     []string
	Min, Max int64 // used when Range
	Range    bool
	Pattern  *regexp.Regexp
}

var ident = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

// DefaultRecipes is the allow-list. Argument ranges are deliberately tight; widen them in a
// spec delta, not ad hoc. Source for the d2d flags: docs/d2d-container/README.md.
var DefaultRecipes = map[string]Recipe{
	// sleep is the dry-run recipe: the runner sleeps for seconds instead of launching anything.
	"sleep": {Name: "sleep", Args: map[string]ArgSpec{
		"seconds": {Required: true, Range: true, Min: 0, Max: 3600},
	}},
	// d2d-soak: a hidden d2d engine under the local slot pool for hours with a class and seed.
	"d2d-soak": {Name: "d2d-soak", Args: map[string]ArgSpec{
		"hours": {Required: true, Range: true, Min: 1, Max: 24},
		"class": {Required: true, Range: true, Min: 1, Max: 7},
		"seed":  {Range: true, Min: 0, Max: 1<<31 - 1},
		"mode":  {Enum: []string{"lockstep", "realtime"}},
		"name":  {Pattern: ident},
	}},
	// d2d-replay: replay one recorded scenario and compare (cross-host determinism check).
	"d2d-replay": {Name: "d2d-replay", Args: map[string]ArgSpec{
		"scenario": {Required: true, Pattern: ident},
		"seed":     {Range: true, Min: 0, Max: 1<<31 - 1},
	}},
}

// Validate checks that recipe is allow-listed and args satisfy its specs (unknown args are refused).
func Validate(recipes map[string]Recipe, recipe string, args map[string]string) error {
	r, ok := recipes[recipe]
	if !ok {
		return fmt.Errorf("recipe %q is not on the allow-list", recipe)
	}
	names := make([]string, 0, len(r.Args))
	for n := range r.Args {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		spec := r.Args[n]
		v, present := args[n]
		if !present {
			if spec.Required {
				return fmt.Errorf("recipe %s: argument %q is required", recipe, n)
			}
			continue
		}
		if err := spec.check(v); err != nil {
			return fmt.Errorf("recipe %s: argument %q: %v", recipe, n, err)
		}
	}
	for n := range args {
		if _, ok := r.Args[n]; !ok {
			return fmt.Errorf("recipe %s: unknown argument %q", recipe, n)
		}
	}
	return nil
}

func (a ArgSpec) check(v string) error {
	switch {
	case a.Range:
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n < a.Min || n > a.Max {
			return fmt.Errorf("want an integer in %d..%d, got %q", a.Min, a.Max, v)
		}
	case len(a.Enum) > 0:
		for _, e := range a.Enum {
			if e == v {
				return nil
			}
		}
		return fmt.Errorf("want one of %v, got %q", a.Enum, v)
	case a.Pattern != nil:
		if !a.Pattern.MatchString(v) {
			return fmt.Errorf("does not match %s", a.Pattern)
		}
	default:
		return fmt.Errorf("argument spec has no rule")
	}
	return nil
}
