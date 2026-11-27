package main

import (
	"d2lod-bot/internal/supply"
	"d2lod-bot/internal/watch"
)

// bindTownPlan mounts GET /api/townplan (watch.TownPlanAPI). The route is always mounted and answers 503 until a source
// is bound; src is nil when the backend has no town (the route then stays 503).
func bindTownPlan(f *watch.Fun, src func() supply.View) {
	if f == nil {
		return
	}
	api := watch.NewTownPlanAPI()
	if src != nil {
		api.Bind(src)
	}
	f.TownPlan = api
}

// townPlanSource reads the town of the runner reference late: the Town exists once wireRepair ran, which is after
// the ladder options are built. Before that (or without a town) the viewer shows NoView.
func townPlanSource(ref *runnerRef) func() supply.View {
	if ref == nil {
		return nil
	}
	return func() supply.View {
		if tw := ref.tw.Load(); tw != nil {
			return tw.PlanView()
		}
		return supply.NoView()
	}
}
