# FCR / FHR / FBR / IAS breakpoints per class: cross-check against LoD sources

> Gap 3 of the 2026-11-27 SME reviews. Extends `mechanics-breakpoints.md` (computed frame tables, formulas, WSM). 25 frames per second. A breakpoint value is the minimum **item** percent (cumulative, before the 120/(X+120) diminishing step is applied, i.e. the "total" you read on the character screen) at which the animation drops one frame.
> Tags: V fetched, S secondary, U unverified. Source for every row below: **V** Amazon Basin Wiki class pages, which state LoD-vs-RotW where they differ (https://www.theamazonbasin.com/wiki/index.php/Sorceress, `.../Necromancer`, `.../Amazon`, `.../Assassin`, `.../Druid_(Diablo_II)`, `.../Paladin_(Diablo_II)`, `.../Barbarian_(Diablo_II)`), fetched 2026-11-27. Cross-checked against `mechanics-breakpoints.md` (computed from Maxroll/Wowhead D2R formulas): **every breakpoint list below agrees** with its tables. Frame counts at 0% come from that doc (the Basin header row is shared across rows in the fetched text, so row-to-frame alignment is not recoverable from it alone).

## FCR (frames, 0% start)
| Class / form | Breakpoints | Frames (0% start) | Tag |
|---|---|---|---|
| Sorceress, all spells but Lightning/Chain Lightning | 0 9 20 37 63 105 200 | 13 down to 7 | V |
| Sorceress Lightning / Chain Lightning (Inferno cannot be changed) | 0 7 15 23 35 52 78 117 194 | 19 down to 11 | V (Basin also lists 20, 63, 200: action-frame-only steps, S) |
| Necromancer | 0 9 18 30 48 75 125 | 15 down to 9 | V; see 1 |
| Necromancer, Trang-Oul vampire | 0 6 11 18 24 35 48 65 86 120 180 | 23 down to 13 | V |
| Druid human | 0 4 10 19 30 46 68 99 163 | 18 down to 10 | V (Basin also lists 11, 63, 142: action-frame-only, S) |
| Werewolf | 0 6 14 26 40 60 95 157 | 16 down to 9 | V |
| Werebear | 0 7 15 26 40 63 99 163 | 16 down to 9 | V |
| Paladin (not Fist of the Heavens) | 0 9 18 30 48 75 125 | 15 down to 9 | V |
| Paladin Fist of the Heavens | 0 15 39 86 | 34 down to 31 | V (cast delay dominates) |
| Amazon bow | 0 7 14 22 32 48 68 99 152 | 19 down to 11 | V |
| Amazon 1HS/2HS/crossbow | 0 7 11 14 22 26 32 48 68 86 99 152 180 | 19..11 w/ repeats | V |
| Amazon 1HT/2HT/staff | 0 7 10 14 22 23 32 42 48 68 72 99 133 152 | 19..11 w/ repeats | V |
| Amazon unarmed | 0 7 13 14 22 29 32 48 56 68 99 113 152 | 19..11 w/ repeats | V |
| Assassin | 0 8 16 27 42 65 102 174 | 16 down to 9 | V (Basin also lists 15, 39, 86: action-frame, S) |
| Barbarian | 0 9 20 37 63 105 200 | 13 down to 7 | V (also lists 15, 39, 86) |

Practical targets (S, community, same as `mechanics-breakpoints.md`): Sorceress 105 (8f) and 200 (7f); Paladin/Necromancer 75 (10f) or 125 (9f); Druid 99 (11f) or 163 (10f); Assassin 65/102/174.
1. **Disagreement:** Wowhead prints 16f at 0% for the Necromancer but its breakpoints match a 15f start; Basin's breakpoints equal ours (0 9 18 30 48 75 125). The 0% frame count is U for Necromancer; the breakpoints are V.

## FHR (breakpoints; frame labels from `mechanics-breakpoints.md`)
| Class / form | Breakpoints | V/S |
|---|---|---|
| Sorceress | 5 9 14 20 30 42 60 86 142 280 | V |
| Necromancer / Druid non-1HS | 5 10 16 26 39 56 86 152 377 | V |
| Necromancer vampire | 2 6 10 16 24 34 48 72 117 208 638 | V |
| Druid 1HS (sword, mace, axe, wand) | 3 7 13 19 29 42 63 99 174 456 | V |
| Werewolf | 9 20 42 86 280 | V |
| Werebear | 5 10 16 24 37 54 86 152 360 | V |
| Amazon | 6 13 20 32 52 86 174 600 | V |
| Assassin / Barbarian / Paladin (other weapons) | 7 15 27 48 86 200 4680 | V |
| Paladin spear / staff / 2H axe-hammer-polearm | 3 7 13 20 32 48 75 129 280 4680 | V |

Common targets (S): 48 or 86 for Barbarian/Assassin/Paladin, 86 for Sorceress, 52 or 86 Amazon; Basin states values above 594 are unreachable in LoD (V).

## FBR (shield-only, plus Guardian Angel/Pattern)
| Class | Breakpoints | V |
|---|---|---|
| Sorceress / Werewolf-form | 7 15 27 48 86 200 | V |
| Necromancer / Druid | 6 13 20 32 52 86 174 | V |
| Barbarian | 9 20 42 86 280 | V |
| Assassin / Paladin (no Holy Shield) / Amazon 1HT | 13 32 86 600 | V |
| Paladin with Holy Shield | 86 (1 frame) | V |
| Amazon 1HS | 4 6 11 15 23 29 40 56 80 120 200 480 | V |
| Werebear | 5 10 16 27 40 65 109 223 | V |
Basin says more than 140 FBR (155 for Paladin) is out of reach (V).

## IAS / Zeal
Not tabulated per class: IAS depends on weapon WSM and skill (V formula in `mechanics-breakpoints.md` section 5, from Maxroll). Basin's "Increased Attack Speed" page lists only item sources, not breakpoints (checked). Zeal is a rollback skill with no closed form; use a calculator (Maxroll attack-speed page, V in `mechanics-breakpoints.md`). Gap: no fetched 1.14d per-class IAS breakpoint table; do not guess numbers in code, compute from the formula.

## Not sourced
- D2R vs LoD: Basin states no LoD/RotW difference for FCR/FHR/FBR except the unreachable-cap line (594 vs 603 FHR). I treat the tables as equal for 1.14d (S).
- Per-class IAS breakpoint lists (above); Teleport/FRW values (not fetched).
