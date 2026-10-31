// Package metrics is a small stdlib metrics registry (counters, gauges, histograms) with Prometheus text
// exposition (format 0.0.4), process statistics read from /proc and structured-log helpers. It exists so
// the soak harness and the dashboard can scrape long runs (gotcha 66) without a dependency.
//
// Mount it on the server: server.Config.Mounts = []server.Mount{{Path: "/metrics/", Handler: reg.Handler()}}
// (the handler answers on any path, so the mount's prefix does not matter).
package metrics

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

var (
	nameRE  = regexp.MustCompile(`^[a-zA-Z_:][a-zA-Z0-9_:]*$`)
	labelRE = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
)

// DefBuckets suit durations in seconds from 5 ms to 10 s.
var DefBuckets = []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}

type kind string

const (
	kCounter   kind = "counter"
	kGauge     kind = "gauge"
	kHistogram kind = "histogram"
)

type series struct {
	labels string // rendered {a="b",c="d"} or ""
	// counter / gauge
	val float64
	fn  func() float64 // gauge func, evaluated at scrape time
	// histogram
	counts []uint64
	sum    float64
	n      uint64
}

type family struct {
	name, help string
	kind       kind
	buckets    []float64
	series     map[string]*series
}

// Registry holds metric families. The zero value is not usable; use NewRegistry.
type Registry struct {
	mu   sync.Mutex
	fams map[string]*family
	cols []func(*Collect)
}

func NewRegistry() *Registry { return &Registry{fams: map[string]*family{}} }

// Counter is a monotonically increasing value.
type Counter struct {
	r *Registry
	s *series
}

func (c *Counter) Inc() { c.Add(1) }

// Add ignores negative values (a counter never goes down).
func (c *Counter) Add(v float64) {
	if v < 0 || math.IsNaN(v) {
		return
	}
	c.r.mu.Lock()
	c.s.val += v
	c.r.mu.Unlock()
}

// Gauge is a value that goes up and down.
type Gauge struct {
	r *Registry
	s *series
}

func (g *Gauge) Set(v float64) { g.r.mu.Lock(); g.s.val = v; g.r.mu.Unlock() }
func (g *Gauge) Add(v float64) { g.r.mu.Lock(); g.s.val += v; g.r.mu.Unlock() }

// Histogram counts observations into cumulative buckets.
type Histogram struct {
	r *Registry
	f *family
	s *series
}

func (h *Histogram) Observe(v float64) {
	if math.IsNaN(v) {
		return
	}
	h.r.mu.Lock()
	defer h.r.mu.Unlock()
	h.s.sum += v
	h.s.n++
	if i := sort.SearchFloat64s(h.f.buckets, v); i < len(h.f.buckets) {
		h.s.counts[i]++
	}
}

// get returns the series for name+labels, creating family and series. Misuse is a programming error: panic.
func (r *Registry) get(name, help string, k kind, buckets []float64, labels []string) (*family, *series) {
	if !nameRE.MatchString(name) {
		panic("metrics: bad metric name " + name)
	}
	if len(labels)%2 != 0 {
		panic("metrics: odd label list for " + name)
	}
	lbl := renderLabels(labels)
	r.mu.Lock()
	defer r.mu.Unlock()
	f := r.fams[name]
	if f == nil {
		f = &family{name: name, help: help, kind: k, buckets: buckets, series: map[string]*series{}}
		r.fams[name] = f
	} else if f.kind != k {
		panic("metrics: " + name + " already registered as " + string(f.kind))
	}
	s := f.series[lbl]
	if s == nil {
		s = &series{labels: lbl}
		if k == kHistogram {
			s.counts = make([]uint64, len(f.buckets))
		}
		f.series[lbl] = s
	}
	return f, s
}

// Counter returns the counter name{labels}, created on first use. labels is a flat key, value list.
func (r *Registry) Counter(name, help string, labels ...string) *Counter {
	_, s := r.get(name, help, kCounter, nil, labels)
	return &Counter{r, s}
}

func (r *Registry) Gauge(name, help string, labels ...string) *Gauge {
	_, s := r.get(name, help, kGauge, nil, labels)
	return &Gauge{r, s}
}

// GaugeFunc registers a gauge read at scrape time. fn must not call back into the registry.
func (r *Registry) GaugeFunc(name, help string, fn func() float64, labels ...string) {
	_, s := r.get(name, help, kGauge, nil, labels)
	r.mu.Lock()
	s.fn = fn
	r.mu.Unlock()
}

// Histogram uses DefBuckets when buckets is nil; buckets must be sorted ascending (+Inf is implicit).
func (r *Registry) Histogram(name, help string, buckets []float64, labels ...string) *Histogram {
	if buckets == nil {
		buckets = DefBuckets
	}
	if !sort.Float64sAreSorted(buckets) {
		panic("metrics: unsorted buckets for " + name)
	}
	f, s := r.get(name, help, kHistogram, append([]float64(nil), buckets...), labels)
	return &Histogram{r, f, s}
}

// Collect lets a collector emit gauges computed at scrape time (for example one per live process).
type Collect struct {
	out []collected
}

type collected struct {
	name, help, labels string
	v                  float64
}

// Gauge emits one sample; the same name with different labels forms one family.
func (c *Collect) Gauge(name, help string, v float64, labels ...string) {
	c.out = append(c.out, collected{name, help, renderLabels(labels), v})
}

// AddCollector registers fn, called on every scrape. It must be quick and must not call back into r.
func (r *Registry) AddCollector(fn func(*Collect)) {
	r.mu.Lock()
	r.cols = append(r.cols, fn)
	r.mu.Unlock()
}

func renderLabels(kv []string) string {
	if len(kv) == 0 {
		return ""
	}
	type p struct{ k, v string }
	ps := make([]p, 0, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		if !labelRE.MatchString(kv[i]) {
			panic("metrics: bad label name " + kv[i])
		}
		ps = append(ps, p{kv[i], kv[i+1]})
	}
	sort.Slice(ps, func(i, j int) bool { return ps[i].k < ps[j].k })
	var b strings.Builder
	b.WriteByte('{')
	for i, x := range ps {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(x.k + `="` + escLabel(x.v) + `"`)
	}
	b.WriteByte('}')
	return b.String()
}

var labelEsc = strings.NewReplacer(`\`, `\\`, "\n", `\n`, `"`, `\"`)

func escLabel(s string) string { return labelEsc.Replace(s) }

func fmtFloat(v float64) string {
	switch {
	case math.IsInf(v, 1):
		return "+Inf"
	case math.IsInf(v, -1):
		return "-Inf"
	case math.IsNaN(v):
		return "NaN"
	}
	return strconv.FormatFloat(v, 'g', -1, 64)
}

// withLE inserts le="x" into a rendered label set.
func withLE(labels, le string) string {
	if labels == "" {
		return `{le="` + le + `"}`
	}
	return labels[:len(labels)-1] + `,le="` + le + `"}`
}

// WriteTo writes the exposition. Families and series are sorted, so output is stable.
func (r *Registry) WriteTo(w io.Writer) (int64, error) {
	r.mu.Lock()
	cols := append([]func(*Collect){}, r.cols...)
	r.mu.Unlock()
	var c Collect
	for _, fn := range cols {
		fn(&c)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	type fam struct {
		name, help string
		kind       kind
		lines      []string
	}
	all := map[string]*fam{}
	for name, f := range r.fams {
		out := &fam{name: name, help: f.help, kind: f.kind}
		keys := make([]string, 0, len(f.series))
		for k := range f.series {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			s := f.series[k]
			switch f.kind {
			case kHistogram:
				var cum uint64
				for i, ub := range f.buckets {
					cum += s.counts[i]
					out.lines = append(out.lines, fmt.Sprintf("%s_bucket%s %d", name, withLE(s.labels, fmtFloat(ub)), cum))
				}
				out.lines = append(out.lines,
					fmt.Sprintf("%s_bucket%s %d", name, withLE(s.labels, "+Inf"), s.n),
					fmt.Sprintf("%s_sum%s %s", name, s.labels, fmtFloat(s.sum)),
					fmt.Sprintf("%s_count%s %d", name, s.labels, s.n))
			default:
				v := s.val
				if s.fn != nil {
					v = s.fn()
				}
				out.lines = append(out.lines, fmt.Sprintf("%s%s %s", name, s.labels, fmtFloat(v)))
			}
		}
		all[name] = out
	}
	for _, x := range c.out {
		f := all[x.name]
		if f == nil {
			f = &fam{name: x.name, help: x.help, kind: kGauge}
			all[x.name] = f
		}
		f.lines = append(f.lines, fmt.Sprintf("%s%s %s", x.name, x.labels, fmtFloat(x.v)))
	}
	names := make([]string, 0, len(all))
	for n := range all {
		names = append(names, n)
	}
	sort.Strings(names)
	var total int64
	for _, n := range names {
		f := all[n]
		var b strings.Builder
		if f.help != "" {
			b.WriteString("# HELP " + n + " " + strings.NewReplacer(`\`, `\\`, "\n", `\n`).Replace(f.help) + "\n")
		}
		b.WriteString("# TYPE " + n + " " + string(f.kind) + "\n")
		for _, l := range f.lines {
			b.WriteString(l + "\n")
		}
		k, err := io.WriteString(w, b.String())
		total += int64(k)
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

// Handler serves the exposition on any path; GET and HEAD only.
func (r *Registry) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet && req.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		if req.Method == http.MethodGet {
			r.WriteTo(w)
		}
	})
}
