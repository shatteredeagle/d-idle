# Open to contributors

> status: current, APPROVED by the maintainer 2026-11-01 (card communitywork). The source list for `tools/contrib/board.py`, which renders [work-board.md](work-board.md) and the GitHub issue bodies. How to take an item: [../contributing/agent-work.md](../contributing/agent-work.md).

Only items that need nothing but the game binary of your own legitimate Diablo II LoD 1.14d install and public sources (this repo, upstream `psi29a/d2decomped`) are listed. Live drills, soaks and anything on the maintainer's private lab are NOT open (see the guide). The maintainer keeps this file in step with `docs/decomp-queue.md` and `docs/BACKLOG.md`; where they disagree those two win.

Format (read by the board script): one `## ID: title` section per item, then `- key: value` lines. Keys: `size` (S, M, L), `status` (`ready` = open for a claim now, `proposed` = open but the maintainer may still reshape it, so ask in the issue first; `claimed` and `done` are shown but get no issue; this is the contributor status, not the queue row's), `kind` (decomp, verify), `source` (`queue:N` = row N of docs/decomp-queue.md, `backlog:NAME` = a line in docs/BACKLOG.md), `question`, `start`, `deliverable`, `needs`. Function addresses are tags only (`FUN_xxxxxxxx` as in the repo's trace docs); never paste code.

## PALSKILL: Smite, Zeal and Sacrifice dispatch in game.exe
- size: M
- status: proposed
- kind: decomp
- source: backlog:SKMELEE+
- question: How does game.exe dispatch Smite, Zeal and Sacrifice (hit count, AR and damage bonus, mana and life cost, knockback and stun, what a held button repeats), and which of these does d2d's `skill_built` branch implement versus falling back to a plain attack?
- start: docs/research/paladin-act1-recon.md section 6 rows 1 and 5; the skill bind trace docs/research/skill-bind-trace.md for the packet side (FUN_0054be70, FUN_00647960); upstream d2decomped fight code (public) for what d2d does.
- deliverable: docs/research/pal-skill-trace.md; a static result (which skills fall back in d2d, read from upstream source, tagged S) is a valid first PR on its own.
- needs: your own game.exe 1.14d, upstream d2decomped checkout, Python 3 with unicorn if you run code.

## PALAURA: how an aura applies in game.exe
- size: M
- status: proposed
- kind: decomp
- source: backlog:PALAURA
- question: Prior work (do not redo): docs/research/pal-skill-trace.md section 3 already has the pulse period (50 frames), the ranges (16 + 2 per level for Might, Prayer, Defiance and Concentration; 6 + 1 per level for Holy Fire) and every calc slot (21,760 cases, 0 differ against d2d). Open: (1) the aura target filter (aurafilter 0x12003 for the friendly auras, 0xa783 for Holy Fire) and the shape of the area search FUN_0056b7e0 (who is passed: self, merc, pets, party, enemies; the exact radius test); (2) the damage an aura does to enemies per pulse (Holy Fire, Shock, Freeze: roll, resist, which units), read and then run; (3) Prayer's per-pulse mana and state 85 (`nomanaregen`): does the regeneration code really stop while it is set. Also useful: the D5 Might mana floor and aura reach to merc and party on d2d (the trace's D1 to D7).
- start: docs/research/pal-skill-trace.md sections 3, 6 (D5, D6) and 8 (drills A2, A3); FUN_005cf010 (friendly aura do), FUN_005cf3a0 / FUN_005d0920 (Holy Fire, Shock, Freeze do), FUN_0056b7e0 (area apply), FUN_005cedc0 (callback), FUN_00644b10 and FUN_0056c110 (mana); the emulator style of tools/emu/pal_aura.py; docs/research/paladin-act1-recon.md section 6 rows 3 and 11; docs/research/d2d-fidelity-riskmap.md.
- deliverable: docs/research/pal-aura-trace.md covering only the open parts above, each claim tagged V (run in the emulator), V read, S or U; for each part end with the exact probe the maintainer should run on d2d (the live half is maintainer-only) and the number you expect. Done when: the filter bits are decoded and the area search shape is stated with a V run on a fake player, merc and monster set; the enemy hit of one aura is run against the trace's element tables; Prayer's mana and state 85 effect are read to the regeneration code; the trace's D5 and D6 rows can be updated from it.
- needs: your own game.exe 1.14d, upstream d2decomped checkout, Python 3 with unicorn.

## PALDUR: durability loss per hit and per block
- size: S
- status: proposed
- kind: decomp
- source: backlog:PALDUR
- question: In game.exe 1.14d, what is the chance and the slot choice for durability loss when the player hits and when the player is hit or blocks, and does d2d implement the same rule (it has `debug wear` verbs but the loss rule is unknown)?
- start: docs/research/price-trace.md (the repair gate and price are done: FUN_0062e660, FUN_00629930, use them as the "durability" anchor); docs/research/paladin-act1-recon.md section 6 row 7.
- deliverable: docs/research/durability-trace.md, small; rule with tags, d2d comparison from upstream source (S) or a differential (V) if you build one.
- needs: your own game.exe 1.14d, upstream d2decomped checkout, Python 3 with unicorn.

## SKGROUND: ground skills (Blizzard, Meteor, Hydra, Fire Wall, Thunder Storm, Frozen Orb)
- size: M
- status: proposed
- kind: decomp
- source: queue:22
- question: Blizzard, Meteor and Fire Wall are already traced and run against game.exe (docs/research/sk-ground-trace.md, sk-ground2-trace.md), including the skill delay. Left open: Hydra (read to the spawn only: its AI and missile are not run), Frozen Orb and Thunder Storm (read depth equals d2d, nothing run natively), and the shape of Meteor's blast. Pick ONE of these and trace it fully with a native run; the others are separate PRs.
- start: docs/research/sk-ground-trace.md and sk-ground2-trace.md (what is done, their section of U lines); docs/research/sk-missile-trace.md and sk-nova-trace.md for the sub-missile timeline style (FUN_0056d400 is the Frost Nova ring spawn).
- deliverable: docs/research/sk-ground-<skill>-trace.md; advanced item, claim it only if you have done a decomp before.
- needs: your own game.exe 1.14d, upstream d2decomped checkout, Python 3 with unicorn.

## VERIFY-TELEPORT-ROOM: independent re-check of the Teleport room rule
- size: S
- status: ready
- kind: verify
- source: backlog:SKTELE
- question: Our claim: game.exe refuses a Teleport hop whose target lies outside the caster's room and the rooms in that room's near list (FUN_00554ea0 with FUN_00463740 over FUN_00619790), so the range is a room rule and not a distance; a per-level Teleport value of 0 refuses and 2 adds a line test (FUN_006229f0). Re-derive the destination rule yourself on a grid of caster positions and targets, class-generic, and compare with the claim.
- start: docs/research/sk-teleport-trace.md (read AFTER your own look at FUN_005ca360 and FUN_00554ea0); tools/emu/sk_tele_spot.py only to compare numbers at the end.
- deliverable: docs/research/verify-teleport-room.md: per claim HOLDS / HOLDS WITH LIMITS / FAILS (accepted and refused counts, grid, seed), the reproduction command, differences from the author's numbers.
- needs: your own game.exe 1.14d install, upstream d2decomped checkout, Python 3 with unicorn.

## VERIFY-AIDEL: independent re-check of the monster think period
- size: S
- status: ready
- kind: verify
- source: backlog:FREEZERESUME
- question: Done (docs/research/verify-aidel.md): the aidel column per difficulty and the think scheduling FUN_005a73e0 (352,800 cases, 0 differ) and the 55,268 equal cases claim (holds with limits); the resume at freeze end + aidel was also re-checked in docs/research/pr-freezeresume-verifier.md. Left open: drive FUN_005a7c20 and the freeze-end callback (FUN_0057b170) with your own harness (the verify-aidel doc marks that claim NOT CHECKED), and a think event due during a get-hit (U). Re-derive those with your own harness.
- start: docs/research/mon-wake-trace.md (the aidel rows, read AFTER your own look at the two functions); docs/research/pr-freezeresume-proof.md for the freeze-end claim.
- deliverable: docs/research/verify-aidel.md: HOLDS / HOLDS WITH LIMITS / FAILS per claim, your case count and seed, the command, differences from the author's numbers.
- needs: your own game.exe 1.14d install (the monster table read only, never shared), upstream d2decomped checkout, Python 3 with unicorn.

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
- question: A hit that deals no damage (an immune target, a full block, a rounded-down roll) may still wake or alert the monster, fire on-hit effects of enchanted monsters and apply knockback. Which of these does game.exe do, in which function, with which condition, and does d2d match?
- start: docs/research/sk-resist-trace.md section 10 (first bullet) and Q9; FUN_0057cb00 (the get-hit gate, S: docs/research/pr-get-hit-rule-proof.md); docs/research/mon-wake-trace.md for the wake rules.
- deliverable: docs/research/sk-zero-hit-trace.md; a static answer (tagged V read) is a valid first PR, a native run (V run) is better.
- needs: your own game.exe 1.14d install, upstream d2decomped checkout, Python 3 with unicorn.

## SKSTATEEV: the state events fired on a hit (FUN_005d3ac0)
- size: L
- status: proposed
- kind: decomp
- source: backlog:SKARMOR
- question: FUN_005d3ac0 walks a unit's states and fires the events their stat lists name (attacked in melee, absorb damage, on-hit effects). Which events fire on a hit and on a miss, in which order against resistances and other absorbs, and where are events 3 and 11 fired? This settles Shiver Armor on a miss and the place of Energy Shield against resistances; advanced item, claim it only if you have done a decomp before.
- start: docs/research/sk-armor-trace.md section 11 (Not traced) and Q3, Q5; docs/research/sk-melee-trace.md Q1 (the pre-events row); docs/research/sk-melee2-trace.md (Not covered).
- deliverable: docs/research/sk-state-events-trace.md in the trace style, plus an oracle script under tools/emu/ if you ran game code.
- needs: your own game.exe 1.14d install, upstream d2decomped checkout, Python 3 with unicorn.
