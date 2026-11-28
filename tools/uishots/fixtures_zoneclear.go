package main

import (
	"net/http"

	"d2lod-bot/internal/env"
	"d2lod-bot/internal/watch"
)

// zoneClearFixture serves /api/zoneclear from an in-memory set of knob values (the real handler, so the fit pages and the
// pictures show exactly what the panel draws) and takes the panel's Save and Reset.
func zoneClearFixture() http.Handler {
	vals, src := map[string]float64{}, map[string]string{}
	def := map[string]float64{}
	for _, k := range env.Knobs() {
		vals[k.Key], def[k.Key] = k.Default, k.Default
	}
	vals["bot.farmResetPct"], src["bot.farmResetPct"] = 90, "dashboard"
	z := &watch.ZoneClear{
		Value: func(k string) (float64, string, error) {
			s := src[k]
			if s == "" {
				s = "default"
			}
			return vals[k], s, nil
		},
		Set: func(over map[string]*float64) error {
			for k, p := range over {
				if p == nil {
					vals[k] = def[k]
					delete(src, k)
					continue
				}
				vals[k], src[k] = *p, "dashboard"
			}
			return nil
		},
	}
	m := http.NewServeMux()
	m.Handle(watch.ZoneClearRoute, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			z.PostHandler(watch.AnswerOptions{}).ServeHTTP(w, r)
			return
		}
		z.Handler().ServeHTTP(w, r)
	}))
	return m
}
