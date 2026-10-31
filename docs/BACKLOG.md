# Backlog: open items only (the work to launch is the cards in `openspec/changes/`)

> Milestones, launch order and the operating rhythm: [roadmap.md](roadmap.md) (proposal, 2026-10-31); the cards are `openspec/changes/m1-*`, `m2-*`, `m3-*`.

> Bot generality study merged (docs/research/bot-generality.md): proposed first card gen-invariants (holdout seeds 16-23 + invariant counters in the M2 gate run); awaiting the user's approval to launch.
> gen-invariants merged: d2dsoak reports per-seed invariants (HANG STALL WEDGE PROGRESS DEATHS RESTARTS EXIT) and a holdout group (seeds 16-23); exit codes unchanged (read INVARIANTS: or -json). Offline validation: navsoak and navsoak2 VIOLATED(WEDGE,PROGRESS), navsoak3 PASS so far; m2early seed 2 VIOLATED(HANG): one real combat no-progress watchdog firing (investigating)
> combatspin merged: the one watchdog firing (baseline seed 2) was abortFight Flee picking a point in a cave mouth; move skipped it without sleeping; fixed (move costs a poll tick, valid rejects mouth points); not verified live; the next gate run will show it via the HANG invariant
> gen-corpus merged: 6 scenarios (wedge pocket, kit-less clvl-7, poverty loop, campaign stall, combat pin, champion group) as TestScenario_* tests with knob-off proofs for 4 (docs/scenarios.md); the town scenarios need the extracted game tables (skip without)

Trimmed 2026-10-31 (doctrim); each item was checked against the code and `git log` that day. The full old list, with the items that have since merged, is [archive/BACKLOG-2026-10-30.md](archive/BACKLOG-2026-10-30.md). Where we are: [plan/STATUS.md](plan/STATUS.md). Live items need a free instance or d2d slot (`tools/d2dslot.sh status`) and a non-NO `tools/capacity.sh --add <cores>` ([process/worker-capacity.md](process/worker-capacity.md)). When a report says "not done", add a line here; when an item gets a card, replace the line with the card id.

## 0. Next milestone: solid bot basics, then the fidelity gate (user, 2026-10-31)
The next milestone is making the bot itself better: the town and inventory basics must be rock solid on d2d, not progression through the game. The d2-sme's 15-scenario checklist (2026-10-31) is the acceptance list; each scenario needs an offline test or a live proof, tracked in a coverage table (first step: the offline basics audit, awaiting the user's go).
1. Gold 50, belt empty, in town: buys 1 healing potion, no scroll or tome, leaves.
2. Gold 120, 1 scroll, belt full: buys no scroll, keeps the gold, no shop visit.
3. Gold 300, no scroll, belt full, clvl 9: buys 1 scroll, keeps at least 150.
4. One junk item worth 20 gold, 0 scrolls: no sell trip.
5. Seven sellable items worth over 30 gold in total: one visit sells all, scroll count unchanged.
6. Charm, rune and gem in the pack: all still held after a sell visit.
7. Unidentified magic item worth under 100: sold unidentified, no ID scroll bought.
8. Durability 28%, gold 200: repaired in the same visit, no extra trip.
9. Durability 90%: no repair attempt.
10. Repair fails: logged once, no retry until gold or durability changes.
11. Gold 1500 at clvl 5: the stash step banks the surplus, the pack keeps 300 + 50 x clvl.
12. Belt has 2 free slots and 1 spare potion in the pack: the spare goes to the belt before leaving town.
13. Vendor armor needs more Strength than the character has: not bought, not bought again after a restart.
14. Pack full (4 or fewer free cells): one sell visit, fresh loot ends in the pack.
15. Blood Raven done, clvl 8 or later: Kashya's reward claimed once, no second hire attempt.
Why this first: the SME read the poverty loop (about 54 % of time farming gold) as a spending-policy problem, so closing these basics is also expected to close most of the fidelity-gate gap (card `xpgap`, archived, section 1). Open unknowns: the starting gold of a new offline 1.14d character, real unidentified-item sell prices.

## 1. Cards that exist (the queue, in `openspec/changes/`)
- **Fidelity gate** (XP/h and kills/h below the original; cause: search/stall, poverty recovery, chicken restarts): card `xpgap` (archived) and `d2d-replicated-soak`; analysis in [research/per-kill-time-split.md](research/per-kill-time-split.md) and [d2d-loop.md](d2d-loop.md). One card per further cause, each gated on the per-kill split improving on 3 seeds (not written yet).
  - m2-earlybase measured 2026-10-31: current d2d bot XP/h 4767 +- 230, kills/h 192 +- 7.5 (8 seeds x 3 virtual h, 0 exclusions) vs Game.exe baseline 9172 XP/h / 398 kills/h (weak: two runs): gate FAIL (-48 % / -52 %); deaths PASS; see [d2d-baselines-early.md](d2d-baselines-early.md)
  - XPGAP result 2026-10-31: gap not closed; none of the three knobs beats noise (see [d2d-xp-gap.md](d2d-xp-gap.md)); next suspects: shorten the town trip (waypoint out of town, LEVELWP), leave emptied Blood Moor sooner, chicken restarts, border ping-pong.
- **d2d container**: card `d2d-container-soak` (the 24 h retirement soak); build and bot cards are archived. Blocked on the licence/GPL review (STATUS decision 1). Remaining from the archived cards: 1 h one-slot bot gate in the container, realtime mode, read-only root filesystem, d2fleet wizard for d2d, viewer source via the screenshot verb, soak driver sidecar.
- **Default backend flip**: card `d2d-default-backend` (small; after the container bot runs once).
- **Mesh and stream**: cards `d2dmesh`, `stream-grid` (in progress).
- **Original-game work (deferred)**: cards `gameexe-retire-soak`, `orig-decision-probes` (Mephisto quit-safety: d2d says the Act 4 gate reopens after a reload, the user says a fresh kill is needed).
- **d2d LAN oracle demo**: archived card `d2d-lan-oracle-demo`; idea doc [research/d2d-whisper-lan-oracle.md](research/d2d-whisper-lan-oracle.md).

## 2. Open, no card yet
1. **detloopchunk** (M): `Pacer.Watch` runs once per step chunk; step in chunks of at most `WatchMs` while a watcher is registered, add a nested-Watch guard, re-baseline deaths/h ([d2d-determinism.md](d2d-determinism.md) section 3.1; review: [research/code-review-2026-10-30.md](research/code-review-2026-10-30.md) section 9).
2. **viewguard** (M): `cmd/d2dview` has no write deadlines or viewer cap on SSE/MJPEG; watch speed floor, sleep outside the lock, `botSock == upstream` refusal, shared deny list with the publish sanitizer (same review doc). Unverified in detail: only the missing deadlines were re-checked.
3. **Champions as elite, not boss** (card `champ`, merged): champions were already not bosses in target.go; the new caution knob combat.championEarlyPct (default 10 pp) and the killerChampion autopsy label are in; d2-sme verdict (2026-10-31) Elite=true, Boss=false. Needs a G1/G2 check (offline only so far).
4. **`restarts` counter**: fixed and merged (botsmall: every game recycle counts, both backends). Still not counted: d2d crash re-entries and save points. Check the next soak's minute lines.
- upstream PR policy: bundle several proven claims per PR (maintainer preference 2026-10-31); renderer + Lut Gholein bundle prepared, not opened ([research/pr-bundle-evidence-pack.md](research/pr-bundle-evidence-pack.md)).
5. **Upstream PR, Lut Gholein tool fix** (drlg-dump opens the folder's 1.14d patch): proof and draft done, NOT posted; waits for the user's go ([research/pr-lut-gholein-evidence-pack.md](research/pr-lut-gholein-evidence-pack.md)). PRs #135 and #136 merged by the maintainer 2026-10-31; #134 open; renderer PR (get-hit frames) and Lut Gholein tool fix are READY to open on the user's go (rebased onto upstream main) ([research/upstream-prs.md](research/upstream-prs.md)).
- PR #143 (bundle: get-hit frames + drlg-dump patch): opened 2026-10-31; CodeRabbit follow-up commit 2c151686 pushed (patch fallback, case-insensitive archives, merc FHR restore in local views); no human review yet; see docs/research/pr-bundle-evidence-pack.md
6. **Merc on d2d**: policy wired behind `merc.enabled` ([merc-wire.md](merc-wire.md)) but never run for hours; needs grill 0003 answers first.
7. **Act 2+ on d2d**: never run on either backend; the Act 2 chain (Staff, tomb 73, Duriel, Summoner) is the first d2d probe set once Act 1 completes.
8. **Launchers and Go processes on the worker**: `tools/d2dloop/*`, `tools/d2dsoak/run.sh`, `tools/soak/fresh.sh` may still not forward `d2_role_env` (only some launchers do); one more fresh-game reset copy at `goal.go:128` to fold into `freshGameToLevel` (needs a G1/G2 check).
9. **Public tools/capacity.sh** is an ops stub, so a CI-visible capacity test fixture needs the script in the public tree (d2dfarm resolves the real file through `ops.sh`; `D2_FARM_CAPACITY` overrides).
10. **pubnames** (M): publish sanitizer name allow-list, junk-reply fallback, UUID/token rules; `tools/publish` now lives in the ops repo, so not checked here; after STATUS decision 2.
11. **potionsDrunk vs belt reads**: confirm on the next log that deaths.jsonl `potions` matches (ROUTEGUARD fix b2e010c).
12. **Frozen Game.exe path** (only if the retirement soak or an orig probe needs them; list from 2026-10-30, only partly re-checked): FARMRESUME (`internal/bot/farm.go` restarts a FarmLoop with the full count on interrupt), KILLBOSS (`kill_boss` goals still resolve to farm_area), duplicate/doubled `ResistPenalty` (`internal/combat/hard.go:30`, open question RESISTCHECK), SAVESNAP (`saves.Snapshot` not wired into the managed d2bot), HELPERFATAL, RESCUEOBJ, HELLFLOOR, LEVELWP, Nav stuck at Burial Grounds 5459,4549 (root cause unproven), the Wine container leftovers (ENTRYFIX, FLEETSEED, CONTAINERLOG, DOCTORNITS, CONTAINERWALK; only if the container is un-paused), and the SME audit list ([sme/audit-bot-vs-human.md](sme/audit-bot-vs-human.md): Act 1 XP cliff, builds stop at clvl 30, unreachable resist gate, ladder-only runewords).
13. **Contiguous map across adjacent areas (viewer)**: make the stream's map pane stitch neighbouring areas into one continuous map where the game does, like the in-game automap: field to town and between adjacent areas whose edges connect. Real discontinuities (caves, dungeon levels, town portals) stay a fresh map. Needs the area coordinate frames and tile offsets at the exits, a keep/clear rule for the explored history, and changes to the `/map.png` rendering, `mapVer` handling and the page (`cmd/d2dview`). Idea from the user, 2026-10-31.
14. **Replay as video**: `d2dreplay render` exists; a public "best death of the day" clip needs the sanitizer applied to frames. Idea only.
15. **Windows worker WSL2 d2d worker** for fidelity seeds in parallel ([process/d2d-worker-host.md](process/d2d-worker-host.md)); needs d2dslot on a Windows worker. Idea only.
16. **slotnits tail** (L): `capacity.sh` numeric validation, hpfields reset on a new game, killtime zero-line note. Unverified.
17. **townwedge** (card `townwedge`: merged; knob nav.badEdgeTTLSec 300; live-verified: pocket left, 0 'no route from level 1' in a 74 min run, clvl 11, 3673 kills, 0 deaths): permanent town-exit wedge seen in both nav soaks ([probes/navsoak-compare.md](probes/navsoak-compare.md)).
18. **nowall** (merged, card `nowall`): campaign stall ladder 30/30 virtual min, exit status 5, knobs bot.stallWarnMin / bot.stallExitMin ([nowall.md](nowall.md)).
19. **goldfloor** (card [goldfloor](../openspec/changes/goldfloor/proposal.md), proposed): goldfloor analysis merged ([goldfloor.md](goldfloor.md)): spend-to-zero, not low income (income 430-1270 gold/h, sales 80-85 percent of it; spend about equal in every band); biggest leaks: TP scrolls 48-55 percent of spend, mana potions 22-34 percent, 83 percent of NAVSOAK trips spent under 30 gold; Blood Raven gold gate is NOT binding; proposed first change lever A (trip purchase floor + no TP rebuy while poor, +150..400 gold/h) in card goldfloor (proposed). Earlier note: gold stays under the 500 kit line and the belt is empty 33 percent of minutes ([probes/navsoak-compare.md](probes/navsoak-compare.md)).
20. m2-earlybase unblocked: townwedge merged 2026-10-31.

## 3. Needs the user (also the board's Waiting-on-you panel, `tools/needs-you.sh list`)
- The five decisions in [plan/STATUS.md](plan/STATUS.md): licence/GPL review, public character names, grill 0003 (merc and Act 2), upstream PR follow-ups (post the Lut Gholein PR; cast follow-ups wait for the maintainer), stream and ports.
- Kick stream stays OFF until asked. The container installer viewer needs a second port. The NAS autostart for d2-board.

## Parked by grill 0003 (ADR 0011, 2026-10-31)
- Build merc targets (each build's target merc and gear): needed before the keeper rule has keepers (card m3-merc).
- Desert merc in Normal (M4): buy or not, at what gold, survival in Act 2.
- Max-life floor for Andariel: waits for m3-andariel's life-per-phase numbers (cards m3-routeguard, m3-readiness-model).
- Live probes: dead-merc forfeit of the free Rogue, one revive quote, 2 to 3 hire windows, merc Str and Dex own-stats rule, merc state after quit and reload (card m3-merc).
- Readiness model weights and safety margin (card m3-readiness-model, deferred until m3-andariel has won once).
- Card board deployed: /cards on the status board (phone friendly, approve/defer/reject, decisions read with tools/cards-decisions.sh pending); see docs/process/card-board.md
- m1-stashcap merged: stash gold cap (2,500,000, unverified until m1-liveproof), partial deposits, learned from a refused deposit; no knob
