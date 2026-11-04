# hirefixlive: PaidHire follows the real menu flow (offline, 2026-11-04)

> status: current. Tags: V = verified, S = secondary, U = unverified. Not run live: this change is offline only.

## Why
D8.h and D8.h2 failed live with `no hire line in Kashya's menu` (V, docs/livedrills2.md on branch agent/livedrills2): the first menu has exactly three lines, `Kashya` (header), `talk`, `cancel`. The old `PaidHire` looked for a `hire` line in that first menu only.

## Change (cmd/d2drill/hire_lab.go)
- `PaidHire` now opens the menu and hands it to `hireFlow`, a pure function over a small `menuClient` (Menu, PickMenuLine) so a scripted fake can drive it.
- `hireFlow`: a `hire` line (or a priced row containing `cost`) in the first menu is used at once; otherwise it clicks `talk`, then polls the menu (8 x 400 ms) for a hire line or priced row. Missing lines end in an error that names the last menu seen. U: that the hire window or hire line sits behind `talk`; the bot's own Reviver (internal/d2dmerc/act.go) only ever read one menu, so no earlier live path shows the hire step for Kashya.
- The price comes from `d2dmerc.ParseQuote` of the row text. No price: cancel and error, never a guess. Purse below the quote: cancel and `ErrHireCantAfford`. Otherwise the row is clicked by its own text.
- The class literal `RG` is removed: `WantCode` stays empty, so the drill only checks "living merc with a code in the dump" and gold down by the quote. (The code to class table in internal/d2dsrc is private and not mine to export; a later card can set WantCode from it.)

## Tests (cmd/d2drill/hire_lab_test.go, scripted fake)
Covered: talk then priced row (picks `talk`, then the row text), a direct hire line, cannot afford (cancel, `ErrHireCantAfford`, quote kept), no hire line after talk (error), no talk line (error). Not covered: real d2d row text, timing of the window opening (U).

## Live command (later d2-live run, same recipe as docs/probes/livedrills2/2026-11-04-run.sh on branch agent/livedrills2)
```
labrun <worker> -- bash docs/probes/livedrills2/2026-11-04-run.sh hirefix D8 D8.h,D8.h2 1
```
(group `D8`, `-only D8.h,D8.h2`, repeat 1; throwaway character without a merc; slot 3.) If it still fails, the error now prints the menu after `talk`: record it and, if the window is open but has no menu items, a screenshot is needed (d2d has no hire window reader, U).
