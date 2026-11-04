# DW.m4 `open: inventory` triage (card dwm4triage)

Offline read of the drill harness, the bot and the logs of DRILLS18, 19, 21, 23. Nothing run live. V = read in the
repo, U = unverified.

## Finding

**Neither a pure harness artifact nor a proven bot bug: undecided, with a narrower cause than before.** No code change.

1. The drill never opens the inventory. V: `fDen` (internal/drill/dfield.go:487-540), `lab.DenPhase`, `Survive`,
   `Invincible`, `ToArea` (cmd/d2drill/field_lab.go:460, 781, 634; game_lab.go:24) send no `i` key and no panel
   click. The gate only reads the UI (cmd/d2drill/live.go:93) after the drill, and `Recover` sends one Escape
   (live.go:113-134).
2. The only product code in that flow that presses `i` is `Fighter.portalAttempt` (internal/combat/fighter.go:508-516):
   the town-portal emergency. It closes with `defer f.e.In.CloseAll()` and ignores the error. V.
3. The drill logs show that path ran in the Den: DRILLS18 m41 line 1279 `opened a town portal, go to town`, DRILLS19
   m41 lines 70-71 `town portal cast` and later `no town portal scroll or tome`; lines `left the game to survive`
   (`ErrChicken`, internal/combat/types.go:52) precede it. V (log read).
4. DRILLS18 m41 line 1283 (and 48 lines in all): `d2dinput: menus still open after 4 rounds`, i.e. the bot's own
   `d2dinput.Controller.CloseAll` (internal/d2dinput/controller.go:367-411) pressed Escape four times and the
   inventory flag stayed set. The harness `Recover` (one Escape) then failed the same way, hence
   `recovery failed, group stopped`. V (log); DRILLS19, 21, 23 logs are filtered and show no such line (U whether
   the same).
5. Why Escape does not clear the flag is **U**. Candidates: after the chicken/portal exit the game has left the
   session and the UI frame is stale (a harness-state artifact); or a panel/dialog state where Escape is swallowed.
   The real bot follows a portal with the town chores, which call `Hand.CloseAll` (internal/town/hand/hand.go:123)
   and the sentinel panel rule (internal/sentinel/rules.go:451), so a left-over panel is cleaned later (S, not
   exercised here).

So the panel is opened by the bot's emergency path in a drill whose character is in a fight it was not built for
(the drill is a mouth-bounce test; the survival path is a side effect). The dirty UI is the tail of that emergency,
not of the mouth logic. DW.m4's mouth verdict (0 bounces) is unaffected.

## Why no fix and no fake test

A fake test needs the failure to reproduce with `input.Fake`/`d2dinput` fakes. The failing step (Escape not clearing
the flag) depends on live game state, which a fake would only assert by construction. The one defect visible in code,
`portalAttempt` ignoring a `CloseAll` error (and `input.Game.CloseAll` returning at once when the context is
cancelled, internal/input/controller.go:356-359), is not shown to cause the log. Changing it without that proof is
out of scope.

## Live recheck recipe (not run)

1. Run DW.m4 alone as in DRILLS23, with `-v`, keep the unfiltered log.
2. Grep the log for `town portal cast`, `left the game`, `menus still open`, `ui flags`. If a portal cast precedes
   the dirty UI and `menus still open` appears: bot gap (CloseAll after the cast); capture `dump ui` right after
   the failed CloseAll and after one more Escape.
3. If there is no cast: something else opens it; add a `dump ui` read after every phase of `fDen` to find the phase.
4. If `dump ui` shows the character is not in game (chicken exit): harness artifact; make the gate skip or wait for
   re-entry instead of failing the drill.
5. Cheapest separation: give the drill a bigger life pool (`debug stat 7`/`6` repeated each cycle) so the portal path
   never fires, then see whether `open: inventory` disappears.
