## Why

The RT retreat drill group (card retreatdrill) was designed and tested offline only. This card runs RT.drink and RT.portal live on the d2d lab fork and compares the numbers with the first-guess bounds.

## What Changes

Evidence only: docs/probes/retreat/2026-11-04-RT-* and docs/retreatlive.md. No drill or bot code is changed. RT.leave is skipped (Fighter.Leave is nil in the rig).

## Impact

None on code. One hidden d2d slot (slot 3) for under 5 minutes, throwaway character, setup cheats only before the Act.
