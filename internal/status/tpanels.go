package status

import (
	"time"

	"d2lod-bot/internal/telemetry"
)

// OverviewData is the contract of the "overview" panel: telemetry.Overview plus area name.
type OverviewData struct {
	telemetry.Overview
	AreaName string `json:"areaName"`
}

func NewOverviewPanel(tr *telemetry.Tracker, areaName func(int) string) Panel {
	if areaName == nil {
		areaName = AreaName
	}
	return Panel{ID: "overview", Title: "Overview", Order: 10, Snapshot: func() any {
		o := tr.Overview()
		return OverviewData{o, areaName(o.Area)}
	}}
}

type TimelineData struct {
	Events   []telemetry.Event   `json:"events"`
	Segments []telemetry.Segment `json:"segments"`
	Now      int64               `json:"now"`
}

// NewTimelinePanel streams events with Append as they happen; the snapshot holds the recent tail.
func NewTimelinePanel(h *Hub, tr *telemetry.Tracker) Panel {
	tr.OnEvent(func(e telemetry.Event) { h.Append("timeline", e) })
	return Panel{ID: "timeline", Title: "Timeline", Order: 50, Every: 5 * time.Second, Snapshot: func() any {
		ev := tr.Events(300)
		if ev == nil {
			ev = []telemetry.Event{}
		}
		sg := tr.Segments(120)
		if sg == nil {
			sg = []telemetry.Segment{}
		}
		return TimelineData{Events: ev, Segments: sg, Now: tr.Now().UnixMilli()}
	}}
}

type AreaRow struct {
	telemetry.AreaStat
	Name string `json:"name"`
}

type MetricsData struct {
	Series []telemetry.Point `json:"series"`
	Areas  []AreaRow         `json:"areas"`
	Drops  []telemetry.Event `json:"drops"`
}

func NewMetricsPanel(tr *telemetry.Tracker, areaName func(int) string) Panel {
	if areaName == nil {
		areaName = AreaName
	}
	return Panel{ID: "metrics", Title: "Metrics", Order: 60, Every: 5 * time.Second, Snapshot: func() any {
		s := tr.Series(5)
		if s == nil {
			s = []telemetry.Point{}
		}
		a := []AreaRow{}
		for _, st := range tr.Areas() {
			a = append(a, AreaRow{st, areaName(st.Area)})
		}
		d := tr.Drops(50)
		if d == nil {
			d = []telemetry.Event{}
		}
		return MetricsData{s, a, d}
	}}
}

// RegisterTelemetry registers overview, timeline and metrics panels fed by tr.
func (h *Hub) RegisterTelemetry(tr *telemetry.Tracker, areaName func(int) string) {
	h.Register(NewOverviewPanel(tr, areaName))
	h.Register(NewTimelinePanel(h, tr))
	h.Register(NewMetricsPanel(tr, areaName))
}
