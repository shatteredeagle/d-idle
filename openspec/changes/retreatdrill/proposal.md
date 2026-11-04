## Why

P29 retreat (chicken, leave game, escape) was never executed live; only the decision was recorded (DRILLS5, DRILLS12).

## What Changes

Design and offline-test a live d2d drill that makes the bot execute the retreat action (pass condition, setup verbs, writes only before the start), runnable later when a hidden d2d slot is free.

## Impact

Offline only (fakes and docs), nothing verified live.

Overlap (declared)

The drill package and cmd/d2drill belong to the drills owner (m1-liveproof). This card adds three new files there (internal/drill/retreat.go, internal/drill/retreat_test.go, cmd/d2drill/retreat_lab.go) and one case line in cmd/d2drill/main.go; no existing drill is changed.
