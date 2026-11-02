# skias: swing speed, IAS and WSM trace

## Why

The swing frame count for a weapon speed modifier, increased attack speed and a skill bonus was never traced: sk-melee-trace.md lists swing speed as not covered. d2d computes it with attack_speed and attack_ticks, and nobody has compared that with game.exe.

## What Changes

Docs only: docs/research/sk-ias-trace.md (d2d formula read, breakpoints tabulated by tools/emu/skias_bp.py, deviations D1-D4 tagged U, bot finding B1, drill recipe). The game.exe attack branch is still unread; the differential is specified, not run. No Go changes, no live game.
