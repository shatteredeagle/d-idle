## Why

Fixed clvl floors (merc-policy `Route guard`, `Andariel floor`) are too prescriptive: the 9 Monastery road deaths came at clvl 11-12 with a 2-potion kit against melee packs, so the cause was crowd, kit and life, not clvl. The user asked for readiness judged as character power against zone danger ([ADR 0011](../../../docs/adr/0011-readiness-power-vs-danger.md), accepted 2026-10-31). No power or danger rating exists in the bot today.

## What Changes

- Power = survival (max life, resists, potions in the belt, town portal, merc alive) + offense (estimated kill speed from our skill damage against the area's real monster HP after resists and immunities) + clvl as a minor input.
- Danger = area monster level (the general guide) + real monster HP and damage + crowd (pack size, density) + threats the build handles badly (immunities, poison, ranged or lightning casters, uniques, bosses) + our own deaths and chicken exits per hour in that area, weighted more as data comes in.
- Ready = power >= danger x a safety margin; weights and margin are set by this card (knobs).
- The model takes over only after offline calibration: replayed on our logs it rates the 9 Monastery road deaths (clvl 11-12, 2-potion kit) as not ready and agrees with the m3-andariel result. After that the floors are a safety net and the model may go at most 2 clvl below them.
- Delta: merc-policy ADDED `Readiness model`.

## Dependencies

Deferred until m3-andariel has won once (ADR 0011 gate 4). m3-routeguard does not wait for this card. Parked from grill 0003 and open here: the model's weights and safety margin; the max-life floor for Andariel (from m3-andariel's life-per-phase numbers).
