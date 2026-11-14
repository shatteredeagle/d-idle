# E. Visual system

> status: design proposal (TASK-416, 2026-11-14). The identity stays: near-black warm stone, gold accent, serif display face, dark by default, dense by default (user decisions 2026-11-09, `docs/ui-design-system.md` section 13). Existing `--d-*` token names keep their names and meanings; everything here is an added token or a rule about using the ones that exist. Contrast figures were computed from the hex values in `web/tokens.css` (V by calculation, U in a browser).

## 1. What changes, in one paragraph

Fewer boxes and fewer weights. Today every tile is a bordered card with a serif title at the same size, every control is an outlined pill, and gold is used for titles, labels, links and values at once, so nothing leads. The refinement: the game picture and the status sentence lead; tiles lose their border and sit on one surface step; titles shrink to small caps labels; gold is kept for the brand, the current target and the primary button; state colour appears only on things that have a state.

## 2. Tokens

### Colour semantics (added aliases over existing values)

| Token | Value (dark) | Use | On `--d-surface-1` (#1c1813) |
|---|---|---|---|
| `--d-ok` | `var(--d-good)` #7cc47e | healthy, done | 8.5:1 |
| `--d-warn` (exists) | #e3a93a | look when you can; waiting; amber band | 8.4:1 |
| `--d-danger` | `var(--d-bad)` #ec7d6c | act now; failed; broken | 6.5:1 |
| `--d-info` (exists) | #7ab0e8 | information, links in text, focus family | 7.7:1 |
| `--d-muted` | `var(--d-text-3)` #8c7b67 | done rows, the plan row, hints | 4.3:1: large or secondary text only; body hints use `--d-text-2` (#b9aa94, 7.8:1) |
| `--d-accent` (exists) | #c7a44a | brand mark, primary button, current target ring, selected | 7.4:1 |
| `--d-state-bg-ok`, `-warn`, `-danger`, `-info` | the colour at 14% over the surface | chip and row tint | text stays `--d-text` |

`--d-good` and `--d-bad` keep working (they are the values); the new names are what new panels use, so the meaning is in the name.

Rule: a state is never colour alone. Every chip has a word, every light has a word under it, every alert row has an icon shape (triangle for amber, octagon for red, circle for blue).

### Type

The scale stays (`--d-fs-0` to `--d-fs-5`). Roles are new and are the only sizes a new panel uses:

| Role | Token | Face | Use |
|---|---|---|---|
| Sentence | `--d-fs-3`, weight 600 | body | the status sentence; one per page |
| State word | `--d-fs-1`, small caps, letter-spacing .06em | body | "FARMING" in the sentence, chips |
| Tile label | `--d-fs-0`, small caps, `--d-text-2` | body | tile titles ("ORDERS", "HEALTH"). The serif stays for the brand and page titles only |
| Body | `--d-fs-1` | body | rows, menus |
| Number | `--d-fs-1` or `--d-fs-3`, `tabular-nums` | `--d-font-num` | every number; fixed width so nothing shifts |
| Hint | `--d-fs-0`, `--d-text-2` | body | reasons under a row |

### Space, radius, line

Existing steps only: `--d-sp-1..6` (4, 8, 12, 16, 24, 32), `--d-r-1..3` (4, 8, 12). Tiles: no border, `--d-surface-1` on `--d-bg`, radius `--d-r-2`, padding `--d-pad`. A 1 px `--d-line` separates rows inside a tile. The only bordered things are inputs, buttons and the focus ring. Added: `--d-hit: 44px` (minimum touch target), `--d-row: 36px` (dense list row; 44 on coarse pointers).

## 3. Components

All built from the existing catalogue (`components.css`) where one exists; the new ones are marked.

| Component | Anatomy | States | Notes |
|---|---|---|---|
| Status sentence (new) | dot, state word, place, reason; chevron | `data-tone` ok, warn, danger, info; `data-state` loading, stale | one line, ellipsis on the reason; the full text in the popover; `aria-live="polite"` on this element and no other in the band |
| Status chip | word, optional dot | ok, warn, danger, info, muted | 22 px high, never clickable (a clickable thing is a button) |
| Meter | track, fill, one number | life, mana, xp, generic; `data-level` for thresholds | the existing meter, with the label removed in the band |
| Light (new) | 10 px disc, word, value word | ok, warn, danger, unknown (hollow) | five in `health`; each is a button opening its popover |
| Stat tile | label, number with unit, optional sparkline | - | used in `progress` only; at most four |
| Sparkline (new) | 30 points, 64 by 20 px, stroke `--d-text-2`, last point dot | flat line when unknown | no axis, no fill; the number next to it is the value |
| Progress | track, fill, text left, text right | determinate; "no limit" shows a moving dash, off under reduced motion | order progress, level meter |
| Alert row (new) | shape icon, one line, one button, dismiss | danger, warn, info | max four; the button is the answer |
| Order row (new) | index or state icon, text, state chip, reason line, progress, cancel | queued, waiting, active, done, failed, cancelled; `next` mark | 36 px collapsed; active row is two lines |
| Order chip | verb + object, compact | same states | used on the act map pins' tooltip and on a phone's collapsed list |
| Verb button | label, optional key hint | default, primary (one per row at most), disabled with reason | 32 px high dense, 44 on coarse pointers |
| Map node | rounded rect, name, band stripe at the left edge | here (blue ring), target (gold ring), queued (number pin), hard (amber triangle), failed (red mark), locked (dashed, muted), cleared (check) | the legend is a popover; the six permanent legend chips go |
| Map menu (new) | header (name, band, waypoint), verb rows, duration chips, route | - | a popover anchored to the node from 640 px; a bottom sheet below |
| Route | polyline along the edges from the hero to the target | active (gold), queued (muted dashes) | drawn under the nodes |
| Toast with Undo | text, Undo, 5 s bar | - | the existing toast stack, top end, max four |
| Popover (new) | anchored panel, 280 px, closes on Esc and outside click | - | the shell owns it (`ctx.popover`), like the sheet; a panel never positions it |

## 4. Accessibility

- Contrast: text at least 4.5:1, state marks and control outlines at least 3:1 (the values above meet it; `--d-muted` is not used for body text).
- Focus: the existing 2 px `--d-focus` ring, offset 2 px, on every control, including map nodes (they are buttons in the SVG, with the arrow-key walk of section C.7).
- Targets: 44 by 44 px on coarse pointers (`@media (pointer: coarse)`), 32 px high on fine pointers with at least 8 px between adjacent targets.
- The map is never the only way: "Go to" has the list view that exists today, and the list is what a screen reader gets (`role="list"`, each zone a button with its band and state in the name).
- Live regions: the sentence (polite) and the alert list (polite; assertive for a red row only). Order rows do not announce on every poll, only on a state change.
- Reduced motion: no pulsing, no moving dash; state changes swap instantly.
- Light theme stays and every new token gets its light value in the same PR (the design system's rule).

## 5. Motion

| Where | What | Duration |
|---|---|---|
| A number changes | none (fixed width); the meter fill eases | `--d-dur-2` |
| A row enters or leaves Orders or Needs you | height and opacity | `--d-dur-2` |
| An alert appears | one fade in; no shake, no loop | `--d-dur-2` |
| The current map target | a steady gold ring; no pulse | - |
| Popover, menu, sheet | opacity and 4 px rise | `--d-dur-1` |
| Undo bar | linear width | 5 s |

Nothing loops. Nothing moves because of a poll that changed no value (the design system's rule 5 stands).
