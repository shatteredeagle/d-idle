package decide

import (
	"fmt"
	"sort"
	"strings"
)

// Obs is the raw state a call site reports. Numbers are banded here, never stored raw.
type Obs struct {
	Class string             // build class, resolved through the catalogue's class table
	Num   map[string]float64 // band and classband fields
	Cat   map[string]string  // enum fields (act, diff, hc, kit, ...)
}

// Context is a decision point plus the banded values of its key fields. Key is the string identity.
type Context struct {
	Point  string
	Fields map[string]string
	Key    string
}

// UrgencyProvider is the stage-2 hook (spec 2.3): combat computes the urgency band from its damage window.
type UrgencyProvider interface {
	Urgency(o Obs) (string, error)
}

// BandLabel bands one numeric value for a field. A value equal to an edge falls into the upper band.
func (c *Catalogue) BandLabel(class, field string, v float64) (string, error) {
	f, ok := c.Fields[field]
	if !ok {
		return "", fmt.Errorf("decide: unknown field %q", field)
	}
	edges := f.Edges
	switch f.Kind {
	case "band":
	case "classband":
		cl, ok := c.Classes[class]
		if !ok {
			return "", fmt.Errorf("decide: unknown class %q", class)
		}
		if edges, ok = cl.Bands[field]; !ok {
			return "", fmt.Errorf("decide: class %q has no band for field %q", class, field)
		}
	default:
		return "", fmt.Errorf("decide: field %q is not numeric", field)
	}
	i := 0
	for i < len(edges) && v >= edges[i] {
		i++
	}
	return f.Labels[i], nil
}

// Build makes the context of a point from observations. stage 2 replaces life by urgency on urgency points and needs
// a provider (or Cat["urgency"]). An unknown point, class, missing field or bad value is an error naming it.
func (c *Catalogue) Build(pointID string, o Obs, stage int, urg UrgencyProvider) (Context, error) {
	p, err := c.Point(pointID)
	if err != nil {
		return Context{}, err
	}
	if _, ok := c.Classes[o.Class]; !ok {
		return Context{}, fmt.Errorf("decide: unknown class %q", o.Class)
	}
	names := append([]string{"act", "diff", "hc"}, p.ContextKeys...)
	fields := map[string]string{}
	for _, n := range names {
		if p.Urgency && stage >= 2 && c.isLife(n) {
			n = "urgency"
		}
		if _, dup := fields[n]; dup {
			continue
		}
		f := c.Fields[n]
		switch f.Kind {
		case "enum":
			v, ok := o.Cat[n]
			if !ok && n == "urgency" && urg != nil {
				var err error
				if v, err = urg.Urgency(o); err != nil {
					return Context{}, fmt.Errorf("decide: urgency: %w", err)
				}
				ok = true
			}
			if !ok && f.Absent != "" {
				v, ok = f.Absent, true
			}
			if !ok {
				return Context{}, fmt.Errorf("decide: point %q: missing field %q", pointID, n)
			}
			if !contains(f.Values, v) {
				return Context{}, fmt.Errorf("decide: field %q: unknown value %q", n, v)
			}
			fields[n] = v
		default:
			src := n
			if f.From != "" {
				src = f.From
			}
			v, ok := o.Num[src]
			if !ok {
				return Context{}, fmt.Errorf("decide: point %q: missing field %q", pointID, src)
			}
			l, err := c.BandLabel(o.Class, n, v)
			if err != nil {
				return Context{}, err
			}
			fields[n] = l
		}
	}
	return Context{Point: pointID, Fields: fields, Key: makeKey(pointID, fields)}, nil
}

func makeKey(point string, fields map[string]string) string {
	ks := make([]string, 0, len(fields))
	for k := range fields {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	parts := make([]string, len(ks))
	for i, k := range ks {
		parts[i] = k + "=" + fields[k]
	}
	return point + "|" + strings.Join(parts, ",")
}

// ParseKey is the inverse of the key format.
func ParseKey(key string) (Context, error) {
	point, rest, ok := strings.Cut(key, "|")
	if !ok || point == "" {
		return Context{}, fmt.Errorf("decide: bad context key %q", key)
	}
	fields := map[string]string{}
	if rest != "" {
		for _, kv := range strings.Split(rest, ",") {
			k, v, ok := strings.Cut(kv, "=")
			if !ok || k == "" {
				return Context{}, fmt.Errorf("decide: bad context key %q", key)
			}
			fields[k] = v
		}
	}
	return Context{Point: point, Fields: fields, Key: key}, nil
}

// Plain renders a key's fields for the rules list ("act 1, diff normal, ...").
func (c Context) Plain() string {
	ks := make([]string, 0, len(c.Fields))
	for k := range c.Fields {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	parts := make([]string, len(ks))
	for i, k := range ks {
		parts[i] = k + " " + c.Fields[k]
	}
	return strings.Join(parts, ", ")
}
