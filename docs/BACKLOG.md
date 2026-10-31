# Agent backlog (the idea queue; the work to launch is the cards in `openspec/changes/`)

Rule: keep 5 to 8 agents running (coordinator practice 2026-10-30). When one finishes: merge it (labrun worker whole-repo check, grep FAIL, push, archive the card), then launch the next READY item below on disjoint files as a new card. Add items here whenever a report says "not done" / "needs wiring". Live items need a free instance or d2d slot (`tools/d2dslot.sh status`) and a non-NO `tools/capacity.sh --add <cores>` (docs/process/worker-capacity.md). Current state: [plan/STATUS.md](plan/STATUS.md). Deduped and reordered 2026-10-30 evening by REVIEWDOC; merged items were dropped (`git log` has them), the 2026-10-29 Wine-era queue was pruned to what still applies to the frozen Game.exe path.

## A. Bug-fix cards from the code review (docs/research/code-review-2026-10-30.md section 9; owners and files there)

1. **soakexit** (H): d2dsoak exit codes incl. `-json`; empty, crashed and short seeds flagged and excluded from exposure; duplicate seeds rejected; min-sample INCONCLUSIVE; NaN/negative guards; parse-failure counting; repeatSites cap; killtime ParseFloat checks.
2. **replayharden** (H, with the agent/replay merge): tap serve goroutine, real busy check, socket chmod and Lstat; decompressed-size and step caps; fatal bad lines; hash/verb validation; playback verb and `Build.Args` allow-lists; d2dslot pool and release order; `Stopped`/0-checkpoint exit codes; spec delta text.
3. **viewguard** (M): d2dview watch speed floor, sleep outside the lock, normalized verb forwarding, `botSock == upstream` refusal, write deadlines and viewer cap on SSE/MJPEG, `deadSeen` carry, shared deny list with sanitize.py.
4. **pubnames** (M, after the user answers decision 2 in STATUS): sanitize.py name allow-list and word-boundary fix, `running` from wall time when served from cache, junk-reply fallback, UUID/token rules, leak-grep gate = `scan_file` + names. `tools/publish/**` is owned by stream-grid until it merges.
5. **detloopchunk** (M, with the agent/detloop merge): step in chunks of at most `WatchMs` while a watcher is registered; nested-Watch guard; note in docs/d2d-determinism.md; re-baseline deaths/h after.
6. **slotnits** (L): d2dslot exit-99 remap, capacity.sh numeric validation, hpfields reset on a new game, killtime zero-line note.

## B. OPEN, ranked (d2d primary per ADR 0009)

1. **Fidelity gate not met** (XP/h -41 % after ECONOMY, kills/h a third; docs/d2d-loop.md, docs/d2d-economy.md): the bot-side causes are search/stall (5.5 s per kill vs 0.3, docs/research/per-kill-time-split.md), poverty recovery (63 % of time, ~20 gold/h), chicken restarts instead of fights, 0-kill recovery minutes, repair failures. Cards exist for the measurement (d2d-replicated-soak) but not for the fixes: one card per cause, each gated on the per-kill split improving on 3 seeds.
2. **Determinism of the full bot loop** (detloop unmerged): merge after detloopchunk; then the replay tool (agent/replay) and the determinism finder can be used for the gate runs.
3. **d2d container** (cards d2d-container-build, -bot, -soak): blocked on the licence/GPL review (STATUS decision 1); the draft Dockerfile and compose are in docs/d2d-container/.
4. **Default backend flip** (card d2d-default-backend): small; after the container bot runs once.
5. **Spell damage truncation PR** (docs/research/pr-missile-damage-fine-draft.md): pushed only after prmeasure's A/B numbers and the user's go.
6. **Merc on d2d**: policy wired behind `merc.enabled` (docs/merc-wire.md) but never run for hours; free Rogue claim on Game.exe is a stub (frozen, low). Needs grill 0003 answers first.
7. **Act 2+ on d2d**: never run on either backend; the Act 2 chain items of the old INDEX section 2 (Staff, tomb 73, Duriel, Summoner) are the first d2d probes once Act 1 completes on d2d.
8. **`restarts` counter** in the soak minute line counts 1 of 17 game recycles (G2 summary): fix in the shared minute line (both backends).
9. **Monster-kind classification** (`d2.Kind()` checks the boss bit first, so champions with flag 0x0c are labelled unique and the normal bit 0x02 is never set; found by origprobes): fix with a table test on the five flag combinations.
10. **`potionsDrunk` counter vs belt reads** (ROUTEGUARD fixed it, b2e010c): confirm on the next log that deaths.jsonl `potions` matches.
11. **Mephisto quit-safety** (d2d says the Act 4 gate reopens after a reload; the user says a fresh kill is needed): test on the original when we get there (user, 2026-10-30), card orig-decision-probes is deferred.
12. **hostnames-not-ips**: docs/d2d-fork.md:176 and docs/process/worker-compute.md carry `nas.example` literals; replace with the `.<domain>` names (user rule). Docs only.

## C. Frozen Game.exe path: still-valid items (run only if the retirement soak or an orig probe needs them)

- SOAKFOR: `tools/soak/soak24.sh` parses `--for` as `*h` then `*m` only (`1h30m` breaks under `set -e`); accept `90m`, `1h30m`, `2h`, `45s`.
- WINESAMPLE / SOAKPARSE2: d2soak does not sample `winedevice.exe` and does not alert on the /status panic and restart counters.
- FARMRESUME: `internal/bot/farm.go:108` restarts a FarmLoop with the full count on every interrupt. KILLBOSS: `kill_boss` goals still resolve to farm_area.
- ANTIDOTECD: `internal/combat/hard.go:80` drinks an antidote every 3 s while poisoned; duplicate `ResistPenalty` definitions. RESISTCHECK (research): does the memory resist stat already contain the difficulty penalty (open-findings EV-M10)?
- SAVESNAP: `saves.Snapshot`/`Verify` are not wired into the managed d2bot. HELPERFATAL: helper death during a fight only logs `move failed`.
- Container (Wine, paused): ENTRYFIX remainder, FLEETSEED, CONTAINERLOG (`d2doctor --bundle`), DOCTORNITS (`/checks` leaks paths without the token), CONTAINERWALK (stranger walkthrough). Do not start unless the Wine container is un-paused.
- Features: RESCUEOBJ (cage picker clicks the same cage three times), HELLFLOOR (Hell floor 75 for every Act 1 quest), LEVELWP (activate waypoints while leveling).
- Nav stuck at Burial Grounds 5459,4549: root cause unproven; NAVSTUCK is a mitigation only.
- From the SME audit (docs/sme/audit-bot-vs-human.md): Act 1 XP cliff (Andariel held at clvl 22), builds stop at clvl 30 for six classes, unreachable resist gate, ladder-only runewords in the data files, oldest-corpse-only handling.

## D. Needs the user (also on the board's Waiting-on-you panel, `tools/needs-you.sh list`)

- The five decisions in [plan/STATUS.md](plan/STATUS.md): licence/GPL review, public character names, grill 0003 (merc and Act 2), upstream PR follow-ups, stream and ports.
- Kick stream stays OFF until asked. The container installer viewer needs a second port. The NAS autostart for d2-board.

## E. Ideas (not cards yet)

- Replay as video: `d2dreplay render` exists on agent/replay; a public "best death of the day" clip needs the sanitizer applied to frames (no window titles, no paths in the overlay).
- Use a Windows worker WSL2 d2d worker for fidelity seeds in parallel with the worker host (docs/process/d2d-worker-host.md); needs d2dslot on a Windows worker.
- Whisper / d2proxy LAN oracle (docs/research/d2d-whisper-lan-oracle.md): a packet-level oracle for town layout and spawns; card d2d-lan-oracle-demo is proposed.

- d2d hang: combat loop spins on `combat move failed: nav: no path: empty path` without stepping the lockstep game (seed 42 walk-only arm freezes at virtual ~90 s; logs ~/d2d-lab/runab/stall-S2off*.log on the worker host): bot-side deadlock under lockstep: a fix agent after the freeze.
- upstream d2d PR candidate: Lut Gholein room-seed shift (two RNG steps): trace the two extra steps with emulator hooks on the level RNG (FUN_00666680 per-room roll; candidates FUN_00667970 in stays, FUN_00666ed0), fix generate_preset/stays, diff_drlg.py 1-20 40 tiles to 0 mismatches, verifier, evidence pack; after the freeze
- d2d lockstep nav stall repro: soakfull seed 11 deterministic hang at virtual min 109 in area 8 (combat crowd: stepping away / combat move failed: nav: no path: empty path; logs on the worker host ~/d2d-lab/soakfull/logs/hung-s11*.log; same spin as runab seed 42): needs a melee/hold fallback; after the freeze
- compare SOAKFULL hotfix (agent/soakfull 4db3f20 internal/bot/task.go: freshGameToLevel resets cleared, sweepKills, needBuff, stepArea after NewGame + test; kept in tag archive/soakfull) with master's RUNAB fix (resets cleared only?): adopt the wider reset if it is the more complete one; after the freeze
- d2d XP/h gap now (soakfull, 12 runs, 24 h pooled, lab6, run policy on): 4985 +- 145 vs 6600 early baseline (-24.5 %, INCONCLUSIVE); kills/h 195 vs 500 late; deaths 0 PASS; 18.0 s per kill (walk 13.8, cast 0.9, wait 3.3); recovery 54 % of time; next suspects: walk 77 % of kill time (border ping-pong, nav stuck), poverty/recovery 54 %, casts per kill, spawn density/XP award, crowd step-away stall

- d2d container remaining tasks (from archived cards d2d-container-build and d2d-container-bot): 1 h one-slot bot gate in the container, realtime mode, read-only root filesystem, d2fleet wizard for d2d, viewer source via the screenshot verb, soak driver sidecar, optional Sentry fork patch (not needed so far)
- tools/d2dfarm-test.sh has 7 failures on master (they passed on the agent branch): investigate after the scrub
- live runner could forward D2_COLD_DIR and the d2dview host words (Go processes do not read local.env)
- docs/nas-container.md heading rename (package C): "Running a soak on the NAS" (code comments quote it)
