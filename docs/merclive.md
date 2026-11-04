# merclive: P18 merc claim, hire and revive live on d2d (2026-11-04)

> status: current. Tags: V = ran live this card, S = secondary, U = unverified. d2d lab fork binary d2d.lab-cheats3 on the worker host, hidden slot 3 (not a soak slot), throwaway character, not game.exe. Evidence: docs/probes/merc/2026-11-04-MC-*.{txt,json}; run script 2026-11-04-MC-run.sh (`labrun orion -- bash docs/probes/merc/2026-11-04-MC-run.sh LABEL DRILLID`). Capacity check said OK.

| step | drill | observed | verdict vs hirefix.md | tag |
|---|---|---|---|---|
| (a) free Rogue claim | D8.c | setup `debug quest set 2 0x0002` (Blood Raven reward pending) then the bot's ClaimRogue: merc present in the dump afterwards, code `RG` at 14.3,21.9, dialog closed. The Den flag was not set (the harness sets only Blood Raven) | PASS: merc != none | V |
| (b) paid hire at the hirer | none | no drill or harness verb hires for gold (hirefix deferred B3, Hirer over `menu`) | BLOCKED | U |
| (c) revive of a killed merc with gold | D8.a | setup: `debug merc hire 0`, `debug merc kill` (dump `present:false, dead:true`, the B2 shape), `debug stat 14 3000` gold. The bot's Reviver via Kashya: merc back (present, code RG, life 40) in the dump, gold 3000 -> 3000, reported price 0 | PASS for the flow and the dead-state shape; price is 0 for a cheat-hired merc, so the quote was not exercised (hirefix B4 `ErrCantAfford` path not reached) | V |

Not shown: the 0A and IW hireling codes (only RG appeared), a nonzero revive price (the cheat merc is level 1; a leveled merc may charge), the revive quote text. `writes after start: 0` (no measured run; the three setup writes above precede the action, each ledgered with a reason, actor drills:D8.a / D8.c).

## Design for the paid hire drill (not built, BLOCKED)
Drill D8.h in internal/drill/dgame.go: setup gold via the logged stat cheat to 5000, walk to the Act 1 hirer (Kashya, hire menu, internal/d2dmerc Hirer), hire the first row, assert merc present with a class code and gold down by the quote read from the window. Needs a GameLab method for Hirer and the quote read; Go code, so out of this card.
