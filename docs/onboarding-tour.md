# Onboarding tour, Help sheet and Now line (TASK-48)

> status: BUILT offline (card onboardtour, 2026-11-27). Node stub-DOM tests only; not looked at in a browser, not verified against a live bot. The welcome card and first-30-minutes list stay `docs/onboarding.md` (TASK-247). Tags: V = verified by a named test, U = unverified.

## What it does

- Tour tips (`web/onboarding.js`, words in `web/onboarding.json`): five dismissible cards tied to real state, one at a time. First launch (always due), first time in town (`/api/hero` area is in `townAreas`), first Vault item (`/api/vault?limit=1` total above 0), first quest handed in (`/api/questlog` has a row in state `done`), first death (`/api/uptime` deaths above 0). Buttons: Got it, Back (this visit), Skip tour. V: `TestOnboardingTourDOM`.
- Resume: what was seen is kept in `localStorage` key `d2tour` (`{seen:{id:1}, off:bool}`); a reload shows the next unseen step whose event is still true. A throwing, missing or corrupt store falls back to memory for the visit (V, same test); the Help sheet then says tips may repeat.
- Now line: one line under the page from `GET /api/explain` (last decision code mapped by `doing` in the json, unknown codes use the default sentence; the first standing refusal's need after "Waiting"). Hidden while explain answers `enabled: false`.
- Help sheet: header `Tour` button opens the shared panel sheet (`dSheet`) with three lines, Replay the tour, and a button that fires `d2:nav` id `help` so the shell opens the Help drawer (welcome card and first-30-minutes list).
- Phone: the tip is a bottom card above the bottom bar (`--d-nav-h`), buttons at the 44 px hit size; desktop: a 24rem card bottom right. Styles are in `@layer components` (`web/onboarding.css`).
- Buttons only read; no API call is made by a button, so the shared busy helper (`toast.js`) has nothing to wrap here.

## Not done / not proven

- U: the town area names, that `done` means handed in, and the decision codes of `doing` (only a few are mapped; the rest read "Playing on").
- U: a real browser at 390 and 1280 px. Checklist: open `/watch?layout=cockpit` with a clean profile; the first tip shows above the bottom bar at 390 wide with no horizontal scroll; Got it moves on; reload keeps seen steps; private window still works; Tour button opens the sheet, Replay restarts, the guide button opens the Help drawer; Escape closes the sheet; the Now line appears once explain answers.
- No sample thought page injection (the design doc's step 2); the tour explains in words only.
