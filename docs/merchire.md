# merchire: drill D8.h, paid hire at the hirer

> status: current. Tags: V verified, S secondary, U unverified. Offline only: nothing here ran on the game (not verified live). Design: docs/merclive.md.

## What was built
- `internal/drill/merchire.go`: `HireLab` (optional lab interface, `PaidHire`), `HireInfo{Quote, Row, WantCode}`, `ErrHireCantAfford`, `D8h` (registered inside `D8`). Tests: `internal/drill/merchire_test.go` (fake lab, no literals of class codes).
- `cmd/d2drill/hire_lab.go`: `lab.PaidHire` over d2dctl: approach Kashya (`ApproachKashya`), open the menu, find the "hire" line, read its price with `d2dmerc.ParseQuote`, stop with `ErrHireCantAfford` below it, else pick the line.
- D8.h: Blocked if a merc is owned; SETUP `SetGold 5000` (logged cheat `debug stat 14`, ledgered before the action); Pass = quote > 0, dump merc present, alive, code equals `WantCode`, gold fell by exactly the quote.
- D8.h2: gold 1; Pass = `ErrHireCantAfford`, quote above the purse, no merc, gold unchanged.

## Live run (later, d2-live, hidden slot, never a Gate slot)
Needs the lab fork with `debug stat` and `debug merc` (V, docs/forkcheats.md). Character without a merc (D8.c grants one; use a second fresh character or run D8.h first).
```
labrun <worker> -- bash docs/probes/merclive/2026-11-04-MC-run.sh LABEL D8.h
labrun <worker> -- bash docs/probes/merclive/2026-11-04-MC-run.sh LABEL D8.h2
```
The recipe script takes the drill id as `-only`; it uses hidden slot 3 (V, merclive). Run capacity check first. Pass the same two ids in one run only after a fresh character (D8.h leaves a merc, D8.h2 does not need one but D8.h blocks after it).

## Risks
- U: Kashya's menu "hire" line may carry no price; the quote may only show in the hire window, which d2d cannot read (`d2dmerc.ErrNoOffers`, V). Then D8.h fails with "no price in the hire line": the right result is a Go window reader or a fork verb, not a guessed price.
- U: picking the "hire" line may open the window rather than hire at once (Greiz needed a row click, V town.md); then gold does not fall and the drill fails with "gold fell by 0".
- S: expected code RG for Act 1 (d2.MercClasses[0]); 0A and IW codes stay U (hirefix).
- The hire replaces nothing: a merc owned already makes the drill Blocked, never a destructive hire.
