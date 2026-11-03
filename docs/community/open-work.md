# Open to contributors

> status: current, APPROVED by the maintainer 2026-11-01 (card communitywork). The source list for `tools/contrib/board.py`, which renders [work-board.md](work-board.md) and the GitHub issue bodies. How to take an item: [../contributing/agent-work.md](../contributing/agent-work.md).

Only items that need nothing but the game binary of your own legitimate Diablo II LoD 1.14d install and public sources (this repo, upstream `psi29a/d2decomped`) are listed. Live drills, soaks and anything on the maintainer's private lab are NOT open (see the guide). The maintainer keeps this file in step with `docs/decomp-queue.md` and `docs/BACKLOG.md`; where they disagree those two win.

Format (read by the board script): one `## ID: title` section per item, then `- key: value` lines. Keys: `size` (S, M, L), `status` (`ready` = open for a claim now, `proposed` = open but the maintainer may still reshape it, so ask in the issue first; `claimed` and `done` are shown but get no issue; this is the contributor status, not the queue row's), `kind` (decomp, verify), `source` (`queue:N` = row N of docs/decomp-queue.md, `backlog:NAME` = a line in docs/BACKLOG.md), `question`, `start`, `deliverable`, `needs`. Function addresses are tags only (`FUN_xxxxxxxx` as in the repo's trace docs); never paste code.

## SKGROUND: ground skills (Blizzard, Meteor, Hydra, Fire Wall, Thunder Storm, Frozen Orb)
- size: M
- status: proposed
- kind: decomp
- source: queue:22
- question: Blizzard, Meteor and Fire Wall are already traced and run against game.exe (docs/research/sk-ground-trace.md, sk-ground2-trace.md, skground-trace.md), including the skill delay and the shape of Meteor's blast. Left open: Hydra (read to the spawn only: its AI and HydraMissile are not run), Frozen Orb and Thunder Storm (read depth equals d2d, nothing run natively), the skill-delay refusal and the unit flag bits 0x4 / 0x8 in FUN_0056b3e0. Pick ONE of these and trace it fully with a native run; the others are separate PRs.
- start: docs/research/sk-ground-trace.md and sk-ground2-trace.md (what is done, their section of U lines); docs/research/sk-missile-trace.md and sk-nova-trace.md for the sub-missile timeline style (FUN_0056d400 is the Frost Nova ring spawn).
- deliverable: docs/research/sk-ground-<skill>-trace.md; advanced item, claim it only if you have done a decomp before.
- needs: your own game.exe 1.14d, upstream d2decomped checkout, Python 3 with unicorn.

## SKIAS: swing speed, increased attack speed and the weapon speed modifier
- size: M
- status: proposed
- kind: decomp
- source: backlog:SKMELEE+
- question: Partly answered (docs/research/sk-ias-trace.md, a desk trace): d2d's `attack_speed` and `attack_ticks` are read and their breakpoints reproduced by a script. Left open: game.exe's side. How many frames does one melee swing take for a given weapon speed modifier, increased attack speed gear, a skill speed bonus and a class animation table (nothing of it was run: every game.exe figure there is U)? Cover the breakpoints for any class (class-generic: pick the animation tables from your own data) and compare with the d2d side in that doc.
- start: docs/research/sk-melee-trace.md (Not covered: swing speed); docs/research/paladin-act1-recon.md section 6 row 4 (the recon measured only the swing length); docs/research/walk-speed-trace.md for the velocity and frame-rate style.
- deliverable: docs/research/sk-attack-speed-trace.md; an oracle script under tools/emu/ if you ran game code; exhaustive boundaries or 100,000 cases for any number you call V.
- needs: your own game.exe 1.14d install, upstream d2decomped checkout, Python 3 with unicorn.

## SKZEROHIT: what a zero-damage hit still triggers
- size: M
- status: proposed
- kind: decomp
- source: backlog:SKRESIST
- question: Static answer done (docs/research/sk-zero-hit-trace.md, V read; independently checked in docs/research/verifyzerohit.md): a zero-damage hit keeps knockback, gives no get-hit and no wake; Lightning Enchanted bolts fire on a no-recovery hit. Left open: the native run of the differentials D1 and D2 against d2d (the second-pass run doc is not on main), and whether d2d matches on each point. Re-derive those with your own harness.
- start: docs/research/sk-resist-trace.md section 10 (first bullet) and Q9; FUN_0057cb00 (the get-hit gate, S: docs/research/pr-get-hit-rule-proof.md); docs/research/mon-wake-trace.md for the wake rules.
- deliverable: docs/research/sk-zero-hit-trace.md; a static answer (tagged V read) is a valid first PR, a native run (V run) is better.
- needs: your own game.exe 1.14d install, upstream d2decomped checkout, Python 3 with unicorn.

