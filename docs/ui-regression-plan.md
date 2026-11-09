# Watch UI regression tests and migration plan (TASK-228, uidesign)

> status: proposal (2026-11-09), nothing here is built. Design: [ui-design-system.md](ui-design-system.md). Research: [research/ui-design.md](research/ui-design.md). Tags: V verified by reading the code named, U not run (every tool in this doc is unbuilt; flags marked U were not checked against a binary).

## 1. Three checks, one gate

| Check | Catches | Runs where | Blocks merge? |
|---|---|---|---|
| `tools/uilint` (Go, stdlib) | a malformed panel manifest, CSS outside a layer or outside a panel's scope, colour literals, inline styles, media queries in panels, fixed positioning, `!important`, globals in panel JS | `tools/ci/check.sh` (hygiene stage), on every machine | yes |
| golden DOM (Go test, `internal/watch`) | a changed data-attribute API, a panel rendering outside its root, a missing state (loading, empty, error, locked) | `go test ./internal/watch -run Golden` | yes |
| screenshots per breakpoint and theme (shell script + headless Chromium under Xvfb) | layout shift, overflow, a panel changing pixels outside its own box, contrast of the token pairs, the phone frame | the worker host, `tools/uishots.sh`, nightly and on request before a UI merge | yes for a UI PR (`owns: internal/watch/web/**`), advisory otherwise |

Rationale: the first two need no browser and run everywhere in milliseconds; the third needs a browser and a display and so runs on the worker host like the live checks (`tools/live.sh` pattern), never on the small box.

## 2. `tools/uilint`: the panel-contract linter

Package `tools/uilint` (`go run ./tools/uilint [-fix=false] [dir]`, exit 1 on findings; one line per finding `file:line: rule: message`). Stdlib only: `encoding/json`, `regexp`, `bufio`, `path/filepath`. No CSS parser dependency: a small tokenizer that tracks `{}` depth and at-rule names is enough for the rules below (CSS in this repo is hand-written and small; the tokenizer fails loudly on an unbalanced brace).

### 2.1 Manifest rules (`panel.json`)

| Rule | Check |
|---|---|
| `manifest-schema` | `schema == "watch.panel/1"`; unknown keys refused; required keys present (`id`, `title`, `band`, `size`, `priority`, `tab`, `endpoint` or `poll == 0` with `endpoint == ""` only for `status`-band shell panels, `owner`, `doc`) |
| `manifest-id` | `id` matches `^[a-z][a-z0-9-]*$`, equals the directory name, unique |
| `manifest-size` | `size`, `minSize`, `maxSize` in `{s, m, l}` and ordered |
| `manifest-band` | `band` in `{status, stage, grid}`; `status` and `stage` ids come from a fixed list in `shell.json` |
| `manifest-priority` | integer 0 to 999, unique within the band |
| `manifest-tab` | `tab` in the closed set |
| `manifest-endpoint` | starts with `/api/`; the path is registered (the lint reads the list exported by `watch.ViewRoutes()`, a Go test asserts it) |
| `manifest-unlock` | empty or an id in `data/unlocks/ladder.json` |
| `manifest-doc` | `doc` exists in the tree |
| `manifest-orphan` | every `panels/<id>/` has a manifest; every manifest has `panel.css` and `panel.js` |

### 2.2 CSS rules

| Rule | Check | Applies to |
|---|---|---|
| `css-layered` | every rule is inside `@layer <name>`; the layer order statement appears once, in `tokens.css`, and lists exactly `reset, tokens, shell, components, panels, utilities` | all |
| `css-layer-file` | `tokens.css` uses only `@layer tokens`, `reset.css` only `reset`, `shell.css` only `shell`, `components.css` only `components`, `panels/*/panel.css` only `panels`, `utilities.css` only `utilities` | all |
| `css-no-important` | no `!important` | all |
| `css-no-id` | no `#id` selectors | all but none (ids are never styled) |
| `css-element-only-in-reset` | a selector that starts with a bare element name (`h2`, `button`, `ul li`) only in `reset.css` | all |
| `css-panel-scope` | in `panels/<id>/panel.css`, every selector of every rule (including inside `@container`) starts with `[data-panel="<id>"]`; `:root`, `*`, `html`, `body`, `.d-*` as the first compound are refused (a panel may style `.d-*` only as a descendant of its root) | panels |
| `css-no-media-in-panel` | no `@media` in `panels/**` or `components.css` (container queries only) | panels, components |
| `css-no-fixed-in-panel` | no `position: fixed`, `position: sticky`, `z-index` in `panels/**` | panels |
| `css-no-root-size` | in a panel file, the rule whose selector is exactly the root attribute has no `width`, `height`, `min-width`, `max-width`, `inline-size`, `block-size`, `grid-column`, `grid-row`, `order`, `margin` | panels |
| `css-no-colour-literal` | outside `tokens.css`: no `#hex`, `rgb(`, `rgba(`, `hsl(`, `oklch(`, `color-mix(` or CSS named colours as values; `transparent` and `currentColor` allowed | all but tokens |
| `css-token-exists` | every `var(--d-...)` names a token defined in `tokens.css`; every `var(--p-<id>-...)` in a panel file is defined in that panel file | all |
| `css-no-vw-in-panel` | no `vw`, `vh`, `dvh`, `svh` units in panels (use `cqi`, `cqb`, rem) | panels |

### 2.3 HTML and template rules

| Rule | Check |
|---|---|
| `html-no-style-attr` | no `style="` in `web/**/*.html` and in the Go template strings of `internal/watch/*.go` (the linter scans Go files for backtick strings containing `<` and applies the HTML rules to them) |
| `html-no-style-element` | no `<style>` outside `<head>`; in production no `<style>` at all (mockups under `docs/mockups/` are exempt by path) |
| `html-no-inline-script` | no `<script>` without `src`, no `on*=` attributes |
| `html-panel-root` | the shell HTML has no `data-panel` elements of its own (panels are created by `shell.js`), except the fixed `status` and `stage` ids |

### 2.4 JS rules (regex, deliberately simple)

| Rule | Check |
|---|---|
| `js-no-globals` | panel files are ES modules (`export function mount`), no `window.<name> =`, no top-level `var` outside the module scope |
| `js-no-timer` | no `setInterval`, `setTimeout` (except the `--allow-timeout` list for debounce), `fetch(`, `EventSource(` in `panels/**` |
| `js-no-query-outside-root` | no `document.querySelector`, `document.getElementById`, `document.body` in `panels/**` |
| `js-no-inline-style` | `.style.` writes only from the whitelist (`gridArea`, `gridColumn`, `gridRow`, `transform`, `setProperty('--p-`); `.style.cssText`, `.setAttribute('style'`, `innerHTML`, `insertAdjacentHTML`, `document.createElement('style')` refused everywhere |
| `js-text-only` | `innerHTML` and `outerHTML` refused everywhere (keeps today's `textContent` rule) |

### 2.5 Tests for the linter

`tools/uilint/uilint_test.go` with a `testdata/` tree: one good panel, and one fixture per rule that must fail with exactly that rule name (table-driven). The linter runs on itself in CI (`tools/ci/check.sh` adds `go run ./tools/uilint internal/watch/web` to the hygiene stage so it runs on the small box too; it is a few hundred files and takes well under a second).

## 3. Golden DOM and layout check (Go, `internal/watch`)

Today `golden_test.go` and `internal/watch/testdata/*.json` hold JSON goldens for API replies (V). The new check renders the DOM without a browser:

- `internal/watch/web/panels/<id>/fixtures/{ready,empty,locked,error}.json`: the API reply each state is rendered from (the `ready` fixture is the same file the API golden uses where one exists, so the two never drift).
- A tiny DOM in Go for tests only (`internal/watch/domtest`): `shell.js` and the panel modules are too small to warrant a JS engine; instead each panel's `panel.js` is mirrored by a Go `Render(state, fixture) *Node` in `internal/watch/panels/<id>.go`? No: that doubles every panel. Instead the golden is produced by the real JS in the browser run (section 4) with `--dump-dom`, normalised (attributes sorted, whitespace collapsed, `data-changed` and timestamps stripped), and stored as `panels/<id>/golden/<state>.html`. The Go test then only diffs the stored golden against the committed one and checks the contract on it with `encoding/xml`-free string rules: every element inside the panel root, every `data-*` attribute from the API set, exactly one `aria-live` at most, the `data-state` present, the headline `h2` with the manifest title. So the DOM golden is refreshed by the browser job and verified by `go test` everywhere.
- Layout golden: the browser job also writes `panels/<id>/golden/<breakpoint>.boxes.json` (`getBoundingClientRect` of the panel root and of every `[data-key]` child, rounded to 2 px). `go test` asserts: no box exceeds its root; the root's width equals the column span the shell computed for that breakpoint within 2 px; no two panel roots overlap; the status band's height is the same at `ready` and `loading`.
- Update: `go test ./internal/watch -run Golden -update` keeps working for the JSON goldens; `tools/uishots.sh --update` refreshes the DOM and box goldens (it needs the browser).

## 4. Screenshot suite on the worker host (`tools/uishots.sh`)

### 4.1 What it does

1. Builds `cmd/d2dview` (the simulated Sorceress and Paladin runners already serve the watch pages offline, V docs/watch-screen.md) and starts it on a free loopback port with `-watch -watch-class <class>` and a fixture mode flag to be added (`-watch-fixtures <dir>`: every viewer route answers from the panel fixtures, so the pictures are deterministic; new Go, small).
2. Starts `Xvfb :2xx -screen 0 1920x1080x24` (private display, never the user's desktop; same rule as the game instances) and a headless Chromium against it.
3. For each page (`watch`, `watch?mode=kiosk`, `watch?mode=overlay`, `inventory`, `policy`, `return`, `pacts`, `hall`, `tokens`), each breakpoint (390x844, 768x1024, 1280x800, 1536x960, 1920x1080) and each theme (dark, light) and both classes: one PNG, one `--dump-dom`, one boxes JSON (a 30-line page script added to the page only under the fixture flag writes `window.__boxes`, read through `--dump-dom`).
4. Compares the PNGs against `internal/watch/web/golden/<page>-<bp>-<theme>-<class>.png` with a per-panel mask: the diff tool (Go, `image/png`, stdlib) reads the boxes JSON, computes the changed-pixel ratio inside each panel box and outside all boxes, and fails when a panel the PR does not own changed more than 0.5 percent of its pixels or the outside-area changed at all. Ownership comes from the task's `owns:`.
5. Contrast check: the same Go tool samples every `d-stat .val`, `d-meter .num`, `d-badge`, `d-list .txt` box (from the boxes JSON) for its foreground and background modal colours and computes WCAG contrast; fails under 4.5:1 (3:1 for badges on solid tones and for the meter fill against its track).
6. Cleanup through `tools/agentrun.sh` so `agentreap` can stop it; hard limit 10 minutes; one run at a time by a flock.

### 4.2 Commands (U: flags from memory, to be checked on the worker host when the script is built)

```sh
Xvfb :230 -screen 0 1920x1080x24 -nolisten tcp &
DISPLAY=:230 chromium --headless=new --no-sandbox --disable-gpu --hide-scrollbars \
  --force-device-scale-factor=1 --window-size=1280,800 --virtual-time-budget=6000 \
  --run-all-compositor-stages-before-draw --screenshot=/tmp/shots/watch-1280-dark.png \
  "http://127.0.0.1:PORT/watch?t=TOKEN&theme=dark&fixtures=1"
DISPLAY=:230 chromium --headless=new --no-sandbox --dump-dom "http://127.0.0.1:PORT/watch?..." > watch-1280-dark.html
```

Determinism: fixed fonts (install the same system fonts on the worker host and name them first in `--d-font-*` stacks: DejaVu Sans, DejaVu Serif, DejaVu Sans Mono are present on the usual Linux images), `--virtual-time-budget` so timers settle, `prefers-reduced-motion` forced by `--force-prefers-reduced-motion` so no animation is mid-frame, `--disable-lcd-text` for stable antialiasing, and the fixture mode so no clock or poll varies. If Chromium is absent on the worker host, Firefox headless (`firefox --headless --screenshot --window-size=W,H URL`) is the fallback; the plan prefers Chromium for `--dump-dom` and `--virtual-time-budget`. Either way nothing is installed by the script; it reports what is missing. (docs/watch.md records that the d2watch page was once rendered in headless Firefox on the worker host, V, so at least Firefox existed there in October.)

### 4.3 Where it runs and how it gates

- Nightly on the worker host and on `labrun <worker> -- tools/uishots.sh` (the worker host) before queueing a UI branch (`MERGEQ` preflight hook for branches whose `owns:` touches `internal/watch/web`); a UI PR whose diff touches goldens must include the updated PNGs (under 1 MB each by the hygiene rule; 1280 and 1920 shots are stored as 8-bit palette PNGs, well under that).
- Separate tools module: no. Stdlib Go for the diff and contrast tool (`tools/uishots/diff.go`), shell for the orchestration; nothing new in `go.mod`.

## 5. Migration plan: ordered small PRs

Each PR is one card (`component: watch`, `kind: feature` unless said), owns only the paths named, lands behind the gate above, and leaves the page working at every step (the old `app.css` and the new layers coexist until PR 8). Sizes are the card `--size` (S about a day, M two to three, L a week); recommended priority in brackets.

| # | PR (card) | Owns | Acceptance criteria | Size |
|---|---|---|---|---|
| 1 | `uifixcsp`: hotfix the gear strip (kind `bug`) | `internal/watch/web/gear.js`, `app.css` | the strip's rules move into `app.css`; no `<style>` injection; a test asserts no `createElement('style')` in any web asset; screenshot shows spaced chips | S [high] |
| 2 | `uilint`: the linter and its tests, run in `check.sh` | `tools/uilint/`, `tools/ci/check.sh` | all rules of section 2 with a failing fixture each; today's tree passes with a documented allow-list (`tools/uilint/allow.txt`) of existing violations that shrinks to empty by PR 8 | M [high] |
| 3 | `uitokens`: `tokens.css`, `reset.css`, layer order; `app.css` wrapped in `@layer shell` with values pointing at tokens; inventory, pacts and hall load the tokens | `internal/watch/web/tokens.css`, `reset.css`, `app.css`, `invui.go`, `pactui.go`, `hall.go` (style blocks only) | visual parity screenshots within 1 percent; the theme toggle changes all five pages; contrast check passes on the token sheet | M [high] |
| 4 | `uishell`: `shell.css`, `shell.js` (placement, scheduler, focus keeping, one sheet, one toast stack, one banner), `panels.json` served from Go, the manifest parser and its Go test | `internal/watch/web/shell.*`, `internal/watch/panels.go`, `panels_test.go` | the shell renders the bands with the existing cards placed from a generated manifest (sizes and priorities of the migration table); phone rail works; a Go test validates every manifest | L [high] |
| 5 | `uicomponents`: `components.css`, `components.js` (12 components) + `tokens.html` page | `internal/watch/web/components.*` | every component has the states of the catalogue; the token sheet page is in the screenshot suite; golden DOM per component | L [high] |
| 6 | `uishots`: the screenshot suite and the diff and contrast tool, fixture mode in d2dview | `tools/uishots.sh`, `tools/uishots/`, `cmd/d2dview` (fixture flag), `internal/watch/web/golden/` | one green run on the worker host for all pages, breakpoints, themes, classes; the first golden set committed; run time under 5 minutes | M [high] |
| 7a | `uipanel-hero`: hero status, whyline, alerts count, video, map as panel modules (status and stage bands) | `internal/watch/web/panels/{hero-status,whyline,alerts-count,video,map}/` | each passes uilint; the status band is one line at laptop and two on a phone; the map canvas sizes by `cqi`; old header strips removed | M [high] |
| 7b | `uipanel-orders`: hero-now, orders-map, queue, thoughts | `panels/{hero-now,orders-map,queue,thoughts}/` | the order sheet uses `ctx.sheet`; focus keeping on redraw is the shell's; golden DOM for ready, empty | M |
| 7c | `uipanel-condition`: gear (condition card), tally (with spend), risk (new) | `panels/{gear,tally,risk}/` | gear rows wrap into two columns above 24 rem by container query; the broken slot is first; `/api/risk` is on the page for the first time behind its unlock | M |
| 7d | `uipanel-story`: chronicle, journal, voice (with diary) | `panels/{chronicle,journal,voice}/` | lists scroll inside `data-max`; serif voice rows; "why?" disclosure | S |
| 7e | `uipanel-discover`: quests, cards, bestiary, atlas, locked list, and the Locked component wired to `/api/unlocks` for every panel with an `unlock` | `panels/{quests,cards,bestiary,atlas,locked}/` | a locked panel renders as a silhouette tile in place; buying through the tile works against the sim; the reveal plays once | M |
| 7f | `uipanel-meta`: rules, alerts (list), update banner through the shell banner | `panels/{rules,alerts}/`, `web/update.js` removal | one toast stack, one banner; `update.js` deleted | S |
| 8 | `uicleanup`: delete `app.css`, `unlockui.js`, `gear.js`, `alerts.js`, `senses*.js` globals; allow-list empty; `watch.html` is the shell only | the files named | `tools/uilint` passes with an empty allow-list; `TestWatchAssetsHaveNoThirdPartyURL` and the no-inline tests extended to `panels/**` | S |
| 9 | `uipage-inventory`: inventory on the shell with `d-cells`, slots with durability meters, repair card shared with the gear panel | `internal/watch/invui.go`, `web/pages/inventory.html`, `web/inventory.js` | cells placed by `gridArea` only; marks and pins as badges; theme toggle works; screenshots at five breakpoints | M |
| 10 | `uipage-policy`: policy sheet on the shell (Button group options, Locked options, preview as a Note with Save and Cancel) | `policysheet.go` template, `web/pages/policy.html` | goldens unchanged for `/api/policy`; the page passes uilint; the preview card no longer needs `return.css` | S |
| 11 | `uipage-return`: return screen on the shell (Stat tile row, Cards for expeditions and missed questions) | `returnscreen.go` template, `web/pages/return.html`, delete `return.css` | single-column `l` layout; `return.css` deleted | S |
| 12 | `uipage-pacts-hall`: pacts and hall pages on the shell | `pactui.go`, `hall.go` | shared tokens, Table component; theme toggle works | S |
| 13 | `uioverlay`: `?mode=kiosk` and `?mode=overlay` | `shell.css`, `shell.js` | overlay is the status band alone on a transparent background at 1920 by 160; kiosk fades to the band after 20 s idle; both in the screenshot suite | S |

Order: 1 at once (it is a bug the user sees); 2 and 3 in parallel (different owns); 4 then 5 (the shell needs no components to place the old cards; the components need the layer order); 6 as soon as 4 is in so every later PR lands with pictures; 7a to 7f in parallel after 5 and 6 (disjoint directories, the point of the contract); 8 after all of 7; 9 to 13 in parallel after 8. Total about 10 agent-days of M and L work plus the S items; the user sees a fixed gear strip on day one and the new status band after PR 7a.

What stays out until the user decides (section 13 of the design doc): the default theme flips to dark in PR 3 only if decision 1 is yes (otherwise PR 3 ships dark as the alternate); Exocet is never added without decision 4; the overlay (PR 13) is built as a shell mode unless decision 3 says separate page.

## 6. Acceptance of the whole migration

- `tools/uilint` passes with an empty allow-list.
- Every page, breakpoint, theme and class has a committed golden and the nightly run is green.
- Adding a test panel (`panels/zz-probe/`, a manifest and two files, no other change) places it in the grid at its priority with the right size on all five breakpoints and changes no pixel outside its box: that is the drill that proves "adding a feature never touches existing CSS or markup", and it becomes a permanent test in the screenshot suite (the probe panel is created and removed by the script).
- CLS measured by the browser job (a 20-line page script reading `PerformanceObserver('layout-shift')` under the fixture flag) stays under 0.05 on every page at every breakpoint from load to the fourth poll.
