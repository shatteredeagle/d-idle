package d2dsrc

import (
	"encoding/json"
	"fmt"
	"math"
)

// obj is a decoded JSON object with presence-aware accessors: a missing key is reported, never a zero.
// The dump schema is ours but d2d's fork may lag it, so every typed read says whether the key was there.
type obj map[string]any

func parseObj(b []byte) (obj, error) {
	var o obj
	if err := json.Unmarshal(b, &o); err != nil {
		return nil, fmt.Errorf("d2dsrc: dump is not a JSON object: %w", err)
	}
	if o == nil {
		return nil, fmt.Errorf("d2dsrc: dump is null")
	}
	return o, nil
}

func (o obj) sub(k string) obj {
	m, _ := o[k].(map[string]any)
	return obj(m)
}

func (o obj) num(k string) (float64, bool) {
	f, ok := o[k].(float64)
	if !ok || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, false
	}
	return f, true
}

func (o obj) int(k string) (int, bool) {
	f, ok := o.num(k)
	return int(f), ok
}

func (o obj) str(k string) (string, bool) { s, ok := o[k].(string); return s, ok }
func (o obj) bool(k string) (bool, bool)  { b, ok := o[k].(bool); return b, ok }

// ints reads an array of numbers; ok is false when any element is not a number.
func (o obj) ints(k string) ([]int, bool) {
	a, ok := o[k].([]any)
	if !ok {
		return nil, false
	}
	out := make([]int, len(a))
	for i, v := range a {
		f, ok := v.(float64)
		if !ok {
			return nil, false
		}
		out[i] = int(f)
	}
	return out, true
}

// pair reads a [cur, max] array.
func (o obj) pair(k string) (cur, max int, ok bool) {
	p, ok := o.ints(k)
	if !ok || len(p) != 2 {
		return 0, 0, false
	}
	return p[0], p[1], true
}

func (o obj) list(k string) []obj {
	a, _ := o[k].([]any)
	out := make([]obj, 0, len(a))
	for _, v := range a {
		if m, ok := v.(map[string]any); ok {
			out = append(out, obj(m))
		}
	}
	return out
}
