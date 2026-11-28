package watch

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"d2lod-bot/internal/env"
)

// fakeZone is the tuner's part: values by key, an edit that checks nothing (the handler does) and sources.
type fakeZone struct {
	v   map[string]float64
	src map[string]string
}

func newFakeZone() *fakeZone {
	f := &fakeZone{v: map[string]float64{}, src: map[string]string{}}
	for _, k := range env.Knobs() {
		f.v[k.Key] = k.Default
	}
	return f
}

func (f *fakeZone) zc() *ZoneClear {
	return &ZoneClear{
		Value: func(k string) (float64, string, error) {
			s := f.src[k]
			if s == "" {
				s = "default"
			}
			return f.v[k], s, nil
		},
		Set: func(over map[string]*float64) error {
			for k, p := range over {
				if p == nil {
					for _, d := range env.Knobs() {
						if d.Key == k {
							f.v[k] = d.Default
						}
					}
					delete(f.src, k)
					continue
				}
				f.v[k], f.src[k] = *p, "dashboard"
			}
			return nil
		},
	}
}

func TestZoneClearViewAndPost(t *testing.T) {
	f := newFakeZone()
	z := f.zc()
	v, err := z.View()
	if err != nil || v.Schema != "watch.zoneclear/1" || len(v.Knobs) != 3 {
		t.Fatalf("view %+v %v", v, err)
	}
	for _, k := range v.Knobs {
		if k.Source != "default" || k.Max <= k.Min || k.Label == "" {
			t.Errorf("knob %+v", k)
		}
	}
	o := AnswerOptions{Token: "tok"}
	post := z.PostHandler(o)
	auth := map[string]string{"Authorization": "Bearer tok"}
	for name, c := range map[string]struct {
		body, field string
		code        int
	}{
		"explored too low":  {`{"exploredPct":10}`, "exploredPct", 400},
		"reset too high":    {`{"farmResetPct":101}`, "farmResetPct", 400},
		"cooldown zero":     {`{"farmResetCooldownMin":0}`, "farmResetCooldownMin", 400},
		"empty":             {`{}`, "", 400},
		"unknown field":     {`{"x":1}`, "", 400},
		"reset with others": {`{"reset":true,"exploredPct":80}`, "reset", 400},
		"retired knob":      {`{"farmReset":true}`, "", 400},
	} {
		w := invPost(post, ZoneClearRoute, c.body, auth)
		if w.Code != c.code || (c.field != "" && !strings.Contains(w.Body.String(), `"field":"`+c.field+`"`)) {
			t.Errorf("%s: %d %s", name, w.Code, w.Body)
		}
	}
	if len(f.src) != 0 {
		t.Fatalf("a refused edit changed %v", f.src)
	}
	w := invPost(post, ZoneClearRoute, `{"exploredPct":80,"farmResetPct":90}`, auth)
	if w.Code != 200 || f.v["bot.exploredPct"] != 80 || f.v["bot.farmResetPct"] != 90 {
		t.Fatalf("save: %d %s %v", w.Code, w.Body, f.v)
	}
	var rep struct{ View ZoneClearView }
	json.Unmarshal(w.Body.Bytes(), &rep)
	if rep.View.Knobs[0].Source != "dashboard" || rep.View.Knobs[0].Value != 80 {
		t.Errorf("reply %+v", rep.View.Knobs[0])
	}
	if w := invPost(post, ZoneClearRoute, `{"reset":true}`, auth); w.Code != 200 || f.v["bot.exploredPct"] != 100 || len(f.src) != 0 {
		t.Errorf("reset: %d %v %v", w.Code, f.v, f.src)
	}
	if w := invPost(post, ZoneClearRoute, `{"exploredPct":80}`, nil); w.Code != 401 {
		t.Errorf("no token: %d", w.Code)
	}
	g := httptest.NewRecorder()
	z.Handler().ServeHTTP(g, httptest.NewRequest("GET", ZoneClearRoute, nil))
	if g.Code != 200 || !strings.Contains(g.Body.String(), "bot.farmResetPct") {
		t.Errorf("get: %d", g.Code)
	}
	g = httptest.NewRecorder()
	z.Handler().ServeHTTP(g, httptest.NewRequest("POST", ZoneClearRoute, nil))
	if g.Code != 405 {
		t.Errorf("viewer POST: %d", g.Code)
	}
}
