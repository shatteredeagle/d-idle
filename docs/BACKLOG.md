# Backlog: open items only (the work to launch is the cards in `openspec/changes/`)

Trimmed 2026-10-31 (doctrim); each item was checked against the code and `git log` that day. The full old list, with the items that have since merged, is [archive/BACKLOG-2026-10-30.md](archive/BACKLOG-2026-10-30.md). Where we are: [plan/STATUS.md](plan/STATUS.md). Live items need a free instance or d2d slot (`tools/d2dslot.sh status`) and a non-NO `tools/capacity.sh --add <cores>` ([process/worker-capacity.md](process/worker-capacity.md)). When a report says "not done", add a line here; when an item gets a card, replace the line with the card id.

## 1. Cards that exist (the queue, in `openspec/changes/`)
- **Fidelity gate** (XP/h and kills/h below the original; cause: search/stall, poverty recovery, chicken restarts): card `xpgap` (in progress, agent/xpgap) and `d2d-replicated-soak`; analysis in [research/per-kill-time-split.md](research/per-kill-time-split.md) and [d2d-loop.md](d2d-loop.md). One card per further cause, each gated on the per-kill split improving on 3 seeds (not written yet).
- **d2d container**: card `d2d-container-soak` (the 24 h retirement soak); build and bot cards are archived. Blocked on the licence/GPL review (STATUS decision 1). Remaining from the archived cards: 1 h one-slot bot gate in the container, realtime mode, read-only root filesystem, d2fleet wizard for d2d, viewer source via the screenshot verb, soak driver sidecar.
- **Default backend flip**: card `d2d-default-backend` (small; after the container bot runs once).
- **Mesh and stream**: cards `d2dmesh`, `stream-grid` (in progress).
- **Original-game work (deferred)**: cards `gameexe-retire-soak`, `orig-decision-probes` (Mephisto quit-safety: d2d says the Act 4 gate reopens after a reload, the user says a fresh kill is needed).
- **d2d LAN oracle demo**: archived card `d2d-lan-oracle-demo`; idea doc [research/d2d-whisper-lan-oracle.md](research/d2d-whisper-lan-oracle.md).

## 2. Open, no card yet
1. **detloopchunk** (M): `Pacer.Watch` runs once per step chunk; step in chunks of at most `WatchMs` while a watcher is registered, add a nested-Watch guard, re-baseline deaths/h ([d2d-determinism.md](d2d-determinism.md) section 3.1; review: [research/code-review-2026-10-30.md](research/code-review-2026-10-30.md) section 9).
2. **viewguard** (M): `cmd/d2dview` has no write deadlines or viewer cap on SSE/MJPEG; watch speed floor, sleep outside the lock, `botSock == upstream` refusal, shared deny list with the publish sanitizer (same review doc). Unverified in detail: only the missing deadlines were re-checked.
3. **Monster kind**: `d2d.MonsterInfo.Kind()` checks the boss bit first, so champions with flag 0x0c read as unique and the normal bit 0x02 is never reached; fix with a table test on the five flag combinations (`internal/d2/monflags.go`).
4. **`restarts` counter** in the soak minute line counted 1 of 17 game recycles (G2 summary); the fix is in the shared minute line (both backends). Unverified whether it was fixed since; check `cmd/d2soak/minutelines.go`.
5. **Upstream PR, Lut Gholein tool fix** (drlg-dump opens the folder's 1.14d patch): proof and draft done, NOT posted; waits for the user's go ([research/pr-lut-gholein-evidence-pack.md](research/pr-lut-gholein-evidence-pack.md)). PRs #134, #135, #136 are open upstream, the maintainer's answer decides the next ones ([research/upstream-prs.md](research/upstream-prs.md)).
6. **Merc on d2d**: policy wired behind `merc.enabled` ([merc-wire.md](merc-wire.md)) but never run for hours; needs grill 0003 answers first.
7. **Act 2+ on d2d**: never run on either backend; the Act 2 chain (Staff, tomb 73, Duriel, Summoner) is the first d2d probe set once Act 1 completes.
8. **Launchers and Go processes on the worker**: `tools/d2dloop/*`, `tools/d2dsoak/run.sh`, `tools/soak/fresh.sh` may still not forward `d2_role_env` (only some launchers do); one more fresh-game reset copy at `goal.go:128` to fold into `freshGameToLevel` (needs a G1/G2 check).
9. **Public tools/capacity.sh** is an ops stub, so a CI-visible capacity test fixture needs the script in the public tree (d2dfarm resolves the real file through `ops.sh`; `D2_FARM_CAPACITY` overrides).
10. **pubnames** (M): publish sanitizer name allow-list, junk-reply fallback, UUID/token rules; `tools/publish` now lives in the ops repo, so not checked here; after STATUS decision 2.
11. **potionsDrunk vs belt reads**: confirm on the next log that deaths.jsonl `potions` matches (ROUTEGUARD fix b2e010c).
12. **Frozen Game.exe path** (only if the retirement soak or an orig probe needs them; list from 2026-10-30, only partly re-checked): FARMRESUME (`internal/bot/farm.go` restarts a FarmLoop with the full count on interrupt), KILLBOSS (`kill_boss` goals still resolve to farm_area), duplicate/doubled `ResistPenalty` (`internal/combat/hard.go:30`, open question RESISTCHECK), SAVESNAP (`saves.Snapshot` not wired into the managed d2bot), HELPERFATAL, RESCUEOBJ, HELLFLOOR, LEVELWP, Nav stuck at Burial Grounds 5459,4549 (root cause unproven), the Wine container leftovers (ENTRYFIX, FLEETSEED, CONTAINERLOG, DOCTORNITS, CONTAINERWALK; only if the container is un-paused), and the SME audit list ([sme/audit-bot-vs-human.md](sme/audit-bot-vs-human.md): Act 1 XP cliff, builds stop at clvl 30, unreachable resist gate, ladder-only runewords).
13. **Replay as video**: `d2dreplay render` exists; a public "best death of the day" clip needs the sanitizer applied to frames. Idea only.
14. **Windows worker WSL2 d2d worker** for fidelity seeds in parallel ([process/d2d-worker-host.md](process/d2d-worker-host.md)); needs d2dslot on a Windows worker. Idea only.
15. **slotnits tail** (L): `capacity.sh` numeric validation, hpfields reset on a new game, killtime zero-line note. Unverified.

## 3. Needs the user (also the board's Waiting-on-you panel, `tools/needs-you.sh list`)
- The five decisions in [plan/STATUS.md](plan/STATUS.md): licence/GPL review, public character names, grill 0003 (merc and Act 2), upstream PR follow-ups (post the Lut Gholein PR; cast follow-ups wait for the maintainer), stream and ports.
- Kick stream stays OFF until asked. The container installer viewer needs a second port. The NAS autostart for d2-board.
