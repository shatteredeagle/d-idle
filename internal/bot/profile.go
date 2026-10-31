package bot

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
	"time"

	"d2lod-bot/internal/env"
)

// Profile degrades an optimal env.Tuning on purpose (PLAN.md: optimal first, handicap by configuration). Each knob is
// moved from its documented default toward the worse end of its documented range by a strength in [0, 1]: 0 leaves the
// default, 1 goes to the end of the range where the bot still survives. Nothing ever leaves [Min, Max].
type Profile struct {
	Name string
	Desc string
	// Level is the strength for every knob not named in Per.
	Level float64
	// Per overrides the strength for one knob ("nav.pathSlop") or a whole group ("nav.", the longest matching prefix wins).
	Per map[string]float64
}

// Strength returns the handicap strength for a knob key.
func (p Profile) Strength(key string) float64 {
	if s, ok := p.Per[key]; ok {
		return clamp01(s)
	}
	best, bestLen := p.Level, -1
	for k, s := range p.Per {
		if strings.HasSuffix(k, ".") && strings.HasPrefix(key, k) && len(k) > bestLen {
			best, bestLen = s, len(k)
		}
	}
	return clamp01(best)
}

func clamp01(f float64) float64 {
	if f < 0 || math.IsNaN(f) {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}

// Profiles are the built-in named profiles, from optimal to deliberately clumsy.
func Profiles() []Profile {
	return []Profile{
		{Name: "optimal", Desc: "Every knob at its default: the best the bot can do."},
		{Name: "relaxed", Level: 0.2, Desc: "Slightly slower reactions and a few extra town trips."},
		{Name: "casual", Level: 0.4, Desc: "Noticeably human pace, still safe."},
		{Name: "sloppy", Level: 0.7, Desc: "Slow decisions, long detours, over-careful levelling."},
		{Name: "rookie", Level: 1, Desc: "The worst setting every knob allows while the bot still survives."},
	}
}

// ProfileByName finds a built-in profile (case-insensitive); "" means optimal.
func ProfileByName(name string) (Profile, error) {
	if name == "" {
		name = "optimal"
	}
	for _, p := range Profiles() {
		if strings.EqualFold(p.Name, name) {
			return p, nil
		}
	}
	var names []string
	for _, p := range Profiles() {
		names = append(names, p.Name)
	}
	return Profile{}, fmt.Errorf("unknown profile %q (have %s)", name, strings.Join(names, ", "))
}

// Apply writes the profile into t. Knobs that cannot be resolved to a field are skipped and returned as warnings, so a
// knob renamed in one domain does not disable the profile for the others.
func (p Profile) Apply(t *env.Tuning) []string {
	var bad []string
	for _, k := range env.Knobs() {
		if k.NoHandicap {
			continue
		}
		v := k.Default + p.Strength(k.Key)*(worstEnd(k)-k.Default)
		if err := SetKnob(t, k, v); err != nil {
			bad = append(bad, err.Error())
		}
	}
	sort.Strings(bad)
	return bad
}

// worstEnd is the end of the range that is least efficient.
func worstEnd(k env.Knob) float64 {
	if k.HigherIsBetter {
		return k.Min
	}
	return k.Max
}

// ApplyOverrides sets knobs by key (user configuration), clamped to each knob's range. Unknown keys are an error, except retired ones (env.RetiredKnobs), which are ignored.
func ApplyOverrides(t *env.Tuning, over map[string]float64) error {
	byKey := map[string]env.Knob{}
	for _, k := range env.Knobs() {
		byKey[k.Key] = k
	}
	keys := make([]string, 0, len(over))
	for k := range over {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		k, ok := byKey[key]
		if !ok {
			if env.RetiredKnobs[key] {
				continue
			}
			return fmt.Errorf("unknown tuning knob %q", key)
		}
		if err := SetKnob(t, k, over[key]); err != nil {
			return err
		}
	}
	return nil
}

// SetKnob writes v (clamped to the knob's range) into the Tuning field named by the knob key
// ("combat.chickenLifePct" is Tuning.Combat.ChickenLifePct). Supported field kinds: ints, floats, bool, time.Duration
// (the knob unit must then be ns, us, ms or s).
func SetKnob(t *env.Tuning, k env.Knob, v float64) error {
	f, err := knobField(t, k.Key)
	if err != nil {
		return err
	}
	if math.IsNaN(v) {
		return fmt.Errorf("knob %s: value is NaN", k.Key)
	}
	v = math.Min(math.Max(v, k.Min), k.Max)
	switch {
	case f.Type() == reflect.TypeOf(time.Duration(0)):
		unit, ok := map[string]time.Duration{"ns": time.Nanosecond, "us": time.Microsecond, "ms": time.Millisecond, "s": time.Second}[k.Unit]
		if !ok {
			return fmt.Errorf("knob %s: Duration field needs unit ns/us/ms/s, has %q", k.Key, k.Unit)
		}
		f.SetInt(int64(math.Round(v * float64(unit))))
	case f.CanInt():
		f.SetInt(int64(math.Round(v)))
	case f.CanUint():
		f.SetUint(uint64(math.Round(v)))
	case f.CanFloat():
		f.SetFloat(v)
	case f.Kind() == reflect.Bool:
		f.SetBool(v >= 0.5)
	default:
		return fmt.Errorf("knob %s: unsupported field type %s", k.Key, f.Type())
	}
	return nil
}

// GetKnob reads a knob's current value as a float64 in the knob's unit.
func GetKnob(t *env.Tuning, k env.Knob) (float64, error) {
	f, err := knobField(t, k.Key)
	if err != nil {
		return 0, err
	}
	switch {
	case f.Type() == reflect.TypeOf(time.Duration(0)):
		unit, ok := map[string]time.Duration{"ns": time.Nanosecond, "us": time.Microsecond, "ms": time.Millisecond, "s": time.Second}[k.Unit]
		if !ok {
			return 0, fmt.Errorf("knob %s: Duration field needs unit ns/us/ms/s, has %q", k.Key, k.Unit)
		}
		return float64(f.Int()) / float64(unit), nil
	case f.CanInt():
		return float64(f.Int()), nil
	case f.CanUint():
		return float64(f.Uint()), nil
	case f.CanFloat():
		return f.Float(), nil
	case f.Kind() == reflect.Bool:
		if f.Bool() {
			return 1, nil
		}
		return 0, nil
	}
	return 0, fmt.Errorf("knob %s: unsupported field type %s", k.Key, f.Type())
}

func knobField(t *env.Tuning, key string) (reflect.Value, error) {
	grp, name, ok := strings.Cut(key, ".")
	if !ok || grp == "" || name == "" {
		return reflect.Value{}, fmt.Errorf("knob key %q is not group.name", key)
	}
	match := func(want string) func(string) bool {
		return func(have string) bool { return strings.EqualFold(have, want) }
	}
	g := reflect.ValueOf(t).Elem().FieldByNameFunc(match(grp))
	if !g.IsValid() || g.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("knob %s: no Tuning group %q", key, grp)
	}
	f := g.FieldByNameFunc(match(name))
	if !f.IsValid() || !f.CanSet() {
		return reflect.Value{}, fmt.Errorf("knob %s: no field %q", key, name)
	}
	return f, nil
}
