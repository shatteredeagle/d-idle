package decide

import (
	"math/rand"
	"sort"
	"testing"
	"time"
)

func mustCat(t testing.TB) *Catalogue {
	t.Helper()
	c, err := Shipped()
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func sortedFields(c *Catalogue) []string {
	var ns []string
	for n := range c.Fields {
		ns = append(ns, n)
	}
	sort.Strings(ns)
	return ns
}

// baseObs fills every field with a valid value (first enum value, a value in band index 1).
func baseObs(c *Catalogue, class string) Obs {
	o := Obs{Class: class, Num: map[string]float64{}, Cat: map[string]string{}}
	for _, n := range sortedFields(c) {
		f := c.Fields[n]
		if f.Kind == "enum" {
			o.Cat[n] = f.Values[0]
		} else {
			o.Num[n] = bandValue(c, class, n, 1)
		}
	}
	return o
}

// bandValue is a number that lands in label index i of a numeric field.
func bandValue(c *Catalogue, class, n string, i int) float64 {
	f := c.Fields[n]
	edges := f.Edges
	if f.Kind == "classband" {
		edges = c.Classes[class].Bands[n]
	}
	if len(edges) == 0 {
		return 0 // unknown class: the caller expects an error
	}
	if i == 0 {
		return edges[0] - 1
	}
	return edges[i-1]
}

func (o Obs) with(cat map[string]string, num map[string]float64) Obs {
	n := Obs{Class: o.Class, Num: map[string]float64{}, Cat: map[string]string{}}
	for k, v := range o.Num {
		n.Num[k] = v
	}
	for k, v := range o.Cat {
		n.Cat[k] = v
	}
	for k, v := range cat {
		n.Cat[k] = v
	}
	for k, v := range num {
		n.Num[k] = v
	}
	return n
}

type rig struct {
	t   *testing.T
	cat *Catalogue
	clk *ManualClock
	s   *Store
}

func newRig(t *testing.T) *rig {
	t.Helper()
	cat := mustCat(t)
	clk := &ManualClock{}
	s, err := New(cat, "p1", clk)
	if err != nil {
		t.Fatal(err)
	}
	return &rig{t, cat, clk, s}
}

func (r *rig) ctx(point string, cat map[string]string, num map[string]float64) Context {
	r.t.Helper()
	c, err := r.cat.Build(point, baseObs(r.cat, "sorceress").with(cat, num), 1, nil)
	if err != nil {
		r.t.Fatal(err)
	}
	return c
}

// ask asks and returns the open thought id.
func (r *rig) ask(c Context) Decision {
	r.t.Helper()
	d, err := r.s.Ask(c)
	if err != nil {
		r.t.Fatal(err)
	}
	return d
}

// say asks and answers once.
func (r *rig) say(c Context, ans string) {
	r.t.Helper()
	d := r.ask(c)
	if d.Thought == nil {
		r.t.Fatalf("no thought for %s (source %s)", c.Key, d.Source)
	}
	if err := r.s.Answer(d.Thought.ID, ans); err != nil {
		r.t.Fatal(err)
	}
}

func (r *rig) rec(c Context) Record {
	rec, _ := r.s.Record(c.Key)
	return rec
}

func (r *rig) pendingConfirm(c Context) int {
	for _, th := range r.s.OpenThoughts() {
		if th.Confirm && th.Key == c.Key {
			return th.ID
		}
	}
	return 0
}

func (r *rig) learned(c Context) bool {
	rec := r.rec(c)
	return rec.Rule != nil && !rec.Rule.Revoked
}

var fade = 46 * time.Second

// randObs gives a random valid observation for any point of the catalogue.
func randObs(rng *rand.Rand, c *Catalogue, class string) Obs {
	o := Obs{Class: class, Num: map[string]float64{}, Cat: map[string]string{}}
	for _, n := range sortedFields(c) {
		f := c.Fields[n]
		if f.Kind == "enum" {
			o.Cat[n] = f.Values[rng.Intn(len(f.Values))]
		} else {
			o.Num[n] = bandValue(c, class, n, rng.Intn(len(f.Labels)))
		}
	}
	return o
}
