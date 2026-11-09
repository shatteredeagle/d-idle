# Watch UI research: dashboards, idle and ARPG game interfaces, design systems (TASK-228, uidesign)

> status: research for the design system in [../ui-design-system.md](../ui-design-system.md) (2026-11-09). Docs only, no production code changed. Tags: V = page read in full or in part this session (URL given), S = search snippet or page index only, U = from memory, not re-read; treat U as a lead, not a fact. Repo facts are V by code reading (file named).

## 0. Answer in ten lines

1. The mess has one root cause, not twelve: every feature shipped its own markup, its own CSS (three stylesheets plus two scripts that inject styles) and its own placement rule in `app.css`. There is no shell that places panels, so each new panel edits the grid of the last one (V `internal/watch/web/app.css` lines 67 to 77, `watch.html`).
2. The gear strip text runs together because the page is served with `Content-Security-Policy: style-src 'self'` and `gear.js` injects a `<style>` element, which that policy blocks; the strip renders with no flex gap and `<b>Gear</b>` glued to "head" (V `internal/watch/http_view.go` `pageCSP`, `web/gear.js` lines 7 to 12). The same policy would block any future inline style; the design system must forbid them.
3. Dashboards that stay readable share three habits: a small number of always-visible primary tiles, progressive disclosure limited to two levels, and a grid whose cells are declared by data (Grafana's 24-column `gridPos`), never by hand-placed CSS (V NN/g, Grafana docs; S Grafana JSON model).
4. Idle games put the few live numbers the player watches in a fixed header or sidebar, give every other system its own tab that unlocks over time, and format large numbers with suffixes in a fixed-width column so they never jump (V Game Developer article, GCAP abstract; S InnoGames; U the games themselves).
5. ARPG HUDs (Diablo II and IV, Path of Exile, Last Epoch, Grim Dawn) keep vitals in the bottom centre, the map top-right, buffs beside the vitals, and everything else behind a key; the stash and inventory are literal cell grids with rarity colour as the only decoration. Nothing on the HUD reflows (U; the gallery indexes at Interface In Game confirm the screens exist, V).
6. Run summaries (Hades, Slay the Spire) are a single centred column of headline, cause, then a stat table; Hades proves the tone of the end screen can carry story (V Kotaku; U layout).
7. The CSS platform now has the tools to make panels isolated: cascade layers (`@layer`, Baseline 2022), container queries (Chrome 106, Firefox 110, Safari 16, about 96 percent of users), subgrid (Baseline 2023), logical properties, and `@scope` (newly available 2026, use with a fallback) (V MDN, caniuse).
8. Tokens are the agreed contract between design and code in Material 3, Radix, shadcn, Carbon, Polaris and Open Props: a reference palette, system roles (surface, on-surface, outline) and component tokens, and every role pairs with its "on" colour so contrast is designed once (V Material web docs, Radix, shadcn, Open Props).
9. Accessibility numbers to build in: text 4.5:1 (3:1 for 18 pt or 14 pt bold), non-text and focus indicators 3:1, focus ring at least a 2 px perimeter, reduce or replace motion under `prefers-reduced-motion`, live regions polite and batched (V WCAG 2.2 understanding pages, MDN; S aria-live guides).
10. No layout shift: reserve space for every late-arriving panel and media (`aspect-ratio`, fixed tile heights), never insert above the reading position, animate with `transform` and `opacity` only, CLS under 0.1 (V web.dev).

## 1. What the current UI is (inventory, V by code reading)

Pages and their styling sources:

| Page | Markup | Styles | Scripts | Writes |
|---|---|---|---|---|
| `/watch` | `web/watch.html` (static) | `app.css`; `gear.js` injects a `<style>` (blocked by CSP); `update.js` sets `style.cssText` (allowed, CSSOM) | `app.js`, `senses_units.js`, `senses.js`, `unlockui.js`, `alerts.js`, `gear.js`, `update.js` (mounted by the bot) | answer listener on another port |
| `/watch/inventory` | `invui.go` `invPage` (Go string) | its own inline `<style>` with hard-coded dark hex colours, no tokens | `inventory.js` (Go string) | `/api/inventory/{mark,equip,unpin}` |
| `/watch/policy` | `policysheet.go` template | `app.css` + `return.css` | `policy.js` | `/api/policy`, `/api/policy/preview`, `/api/unlocks/buy` |
| `/watch/return` | `returnscreen.go` template | `app.css` + `return.css` (`returnCSS`) | `return.js` | `/api/return/{answer-kind,notes}` |
| `/watch/pacts` | `pactui.go` `pactPage` | own inline `<style>`, hard-coded colours | `pacts.js` | `/api/pacts/*` |
| Hall of Fame | `hall.go` builds HTML by `WriteString` | own inline `<style>` | none | none |

Panels on `/watch` today (each is a `<section class="card" data-v=...>` with a hand-written grid-area in `app.css`, or an element inserted by a script):

| Panel | Where placed | Data | Unlock | Notes |
|---|---|---|---|---|
| Header: brand, state dot, uptime, deaths badge, currency strip, tally strip, Inventory and Return links, clock, theme button | `header.top`, flex-wrap | `/api/uptime`, `/api/currency`, `/api/tally` | always | wraps to three lines on a phone; strips are text blobs |
| Gear strip | inserted after the header by `gear.js` | `/api/repair` (5 s poll) | always | unstyled under CSP (finding 2) |
| Voice strip | `#voicestrip` | `/api/voice` | `voice` | italic serif quote |
| Message banner | `#msgbar` | errors | always | |
| Game: video + map side panel | `.videotop` fixed `max-width: 840px`, map `width: 338px` | `/stream`, `/api/senses` (0.5 s) | video flag; senses tier 1 | collapse and full screen; map hide and zoom |
| Whyline | `p.whyline` | `/api/whyline` | always | one line |
| Tab strip (phone only) | `nav.views` | | | hidden at 900 px and wider |
| Hero now | `.o-now` col 6 to 13 row 1 | `/api/orders`, `/api/orders/map` | always | order progress bar, crumbs, Cancel, Come home |
| Where to (Act map or list) | `.o-map` col 1 to 6 rows 1 to 3 | `/api/orders/map` | always | SVG nodes, order sheet |
| Up next (queue) | `.o-queue` col 6 to 13 row 2 | `/api/orders` | always (depth by `orderqueue*`) | |
| Thoughts (questions + feed) | `.thoughts` col 6 to 10 row 3 | `/api/thoughts` | always | fading countdown bars |
| Rules | `.rules` col 1 to 6 | `/api/rules` | always | foldable |
| Journal | `.journal` col 6 to 13, max-height 420 px | `/api/journal` | always | "why did you do that?" links |
| Spend | `.spend` full width | `/api/currency` | currency mode | |
| Chronicle (+ tape, recent reasons) | span 6 | `/api/chronicle` | `chronicle`, `chronicle_tape` | serif lines |
| Diary | span 6 | `/api/voice` | `voice` | |
| Quests | span 6 | `/api/questlog` | `questlog` | |
| Find cards (identify reveal) | span 6 | `/api/reveal` | `idreveal` | modal sheet, sound |
| Still locked (silhouettes) | span 6 | `/api/unlocks` | always while something is locked | folded by default |
| Cain's notes (bestiary) | span 6 | `/api/bestiary` | `bestiary`, `bestiary_resists` | folded |
| Waypoint atlas | span 6 | `/api/atlas` | `atlas` | folded |
| Alerts (+ toasts) | span 6; toasts fixed top-right | `/api/alerts` (4 s) | `alerts` (bought) or free rules | |
| Risk meter | not on the page yet (`/api/risk` exists) | `/api/risk` | `risk` | |
| Order sheet, reveal sheet, scrim | fixed overlays | | | bottom sheet on phone, centred dialog at 900 px |
| Update banner | fixed bottom, inline style via CSSOM | `/api/update/status` | always | own colours (`#222`) |
| Toast (orders) and toasts (alerts) | two different fixed elements | | | duplicated component |

Observed defects that a design system must make impossible, not just fix once:

- Placement by hand: `app.css` lines 73 to 76 name grid columns per class; a panel without a rule falls to `span 6`. Adding a panel means editing this block, which is where "new features break existing elements" comes from.
- Three widths are literals: video card 840 px, map panel 338 px, canvas 320 px. On a 1024 px laptop the video and the map do not fit beside each other (`.gamerow` is `flex-wrap: nowrap`), so the map shrinks the video.
- The header is one flex-wrap row holding nine unrelated things; on a phone it is the tallest element on the page.
- Two toast implementations, two colour sets (`--ok` is referenced in `.toast.good` but never defined; `#d90` is hard-coded), two sheets, two "hint" styles.
- Page-level styles leak into panels (`h2` margin, `.log-list li` border, `button` min-height 36 px, `.badge` uppercase) so a panel cannot change its own look without a more specific selector, and more specific selectors are how the cascade war starts.
- The inventory, pacts and hall pages have their own palettes; the theme toggle does not reach them.
- Nothing reserves space: panels appear when their first poll answers (4 to 5 s after load), which pushes everything below them (layout shift).
- No locked-state component: a locked feature is either hidden or one line in "Still locked"; a bought feature (alerts) has a third treatment (teaser plus Buy button inside its own panel).

## 2. Dashboard information architecture and density

### 2.1 Progressive disclosure (V, NN/g, https://www.nngroup.com/articles/progressive-disclosure/)

- "Initially, show users only a few of the most important options." At most two disclosure levels; beyond two "typically have low usability".
- "The very fact that something appears on the initial display tells users that it's important": the primary level is a statement of priority. Pick it by frequency of use, confirm by watching users.
- Multiple secondary displays are fine (each behind its own clearly labelled control); staged disclosure (wizards) only when steps are independent.
- For us: level 1 is the always-visible band (hero status, why, video, map, alerts); level 2 is one tap (a tab, a fold, a sheet). Nothing deeper on the same page; anything deeper is a separate page (inventory, policy, return).

### 2.2 Grafana dashboards (V, https://grafana.com/docs/grafana/latest/dashboards/build-dashboards/best-practices/; S, JSON model `gridPos` via https://docs.aws.amazon.com/grafana/latest/userguide/v10-dash-dashboard-json-model.html)

- A dashboard "tells a story or answers a question", general to specific; keep only data that answers the question; reduce cognitive load so a viewer knows what each panel is without effort.
- Colour carries one consistent meaning across the whole board (blue good, red bad); normalise units (CPU as a percent of cores) so 100 always means the same thing.
- Hierarchy of dashboards: a high-level board links down to detail boards; alerts link to the board that explains them. Avoid sprawl: template variables instead of copies.
- Layout model (S): a 24-column grid, panel `gridPos {x, y, w (1 to 24), h}` in 30 px row units, with "negative gravity" that floats panels up into empty space. Panels are data, placed by the engine. This is the model for our panel manifest: a panel declares its size class and priority; the shell places it.

### 2.3 Datadog, Polaris, Carbon (V Carbon, https://carbondesignsystem.com/elements/2x-grid/overview/; U Datadog and Polaris)

- Carbon 2x grid: mini unit 8 px; breakpoints Small 320 (4 columns), Medium 672 (8), Large 1056 (16), X-Large 1312 (16), Max 1584 (16); 16 px padding, gutter 32 px; fluid, fixed and hybrid grids; "test at each standard breakpoint" (V).
- Datadog (U): ordered layout (widgets flow and reflow) versus free layout (absolute); groups with a header; notes as section titles; a timeframe shared by the board. Polaris (U): a primary column about two thirds wide and a secondary column one third, cards as the unit of grouping, a 4 px base spacing scale.
- Takeaway: every mature system fixes a breakpoint table and a spacing unit once and then only ever composes from them.

### 2.4 Density

- Tufte's rule (U): maximise data-ink; small multiples; sparklines inline with text. For a watch screen that means stat tiles with a one-line sparkline or delta, not a chart per number.
- Game Developer on layout (V, https://gamedeveloper.com/design/ux-design-alleviating-burdens-on-playing-gaming-through-better-interface-layout): tabs show a subordinate structure; first and second level tabs go on different edges or differ in shape, colour or size; "reveal information gradually. New players should see basic controls first"; group by category and offer filters when categories follow a pattern; lighter tones read as nearer, darker as further, so colour blocks show parent and child; animate the parent first, then the children.

## 3. How idle, incremental and ARPG games show data

Nothing in this section was read from the games this session; the Interface In Game indexes confirm the screens exist (V https://interfaceingame.com/games/hades/, /slay-the-spire/, /path-of-exile/, /diablo-iv/, /diablo-ii-resurrected/: the pages list "In game", "Inventory", "Stash", "Map", "Game over", "Overall statistics", "Waypoint", "Quests" screenshot sets), the descriptions are U unless tagged.

### 3.1 Idle and incremental

| Game | Pattern worth borrowing | Tag |
|---|---|---|
| Cookie Clicker | Left column: the big number and its rate (always visible); centre: the news ticker and the building rows; right: the store. Numbers use word suffixes (million, billion) and never change width enough to reflow. Achievements and upgrades appear as a growing tile grid with a locked silhouette state (dimmed tile, "???"). | U |
| Antimatter Dimensions | A fixed header with the one resource that matters (antimatter and its rate) and a two-level tab bar: top-level tabs (Dimensions, Options, Statistics, Achievements, Challenges...) and sub-tabs per tab. Tabs appear as they unlock. Every row is "name, amount, multiplier, buy button" with tabular numbers. | U |
| Melvor Idle | Sidebar navigation of skills with the current level beside each; the active skill shows a progress bar in the sidebar, so "what is happening" is visible from every page. Mastery and unlock states use a dim tile with a lock icon and the requirement as text. | U |
| Realm Grinder | A central tabbed panel (Buildings, Upgrades, Research, Trophies) around a fixed resource header; faction colour themes the whole panel. | U |
| NGU Idle | Many features, each its own tab that unlocks; a left-hand tab rail with a tiny progress bar per tab so the player sees activity without visiting. Dense numbers, little decoration. | U |
| Kittens Game | Resource table left (name, amount, max, rate per tick in a fixed-width column), tabbed workspace centre, log right. The log is the story feed; everything else is a table. | U |
| Dungeon, Inc. (GCAP 2017 "Idle By Design") | "Incremental games lean heavily on UI to convey the vast majority of their gameplay, which is often complex and number-heavy"; a satisfying UI is what makes an idle game stand out. | V abstract, https://gcap2017.sched.com/event/2961e59bb5e8bad84ff6c38cb3ed290c |
| Number formatting (InnoGames) | Letter notation K, M, B, T, AA, AB over scientific; rounding during formatting was the main source of bugs. | S https://blog.innogames.com/dealing-with-huge-numbers-in-idle-games/ (fetch failed, snippet only) |
| Mobile idle guides | Unlock category tabs after a milestone "to reduce cognitive load" and make new UI feel like a reward; large tap targets (one guide says 100 px; WCAG 2.5.8 says 24 px minimum, our own rule is 44 px). | S search results |

Common denominators: (a) one fixed, small "vitals" header; (b) tabs that unlock; (c) tables and rows, not prose; (d) tabular numbers with suffixes; (e) a log as the narrative; (f) locked content shown as a dim silhouette with its requirement, never hidden.

### 3.2 ARPG HUDs, inventory and stash

| Game | HUD | Inventory and stash | Tag |
|---|---|---|---|
| Diablo II (and Resurrected) | Bottom bar: life orb left, mana orb right, belt (4 columns) and the two skill buttons centre, XP bar as a thin strip above the bar, stamina bar. Everything else behind a key; the automap overlays the world. The look: dark stone and iron frames, gold trim, Exocet for titles, a plain readable serif for body text, parchment-coloured text on near-black, rarity colours as the only semantic colour (white normal, grey ethereal/low, blue magic, yellow rare, green set, gold unique, orange crafted, also gold for runewords in text). | Paper doll of slots above a 10 by 4 grid; the stash is a 6 by 8 grid (Resurrected 10 by 10 and shared tabs). Items are cell images; the tooltip carries all numbers. | U; typeface V (section 3.4) |
| Diablo IV | Action bar bottom centre with orbs at both ends, buff row above it, minimap top right with the objective tracker beneath it. Inventory: paper doll left, grid right, tooltip on hover with the rarity tier colouring the item name and the frame. | Grid plus paper doll; stash tabs | U |
| Path of Exile | Orbs bottom corners, flask bar bottom centre, skill bar bottom right, the XP bar as a full-width hairline at the very bottom, the minimap top right or as a world overlay. The loot filter decides what the player sees at all, and its rule list is the real inventory UI (that point is V in docs/research/incremental-arpg.md). | Stash tabs by type with quad tabs | U |
| Last Epoch | Same anatomy as PoE with a cleaner, flatter style; the loot filter is in-game and rule-based with colours per rule. | | U |
| Grim Dawn | Orbs at the bottom corners, the skill bar centre, a textured parchment inventory with a grid and a separate "component" grid. | | U |
| Warframe | A HUD that fades when nothing happens and returns on change (health, shields, energy top right; ability row bottom right); the inventory and arsenal are tile grids with a left-hand filter rail. | | U |

Takeaways: vitals stay in fixed positions; everything else is behind one key or tab; the grid is the inventory metaphor and rarity colour is the one semantic colour; the HUD never reflows; Warframe's fade-on-idle is the right model for a spectator overlay.

### 3.3 Run summaries and spectator screens

- Hades (V, https://kotaku.com/let-s-celebrate-the-silly-death-screens-at-the-end-of-h-1845486673): the end screens pair one image with one line of narration; after many runs the lines become unrelated jokes; the author argues the tone works because it keeps a grim game from "taking itself too seriously". Layout (U): a centred column, headline, the cause, then a stat row (chambers, time, kills) and the boon list. Our return screen's "three sentences" and Chronicle already lean this way; the lesson is "headline first, numbers second, story voice allowed".
- Slay the Spire (U; V that "You are slain" is tagged Game over, Progress, Stats at Interface In Game): a fixed top bar (HP, gold, potions, floor, deck, map button) that never changes layout; the death screen is one column: cause, score breakdown as a list with totals, then the deck and relics as tile rows.
- Stream overlays (S, Elgato, Nerd or Die, MacMyths guides): design at 1920 by 1080; keep the centre clear; webcam in a corner at 15 to 20 percent of width; colour-coded safe zones (red avoid, blue sparingly, green safe); keep fine text out of the camera area; "give the main content the largest, clearest area". None of the guides covers downscaled 480p readability: test by downscaling to 854 by 480 (my rule from that gap: nothing on an overlay under 20 px at 1080p, and no line longer than 40 characters).
- Spectator culture (V in docs/research/incremental-arpg.md section 2.4): watchers love emergent stories and nicknames, not optimal play. The whyline and the Chronicle are the spectator content; the overlay should show them, not the tally.

### 3.4 The Diablo II look itself

- Type: the logo and menus use Exocet (Jonathan Barnbrook, Emigre, 1991), "inscriptions that are short and large in size typically utilize Exocet Heavy"; the page does not identify the in-game body face (V https://fontsinuse.com/uses/60790/diablo-ii-logo-and-menu). Exocet is a commercial Emigre face; one forum post calls it free, the font blogs say paid (S, conflicting). We cannot ship it; the design system uses a system serif for display and system sans for body, with Exocet named as an optional self-hosted display face if the user owns a licence (open decision).
- Palette (U, from the game): near-black backgrounds (#0b0a08 to #1a1612), warm dark browns for frames, gold for ornament and unique items (#c7a44a range), parchment text (#e8dcc0), the rarity set above, red life and blue mana orbs, a desaturated green for set items. Blizzard describes gold text for unique item names (S wiki mirror).
- Ornament (U): carved stone frames, iron rivets, gold filigree corners, chains. For a web dashboard the ornament should be a single device (a 1 px gold hairline with notched corners, or a top rule in gold) used on exactly one tier of surface, else the page becomes a texture.

## 4. Design systems and tokens

### 4.1 Material 3 (V, https://raw.githubusercontent.com/material-components/material-web/main/docs/theming/color.md; U the m3.material.io pages, which did not render)

- Roles: `primary`, `primary-container`, `secondary`, `secondary-container`, `tertiary`, `tertiary-container`, `error`, `error-container`, `background`, `surface`, `surface-dim`, `surface-bright`, `surface-container-lowest/low/-/high/highest`, `outline`, `outline-variant`; every token has an `on-<token>` for content placed on it, designed for accessible contrast (the doc says intended, not guaranteed). Dark and light schemes are generated from the same key colours and swapped wholesale.
- Lesson: five surface levels replace shadows for elevation in dark themes; the "on" pairing means a component never picks a text colour, it picks a surface and gets the text colour for free.

### 4.2 Radix Colors (V, https://www.radix-ui.com/colors/docs/palette-composition/understanding-the-scale)

12 steps with fixed meaning: 1 app background, 2 subtle background, 3 UI element background, 4 hovered, 5 active or selected, 6 subtle borders (cards, separators), 7 interactive borders, 8 strong borders and focus rings, 9 solid background (highest chroma), 10 hovered solid, 11 low-contrast text, 12 high-contrast text. Steps 11 and 12 are designed to meet APCA Lc 60 and Lc 90 on step 2. Dark mode: map the app background alias to step 1 or 2 of a grey or tinted scale. Lesson: one scale per hue with role-by-step gives a dark D2 palette that is still systematic.

### 4.3 shadcn/ui (V, https://ui.shadcn.com/docs/theming)

Variables as background and foreground pairs: `--background/--foreground`, `--card`, `--popover`, `--primary`, `--secondary`, `--muted`, `--accent`, `--destructive`, `--border`, `--input`, `--ring`, `--chart-1..5`, `--sidebar-*`, one `--radius` with a derived scale; colours in `oklch()`; dark mode by redefining the same variables under `.dark`. Lesson: a small, flat role set is enough for an application UI; the `-foreground` suffix is the same "on" rule as Material.

### 4.4 Open Props (V, https://open-props.style/)

Numbered scales: `--size-1..15`, `--font-size-0..8` and fluid variants, `--radius-1..6`, `--shadow-1..6`, `--ease-*`, `--animation-*`, `--layer-1..5` (z-index), `--gray-0..12`, `--duration-*`; the author recommends theme-agnostic aliases (`--surface-1`, `--text-1`) pointing at light or dark values so components never name a theme. Lesson: numbered scales for primitives, named roles for semantics; components use only roles.

### 4.5 Carbon and Polaris

Carbon's breakpoint table and 8 px unit (V, section 2.3). Polaris (U): tokens exported as CSS custom properties with a `--p-` prefix (`--p-space-400`, `--p-color-bg-surface`), a 4 px spacing base, "two thirds plus one third" page columns. Lesson: namespace the tokens so a panel cannot collide with them, and never let a panel define a token.

## 5. CSS architecture that resists regressions (V MDN unless tagged)

| Mechanism | What it gives us | Support | Source |
|---|---|---|---|
| Cascade layers `@layer` | Layer order beats specificity: `@layer reset, tokens, shell, components, panels, utilities;` means a panel can never out-rank the shell by writing a longer selector, and unlayered styles (which beat every layer) are forbidden by the linter. `!important` inverts the order, so it is also forbidden. Re-declaring a layer appends to it, so each panel file can open `@layer panels { ... }`. | Baseline widely available since March 2022 | https://developer.mozilla.org/en-US/docs/Web/CSS/@layer |
| Container queries | A panel sizes its content from its own box (`container-type: inline-size` on the panel root, `@container (width > 28rem)` inside), so a panel looks right in a 1-column phone, a 4-column kiosk or a sidebar without knowing the viewport. Units `cqi`, `cqb` for fluid type inside a panel. | Chrome and Edge 106, Firefox 110, Safari 16 (iOS 16); about 96 percent of users | https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_containment/Container_queries, https://caniuse.com/css-container-queries |
| Named grid areas and `grid-template-areas` | The shell declares the bands by name per breakpoint (`"status status" "stage side" "panels panels"`); panels are placed into the auto-flow area by the shell's placement rules, not by their own CSS. | Baseline | MDN grid (U, not re-read) |
| Subgrid | Cards in a row share header, body and footer tracks so headers line up even when titles wrap; gap is inherited. | Baseline since September 2023 | https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_grid_layout/Subgrid |
| Logical properties | `padding-inline`, `margin-block`, `inset`, `inline-size`, `border-start-start-radius`: one rule set for any writing mode and fewer overrides. | Baseline since about 2020 | https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_logical_properties_and_values |
| `@scope` | `@scope ([data-panel="gear"]) to ([data-panel])` styles a panel without reaching into nested panels; proximity beats source order. Newly available 2026, so the design system scopes by an attribute prefix in the selector today and adopts `@scope` once the tester browsers have it. | Baseline 2026 newly available | https://developer.mozilla.org/en-US/docs/Web/CSS/@scope |
| Custom properties as the API | A panel reads `var(--c-text)`; it never writes a token. A panel may expose its own knobs under its prefix (`--p-gear-cols`) and set them from a container query. | Baseline | |
| Content-Security-Policy | `style-src 'self'` stays. So: no `<style>` injection (gear.js today), no `style=""` attributes; CSSOM writes (`el.style.width = ...`) are allowed and are the one legal way to set a computed length (the inventory grid cells). | | V `internal/watch/http_view.go` |

Patterns that cause regressions and will be linted out: unlayered rules, `!important`, element selectors outside the reset layer (`h2 {}`, `button {}`), ids in selectors, selectors that start outside the panel's root attribute, fixed pixel widths on panel roots, `position: fixed` inside a panel (the shell owns overlays), media queries inside a panel file (container queries only), hard-coded colours.

## 6. Accessibility (V W3C understanding pages, MDN)

- 1.4.3 Contrast (Minimum): text 4.5:1; large text (18 pt, or 14 pt bold; 1 pt is 1.333 px so 24 px or 18.5 px bold) 3:1; exempt: incidental, logotypes, inactive components (https://www.w3.org/WAI/WCAG22/Understanding/contrast-minimum.html).
- 1.4.11 Non-text Contrast: 3:1 for the visual information that identifies a component and its states, and for graphical objects needed to understand content; 2.999:1 fails; focus indicators need 3:1 against the adjacent background; hue alone for state also fails 1.4.1 (https://www.w3.org/WAI/WCAG22/Understanding/non-text-contrast.html).
- 2.4.13 Focus Appearance (AAA, we adopt it because it is cheap): the indicator is at least the area of a 2 px perimeter and changes by 3:1 between focused and unfocused; "the simplest way to comply is a solid 2px outline" (https://www.w3.org/WAI/WCAG22/Understanding/focus-appearance.html). Today's `:focus-visible { outline: 2px solid var(--focus) }` already passes; keep it in the reset layer and forbid `outline: none` in panels.
- 2.4.11 Focus Not Obscured (AA, U): sticky headers and the bottom sheet must not cover the focused element; scroll-padding on the shell equal to the header height.
- 2.5.8 Target Size (AA, U): 24 by 24 px minimum; our rule stays 44 px for primary actions (docs/research/orders-ux-design.md).
- Motion: `prefers-reduced-motion: reduce`, replace scaling and panning with opacity changes; MDN's example swaps a `scale` pulse for an opacity `dissolve` (https://developer.mozilla.org/en-US/docs/Web/CSS/@media/prefers-reduced-motion). The current `* { transition: none !important }` is the blunt form; the design system puts motion tokens to zero instead.
- Live regions (S, several guides): `aria-live="polite"` for almost everything, regions created empty at load, `aria-atomic="true"` for numbers so "47 kills" is read whole, batch updates (debounce, `aria-busy`), never assertive for routine data; no guide gives a throttle, so ours: a live region may change at most once per 3 s and only on a threshold crossing (life under 25 percent, a death, an unlock), never on every tick.
- Keyboard: every tab, fold, sheet and map node reachable; Escape closes any sheet (app.js already does this); a redraw keeps focus (app.js `renderOrders` keeps the focused button index; make that the shell's job for every panel).

## 7. Responsive strategy

Breakpoints are the shell's business only (media queries) and panels never see them (container queries). Proposed (reasoned from Carbon's table and our content: the video is 4:3 and 440 px wide at its smallest useful size, the map 320 px, a stat tile 9 rem):

| Name | Min width | Columns | Who |
|---|---|---|---|
| phone | 0 | 1 (tabs for level 2) | the user on the couch |
| tablet | 640 px | 2 | iPad, phone landscape |
| laptop | 1024 px | 3 (stage 2 + side 1) | 13 inch laptop, the desktop-app window |
| wide | 1440 px | 4 | monitor |
| kiosk | 1920 px, `?mode=kiosk` or `?mode=overlay` | 4, no chrome, fade-on-idle | stream overlay, TV |

Rules: fluid type with `clamp()`; no fixed widths on roots; the video keeps `aspect-ratio: 4 / 3` and takes the stage column; the map is a side panel at laptop and wider and stacks under the video below that; the header is one line (status, why) at every width and the rest moves into the side rail or the tab strip.

## 8. Theming

The existing toggle (auto, light, dark; `d2theme` in localStorage; `data-theme` on `<html>`) stays as the mechanism (V app.js lines 493 to 504). What changes: the token file defines light and dark once at the root and the pages that have their own palettes (inventory, pacts, hall) load the shared tokens. `color-scheme: light dark` plus `data-theme` is the shadcn pattern with the Material swap. Decision for the user (section 9 of the design doc): the user's 2026-11-05 look-and-feel decision chose the original dashboard's light-first palette (V docs/research/watch-screen-design.md section 7); this research argues for a dark D2 palette as the default with the light theme kept as a real, tested alternative, not removed.

## 9. Motion and live updates

- CLS (V, https://web.dev/articles/cls): good is 0.1 or less at the 75th percentile; causes are media without dimensions, late content inserted above the reading position, fonts; fixes are reserved space, loading indicators in the reserved space, `transform` and `opacity` instead of layout properties, and `prefers-reduced-motion`.
- Skeletons (V, https://www.nngroup.com/articles/skeleton-screens/): under 1 s nothing; 1 to 10 s a skeleton for a page or a spinner for one module; over 10 s a progress bar; a skeleton must mirror the final layout ("frame-display" skeletons that show only the chrome are not recommended). Our panels answer within 4 to 5 s of load, so each panel renders its skeleton at its final size immediately and swaps content in place.
- Number tickers (U, common practice): tabular figures (`font-variant-numeric: tabular-nums`), a fixed character width per tile, animate only the colour (a 600 ms highlight that fades) and never the digits for values that change every tick; animate digits only for rare celebratory values (an unlock price paid). Direction colour: up green, down red, neutral otherwise; never colour alone (an arrow glyph too).
- Change highlights: one `data-changed` attribute set by the shell for 600 ms; the component layer styles it; reduced motion drops the fade and keeps the colour.
- Streams versus polls: the page already has `/api/events` (SSE) plus 5 s polls; the shell should own one fetch scheduler so that eight panels do not each run their own timer (today: six intervals and timeouts at 250 ms, 500 ms, 4 s, 5 s, 5 s, 5 s).

## 10. Information scent: "what is the hero doing and why"

- The whyline is the answer and it is the right size: one sentence, present tense, the hero's own reason (V docs/whyline.md). The research on spectators (section 3.3) says it is also the overlay's headline.
- It needs three companions in the same band: the state word (fighting, walking, in town, dead, idle: from `/api/uptime` `label`), the place (area name from `/api/senses` or orders map), and the vitals (life, mana percent from the status poll; gear condition as one word: fine, worn, broken). Those four form the "hero status" tile; everything else is explanation.
- The second level of scent is the Hero now card (order, progress, route crumbs) and the latest Thought; the third is the Chronicle and the Journal with "why did you do that?". That is already the shape of the data; the design system makes it the shape of the page.

## 11. Dead ends and gaps

- Material 3's own site (m3.material.io) renders by script and returned only titles; the material-web Markdown was used instead.
- Interface In Game pages list screens but their text has no layout detail; game HUD descriptions above are U.
- The InnoGames number-format post reset the connection; its snippet was used.
- Polaris redirects to shopify.dev (not fetched); Datadog's best-practices page is 404 at the guessed URL; both are U.
- No fetch of the headless Chrome screenshot flags beyond `--headless` (the Chrome doc covers only the mode); the flags in the regression plan are from memory (U) and must be checked on the worker host when the script is built.
