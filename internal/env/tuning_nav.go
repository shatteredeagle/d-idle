package env

import "time"

// NavTuning holds the nav knobs. Owner: the nav agent; add fields, defaults and Knob entries here only.
// Defaults are the best profile; every range below is where the character still reaches its goals (tests in
// internal/nav prove paths get longer, never unreachable, when PathQuality drops).
type NavTuning struct {
	// PathQuality 1 = exact weighted A* with string-pulling and straight waypoints. Lower values inflate the
	// A* heuristic, skip smoothing below 0.5 and displace waypoints sideways (Sloppiness): longer, wobblier paths.
	PathQuality float64
	// Sloppiness is the largest sideways displacement of a waypoint (subtiles) at PathQuality 0.
	Sloppiness float64
	// WaypointSpacing is the distance between walking clicks (subtiles). Kolbot walks at 5, KO at 15..30.
	WaypointSpacing   float64
	ArriveTolerance   float64 // GoTo is done within this many subtiles
	PollMs            float64 // observation tick while walking; also the reaction time
	ReplanMs          float64 // earliest a changed grid or monster layout triggers a new plan
	StuckMs           float64 // no progress for this long counts as stuck
	MaxRecoveries     int     // rungs of the stuck ladder to climb (re-click, side step, detour, door, ...) before giving up
	WallClearance     float64 // cells around walls that cost extra
	WallWeight        float64 // extra cost per cell of missing clearance
	MonsterRadius     float64 // subtiles around a monster that cost extra
	MonsterPenalty    float64 // extra cost at the monster itself, fading to 0 at MonsterRadius
	ExploreInformed   bool    // true: prefer big, forward frontiers; false: nearest frontier
	ExploreBucket     float64 // visited-set resolution and the radius that counts a frontier as inspected (subtiles)
	WalkTimeoutSec    float64 // hard cap for one GoTo
	ExploreTimeoutSec float64 // hard cap for one Explore
	ExitPush          float64 // how far past a border gap or exit tile to walk (subtiles)
	LoadWaitSec       float64 // how long to wait through a loading screen
	PanelWaitMs       float64 // how long to wait for a clicked panel (waypoint) to open
	UseWaypoints      bool    // travel, the corpse run and town returns take activated waypoints (docs/waypoints.md)
	WpMinDistance     float64 // a waypoint hop must skip at least this many levels, otherwise the bot walks
	WpCooldownSec     float64 // game.exe refuses a waypoint travel this long after a level change (D2D-V: 10 s)
	CautionRadius     float64 // CorpseApproach stops when a living monster is this close (subtiles); its paths avoid monsters at twice the usual radius
}

func DefaultNavTuning() NavTuning {
	return NavTuning{
		PathQuality: 1, Sloppiness: 8, WaypointSpacing: 8, ArriveTolerance: 2, PollMs: 100, ReplanMs: 1500,
		StuckMs: 1200, MaxRecoveries: 6, WallClearance: 3, WallWeight: 0.75, MonsterRadius: 8, MonsterPenalty: 6,
		ExploreInformed: true, ExploreBucket: 10, WalkTimeoutSec: 120, ExploreTimeoutSec: 300, ExitPush: 8,
		LoadWaitSec: 20, PanelWaitMs: 2500, CautionRadius: 20,
		UseWaypoints: true, WpMinDistance: 2, WpCooldownSec: 10,
	}
}

func (t NavTuning) Poll() time.Duration { return time.Duration(t.PollMs * float64(time.Millisecond)) }
func (t NavTuning) Replan() time.Duration {
	return time.Duration(t.ReplanMs * float64(time.Millisecond))
}
func (t NavTuning) Stuck() time.Duration { return time.Duration(t.StuckMs * float64(time.Millisecond)) }
func (t NavTuning) WalkTimeout() time.Duration {
	return time.Duration(t.WalkTimeoutSec * float64(time.Second))
}
func (t NavTuning) ExploreLimit() time.Duration {
	return time.Duration(t.ExploreTimeoutSec * float64(time.Second))
}
func (t NavTuning) LoadWait() time.Duration {
	return time.Duration(t.LoadWaitSec * float64(time.Second))
}
func (t NavTuning) PanelWait() time.Duration {
	return time.Duration(t.PanelWaitMs * float64(time.Millisecond))
}

func b2f(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func navKnobs() []Knob {
	d := DefaultNavTuning()
	return []Knob{
		{Key: "nav.pathQuality", Label: "Path quality", Unit: "0..1", Default: d.PathQuality, Min: 0.1, Max: 1, HigherIsBetter: true,
			Desc: "1 = shortest, smoothed paths. Lower values make paths longer and wobblier (inflated A* heuristic, no smoothing below 0.5, sideways waypoint jitter) but still reach the goal."},
		{Key: "nav.sloppiness", Label: "Waypoint sloppiness", Unit: "subtiles", Default: d.Sloppiness, Min: 0, Max: 14, HigherIsBetter: false,
			Desc: "Largest sideways displacement of a waypoint at path quality 0; scaled by (1 - quality)."},
		{Key: "nav.waypointSpacing", Label: "Walking click spacing", Unit: "subtiles", Default: d.WaypointSpacing, Min: 4, Max: 24, HigherIsBetter: true,
			Desc: "Distance between walking clicks. Smaller spacing follows the path more exactly but clicks more often."},
		{Key: "nav.arriveTolerance", Label: "Arrival tolerance", Unit: "subtiles", Default: d.ArriveTolerance, Min: 1.5, Max: 6, HigherIsBetter: false,
			Desc: "A walk ends within this distance of the goal."},
		{Key: "nav.pollMs", Label: "Walk reaction time", Unit: "ms", Default: d.PollMs, Min: 50, Max: 400, HigherIsBetter: false,
			Desc: "Observation period while walking; larger values react later to stops and monsters."},
		{Key: "nav.replanMs", Label: "Replan interval", Unit: "ms", Default: d.ReplanMs, Min: 500, Max: 8000, HigherIsBetter: false,
			Desc: "Earliest time a changed map or monster layout triggers a new plan."},
		{Key: "nav.stuckMs", Label: "Stuck timeout", Unit: "ms", Default: d.StuckMs, Min: 600, Max: 4000, HigherIsBetter: false,
			Desc: "No progress for this long starts the recovery ladder."},
		{Key: "nav.maxRecoveries", Label: "Recovery attempts", Unit: "count", Default: float64(d.MaxRecoveries), Min: 2, Max: 12, HigherIsBetter: true,
			Desc: "Stuck-ladder rungs tried before the walk fails with ErrStuck."},
		{Key: "nav.wallClearance", Label: "Wall clearance", Unit: "cells", Default: d.WallClearance, Min: 0, Max: 6, HigherIsBetter: true,
			Desc: "Cells next to walls that cost extra, keeping paths off corners."},
		{Key: "nav.wallWeight", Label: "Wall penalty", Unit: "cost/cell", Default: d.WallWeight, Min: 0, Max: 3, HigherIsBetter: true,
			Desc: "Extra path cost per cell of missing wall clearance."},
		{Key: "nav.monsterRadius", Label: "Monster avoid radius", Unit: "subtiles", Default: d.MonsterRadius, Min: 0, Max: 16, HigherIsBetter: true,
			Desc: "Paths keep this distance from monsters and NPCs when a detour is cheap."},
		{Key: "nav.monsterPenalty", Label: "Monster penalty", Unit: "cost", Default: d.MonsterPenalty, Min: 0, Max: 20, HigherIsBetter: true,
			Desc: "Extra cost of a cell at a monster, fading to zero at the avoid radius."},
		{Key: "nav.exploreInformed", Label: "Informed exploration", Unit: "bool", Default: b2f(d.ExploreInformed), Min: 0, Max: 1, HigherIsBetter: true,
			Desc: "1 prefers large, forward frontiers; 0 always walks to the nearest frontier."},
		{Key: "nav.exploreBucket", Label: "Explore resolution", Unit: "subtiles", Default: d.ExploreBucket, Min: 5, Max: 30, HigherIsBetter: false,
			Desc: "A frontier within this distance of a place already visited counts as inspected."},
		{Key: "nav.walkTimeoutSec", Label: "Walk timeout", Unit: "s", Default: d.WalkTimeoutSec, Min: 30, Max: 600, HigherIsBetter: true,
			Desc: "Hard cap for one GoTo."},
		{Key: "nav.exploreTimeoutSec", Label: "Explore timeout", Unit: "s", Default: d.ExploreTimeoutSec, Min: 30, Max: 1800, HigherIsBetter: true,
			Desc: "Hard cap for one Explore."},
		{Key: "nav.exitPush", Label: "Exit overshoot", Unit: "subtiles", Default: d.ExitPush, Min: 3, Max: 20, HigherIsBetter: true,
			Desc: "How far past a border gap or exit tile the walk continues to trigger the level change."},
		{Key: "nav.loadWaitSec", Label: "Loading wait", Unit: "s", Default: d.LoadWaitSec, Min: 5, Max: 90, HigherIsBetter: true,
			Desc: "How long to wait through a loading screen."},
		{Key: "nav.panelWaitMs", Label: "Panel wait", Unit: "ms", Default: d.PanelWaitMs, Min: 800, Max: 8000, HigherIsBetter: true,
			Desc: "How long to wait for a clicked panel (waypoint) to open."},
		{Key: "nav.useWaypoints", Label: "Use waypoints", Unit: "bool", Default: b2f(d.UseWaypoints), Min: 0, Max: 1, HigherIsBetter: true, NoHandicap: true,
			Desc: "1: travel, the corpse run and the way back from town take an activated waypoint instead of walking (saves minutes and walking deaths; the route guard still applies to every area on the way). 0 walks everything."},
		{Key: "nav.wpMinDistance", Label: "Waypoint minimum hops", Unit: "levels", Default: d.WpMinDistance, Min: 2, Max: 8, HigherIsBetter: false, NoHandicap: true,
			Desc: "A waypoint trip is taken only when it skips at least this many level hops; shorter routes walk. 2 is the shortest trip that saves a level."},
		{Key: "nav.wpCooldownSec", Label: "Waypoint cooldown", Unit: "s", Default: d.WpCooldownSec, Min: 10, Max: 20, HigherIsBetter: false, NoHandicap: true,
			Desc: "game.exe silently refuses a waypoint travel for 10 s after any level change; the bot waits this long since arriving before clicking a row. Lower is not useful."},
		{Key: "nav.cautionRadius", Label: "Corpse approach caution", Unit: "subtiles", Default: d.CautionRadius, Min: 8, Max: 40, HigherIsBetter: true,
			Desc: "A corpse run stops and reports when a living monster comes this close; its paths avoid monsters at twice the usual radius."},
	}
}
