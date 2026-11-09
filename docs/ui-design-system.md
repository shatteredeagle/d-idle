# d.idle watch UI design system (TASK-228, uidesign)

> status: proposal (2026-11-09), docs and mockups only; nothing in `internal/watch` changed. Research and sources: [research/ui-design.md](research/ui-design.md). Mockups: [mockups/ui/](mockups/ui/). Test and migration plan: [ui-regression-plan.md](ui-regression-plan.md). Decisions the user still owns are in section 13; until taken, the safe defaults named there stand.

## 1. Why a system, in one paragraph

The watch screen grew one panel per card, each with its own markup, CSS and grid rule, so every new panel edited the layout of the last one and the page is now a header that wraps three deep, a gear strip with no spacing (its injected `<style>` is blocked by the page's own CSP), three fixed widths that fight on a laptop, two toasts, two sheets and three palettes. The fix is not another pass over `app.css`. It is a shell that owns layout, tokens that own look, a short component catalogue that owns behaviour, and a panel contract under which a feature ships as one self-contained module that the shell places. After that, adding a feature touches one new directory and one line in a manifest, and the regression suite in the plan doc proves the rest of the page did not move.

## 2. Principles

1. **The hero first.** What is the hero doing, where, how is it, and why: four facts, always visible, same place at every width. Everything else is explanation and one tap away.
2. **Layout is the shell's; look is the tokens'; behaviour is the components'.** A panel owns only its content. It never positions itself, never names a colour, never knows the viewport.
3. **Data places panels.** A manifest entry (size, priority, band) decides where a panel goes; the shell computes the grid. No hand-written grid rules per panel, ever.
4. **Two levels, then a page.** Level 1 is visible; level 2 is a fold, a tab or a sheet; anything deeper is its own page (inventory, policy, return, pacts, hall) built on the same shell.
5. **Nothing moves unless the user or an event moved it.** Every panel renders at its final size before its data arrives (skeleton), late panels never push earlier ones, numbers keep their width, and the only layout change is an unlock, which is an event with a reveal.
6. **Locked is a state, not an absence.** A locked feature is a silhouette tile in its future place with its teaser and price. The ladder opens it in place.
7. **The palette is Diablo II's: dark, warm, gold on near-black, rarity colours as the one semantic colour family.** One ornament device, on one surface tier. A light theme stays and is tested.
8. **Readable by a stream viewer at 480p and by a screen reader.** Contrast 4.5:1 for text, 3:1 for everything else, a 2 px focus ring, reduced motion honoured, live regions polite and rare.
9. **Stdlib and CSP stay.** No framework, no CDN, no inline styles or scripts; `style-src 'self'` is a feature, and the linter enforces what the CSP would otherwise break at runtime.
10. **A panel that breaks the rules fails CI, not the user's screen.** The contract linter, the golden DOM and the screenshot diff are the design system's teeth.

## 3. Tokens

All tokens live in `internal/watch/web/tokens.css` inside `@layer tokens`. Prefix `--d-`. Panels read tokens, never define them. Primitives are numbered scales; roles are named; components use roles only.

### 3.1 Colour

Two scales of twelve steps (Radix model: 1 app background ... 6 card border, 8 focus, 9 solid, 11 low-contrast text, 12 text), a warm neutral (`--d-stone-N`) and a gold (`--d-gold-N`), plus fixed semantic and rarity hues. Values below are the dark theme (the proposed default); the light theme remaps the same roles. Contrast figures were computed for the pairs the components use (text on surface tiers, badges, meters); they are U until the token sheet is rendered and measured in the screenshot suite.

```css
@layer tokens {
  :root {
    /* primitives, dark */
    --d-stone-1: #0d0b09; --d-stone-2: #14110e; --d-stone-3: #1c1813; --d-stone-4: #241f19;
    --d-stone-5: #2d2720; --d-stone-6: #3a322a; --d-stone-7: #4a4036; --d-stone-8: #6a5c4d;
    --d-stone-9: #8c7b67; --d-stone-10: #a08f7a; --d-stone-11: #b9aa94; --d-stone-12: #ece2cf;
    --d-gold-6: #5a4718; --d-gold-8: #8a6d22; --d-gold-9: #c7a44a; --d-gold-10: #dcb85e;
    --d-gold-11: #e6c76f; --d-gold-12: #f3e2a8;

    /* roles: surface tiers (Material surface-container model; elevation = tier, not shadow) */
    --d-bg: var(--d-stone-1);              /* page */
    --d-surface-0: var(--d-stone-2);       /* bands (status, stage) */
    --d-surface-1: var(--d-stone-3);       /* card */
    --d-surface-2: var(--d-stone-4);       /* inset inside a card: list rows, meters, inputs */
    --d-surface-3: var(--d-stone-5);       /* sheets, popovers, toasts */
    --d-line: var(--d-stone-6);            /* card and separator borders */
    --d-line-strong: var(--d-stone-7);     /* interactive borders */
    --d-text: var(--d-stone-12);
    --d-text-2: var(--d-stone-11);         /* secondary */
    --d-text-3: var(--d-stone-9);          /* placeholder, locked */
    --d-accent: var(--d-gold-9);           /* the one brand colour: titles, active tab, primary button */
    --d-accent-text: var(--d-gold-11);     /* accent used as text on surface-1 */
    --d-on-accent: #1a1408;
    --d-focus: #8db8ff;                    /* 3:1 on every surface tier */
    --d-ornament: var(--d-gold-8);         /* the hairline device, surface-1 only */

    /* semantic (state) with their on-colours */
    --d-good: #7cc47e; --d-on-good: #0c1a0d;
    --d-warn: #e3a93a; --d-on-warn: #1a1204;
    --d-bad:  #ec7d6c; --d-on-bad:  #1c0806;
    --d-info: #7ab0e8; --d-on-info: #06111c;

    /* vitals */
    --d-life: #d2453a; --d-mana: #4f7fe0; --d-xp: var(--d-gold-9); --d-stamina: #c9b26a;

    /* rarity (Diablo II text colours, adjusted to 4.5:1 on surface-1) */
    --d-q-normal: #e6ddc8; --d-q-low: #9d968a; --d-q-magic: #7f9fff; --d-q-rare: #f0dc5a;
    --d-q-set: #62c070; --d-q-unique: #d6ad5a; --d-q-crafted: #f0a040; --d-q-runeword: #d6ad5a;
  }
  :root[data-theme="light"], :root:not([data-theme]) { /* light remap under prefers-color-scheme: light */ }
}
```

Rules: `--d-text` on `--d-surface-1` and `--d-surface-2` is at least 4.5:1; `--d-text-2` on `--d-surface-1` at least 4.5:1; `--d-text-3` is for locked and placeholder text only and must still meet 3:1; every `--d-<state>` has an `--d-on-<state>` for solid badges and buttons; rarity colours are used only for item names and the card top rule, never as a UI state.

Theme switching keeps today's mechanism (`data-theme` on `<html>`, `d2theme` in localStorage, the auto/light/dark button, `color-scheme: light dark`); only the values move into `tokens.css`, and the inventory, pacts and hall pages load the same file.

### 3.2 Type

System stacks only (no web font, CSP and offline). Display: `--d-font-display: Georgia, "Iowan Old Style", "Palatino Linotype", serif` (the Diablo flavour; Exocet may be self-hosted later as an opt-in, see section 13). Body: `--d-font-body: system-ui, -apple-system, "Segoe UI", Roboto, sans-serif`. Numbers: `--d-font-num: ui-monospace, "SF Mono", Menlo, Consolas, monospace` with `font-variant-numeric: tabular-nums` for every live value.

Scale (fluid, `clamp(min, preferred, max)`, Open Props style numbering):

| Token | Size | Use |
|---|---|---|
| `--d-fs-0` | `clamp(.72rem, .7rem + .1vw, .78rem)` | badges, captions |
| `--d-fs-1` | `clamp(.84rem, .8rem + .15vw, .9rem)` | secondary text, list meta |
| `--d-fs-2` | `clamp(.95rem, .9rem + .2vw, 1.05rem)` | body |
| `--d-fs-3` | `clamp(1.05rem, 1rem + .3vw, 1.2rem)` | card titles |
| `--d-fs-4` | `clamp(1.25rem, 1.1rem + .6vw, 1.6rem)` | stat tile value, page title |
| `--d-fs-5` | `clamp(1.6rem, 1.3rem + 1.2vw, 2.4rem)` | hero status word, kiosk whyline |
| `--d-lh-tight` 1.2, `--d-lh` 1.45 | | |

Inside a panel, a component may size from its container with `cqi` (for example the stat tile value: `font-size: clamp(var(--d-fs-3), 8cqi, var(--d-fs-5))`), never from `vw`.

### 3.3 Space, radius, line, elevation, z

| Token | Value | Token | Value |
|---|---|---|---|
| `--d-sp-1` | 4px | `--d-r-1` | 4px (badge, cell) |
| `--d-sp-2` | 8px | `--d-r-2` | 8px (button, tile) |
| `--d-sp-3` | 12px | `--d-r-3` | 12px (card, sheet) |
| `--d-sp-4` | 16px | `--d-r-round` | 999px (chip, dot) |
| `--d-sp-5` | 24px | `--d-bw` | 1px |
| `--d-sp-6` | 32px | `--d-shadow-1` | `0 1px 2px rgb(0 0 0 / .4)` (toasts and sheets only; cards use surface tiers) |
| `--d-gap` | `var(--d-sp-4)` (grid gap, cards) | `--d-shadow-2` | `0 8px 24px rgb(0 0 0 / .45)` |
| `--d-pad` | `var(--d-sp-4)` card padding; `--d-pad-dense: var(--d-sp-3)` | | |
| `--d-z-band` 10, `--d-z-sheet` 20, `--d-z-toast` 30, `--d-z-banner` 40 | | | |

Density: one knob, `data-density="comfortable|compact"` on `<html>` (default comfortable; compact sets `--d-pad` and list row padding to the dense values and `--d-fs-2` to `--d-fs-1`). Panels never set padding literals; they use `--d-pad` and `--d-sp-N`.

### 3.4 Motion

| Token | Value | Use |
|---|---|---|
| `--d-dur-1` | 120ms | hover, press |
| `--d-dur-2` | 240ms | fold, tab, sheet |
| `--d-dur-3` | 600ms | change highlight fade |
| `--d-ease` | `cubic-bezier(.2, 0, 0, 1)` | all |

`@media (prefers-reduced-motion: reduce) { :root { --d-dur-1: 0ms; --d-dur-2: 0ms; --d-dur-3: 0ms; } }` replaces today's `* { transition: none !important }`; colour changes stay, movement stops. Only `opacity` and `transform` are animated. Digits are never animated for per-tick values.

## 4. The shell

`internal/watch/web/shell.css` (`@layer shell`) and `shell.js`. The shell owns: the document grid, the bands, the tab rail on a phone, overlays (sheet, toasts, banner, scrim), the fetch scheduler, panel placement, focus keeping, theme and density, the kiosk and overlay modes.

### 4.1 Bands and the named grid

```
status   one row, sticky: brand, hero status tile, whyline, alerts count, links, theme
stage    video + map (the two "see it" panels); hidden entirely when neither is available
grid     every other panel, auto-placed by priority and size class, dense packing
rail     phone and tablet only: the level-2 tab strip (bottom, thumb reach)
```

```css
@layer shell {
  .d-shell { display: grid; min-block-size: 100dvh; gap: var(--d-gap);
    grid-template-columns: repeat(var(--d-cols), minmax(0, 1fr));
    grid-template-areas: "status" "stage" "grid"; grid-template-rows: auto auto 1fr; }
  .d-band-status { grid-area: status; position: sticky; inset-block-start: 0; z-index: var(--d-z-band); }
  .d-band-stage  { grid-area: stage; display: grid; grid-template-columns: subgrid; }
  .d-band-grid   { grid-area: grid; display: grid; grid-template-columns: subgrid; grid-auto-flow: dense; align-items: start; }
  [data-size="s"] { grid-column: span 1; } [data-size="m"] { grid-column: span min(2, var(--d-cols)); }
  [data-size="l"] { grid-column: 1 / -1; }
}
```

### 4.2 Breakpoints (the only media queries in the codebase)

| Name | Min width | `--d-cols` | Stage | Level 2 |
|---|---|---|---|---|
| phone | 0 | 1 | video full width, map stacked under it (collapsed by default) | bottom tab rail; the grid shows the active tab's panels |
| tablet | 40rem (640 px) | 2 | video 2 cols, map under it | bottom tab rail |
| laptop | 64rem (1024 px) | 3 | video 2 cols, map 1 col beside it | all panels in the grid; folds |
| wide | 90rem (1440 px) | 4 | video 2, map 1, hero status detail 1 | same |
| kiosk | 120rem (1920 px) or `?mode=kiosk` | 4, larger type (`--d-fs-*` +1 step), no links, no buttons | video 3, map 1 | none; fades to the status band after 20 s idle |
| overlay | `?mode=overlay` | 1 column, transparent `--d-bg`, status band only (hero status, whyline, last alert), max 40 characters per line, nothing under 20 px | | none |

### 4.3 Placement algorithm (shell.js, about 60 lines)

1. Load `/watch/panels.json` (the manifest, served from the embedded `web/panels/*/panel.json` files; a Go test and the linter validate it).
2. For each entry, create the panel root at once: `<section class="d-panel" data-panel="<id>" data-size="<size>" data-state="loading" aria-labelledby="...">` with the title and a skeleton of the declared `skeleton` rows, so the grid has its final shape before any data arrives.
3. Sort by `priority` (ascending), then `id`; append to the band named by `band`. The grid's `dense` auto-flow fills holes; the DOM order is the reading order and the tab order.
4. Resolve unlock state from `/api/unlocks`: a panel whose `unlock` key is locked renders the Locked component in a size-`s` root (`data-state="locked"`) at the same priority. When it opens, the shell swaps the state and size in place and plays the reveal.
5. Import the module `/watch/panels/<id>/panel.js`; call `mount(root, ctx)`. The module returns `{ update(data), destroy() }`. The shell schedules `GET endpoint` at `poll` seconds (or subscribes it to the SSE change event when `poll` is 0) and calls `update`. A panel never creates a timer or a fetch of its own.
6. Before every `update` the shell records the focused element by `data-key` inside the panel and restores it after (today's `renderOrders` trick, generalised).
7. On a phone, each panel is listed in the tab named by its manifest `tab`; the rail shows only tabs that have at least one open or locked panel.

### 4.4 Overlays the shell owns

One sheet (`d-sheet`: bottom sheet under laptop, centred dialog from laptop; focus trap, Escape, returns focus), one toast stack (`d-toasts`, top end, max 4, severity border), one banner slot (`d-banner`, for update and connection messages). Panels request them through `ctx.sheet(content)`, `ctx.toast(sev, text)`, `ctx.banner(text, actions)`; a panel that renders `position: fixed` fails the linter.

### 4.5 Fetch scheduler

One scheduler with the existing SSE `/api/events` as the trigger and a 5 s safety poll; per-panel intervals from the manifest; a panel hidden by a tab or a fold is polled at a quarter rate; `document.hidden` pauses everything but the status band. Replaces the six independent timers of today.

## 5. Component catalogue

Each component is a class `d-<name>` in `components.css` (`@layer components`), a factory in `components.js` that builds DOM with `textContent` only, a data-attribute API for state, and documented anatomy. Panels compose components; they do not restyle them. Props are data attributes; states are `data-state` or ARIA attributes.

| # | Component | Anatomy | Props (attributes) | States | Used by |
|---|---|---|---|---|---|
| 1 | **Card** `d-card` | header (title `h2`, meta slot, actions slot), body, footer slot; optional top ornament rule | `data-size s/m/l` (set by the shell), `data-ornament` (one tier only), `data-foldable` | `data-state loading/ready/empty/error/locked`, `aria-expanded` when foldable | every panel root |
| 2 | **Stat tile** `d-stat` | label, value (tabular, fixed `ch` width from `data-width`), unit, delta (arrow glyph + text), optional 24-point sparkline `<svg>` | `data-width` (characters reserved), `data-trend up/down/flat` | `data-changed` (600 ms highlight), `data-state` | hero status, uptime, tally, currency, kills, XP/h |
| 3 | **Meter** `d-meter` | track, fill, label, value text; `role=meter` or `progressbar` with `aria-valuenow/min/max` | `data-kind life/mana/xp/stamina/durability/order/fade` (colour role), `data-thresholds "25,50"` | `data-level ok/warn/bad` computed from thresholds; `data-changed` | life, mana, order progress, Thought fade bar, gear durability, risk |
| 4 | **Strip** `d-strip` | an inline row of chips, each `label value`, wraps at container width, `overflow-x: auto` with fade edges when `data-nowrap` | `data-nowrap` | per chip `data-level ok/warn/bad` | currency, tally, route crumbs, legend |
| 5 | **List** `d-list` | `ul` of rows: leading (time or icon), primary text, secondary text, trailing (badge or button); `aria-live` optional | `data-dense`, `data-max-height` (scroll inside, never grows the page) | row `data-kind` for the journal kinds | journal, feed, rules, chronicle, diary, quests, alerts, atlas, bestiary |
| 6 | **Table** `d-table` | `table` with `thead`, numeric columns right-aligned tabular, first column sticky under `data-sticky` | `data-sticky` | | hall of fame, spend, repair estimate, policy preview deltas |
| 7 | **Disclosure** `d-tabs` and `d-fold` | tabs: `role=tablist`, arrow keys, `aria-selected`; fold: button in the card header, `aria-expanded`, body hidden with `hidden` | `data-remember` (localStorage key) | | phone rail (shell), Thoughts/Feed, Chronicle/Tape/Reasons, card folds |
| 8 | **Alert** `d-toast` and `d-note` | toast: severity edge, text, Dismiss; note: inline banner inside a card | `data-sev info/good/warn/alarm` | | shell toasts, update banner, "too hard" warnbox, repair request state |
| 9 | **Badge** `d-badge` | one-line label, optional count | `data-tone neutral/good/warn/bad/accent/rarity-*` | | deaths, alert count, locked, default, "your call", queue position, quest state |
| 10 | **Button set** `d-btn`, `d-btngroup`, `d-chip` | button: label, optional sub-label (`does`); group: segmented, one `aria-pressed`; chip: toggle pill | `data-variant primary/ghost/danger`, `data-size sm/md`, min target 44 px for `md` | `disabled`, `aria-pressed` | every action; duration chips; policy options |
| 11 | **Empty and Locked** `d-empty`, `d-locked` | empty: icon slot, one sentence, optional action; locked: dashed outline, silhouette title, teaser, price badge, Buy button when purchasable, "opens when ..." text otherwise | `data-price`, `data-purchasable` | `data-state locked/opening/open` (reveal) | every panel's empty state; the unlock ladder |
| 12 | **Ticker** `d-ticker` | a single line that replaces its text (whyline, voice), `role=status aria-live=polite aria-atomic=true`, min 3 s hold, optional serif voice style | `data-voice` | `data-changed` | whyline, voice strip, Chronicle latest, kiosk headline |

Inventory grid cells (`d-cells`) are a thirteenth, page-specific component for the inventory page: a CSS grid of `repeat(cols, var(--d-cell))` with items placed by `grid-area` set from the CSSOM (allowed under CSP), rarity colour as the cell's top rule, marks (sell, keep, pinned) as badges. It replaces the absolute-positioned 34 px cells.

## 6. The panel contract

A feature ships as `internal/watch/web/panels/<id>/` with exactly three files, and one line nowhere else.

```
panels/<id>/panel.json   the manifest (below)
panels/<id>/panel.css    @layer panels { [data-panel="<id>"] ... }   only
panels/<id>/panel.js     export function mount(root, ctx) -> { update(data), destroy() }
```

### 6.1 Manifest

```json
{
  "schema": "watch.panel/1",
  "id": "gear",
  "title": "Gear condition",
  "band": "grid",
  "size": "s",
  "minSize": "s",
  "maxSize": "m",
  "priority": 30,
  "tab": "hero",
  "defaultOpen": true,
  "endpoint": "/api/repair",
  "poll": 5,
  "unlock": "",
  "skeleton": 3,
  "owner": "repairmanual",
  "doc": "docs/repair.md"
}
```

| Key | Meaning | Checked by the linter |
|---|---|---|
| `id` | directory name, `[a-z][a-z0-9-]*`, unique | equals the directory, unique across panels |
| `title` | the card title, plain text | non-empty, no HTML |
| `band` | `status`, `stage` or `grid` | `status` and `stage` hold only the ids the shell lists (hero status, whyline, alerts count; video, map) |
| `size`, `minSize`, `maxSize` | `s` (1 col), `m` (2), `l` (full row); the shell may grow a panel between min and max to fill a row | `minSize <= size <= maxSize` |
| `priority` | reading order, 0 first; tens are reserved per group (0 to 9 hero, 10 to 29 orders, 30 to 49 condition, 50 to 69 story, 70 to 89 discovery, 90 and up meta) | integer, unique within a band |
| `tab` | the phone rail tab: `hero`, `orders`, `story`, `discover`, `settings` | closed set |
| `defaultOpen` | folded or open at laptop and wider (phones never fold) | boolean |
| `endpoint`, `poll` | one GET on the viewer origin and its interval in seconds (0 = on SSE change only) | path starts with `/api/`; the handler exists (Go test) |
| `unlock` | ladder id or empty (always open) | the id exists in `data/unlocks/ladder.json` |
| `skeleton` | rows of placeholder in the loading state | 1 to 8 |
| `owner`, `doc` | the card name and the doc that explains the feature | doc path exists |

Writes are not in the manifest: a panel posts to the answer origin through `ctx.post(path, body)`, which carries the base URL, the token and the error banner, so no panel reads `/api/config` on its own (today four scripts do).

### 6.2 What a panel may and may not do

May: build DOM with the component factories and `textContent`; read tokens; set `data-*` on its own elements; read its container width with `@container`; expose panel-local custom properties under `--p-<id>-*`; request `ctx.sheet`, `ctx.toast`, `ctx.banner`; store a preference under `ctx.store(key)` (prefixed by panel id).

May not: write to any element outside its root; use `document.querySelector` (the shell hands it `root`); use `setInterval`, `fetch`, `EventSource` (the scheduler does); write `style` attributes or inject `<style>`; use media queries, `!important`, ids, element or global selectors, unlayered CSS, `position: fixed/sticky`, fixed `width` or `height` on its root, colours other than tokens; set `aria-live` on more than one element.

### 6.3 Stable data-attribute API

The DOM the shell and the components produce is the test surface and the hook for screenshots and golden files: `data-panel`, `data-size`, `data-state`, `data-tab`, `data-sev`, `data-tone`, `data-kind`, `data-level`, `data-changed`, `data-key` (stable identity of a row or button for focus keeping and goldens). Classes are for styling only and may change; data attributes are a contract and change only with a schema bump in the manifest `schema`.

## 7. Information hierarchy

Level 1 (always visible, the `status` and `stage` bands): hero status tile (state word, area, life and mana meters, gear word), whyline ticker, alerts count, video, map. Level 2 (the `grid`, one tap on a phone): Hero now and the order map, Up next, Thoughts, condition (gear, tally and spend, risk), story (Chronicle, Journal, Diary, Voice), discovery (Quests, Find cards, Cain's notes, Atlas, Still locked), Rules, Alerts list. Pages: Inventory, Policy sheet, Return screen, Pacts, Hall, each a `d-shell` with the same status band and a `main` of cards.

### 7.1 Wireframes

Wide (4 columns):

```
+-- status ------------------------------------------------------------------------------+
| d.idle | FIGHTING  Blood Moor  life ####--- 72%  mana ##----  gear fine | why: "Clearing  |
|        | Blood Moor, my life is 72%."                     | alerts 2 | Inventory | Return | Theme |
+-- stage -------------------------------------------------------------------------------+
| video 4:3 (2 cols)                         | map (1 col)            | hero detail (1)    |
|                                            | fog, cross, units      | lvl 9  xp/h  kills |
|                                            |                        | uptime  deaths     |
+-- grid (dense, by priority) -----------------------------------------------------------+
| Hero now (m)                | Where to? map (m)                                        |
| Up next (s) | Thoughts (m)                | Gear condition (s)                         |
| Tally & spend (s) | Risk (s) | Chronicle (m)               | Journal (m)              |
| Quests (s)  | Find cards (s) | Cain's notes (s, folded) | Atlas (s, folded)            |
| Rules (m, folded)           | Alerts (m)                 | Still locked (s) | Diary (s) |
+----------------------------------------------------------------------------------------+
```

Laptop (3 columns): same bands; stage is video (2) + map (1); the hero detail tile folds into the status band as a second line; grid packs three across.

Tablet (2 columns): status band two lines (status word and vitals, then why); stage video full width with the map under it (collapsed to a 96 px strip, tap to expand); grid two across, tab rail at the bottom.

Phone (1 column):

```
+------------------------------+
| d.idle  FIGHTING  72% | 41%  |   <- status band, 2 lines, sticky
| "Clearing Blood Moor..."     |
+------------------------------+
| video 4:3                    |
| [map: collapsed strip]       |
+------------------------------+
| (active tab's panels, stacked, each a full-width card)
|                              |
+------------------------------+
| Hero | Orders | Story | Discover | More |   <- bottom rail, 44 px, badges on tabs
+------------------------------+
```

Kiosk and overlay: see section 4.2; the overlay is the status band alone on a transparent background, 2 lines, for OBS as a browser source at 1920 by 160.

## 8. Naming and file layout

```
internal/watch/web/
  tokens.css        @layer tokens      (light and dark values, density, reduced motion)
  reset.css         @layer reset       (box-sizing, element defaults; the ONLY element selectors)
  shell.css         @layer shell       (grid, bands, rail, sheet, toasts, banner; the ONLY media queries)
  components.css    @layer components  (d-* classes)
  shell.js          placement, scheduler, focus keeping, theme, modes
  components.js     factories: card(), stat(), meter(), strip(), list(), table(), tabs(), fold(), toast(), badge(), btn(), locked(), ticker()
  panels/<id>/panel.{json,css,js}
  pages/{watch,inventory,policy,return,pacts,hall}.html   each loads tokens, reset, shell, components, then the page's panels
```

Layer order, declared once at the top of `tokens.css`: `@layer reset, tokens, shell, components, panels, utilities;`. `utilities` holds `d-sr-only`, `d-num`, `d-muted`, `d-hidden` and nothing else. CSS class prefixes: `d-` system, `p-<id>-` panel-local. JS: ES modules, no globals (today `window.sensesUnitsDraw` and `window.sensesPanel` are globals; they become the map panel's own module).

Go side: `internal/watch/panels.go` embeds `web/panels/**`, parses every `panel.json` at init into `[]PanelManifest`, serves `/watch/panels.json` and the per-panel assets, and exposes `Manifests()` for the linter and tests. Pages built in Go (`invui.go`, `pactui.go`, `hall.go`, the policy and return templates) become templates that include the shared `<head>` and status band partials.

## 9. Rules that prevent regressions

1. Every stylesheet rule is inside a layer; `tools/uilint` fails on an unlayered rule or `!important`.
2. A panel's CSS file contains only rules whose every selector starts with `[data-panel="<its id>"]` (or `@container` blocks of such rules). No ids, no element selectors, no `*`, no `:root`, no media queries, no `position: fixed|sticky`, no `width|height|min-width|max-width` on the root selector itself, no colour literals (`#`, `rgb(`, `hsl(`, named colours) outside `tokens.css`.
3. No `style=""` attributes in any HTML or template; no `<style>` elements outside the page `<head>`; JS may set CSSOM properties only from a whitelist (`gridArea`, `gridColumn`, `gridRow`, `width` and `height` on `d-cells` items, `transform`, `--p-*` custom properties).
4. Only `shell.css` has media queries; everything else uses `@container`.
5. The manifest is validated at Go init and by the linter; a panel directory without a valid manifest, or a manifest without a directory, fails CI.
6. The shell places; a panel that sets `grid-column`, `grid-row` or `order` fails the linter.
7. The data-attribute API is a schema: a golden DOM test per panel (state `ready` with a fixture, `empty`, `locked`, `error`) catches a changed attribute; a changed class does not fail anything.
8. Screenshots per breakpoint and theme of every page, diffed against goldens with a tolerance; a panel PR may only change pixels inside its own panel's box (the diff tool masks the other panels' boxes from the DOM geometry).
9. New token or component: a PR to `tokens.css` or `components.css` with a mockup update and a screenshot golden update; never added from a panel PR.
10. Text through `textContent` only (kept from today; `TestWatchAssetsHaveNoThirdPartyURL` and the no-inline tests stay and extend to panels).

## 10. Migration table

| Today | Becomes | Component(s) | Band / size / priority | Notes |
|---|---|---|---|---|
| header brand, state dot, uptime, deaths, clock | `hero-status` panel | Stat tile, Meter, Badge | status / - / 0 | state word from `/api/uptime` label; area from senses or orders; life and mana from the status poll (new fields on `/api/uptime` or a small `/api/hero`); gear word from `/api/repair` |
| whyline `p.whyline` | `whyline` panel | Ticker | status / - / 1 | unchanged data |
| alerts badge on the tab, toasts | shell toasts + `alerts-count` | Badge, Toast | status / - / 2 | the list stays in the `alerts` panel |
| Inventory, Return links, Theme button | shell status band links | Button set | status | kiosk hides them |
| currency strip | `tally` panel | Strip, Stat tile | grid / s / 40 | header strip removed |
| tally strip | same `tally` panel | Strip | | one panel, two rows |
| gear strip (`gear.js`) | `gear` panel: compact condition card | Meter per slot (durability), Badge, Button | grid / s (m when > 6 slots) / 30 | fixes the CSP break; see section 12 |
| voice strip | `voice` panel | Ticker (voice style), Button (mute) | grid / s / 52 | diary pages inside, folded |
| `#msgbar` | shell banner | Alert (note) | shell | |
| Game video + collapse + full screen | `video` panel | Card (actions), the `<img>` with `aspect-ratio: 4/3` | stage / m (l in kiosk) / 0 | reserved height before the first frame |
| map side panel (`senses.js`, `senses_units.js`) | `map` panel | Card, canvas sized by container, Button set (zoom, hide) | stage / s / 1 | canvas width from `cqi`, not 320 px |
| Hero now | `hero-now` panel | Card, Meter (order), Strip (crumbs), Button set | grid / m / 10 | |
| Where to (SVG map, list toggle, sheet) | `orders-map` panel | Card, Button set, shell sheet | grid / m / 11 | the order sheet uses `ctx.sheet` |
| Up next | `queue` panel | List, Badge | grid / s / 12 | |
| Thoughts + feed | `thoughts` panel | Card, Tabs (Questions, Feed), Meter (fade), Button set | grid / m / 20 | |
| Rules | `rules` panel | List, Button (danger) | grid / m / 92, folded | |
| Journal | `journal` panel | List (max-height), Disclosure (why) | grid / m / 60 | |
| Spend | inside `tally` | Table, Button | | |
| Chronicle, tape, recent reasons | `chronicle` panel | Tabs (Lines, Expeditions, Reasons), List | grid / m / 50 | serif stays as a List `data-voice` row style |
| Diary | inside `voice` | List | | |
| Quests | `quests` panel | List, Badge (state) | grid / s / 70 | |
| Find cards + reveal sheet | `cards` panel | List, Button, shell sheet (rarity top rule) | grid / s / 71 | |
| Still locked | `locked` panel (and the per-panel Locked tiles) | Locked | grid / s / 95 | lists only locked panels that have no tile of their own (senses tiers, pacts) |
| Cain's notes | `bestiary` panel | List, Badge | grid / s / 72, folded | |
| Waypoint atlas | `atlas` panel | List | grid / s / 73, folded | |
| Alerts | `alerts` panel | List, Button set, Alert (locked teaser becomes Locked) | grid / m / 90 | |
| Risk (`/api/risk`, no panel yet) | `risk` panel | Meter, Strip | grid / s / 31 | new |
| Update banner (`update.js`) | shell banner | Alert | shell | drops `style.cssText` |
| Order toast + alert toasts | shell toasts | Toast | shell | one implementation |
| Order sheet + reveal sheet | shell sheet | | shell | one implementation |
| `/watch/inventory` (own palette, absolute cells) | `pages/inventory.html` | Cells, Card, Table, Badge, Button set, Alert | own page | durability as Meter per slot; the Repair block reuses the `gear` panel |
| `/watch/policy` | `pages/policy.html` | Card per row, Button group (options), Locked (locked options), Table (preview deltas) | own page | |
| `/watch/return` | `pages/return.html` | Card, List, Stat tile row (headline numbers), Button set, textarea | own page, single column `l` | |
| `/watch/pacts`, hall | `pages/pacts.html`, `pages/hall.html` | Table, Card, Badge | own page | shared tokens; theme toggle works |

## 11. Locked state

A panel whose `unlock` is closed renders as `d-locked` at size `s` in its own future place: dashed `--d-line-strong` outline, title in `--d-text-3` with a lock glyph, the ladder's `teaser`, and either a `d-badge` with the price and a Buy button (`tally_spend`) or a one-line "opens when ..." from the ladder's trigger text. On open: the shell sets `data-state="opening"`, the tile grows to its manifest size over `--d-dur-2` (height via `grid-template-rows` transition on the card, opacity for the content), one toast "Unlocked: Title", the Chronicle line the engine already writes, then `data-state="ready"`. Phones never show locked tiles in the rail count; the `locked` panel on the Discover tab lists them. Senses tiers and pacts (not panels) stay in the `locked` panel's list. Nothing a locked panel would show is in the DOM (the data stays absent, as today).

## 12. Overflow and long text

- The gear strip becomes a **condition card**: one Meter row per worn slot (`slot` label, durability fill, `cur/max` tabular at a fixed `ch` width, level colour from `warnPct`), rows wrap into two columns when the container is wider than 24rem (`@container`), the worst slot first, a one-line summary in the header meta ("2 worn, 1 broken, repair about 180 gold"), and the Repair now / Cancel button in the footer. It is never a single line of inline text again.
- Every List row uses `overflow-wrap: anywhere` and a `data-max-height` with inner scroll; a card never grows past its band.
- Strips wrap by default; `data-nowrap` strips scroll horizontally with fade edges and never clip a chip in half.
- Stat tiles reserve `data-width` characters; a value longer than that switches to suffix notation (K, M, B; the InnoGames convention) rather than widening the tile.
- Titles truncate with an ellipsis on one line at size `s`, wrap to two lines at `m`, and the full text is in `title=` and the accessible name.
- Numbers: `--d-font-num`, `tabular-nums`, right-aligned in tables, suffixes above 99,999, a thin space as the thousands separator below that.

## 13. Open decisions (safe default named; the mockups use the default)

1. **Palette direction.** Default here: the dark D2 palette (section 3.1) as the default theme, light kept and tested. The user's 2026-11-05 decision chose the light-first dashboard look; this proposal reverses the default and keeps both. Needs a yes.
2. **Density.** Default: comfortable, with the `data-density="compact"` switch available in the theme menu. Alternative: compact by default for the laptop and wide breakpoints.
3. **Stream overlay shares the shell?** Default: yes, `?mode=overlay` renders the status band alone from the same code. Alternative: a separate tiny page with its own CSS (less risk of the shell's chrome leaking into OBS, more duplication).
4. **Display face.** Default: system serif. Alternative: self-host Exocet if the user holds a licence (it is a commercial Emigre face; the research found conflicting claims about free use).
5. **Phone level 2.** Default: a bottom tab rail with five tabs. Alternative: one long scroll with folds (fewer taps, more scrolling).
6. **Hero status data.** Default: compose it from `/api/uptime`, `/api/senses`, `/api/repair` and the status poll on the client. Alternative: one `GET /api/hero` on the viewer that returns the four facts (cleaner, needs a small Go change in the first migration PR).

## 14. Proposed ADR text (for the coordinator or the user's grill session; not written to docs/adr/)

> ADR 00NN: Watch UI design system and panel contract. The watch screen and its pages are built on a token layer, a shell that owns layout and overlays, a component catalogue, and a panel contract: a feature ships as `web/panels/<id>/{panel.json,panel.css,panel.js}` and is placed by the shell from its manifest. Panels never position themselves, never define colours, never use media queries, never write inline styles (the page CSP `style-src 'self'` stays). CSS is layered (`reset, tokens, shell, components, panels, utilities`); `tools/uilint` fails CI on a rule outside a layer, a selector outside a panel's scope, a colour literal outside tokens, or an invalid manifest. Default theme: dark, Diablo II palette; light theme kept. Decision owner: user. Supersedes the look-and-feel line of 2026-11-05 in docs/research/watch-screen-design.md section 7 for the palette default only.
