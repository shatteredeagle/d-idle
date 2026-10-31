## Why

User decision 2026-10-30 (dialog): the 20 percent relative rule on deaths cannot be resolved at 0 to 0.2 deaths/h, and a replicated soak sees only a handful of deaths. Tighten deaths to an exposure-based Poisson judgment with an absolute cap, and add low-variance proxies (potions, life, damage) that move before deaths do.

## What Changes

- delta spec for fidelity-gate: deaths bound, 25 percent proxies, deaths per 1000 kills in bound form
- d2dsoak `compare -gate v2`; v1 stays the default and is unchanged
- docs/d2d-soak.md section 3b
