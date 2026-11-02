# lockstepp4: live re-measure of the lockstep hook with a scaled bot clock

## Why

The P3 measurement (docs/research/lockstep-p3.md) showed the clock hook is exact for the game but the bot, whose waits are real time, loses chores and kills at k=2 and fails at k=4. The branch agent/botclock scales the bot's game-clock waits; whether that recovers the rates was untested live.

## What Changes

Docs only: docs/research/lockstep-p4.md records the live k=2 and k=4 runs (10 min Blood Moor, 5 min town frames at k=4) on hidden instance i4 with the throwaway Booti, under the user's go of 2026-11-02, with the verdict and next steps. No code, no spec delta.
