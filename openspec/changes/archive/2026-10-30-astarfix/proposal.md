## Why

KILLFIX saw a recovered panic `index out of range` in nav/astar.go pop (1 of 24 hidden d2d runs): the guard's Fighter.Safety (PathTo) and the acting goroutine planned on one Navigator/Planner. Same race exists on the Game.exe path.

## What Changes

- Planner.Plan and Navigator.planHard are serialized by a mutex (no per-call scratch: the Field is shared too); race test.
- d2denv Proc.Start waits for an already launched d2d instead of failing with "d2d already running" (restart noise).
