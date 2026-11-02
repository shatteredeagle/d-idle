# Changelog

Every release needs notes here. Newest first; one section per day, grouped by area. There are no tagged releases yet: the
units are the merge commits on `master` (`git log --first-parent`), named by their agent branch (`agent/<name>`).

Marks: **live** = a live result is named in the merge subject or in `docs/` (see `docs/progression/results.md`, `docs/INDEX.md`
section 1); **offline** = unit tests, simulation or fake memory only; **research** = documents only. When a line says nothing
else, it is offline-tested. This file was written from merge subjects and `docs/INDEX.md`; it does not claim more than they do.

## Unreleased / known issues
- verifyzerohit: independent verification of the zero-damage hit trace; game side V run, d2d read only, no PR candidate
- Barbarian skill trace (docs/research/skbarb-trace.md) and emulator runs tools/emu/skbarb_*; coverage skills.bar 0 of 30 to 30 of 30 (calc and physical aspect).
- Amazon skill trace (docs/research/skamazon-trace.md) and emulator tools tools/emu/ama_*; coverage skills.ama 0 of 30 to 30 of 30 (calc aspect, 8 rows with the roll deviation), missiles 12, 28, 40, 41, 85, 87, 205, 206.
- levelsareas: area-level rows of Levels.txt compared with game.exe (1,088 cases, 0 differences); trace docs/research/levelsareas-trace.md
- docs: itemweapons first-pass trace (docs/research/itemweapons-trace.md).
- Assassin skills first-pass trace (table-derived, no ledger rows)
- skdruid: docs/research/skdruid-trace.md, Druid calc differential 0 of 27,800 cases (tools/emu/skdruid_*)
- Docs: branch triage of the 15 unmerged local agent branches (docs/research/branch-triage-20261102.md).
- crfixbundle: bundle 3 (CodeRabbit review fixes: charge-repair base, confidence tags, ground-roll comparison) and the judged comments
- Added docs/research/pr-smite-block-draft.md and tools/emu/sbp_*: the Smite-block upstream candidate built on the private fork with a counted differential against game.exe.
- Independent verification of the Teleport room rule (docs/research/verify-teleport-room.md, tools/emu/vt_*)
- verifyplayerroll: independent verification of the player-roll PR candidate (docs only)
- verifyalign: docs/research/verify-alignment2.md and tools/emu/vfal_run.py (own harness: gate 1200 runs, spawn map 734 MonStats rows, 0 differ)
- stateevrun: docs/research/stateevrun-trace.md, tools/emu/ser_events.py (game.exe swing gate / resolver / damage applier run natively: event order, event 11 vs resistance, event 3 on a miss; counted cases)
- sorcrest2: trace of Inferno, Blaze and Enchant run in game.exe (one flame per do call, Enchant picks the ally else the caster, Blaze's cast makes only the state) and the Fire Ball burst rim on the real d2d build (549 of 16,875 cases differ); docs and emulator scripts only
- reviewprs: review of community PRs 19, 22 and 23 with emulator spot checks and draft comments (docs only)
- verifysmite: independent verification of SMITEBLOCK; block does not stop Smite in game.exe (V run), d2d fix proposed, docs only
- Docs: PR draft for the player-side missile damage roll bundle; det scenario hit_phys.
- Docs: PR draft and population comparison tool for the Nightmare / Hell monster level rule (docs/research/pr-nmlevel-draft.md, tools/emu/nm_pop.py).
- Docs: the monster movement pace PR draft split into two (think paces + Blood Raven + shaman corpse reach; base velocity 75), rebased on current upstream main.
- bundlefinal: final PR body for bundle 1 and credit blocks on its cards
- verifybundle: independent verification of PR bundle 1; ground footprint and burst roll confirmed (base 476 of 3,726 hit cases differ, head 0), docs only
- lockstepbase: d2bot logs `gclock: timescale K` at start when the clock scale is not 1; runbook section 10 for hooked baseline lanes (tooling in the ops repo, not run live).
- Added docs/research/rebase174-20261102.md and pr-bundle-1-draft.md: the four PR candidates rebased onto upstream main after PR 174 and assembled into one bundle branch with counted differentials.
- Lockstep P4 live measurement: with the bot clock scaled the time hook at k=2 and k=4 no longer breaks shop chores or kill rate (docs only)
- crsetup: .coderabbit.yaml revised and schema-validated; CodeRabbit pre-scan runbook (docs and config only)
- docs: credit kit for upstream PRs (credit: card block, tools/credit/credit.py, credits ledger, GPL carry-over sentence in the contributor docs).
- docs/research/coderabbit-limits-20261102.md added (CodeRabbit review limits, upstream batch sizes, bundle budget)
- Research: alignment2-trace.md, game.exe gives alignment 2 to mercs, Shadows and all player-side units, so the undead / Blunt blow bonus gate (player or alignment 2) covers them; tools/emu/al2_run.py native runs.
- Added docs/research/smiteblock-trace.md and tools/emu/sb_* (SMITEBLOCK): a block does not stop Smite in game.exe 1.14d, d2d deviates (V run, no PR candidate).
- botclock: internal/gclock time-scale facility (D2BOT_TIMESCALE, default 1 = unchanged); env.RealClock and the game-clock timeouts use it; guard test for new direct time calls
- Research: the original-game clock hook scales the frame rate exactly up to 4x (100 fps), but the bot's own real-time waits make k=2 yield half the work per game minute and k=4 unusable
- contribid: AGENTS.md no longer tells community contributors to commit as Claude; reproducibility rule and PR template checkboxes
- Added docs/research/pr-skmelee3-d1-draft.md and pr-skmelee3-d2-draft.md: the SKMELEE3 candidate split in two upstream-ready branches with counted differentials.
- A looted bow is taken off a non-Amazon the game wore it on, the sell chore never offers worn or start items, and a removed start weapon is worn again.
- CONTRIBUTORS.md added; Credit section in docs/process/chunker.md
- Fixed: town walk no longer loops on a gate that stalled every approach; a Sorceress with a stray bow keeps casting and no longer spends her gold on Arrows.
- prbuildprices2: repair-gate PR draft and branch (docs only)
- Added docs/research/pr-skmelee3-draft.md and sm3_run.sh / sm3_probe.py: the SKMELEE3 candidate built in the private fork with a counted differential.
- Docs: freeze rules PR draft and prep notes (docs/research/pr-freeze-rules-draft.md).
- Added docs/research/verifypalskill2.md and tools/emu/vp2*: independent check of the Smite, Zeal and Sacrifice deviations.
- Added docs/research/pr-skmelee3-verifier.md: independent check of the SKMELEE3 fork candidate (fine physical, undead/Blunt bonus).
- verifyprices: independent Opus verification of the vendor prices follow-on (docs and tools/emu/vpr_* only)
- docs: verifypaldur independent check of the durability-loss trace (docs/research/verifypaldur.md).
- Added docs/research/sk-zero-hit-trace.md (SKZEROHIT: zero-damage hit effects in game.exe vs d2d, static)
- verifypalaura: independent check of PALAURA (docs/research/verifypalaura.md, tools/emu/vp_check.py)
- SKSTATEEV state-event decomposition: docs/research/sk-state-events-trace.md (static; nothing run)
- verifywaypoint2: docs/research/verifywaypoint2-trace.md and tools/emu/vw2_*: waypoint refusal claim re-verified against d2d's own test binary (docs and emulator scripts only)
- Teleport room rule re-checked independently (verifyteleroom): holds on 150,225 emulator cases; limits and extra findings in docs/research/verifyteleroom-trace.md.
- Docs: upstream PR candidate inventory and ranking for the next five slots (docs/research/pr-candidates-20261102.md).
- docs: verifier report for the curse-value, defender and shield first-pass traces (verify-cursedef.md)
- Added docs/research/palskill-trace.md and tools/emu/pal2*: dispatch of Smite, Zeal and Sacrifice, calc and Smite damage differentials, deviations D1 to D9, bot findings B1 to B5.
- ci: fix board archive research paths and chunker groom privacy-scan false positive (Scripts job).
- docs: palaura-trace, the Paladin aura apply decomposed (area search disc, aurafilter bits, enemy roll and resist, Prayer mana and state 85) with game.exe run under the emulator (docs/research/palaura-trace.md).
- VERIFYFREEZEEND2: docs/research/verifyfreezeend2-trace.md, own emulator harness tools/emu/vfe2_*; game.exe runs the AI think during a get-hit, d2d holds it (D1)
- skiasgame2: docs/research/skiasgame2-trace.md, tools/emu/skiasgame2_rate.py: game.exe attack rate equals d2d over 198,988 cases; sequence-mode -30 and dual-wield rule listed as deviations; ledger F07 updated
- docs: paldur-trace, durability loss per hit and per block traced in game.exe and compared with d2d (docs/research/paldur-trace.md).
- docs: original-game baseline runs 20261102 (4 rows, docs/probes/baseline-runs/20261102/record.md).
- Docs: rebased the three open upstream PRs (#159, #160, #163) on current upstream main, held for a force-push decision; log in docs/research/openpr-log.md.
- Research: lockstep probes P1 and P2 run on a hidden instance: the game frame counter runs at 25.00 fps, a null time hook changes nothing measurable and k=2 doubles the frame rate
- Research: lockstep probes P1/P2 prepared (time-hook proxy builds against the game's own DLL and passes the offline tests); live runs pending
- boardclean: contributor board cleanup, finished items closed and archived (docs/community only)
- Docs: opened upstream PR #171 (monster base velocity 75 percent); posted log in docs/research/openpr-log.md.
- no change
- Docs: opened upstream PR #170 (Nightmare and Hell monster start level); posted log in docs/research/openpr-log.md.
- Decomp item 15 (SKSHIELD): shield and off-weapon gear in the player's stat list traced and run (docs/research/skshield-trace.md, tools/emu/skshield_run.py, skshield_base.py); SKMELEE2's D7 settled, D8 given its game-side input.
- docs: SKDEFENDERS trace (docs/research/skdefenders-trace.md) and emulator oracle tools/emu/skdef_*; docs/coverage ledger row F04 (defender side) and regenerated rule-coverage.md
- Curse value calcs traced and differentialled (skcurseval-trace.md, tools/emu/sk_curseval*.py): equal d2d at every level 1..99; bot finding on Lower Resist vs magic
- docs: original-game baseline runs 20261102 (1 rows, docs/probes/baseline-runs/20261102/record.md).
- Docs: opened upstream PR #167 (monster think paces); posted log in docs/research/openpr-log.md.
- SKIAS partial desk trace of the swing frame count (docs/research/sk-ias-trace.md, tools/emu/skias_bp.py)
- docs/research/geararmore.md: second pass of gear attack rating terms (stats 179, 120 run on game.exe, 0 differ), tools/emu/gar_run.py gar_census.sh
- Added docs/research/verifyresistq.md and tools/emu/vq_*.py: second-pass check of the monster modifier resist adds, curse values and the no-get-hit rule.
- docs: verify-skmana, independent re-check of the skill mana check and payment against game.exe (docs/research/verify-skmana.md).
- Independent verification of the resistance rules (verify-skresist.md, tools/emu/vr_resist.py): no rule fails
- Added docs/research/verify-skmelee2.md and tools/emu/vm2_*: independent check of the player-blow deviations D1, D2, D4, D5, D6.
- docs: VERIFY-AIDEL independent check of aidel and FUN_005a73e0 (0 differ in 352,800 cases)
- SKGEARAR: trace of the gear attack-rating terms, FUN_0057d8a0 run natively (docs/research/sk-gearar-trace.md, tools/emu/sk_gearar.py)
- Added docs/research/pr-nmlevel-verifier.md and its harness: independent verification of the nmlevel candidate.
- no change
- Docs: opened upstream PR #166 (player-side missile damage roll); posted log in docs/research/openpr-log.md.
- bot logs: countable lines explore-giveup, tp-debounce-hold, tp-debounce-flip, kitgate-trip (docs/botlogs.md)
- kit gate by character level: from clvl 8 an incomplete kit makes a town trip instead of holding the character in Blood Moor and Cold Plains
- no change
- docs: Gate 1 PASS review (docs/soaks/gate1-20261102.md, Review 3).
- Docs: hireling trace (mercstats): game.exe's merc level-up, hire offer, resurrection price, experience award, think and attack code equal d2d's rules in 411,196 emulator runs; merc life regeneration and Act 2 auras are missing in d2d (effect U); Cold Arrow, Defiance and Thorns need merc level 6 so low-level offers can lack their skill; the merc accepts Rejuvenation Potions
- Docs: monster skill use trace (monsterskill): monster skill levels, free casting and the Corrupt Archer think run against game.exe and d2d (no deviation proven); usability-gate deviation read only; bot shooter model findings; ledger row CorruptArcher
- Docs: monster AI trace, second pass (monsterai2): ports of the six thinks d2d lacks equal game.exe on 36,000 cases; d2d's andariel_think equals game.exe on 3,830; Fallen flee pace 50, Blood Raven pace and home rules, Fallen Shaman corpse reach proven deviations; the pass 1 mismatch families root-caused (harness labels); coverage ledger monster.ai 23 of 146 rows
- no change
- Docs: opened upstream PR #164 (monster missile launch record and poison units); posted log in docs/research/openpr-log.md.
- Private fork: a monster missile's owner poison reads SrcDamage as game.exe's unsigned byte; the two missile PR drafts follow the third check.
- botsupply: gold per kill follows the monster level (Blood Moor 0.39, not 6), early gold piles are no longer skipped, recovery XP table from the real means, simulator monsters with real life.
- docs: monster-missile PR drafts updated after the re-verifier (owner poison rule, trap sibling, merge order), poison-units PR draft added
- verifywaypoint: independent verification of the waypoint refusal claim (docs and tools/emu/wpv_* only)
- playerdef: docs/research/playerdef-trace.md, trace of the player's life, mana, regen, defence, block and resistance rules with 8 d2d deviations and 4 bot findings (docs only, nothing run)
- uniqmods (research): game.exe pick rules for champion / unique / minion mods, pack and minion counts and Act 1 superunique records traced against d2d and the bot (docs/research/uniqmods-trace.md); no code change.
- monsterhit (docs only): docs/research/monsterhit-trace.md and tools/emu/mh_rows.py (table helper); read-only decomposition of the monster swing on the player, no oracle run, no ledger row.
- docs: monster-missile record PR candidate fixed after the independent verifier (poison units), rebased candidate and Andariel sibling named, PR draft added
- tools/emu/run-on-worker.sh: audited wrapper to run a tools/emu script on the worker host; docs/api-agent-cookbook.md
- Contributor board: the maintainers now claim items their own work covers, close finished ones with a citation, and watch issue comments each cycle (tools/contrib/claim.py).
- tpglitch: town-portal verdict debounced at its single source (kit gate and poverty check); pingpongrate: ping-pong guard resets on XP progress, not on any kill
- docs: licence research for a standalone Game.exe time hook (not legal advice); nothing run
- docs: Gate 1 rerun 2 record and review (XP/h per band, stall windows, HasTP root cause), cards tpglitch, pingpongrate, kitgatelevel proposed.
- docs: research on running d2d and Game.exe from one seed in lockstep (seed pin by save map id, RNG stream, parity, plan P3, verdict); nothing run
- treasure: docs/research/treasure-trace.md (class selection, no-drop and picks, potion, scroll and gold rates per kill for Act 1 and 2, chests; d2d differential re-run), tools/emu/tre_* oracles, coverage ledger row for the treasure classes.
- monsterstats (docs only): docs/research/monster-stats-trace.md and tools/emu/ms_* (stat-row and unit differential vs d2d's rules::monster_stats: 242,220 + 85,213 cases, Normal 0 differ); coverage row monster.stats TESTED-DEVIATION.
- memory rule (ADR 0014): memory writes allowed for setup, instrumentation, diagnosis, fixtures; crutch rule; d2cheat runstart and ledger --since-run
- tools: d2restore snapshot library (snapshot, verify, list, place) and a prefix-keyed cheat ledger (docs/snaplib.md).
- Research: lockstep probe P0 (no game) shows an import-table time hook scales a 25 fps frame gate under Wine; probes in tools/lockstep/
- docs: research on lockstep / speed-up for the original Game.exe 1.14d (no prior art found; proxy-DLL clock hook plan, probes, policy); nothing run
- docs: runbook for the extra original-game baseline runs (docs/process/baseline-runs.md).
- docs: character factory research (docs/research/character-factory.md).
- docs: missilediff (monster missile record, real d2d build vs game.exe, 120,000 cases), tools/emu/mm_diff.py, ledger rows for 22 missile rows
- docs: backend parity audit for resuming the bot on the original game (docs/research/backend-parity.md).
- tools: `d2dsoak bands` (XP/h per clvl band with a baseline verdict), baseline files extended with the whole a14 run, G2 and the Fresha chain plus provenance fields; soak-gates delta spec for the per-band Gate 1 XP clause.
- combat: missile threat model, kill-first for unblockable shooters, no walk-away retreat from runners (knobs combat.threatDodge, shooterKillMin, runnerGuard, standOK)
- Docs: monster AI trace (monsterai): 25 Act 1/2 AI thinks run in game.exe under the emulator and replayed through d2d's rules::mon_think (Skeleton, Goatman, Quill Rat, Griswold, Smith 0 differ; Zombie, Brute, Blood Hawk, Corrupt Rogue/Lancer pace and run deviations); coverage ledger monster.ai 11 of 146 rows
- Research: sor-rest-trace.md closes the Sorceress ledger (skills.sor 14 run-tested, 16 read-only, 0 untested); d2d rolls damage per monster in a Fire Ball / Glacial Spike burst where game.exe rolls once (proven on the real build, no fix yet); oracles tools/emu/sk_rest_*.
- docs: Gate 1 run 20261101 review (stall windows, gap analysis, rerun recipe), cards soakgiveup and povertyflip proposed.
- docs: baseline trust audit (docs/research/baseline-audit.md).
- Necromancer skill trace (docs/research/nec-skill-trace.md) and emulator differentials tools/emu/nec_*; coverage skills.nec 0 of 30 to 30 of 30 (calc aspect), missiles 114, 192, 193.
- docs: monster-missile-trace (Acts 1 and 2 monster missiles, 213 game.exe launches under the emulator), tools/emu/mm_* oracle, ledger line for 48 missile rows
- Added a generated rule-coverage table (docs/coverage) that counts, per data-table subsystem, which rows were differentially tested against game.exe 1.14d, with a CI freshness check.
- palskill: docs/research/pal-skill-trace.md and tools/emu/pal_* (Paladin skill decomposition; undead / demon flags, element damage and aura calcs equal d2d on 734 / 8,640 / 21,760 cases)
- SKABRECHECK: docs/research/skabrecheck.md settles two SKRESIST A/B claims against game.exe (Fire / Lightning Mastery are not pierce; d2d truncates a missile's physical roll to whole points, 84,271 of 101,626 cases)
- Added docs/agents/claude-tracing-guide.md, a guide for Claude users on tracing claims and the agent setup.
- none (docs only)
- docs: Blizzard shard-offset PR opened upstream (#163); posted log and queue row.
- docs: Blizzard shard-offset PR candidate brought to READY (names, Eruption, ruff), draft updated with the verifier's counts.
- The town trip no longer loops when the tome count is stale: an empty portal source marks the tome empty for the next visit (FIXPROGRESS).
- A nearly empty potion kit now starts recovery inside the 15 minute cooldown (recoveryPotionFloor, default 2).
- SKGROUND2: Fire Wall, Hydra and the Blizzard hit area traced; a one-line shard offset fix is a local d2d PR candidate (docs/research/sk-ground2-trace.md).
- Gate 1 rerun on the rebuilt d2d stack: not passed on one seed (progress and belt window); fixes in progress (docs/soaks/gate1-rerun.md).
- forkstats: the private fork's dump stats can show worn-gear property sums and weapon base damage (read only); d2d has no player stats 17, 18, 21, 22 (docs/forkdbg.md).
- docs: PR vetting can run on a second machine (portable differentials, Python checks, clang-tidy, merge check); parity checked, one float parity bug (fused multiply-add on arm64) found and fixed by -ffp-contract=off (docs/review-strategy.md).
- linuxbuild: a Linux builder container (limited CPU and memory) builds the fork binaries and tests; results match the game host's builds.
- oriontune: capacity rule counts I/O wait only when a local disk is busy; worker-capacity doc updated.
- SKGROUND: ground skills traced (Blizzard, Meteor): d2d deviations and bot delay-timer findings recorded in docs/research/sk-ground-trace.md.
- Waypoint trips no longer wait 10 s after arriving (nav.wpCooldownSec default 0, deprecated).
- capdocs: capabilities.md, README, running.md and the sorc skill matrix refreshed (Teleport hops and the Frozen Armor retry proven live, #159 and #160 open, melee, resist and Paladin offline-verified).
- tickpolish: the Discord tick wrapper repairs privacy-guard runs by changing separators only, and renders an empty waiting list without asking the model.
- palfix (offline): Paladin fixes: mana-heavy flag in the build table (hammerdin included), Blessed Hammer band from clvl 18, simulator takes the class's attack names, legacy Paladin table trains Zeal 4 and Prayer, data tables can name themselves and `--build paladin-zealot-lite` exists (docs/palfix.md); not verified live.
- palsme (offline, not live): Zealot-lite table per the SME review (Prayer 3 -> 1, Cleansing and Vigor dropped, Defiance 1 kept, freed points to Might and Holy Fire 2; clvl 10/15/20-24 Prayer/Might/Holy Fire/Defiance/Zeal 1/7/0/0/0, 1/8/0/0/4, 1/10/2/1/4); legality incl. the install check passes on the worker host; docs/palsme.md records the ready cards PALRETREAT, PALBELT, PALBRGATE, PALMERC1, SMEDOCS (policy features paused until the primitives pass live).
- boardgrow (offline, tooling and docs): work archive (docs/community/work-archive.md; `board.py --archive` moves done items out of open-work.md, `--check` fails while one is left); the chunker sync and groom read archived items too, so they are never withdrawn or re-created (an open issue of an archived item is closed once); SKMELEE and SKRESIST archived; 12 new work items (5 verify, 7 decomp). Chunker sync verified live: it closed issues #5 and #6 with one comment each.
- docs: opened upstream PRs #159 (Frozen Armor and freeze-resume bundle) and #160 (curse lowering on immune monsters); posted log in docs/research/openpr-log.md.
- docs: FORKREBASE: the private d2d fork is rebuilt on upstream main as lab2-freezeresume (docs/forkrebase.md).
- verifycurseimmune (research, docs and tools/emu only): independent verifier of the curse-immune PR candidate (own harness running game.exe's FUN_005c3540 / FUN_00645b20 natively and the real `Fight::target_of`): READY, 0 differ on head across 5 classes, hirelings and curse calcs checked; draft text edits applied; nothing pushed or posted.
- skmeleewire (offline, not live): the resource decision now gets the melee model: weapons.txt StrBonus / DexBonus in data.ItemBase (schemaVersion 3, older caches re-extract), combat.meleeBlow / meleeFoe fill Input.Blow / Input.Foe as a pair (nil on any missing input); stock replay unchanged (22 kills / 28 swings / 66 casts), undead and high-defense packs change rung as designed; awaits an early-game soak and drill R1 (docs/skmeleewire.md).
- skmeleebot (offline, not live): the resource simulator's melee rolls to-hit and damage instead of a flat 3 that always landed (internal/resource/melee.go; stock replay 23 kills / 37 swings before, 22 / 28 after), weak() needs to-hit >= 50 % when the model is known, Blunt vs undead raises the melee rung's pack limit; production unchanged until SKMELEEWIRE feeds Input.Blow / Input.Foe (docs/skmeleebot.md).
- upstreamsync (research, docs only): mergeability sweep after #152 and #153 merged upstream (and #155, #156, #157, #158 later the same day), four clean rebased copies of local candidates, fork rebase plan (new branches, no force-push), outdated status lines marked merged. Offline, nothing pushed or posted.
- skmelee2 (research, docs and tooling): the player's plain Attack differentialled against the real d2d build (docs/research/sk-melee2-trace.md, tools/emu/sk_melee2.py, tools/emu/sk_melee2_test.cpp): D1, D2, D4, D5, D6 real, new D8 (off-weapon ED ignored by d2d), D7 needs drill R1; no PR candidate.
- bundlearmor, bundledet (research, docs only): combined PR body for the Frozen Armor and freeze-resume claims as two local commits (docs/research/pr-bundle-armor-resume-body.md; differentials 0, det state hashes 14 of 14 identical, docs/research/pr-bundle-armor-resume-det.md); held for the user's go, not opened upstream; the body's base hash 72132b99 may be stale after upstream main moved.
- skresistbot (offline): bot resist handling per the SKRESIST trace: expFactor floors at -100 (Res -250: factor 3.5 -> 2.0), new Target.Pierce / EffRes subtract pierce only while the total is under 100 (cold 75 with pierce 20: 0.25 -> 0.45; 100 or more stays immune); B3 left as a pinned fallback; not verified live (docs/skresistbot.md).
- chunker (offline, `sync --apply` against GitHub not yet run): `tools/contrib/chunker.py` keeps the work-item issues in step with the board (sync, dry run by default, 10 writes a run cap) and proposes new items (groom, one Sonnet call, proposals only); 35 contrib tests; 3 real groom calls, last proposal empty (docs/process/chunker.md).
- skresistfix (research, docs only): local upstream candidate fix/curse-immune for curses and auras on immune monsters (differential 88,101 -> 0 of 147,628; docs/research/pr-curseimmune-proof.md, draft marked DRAFT); not independently verified, nothing pushed or opened upstream.
- Decomp row 21 (SKRESIST, research and tooling): monster resistances, immunities, curse 1/5 rule and pierce in game.exe 1.14d vs d2d (real build) vs the bot (docs/research/sk-resist-trace.md, tools/emu/sk_resist.py); d2d curse-on-immune deviation proven, fix is the follow-up SKRESISTFIX.
- Decomp row 20 (SKMELEE): the player's plain Attack in game.exe 1.14d vs d2d (docs/research/sk-melee-trace.md, tools/emu/sk_melee.py).
- drills29 (live, harness only): drill D29-C (group W) proves the armorcut retry live on both fork binaries (cut at 114.0 s, retry 1.6 s later, armor never absent; a cut needed a logged debug damage boost); D26-A now passes (Buff pass also runs in the lure loop).
- armorcut (offline; retry proven live in DRILLS29): the bot retries an armor recast that a monster hit cut (up to 3 tries per skill, mana-checked, recast still starts before the data-driven duration) and casts no buff during hit recovery; 3 new unit tests. drills28 (live, harness only): d2d refreshes a set Frozen Armor state; the armorcut code held the armor for 300 game s (53 of 53 samples), no cut seen, so the retry is unexercised live (BACKLOG ARMORCUT2).
- teletune3 (live, DRILLS27): `Navigator.ResetStallMemory()` (drill harness only; the bot never calls it), `why=` field on the `nav plan` debug line, drills D22-BL and D22-S; no walking behaviour change. Stall escape proven live (hops after a recorded stall), D22-PC then D22-P in one process now 229 ticks instead of a 7560-tick timeout; the border-leg gap of teletune2 is replan spread, not the Teleport key.
- drills26 (live, harness only): drill D26-A (group U) proves the Frozen Armor recast in a real fight on lab-armorhit and lab-nexthit2 (identical; armor up 296.1 of 300 s, 44 kills, 0 refills); a recast cut at 232.4 s left the armor down about 3.5 s (BACKLOG ARMORCUT); champion pack and boss check not run.
- docs: FREEZERESUME differential: d2d lets a frozen monster act at the freeze end, game.exe at the end + aidel; local upstream candidate fix/freeze-resume brings the frozen cases from 12,503 to 0 (docs/research/pr-freezeresume-proof.md); independent verifier READY WITH CHANGES (docs/research/pr-freezeresume-verifier.md); not opened upstream.
- teletune2 (live, DRILLS25): Teleport hops now land on the planned walking path and are taken when they skip more than 13 subtiles of it (was 36 subtiles of walk to a straight-line cell); Stony Tomb 211 ticks against walking 254, Halls of the Dead 533 against 578, border leg 717 against 708 (9 over, 0 hops: TELETUNE3); after a walk stall the leg rule is off and mana may fall to 15 percent (new knob nav.teleEscapeReservePct; unit-tested, not exercised live).
- agentprompt (research, docs only): a paste-in hands-off prompt for contributors' own AI agents (docs/contributing/agent-prompt.md), linked from the guide and the work board.
- fixtelekey, teletune (live, DRILLS22 and DRILLS24): the Act 2 Teleport hop now fires in production (the key comes from the bound Teleport skill; towns and levels without Teleport refuse; no key for builds without it); hops only when the walk to the landing cell exceeds 36 subtiles and mana after the hop stays above 40 percent, no idle waits for mana. Near walking speed with the defaults, 18 ticks per hop; pockets and path-aware hops are the next card (TELETUNE2).
- tickfix (offline, two real dry runs): The Haiku dispatcher tick no longer falls back every time: thinking is disabled and the fact check tolerates rewording (tools/discord/tick_post.py).
- roamleg (live, DRILLS23): a mouth-gate hold ends the roam episode in one leg instead of failing ten targets; DW.m3 x3: failed legs 50 to 0, step-aways 62 to 11, bounces stay 0, plan lines 559-664 to 425. Sweep completeness unverified (the drill character kills nothing).
- communitywork (offline, docs and stdlib scripts; nothing switched on): Contributor guide, work board, issue and PR templates, outside-PR review gate proposal and a Discord work-channels draft (nothing switched on).
- armorhit (research, docs only): Frozen Armor's answer to a melee striker on d2d differs from game.exe on every damaging case (95,920 of 108,506); local upstream candidate fix/armor-hit brings it to 0 (docs/research/pr-armorhit-proof.md).
- buildv3 (offline, UNVERIFIED live): Added the opt-in Sorceress table `fallback-lightning-v3` (Chain Lightning from clvl 19) and a Chain Lightning attack band gated by the learned skill; default stays v2 until a live soak; see docs/buildv3.md.
- tickwire (offline, one live wiring test): The dispatcher tick can be written by Haiku (tools/discord/tick_post.py, plain fallback); see docs/process/api-credits.md.
- fixskilllvl (offline, not verified live): +skills (all, class, tab, oskill, elemental, shrine) now raise the cast level the bot uses for mana cost and the known-skill gate; synergies keep base points; stat ids UNVERIFIED live (card open).
- fixmouthloop, fixmouthloop2 (live, partial): one step-away attempt per outer walk and a Fighter hold/back-off after a mouth-gate error; walk timeouts 173 to 0, step-aways 129 to 62 per drill run, bounces stay 0 (DRILLS21); remaining roam leg retries are bounded (BACKLOG ROAMLEG).
- fixmouth4 (live): every navigator world click and movement swing passes one exit gate while a stay guard holds in a cave (clip, away rewrite, bounded hold, step-away re-check); proven live in DRILLS19 (Den of Evil: 0 bounces in DW.m3 x3 and DW.m4 x3); known limit: a blocked away route can make the fighter retry (card fixmouthloop).
- forknext (research): the private fork gets lab-nexthit: Nova and Frost Nova strike a monster once per cast like game.exe (differential 84,779 -> 0 of 100,198).
- readmedocs (research): README sections "What it can do today (honestly)" and "Running it", docs/capabilities.md (capability table with evidence links), docs/running.md.
- apireview (offline): automatic advisory PR review (owner PRs from this repo only, needs the ANTHROPIC_API_KEY secret) and an optional Haiku tick helper; see docs/process/api-credits.md.
- fixmanamodel (offline, not verified live): the bot's mana model follows game.exe/d2d: early pools regen 1.2x-1.8x slower than the old 100 s line (one `ManaPerFrame256`), casts are booked when paid (`ManaBook`, per backend), mana potions arrive over 5 s; `restMaxSec`, `restStartManaPct`, `castsPerMonster` and the Hell `ManaBurn*` constants are not retuned (soak review pending).
- actgate (research): draft per-act Sorceress skill gate (verdict rows from the traces and DRILLS11 to 15, gate rule, ordered cards); proposed soak-gates delta is text only, nothing applied to a spec.
- fixcastgap (offline): the bot waits one cast length per skill between casts: Lightning and Chain Lightning 890 ms on game.exe (650 ms on d2d) instead of the flat 650 ms (new knobs combat.castPerSkill, castMarginMs, castFCR); live proof still open.
- drills15 (live): first live skill-smoke drills (group R, 13 drills on d2d, game.exe not run): Teleport, mana, missile and armor traces compared with d2d; deviations of the traces confirmed with numbers (no room limit, blocked target pays, lands on monsters, merc not warped, cut cast keeps mana, no skill delay, Energy Shield absorbs 1.0); drill harness only, no product change.
- fixbuff (offline): the bot recasts Frozen Armor and the other armors during a long fight (before: only after a town trip or new game), reads the armor state on d2d, and a stale skills cache is re-extracted; live proof open (agent/fixbuff).
- drills13 (live): live proof of fixrecast (Frozen Armor recast at 114 s / 126 s, real duration 120 s / 132 s, a recast refreshes the state) and the M1 mana trace (d2d pays 1 tick after the command, 7 ticks before the action frame, a cut cast is not refunded); finds a stale tables cache disables the fix and that Buff is only called on needBuff (agent/drills13).
- skarmor (research): SKARMOR trace of the Sorceress armors and Energy Shield (docs/research/sk-armor-trace.md, tools/emu/sk_armor.py): Frozen Armor's striker effect follows the freeze rule in game.exe (V run), d2d stuns and chills every striker (D1); bot finding B1 (buffs only recast on needBuff edges).
- drills12 (live): live proof of fixcrowd: the 3-adjacent portal drill passes at crowdAdjacent 8 (Safety chicken, scroll kept); the control still casts; the drill measures in the bot's own unit (agent/drills12).
- fixlair (offline): Level data models Teleport 0/1/2 (lair = allowed on a clear line, towns refuse); comments corrected; no behaviour change, the Duriel script still walks (docs/fixlair.md, agent/fixlair).
- Teleport decomposition (SKTELE, research): game.exe's destination rule traced and run under the emulator; d2d deviations and bot findings listed (docs/research/sk-teleport-trace.md).
- feedtick2 (offline, not posted): the whole tick report is posted (cap 3800 chars, was 900); over the cap it is cut at a line with a final "(cut: N chars)" line (tools/discord/feed.py, agent/feedtick2).
- forkdbg (live, lab only): four lab debug verbs (`debug mana full`, `debug states set|clear`, `debug mon set`, `debug cast`) in the private fork branch `lab-debug-skills`, and `internal/d2dctl` `SkillDebug` client that writes the cheat ledger first (an unwritable ledger sends nothing); each verb proven once on a hidden slot, the M1-M5 skill drills not run yet (docs/forkdbg.md).
- openmods (research, upstream PR): boss and minion spawn stats opened as psi29a/d2decomped PR #157 (user go 2026-11-01); gates and differential re-run before opening; posted log in `docs/research/pr-mods-evidence-addendum.md`.
- auditfix (offline): README states the repo holds no game files; docs/fixtures.md describes the 5 recorded fixtures; `tools/ci/gamecode_guard.sh` (in `check.sh`, with `gamecode_test.sh` and a size allow list) fails on pasted x86 listings, Ghidra pseudo-C, hex dumps, game table headers and docs files over 256 KB; leakcheck gains the `labpath` class for docs/probes and `tools/ci/scrub_paths.py` replaced 801 worker paths in 105 probe logs with placeholders (same line counts; docs/process/gamecode-guard.md).
- brandfinal (offline): d.idle brand = cursor concept (user decision 2026-11-01); final asset set (SVG, PNG renders, favicon.ico, banners) in `assets/branding/final/` with a stdlib-only PNG renderer (`assets/branding/render.py`); nothing applied to the README, web pages, GitHub or Discord yet (docs/branding.md).
- feedtick (offline, not posted): `tools/discord/feed.py tick` builds the dispatcher tick report (running, merged, launched, upstream, blocked, waiting) as one `#progress` post, dry run by default, refuses an idle tick; docs/process/feed-after-events.md, feeds.md (e).
- soak-minutes-virtual (offline, not verified live): soak-gate minutes are virtual (game) minutes (user decision 2026-11-01); `d2dsoak` reports the Gate 1 belt window (BELT, over 5) and no kill and no XP stretch (NOKILL, over 10) in virtual minutes; Gate 1 soak 2 now fails both; spec delta in the card, applied at archive.
- modsprep (research, no live run): boss-stat upstream candidate prepared: draft v2 (verifier edits: champion life 1.1-2.1x, traced follow-on callers, area-level limit, siblings), det state hashes on real patched vs base d2d builds (14 scenarios on one hidden slot: 10 equal, 4 differ only by the claimed unique / champion life and minion level, 0 unexplained); nothing posted upstream.
- fixrecast (offline, not verified live): Frozen Armor lasts 120 s (level 1) and 132 s (level 2) but was recast at 150 s, leaving it down about a fifth of the time; armors are now recast `combat.buffMarginSec` (new, 6 s) before their Skills.txt duration, Frozen Armor by its state (id 10), `buffRecastSec` kept as upper bound and fallback; `docs/fixrecast.md` (agent/fixrecast).
- fixcrowd (offline, live proof pending): DRILLS11 F18, the melee cast guard counted attackers within 3 subtiles and never fired live; new knob `combat.crowdAdjacent` (default 8 subtiles, the live zombie reach) replaces the constant, decision unchanged; the weak-hit premise stays unverified; `docs/fixcrowd.md` (agent/fixcrowd).
- fixportalmelee (offline, not verified live): a town portal cast with 3 or more adjacent attackers is cut by melee (Gate 1 soak 2 death at clvl 4); `Decide` now drinks a ready potion, else chickens, and `AbortFight` leaves the game; fewer attackers, quiet trips and all defaults unchanged; `docs/fixportalmelee.md` (agent/fixportalmelee).
- feeds (offline, one real post to #progress): `tools/discord/feed.py` builds soak, drill, upstream, pr-status and release summaries from our docs or flags (dry run by default, `--send` posts, refuses a soak without a verdict, validates the upstream and release URLs, fails closed for a channel without a webhook file); tests in `tools/discord/tests/test_feed.py`; rule `docs/process/feed-after-events.md`; not wired yet: CI-failure feed (needs an Actions secret) and a `#d2d-upstream` webhook (agent/feeds).
- fixtalk (offline, not verified live): the town talk tables (which row for which NPC and pick, NPC class per act) and the dialog adapter with the Akara reward retry moved from cmd/d2bot (package main) to internal/talk with table tests, behaviour identical; drills D8.q and D8.w now call the bot's own talk path; `docs/fixtalk.md` (agent/fixtalk).
- fixf17 (offline, not verified live): DRILLS9 F17 (DW.m1 BLOCKED in run 1 of the full F group) is a harness defect: the lab stepped the game with `Client.Step`, the navigator's Pacer clock lagged and the next walk's first poll jumped by the lag, expiring its 2 minute deadline; `lab.Tick` re-syncs the Pacer, the waypoint trip syncs and logs the lag; no product change; `docs/fixf17.md` (agent/fixf17).
- fixstall4 (offline, not verified live): the NPC approach route walks legs along line of sight (the old 12-step waypoints cut the camp block corner; reproduced on the real grid), routes at every stall instead of odd ones with a blind side step between, detects a stop mid-round after 1.2 s, grows the budget from the planned route length and reports the final position (DRILLS8 F13, H.pause); `docs/fixstall4.md` (agent/fixstall4).
- fixbigfiles (offline): five DRILLS9 run logs (1.0 to 1.1 MB) broke the public CI size gate; capped to about 218 KB in place, `d2drill -cap FILE` added, check.sh now fails probe results over 512 KB (docs/fixbigfiles.md).
- fixstatus (offline, board update verified): `tools/agentstatus/agentstatus.py` was a shell script under a `.py` name and died with a SyntaxError; it is now a Python stub that execs `ops.sh`; test `tools/agentstatus/stub-test.sh`, notes `docs/fixstatus.md` (agent/fixstatus).

- fixrun (offline, live proof pending): the d2d run-toggle belief is reset on every game start in both lifecycle paths (the d2denv path already was), the bot `minute` line logs `runBelieved`, `stam`, `stamMax`, `runKeys`, `runResets`, `walkingRaw`; the 400 ms MoveTo stall wait is unchanged; recipe in `docs/fixrun.md` (agent/fixrun).

- DRILLS5 F4 fix (offline, live proof pending): `Navigator.Sync` plus `combat.Syncer`; `Fighter.Clear`/`Kill` sync the map first, so the first fight after a position jump (debug goto, portal or waypoint arrival, restart) no longer calls every monster unreachable. F3 (poor melee character rests instead of drinking mana) is the intended economy rule, now pinned by a test. `docs/fixmisc.md` (agent/fixmisc).
- fixstatic (offline): Static Field is dropped for fresh characters (user, SKILLINV F1): the opener and cold-immune fallback stay only for a character that already learned it, the Andariel card no longer claims it, and the bot `minute` line carries `table=<skill table>` (d2dsoak prints it), [fixstatic.md](docs/fixstatic.md).\n- fixemptyplan (live proof PASS x3, DRILLS11): a goal that snaps onto the player's own cell is the typed `ErrAlreadyThere` (matches `ErrNoPath` too); back-off candidates that snap onto the player are skipped so the fight falls through to its next option, approach goals count as arrival, `move` no longer logs `move failed` for it (soak 2 F4: 1,191 `empty path` lines), [fixemptyplan.md](docs/fixemptyplan.md).
- fixmeter (offline, live proof pending): `d2dcombat.Attach` builds the combat Meter (off until `Enable`), which now derives damage dealt and kills from hostile monster life and prints `n/a` for hits, misses and anything d2d cannot supply (DRILLS5 P28 BLOCKED note), [fixmeter.md](docs/fixmeter.md).
- fixstall3 (offline, not verified live): the NPC grid route falls back from clearance 1 to 0 so the H.pause stall cells get a route (DRILLS7 F10, regression test on the real camp grid), the bot and the drill harness build the town on `d2dtown.Input` so the portal entry really skips the pre-walk (F11), and the approaches-ignore reason carries geometry for the open D-sync.wp question (F12); `docs/fixstall3.md` (agent/fixstall3).
- fixstall2 (offline, not verified live): the NPC approach plans a route over the level collision grid after a stall (blind side step stays as fallback; DRILLS6 F7), and the town portal entry no longer walks to the portal first when the input's Interact walks itself, so a pinned character takes the portal it opened (F8); `docs/fixstall2.md` (agent/fixstall2).
- fixstall (offline, not verified live): the d2d NPC approach detects a stalled walk after 1.2 s and sizes its game-time budget from the distance (DRILLS5 H.pause: Kashya walk 16 to 38 s); a plain `GoTo` stalled in a pocket leaves by town portal instead of ending in ErrStuck after the full ladder (D6.d); `docs/fixstall.md` (agent/fixstall).
- Discord server built (docs only, **research**): `docs/community/discord/` plan, server, roles and feeds now record the live state of 2026-10-31 (hidden STAFF category, Announcement channel, AutoMod, feeds, the bot keeps Administrator as the user's accepted risk) and the remaining user steps; `server.json` gains STAFF/`discord-updates` and `"type": "news"`, `apply_server.py` accepts it (agent/discorddocs).
- Discord feed webhooks from private files: `common.load_webhook_url()` (env, then `~/.local/share/d2idle/discord-webhook-<channel>` mode 600, then keyring; same fail-closed rules as the bot token, URL shape and channel name checked); `notify.py` uses it; `D2IDLE_DISCORD_HOME` overrides the directory so tests never read real files; headless recipe in `feeds.md` and `server.md` (agent/discordhooks).
- Discord ops bot (offline): long-lived bot model with least privilege (`d.idle-ops` role, allowed and always-ask lists in `roles.md`); `tools/discord/common.py` `load_bot_token()` reads env, then a private file (`~/.local/share/d2idle/discord-bot-token`, mode 600, no symlink, own user, private parent, fail-closed), then the keyring; `apply_server.py` uses it; setup and rotate/revoke in `server.md` (agent/discordtoken).
- Stream map contiguity (offline): the d2dview map pane draws one stitched map across levels the game puts on one automap layer (town and wilderness, Act 2 desert, Kurast, Harrogath fields), caves and dungeons stay alone; act-frame positions, swap-in map image, trail rule; rule and live check in `docs/stream/map-contiguity.md`. Live check pending (agent/mapitem).
- 2026-10-31 README refreshed (status on d2d, how we check it, layout, retired Game.exe gates) and the project name styled **d.idle** (lowercase d) in public prose, the setup wizard and the container templates. Docs and UI text only (agent/readme).
- Discord roles guideline (research, docs only): `docs/community/discord/roles.md` (principles, role set with optional `mod`, permission table, safety settings, raid runbook, sources); server.md, server.json and plan.md aligned (attach files only in two channels, `member` raid-fallback permissions, `optional_roles`). Support pages were not re-readable (403): official claims marked unconfirmed (agent/rolesdoc).
- Discord drafts (research plus tooling, nothing created or posted): `docs/community/discord/` (plan, layout and `server.json`, rules, welcome, FAQ, templates, feeds, public-page proposal, maintainer message draft) and `tools/discord/` (`notify.py` webhook notifier with a fail-closed sanitizer, `apply_server.py` dry-run-by-default server setup). Offline-tested against local mock servers (agent/discord).

- Container dashboard: fleet slots are server instances (fixes 'No Instances' on the NAS), per-slot reverse proxy to the current d2bot address, real fleet status from the bots, per-slot view-only game video (`/api/v1/instances/{id}/viewport/stream`, `xwd` capture, no new package) and a Game view card with reconnect. Offline-tested (agent/fleetview).

- Container soak: `d2soak sidecar` (D2FLEET_SOAK=1) judges each fleet slot inside the d2idle container; `tools/soak/contsoak.sh`, docs/nas-container.md. Offline-tested.

- Repo diet (ARCHIVE): `cmd/spike` deleted; eight one-off live rigs moved to `tools/rigs/` (`//go:build rigs`, `tools/live.sh` still finds them); stale planning docs moved to `docs/archive/` (see its README); `tools/ci/doclinks.py` checks doc links.

Source: `docs/INDEX.md` sections 1, 2 and 4 (as of 2026-10-29), `docs/BACKLOG.md`.

- No 2 h or 24 h soak has ever completed; the longest unattended run is about 1 h 15 (a 90+ min managed run is recorded in `docs/launch.md`). The 24 h runbook, `soak24.sh`, `m24` profile, `d2bot --supervise`, panic recovery and `d2doctor` have never run live.
- Acts 3, 4 and 5, Nightmare and Hell: code and simulations only; no step has been played by the bot. Immunity, affix and missile reads for Nightmare/Hell are unverified.
- Act 1 is not played through end to end by the bot: Blood Raven, Tristram chain, Cain rescue, Smith and Andariel are unproven live. Smith was never found live (the preset 754 anchor is unverified); Andariel is gated off for lack of potions.
- Act 2: staff placement, tomb 73, Duriel, Tyrael and the Summoner kill are unproven (the Arcane maze is not explorable by nav alone); the real Act 1 to 2 hop was never walked.
- Mercenary: revive, equip and farming runs are unverified; hire rows were probed live only for Kashya and Greiz.
- Classes: Barbarian has no live run; Druid and Assassin reached only clvl 4; Amazon ammo-low and empty-quiver paths not run; Necromancer revive, Corpse Explosion and Weaken/Lower Resist state ids are open live checks.
- Fleet and container: the stop chain (`docker stop` to Save and Exit within the 70 s grace), the 4-slot hour, N games per container and the setup wizard with a stranger's files are untested live.
- Stream: Kick OAuth title flow and Home Assistant deployment not run live.
- Idle-game layer (upgrades, prestige) is parked by the user until the bot plays well.
- Review fixes: all High findings of `docs/archive/review-overnight.md` (H1 to H3), `docs/archive/review-day.md` (H1 to H5) and `docs/archive/review-evening.md` (H1, H2) are marked fixed offline in INDEX section 4 (source S: read from commit messages, none re-run live). Still open per INDEX: evening M10 (resist penalty possibly applied twice in `rules.go`, `hard.go`, `d2/player.go`) and some Lows (for example `soak24.sh --for 1h30m` arithmetic) have no fixing commit; the review-day H3 chicken-loop fix is not re-checked against a Diablo or Baal attempt. The fixes for review-day H4 and evening H1 and H2 are live-untested.
- The four merge-risk reviews (overnight, day, evening, night) are archived in `docs/archive/`; their open findings, fix commits and live checks are one table: `docs/open-findings.md`.
- The full prioritised list of open live checks is `docs/INDEX.md` section 2 (P0 gates the alpha: 2 h soak, then 24 h; Andariel; Summoner; merc; quest events; Larzuk/Charsi; Nightmare/Hell probes; cube chore; farming runs; fleet; wizard).

## 2026-10-29

### Bot core and failsafe
- `agent/panicsafe`: d2bot recovers from panics instead of dying.
- `agent/logclean`: log deduplication and a one-line minute summary; supervise and quest polling lines kept.
- `agent/nopause`: the bot no longer pauses itself on low life; the failsafe auto-resumes; the planner no longer picks far-ahead goals.
- `agent/restartfix`: a planned restart counts only once against the failsafe budget.
- `agent/chickfatal`: an unplanned game loss restarts in managed mode; `Flow.BeforeSave` hook.
- `agent/stallfatal`: a rung-4 restart is no longer fatal; clearing idle limit; first stall rung skips the target.
- `agent/loopfix`: blocked inserted steps are dropped; a fruitless explore gives up.
- `agent/botgaps`: stall recoverer, GuardSave, NearTown, Frenzy pair, far NPC.
- `agent/harden`, `agent/harden2`: fixes for the overnight review (H1-H3, M1, M2, M4) and the day review (H1-H3, M3, M6 to M8).
- `agent/lows`: d2bot shuts down saving first, log rotation, flee chicken, guard applies to allow-listed characters only.
- `agent/race`: Recycler lock, no nested ladder escalation, clean `-race` sweep.
- `agent/chickloop`: heal-first gate in town, waypoint to town, pause on re-entry loops.
- `agent/stopchain`: `docker stop` / SIGTERM goes through Save and Exit within the fleet grace; Save happens before the runner is cancelled (live untested).
- `agent/gatewire`, `agent/fuzz`, `agent/coverage`: entry gates wired; fuzz and new tests for unattended-run paths, which found and fixed a grid area cap, a `SetKnob` NaN and a nil snapshot in `ToTown` (direct commit a305319).
- `agent/integrate2` (checkpoint): quest memory and boss adapter wired, game recycler between goals, leveler pause with a visible reason, class-aware start.
- `agent/doctor`: `d2doctor` health checks (config, paths, tables, helper, token permissions, load, clock, difficulty line).
- `agent/chainfix`: the resist gate is rechecked at campaign start; MaxDeaths auto-resume.

### Classes and combat
- `agent/friendly`, `agent/friendly2`, `agent/flive`: `IsFriendly` for every class (pets, mercs); callers use it; rest loop and hostile-near fixes. Live friendly results, `cmd/friendlive`.
- `agent/summon`, `agent/necrolive`: Necromancer summon upkeep, curses, Bone Armor by state. Live results on Necki (Bone Armor state 14, skeleton cap, golem), mage hotkey, army skips rest.
- `agent/amalive`: Amazon live results (19 kills), shop stock pages fix, ammo bought before shopping. Live.
- `agent/assalive`, `agent/druidlive`, `agent/druidassa`: Assassin and Druid builds, profiles, pet and trap upkeep; live runs of 20 to 26 min to clvl 4 only.
- `agent/palalive`, `agent/barblive`: Paladin live results (Palli clvl 13, 63 kills); Barbarian build and profile manaReserve with live results recorded in docs (INDEX still lists no Barbarian live run: see Unreleased).
- `agent/classcombat`: Necromancer corpse skills (Revive, Corpse Explosion), assassin trap planner.
- `agent/casterfix`: caster staff swing on weak packs, nearTown, Den kit floor.
- `agent/fightwire`, `agent/missiles`: curse danger and retreat, missile and curse readers, sidestep hook, fight telemetry (reads unverified).
- `agent/affix`: monster flag reader, affix resists, danger rules, entry gate (flag layout unverified).
- `agent/gear`, `agent/gearwire`, `agent/armordata`, `agent/scorewire`: weapon and ammo model, build tables, armor data, leveler gear scorer uses resist/FCR/ethereal helpers.
- `agent/endbuild`, `agent/builds80`: endgame build tables for all seven classes registered opt-in, Smite weapon fix, ResistGate.
- `agent/hellfix`: mana-burn chicken and hard-tick order fixes.
- `agent/diffp2`, `agent/wirefarm`: Nightmare/Hell rules in the fight loop (Normal unchanged), tuning, death telemetry, `runs:NAME:N` goals.

### Town, loot and items
- `agent/townfix`: greeting stall, affordability-aware advice, futile-trip memory, jobs kill, ring retry (from live soak findings).
- `agent/scrollfirst`: buy the town portal scroll before potions.
- `agent/itempick`: runeword bases, gem keep, charm filter, gear score helpers.
- `agent/cubejobs`: cube chore (gem and rune upgrades), knob `Bot.CubeJobs`, default off (simulator only).
- `agent/stashfull`: full stash and pack handling; `SetBuild` forwards to the judge.
- `agent/lootpotion`: potion pickup uses the build-aware belt plan.
- `agent/yieldwire`: nav Yield wired to a bounded pack clear.
- `agent/mercread`, `agent/wireup`: merc reader and merc chore, EntryGate in the planner, NM Act 1 floor 36 (hire rows were probed live earlier).

### Navigation
- `agent/navstall`: LevelTo guard fix, waypoint activation in passing.
- `agent/tombfind`, `agent/tombhint`, `agent/tombnav`: waypoint icon column, tomb level hint, far probe, gate click fallback, waypoint load wait (partly live in Act 2 runs).
- `agent/teleexplore`, `agent/arcane`: teleport-aware explorer, island-hopping teleport planner for the Arcane Sanctuary.
- `agent/portalnav`: `openPortal` on a class-aware grid, exit-tile cost (live Act 2 context).
- Direct commit f5b7197: hop from Burial Grounds and Cave Level 1 via Cold Plains, found in a live soak.
- `agent/wpcheat`: the waypoints cheat opens act tabs via quest words. Live verified.

### Quests and acts
- `agent/earlyfix`: Den of Evil quest word gating, walk home without scroll, Akara greeting, area hysteresis (the Den claim itself was live-verified on 2026-10-28).
- `agent/act2live`, `act2live2` to `act2live6`: Act 2 live results 1 to 6 (Radament, Book of Skill, cube Transmute of staff pieces, Arcane portal, waypoint rows, Canyon, real tomb found). Live, Navi.
- `agent/act2code`: real Transmute, quest items stay in pack, TakeWaypoint walk, Act 2 stall recovery.
- `agent/act3plan`, `agent/act3data`, `agent/act45data`: Act 3 to 5 plan docs, quest data, task nodes, boss scripts, nav hints.
- `agent/actwire`, `agent/actbot`, `agent/act45bot`: campaign templates, Acts 2 to 5 wired into d2bot, `normal_all` covers Acts 1 to 5.
- `agent/act1gaps`: Act 1 quest objects (tree, cairn stones, cages), Akara and Charsi rows.
- `agent/nmcampaign`, `agent/nextdiff`, `agent/nmsim`, `agent/questevent`: Nightmare and Hell campaigns chain after completion; manager-level chain simulation; held goals persist; quest events resume held goals.

### Bosses
- `agent/bosslive`, `agent/bosslive2`: Countess approach, locate, fight and loot under god mode, preset anchor fix, critical-life escape, `bossrun --god/--nokite`. Live, Combi, god mode.
- `agent/bosswire`, `agent/bossrun`: boss script robustness, potion gates, `cmd/bossrun` single-script runner.
- `agent/bossfix`: pinned-kite fix, cursed retreat, weak-pack strictness, Smith preset anchor, belt-aware potion floor (pinned-kite fix not re-run live).
- `agent/sealsim`, `agent/sealfix`: fight-loop simulation of seals and Baal waves; seal boss timeouts continue and resume from the nearest node, death checks.

### Resource and poverty
- `agent/manapoverty`, `agent/poverty`: energy rows, cheap rest, poverty advice, flee without potions, poverty mode, resource knobs.
- `agent/danger`: poverty pack-danger limit, act-aware starveArea.
- `agent/earlysorc`, `agent/levelrate`: free heal trips from the Den, caster starvation, planner floor so low areas are not fallback farming at high level.
- `agent/holdrun`: locked runs are held; town belt by build.

### Fleet, container and setup
- `agent/slotgame`, `agent/botlaunch`: bot-owned game per fleet slot (fixes the Yama sibling problem); fleet launches and supervises `d2bot` per slot.
- `agent/fleetfix`: 70 s stop grace, goals marker after enqueue, StopGame, slot instance env (live untested).
- `agent/image`: container image carries `d2fleet` and `d2doctor`, wizard default CMD.
- `agent/wizard`, `agent/wizardwire`, `agent/wizui`, `agent/wizgoals`: setup wizard API mounted in the server, wired into `d2fleet`/`d2dash`, goals picker, container user-test checklist.
- `agent/reapfix`, `agent/reapgo`, `agent/winefix`, `agent/wineleak`: Wine reaping with layout-aware prefixes, safe winedevice reaping in Go at managed stop, wineserver fd investigation (steps, not a leak). Shell dry run only.
- `agent/helperleak`, `agent/helperleak2`: all Win32 calls in the d2input helper on one locked OS thread, thread heartbeat.
- `agent/soakpid`: per-instance soak process matching.

### Soak and tooling
- `agent/soaktools`, `agent/soak24`, `agent/logparse`, `agent/dashwire`, `agent/dashalerts`: soak report level, status and alerts files, 24 h runbook and `m24` profile, JSON-log fight/death parsers, `--soak-dir` and dashboard alert strip (never run live).
- `agent/probes`: prebuilt live probe tools (runewords, monflags, mercdump, missiledump, wpdump); never run.
- `agent/viewhang`: d2watch per-client stream writer, reconnect on load, hidden-tab pause.
- `agent/configdoc`: `docs/knobs.md` reference of tuning knobs, with a generator.
- Cheat allow-list gains Necki; Palli, Palli2, Amazi documented (user approved).

### Docs and research
- Reviews: `agent/review` (overnight), `agent/review2` (day), `agent/review3` (evening) merge-risk reviews.
- Research and doc upkeep: `agent/mercresearch`, `agent/diffresearch`, `agent/itemresearch`, `agent/docindex`, `agent/docsync` (INDEX and PLAN status refresh), backlog refreshes for the dispatcher cron.

## 2026-10-28

### Bot core and failsafe
- `agent/bot`, `agent/core`, `agent/wirebot`: bot runner, death policy, profiles, `cmd/d2bot`; verified death handling and deflaked failsafe tests; d2bot on the unified server. First live runs.
- `agent/sentinel`: out-of-process failsafe sentinel with rules and drills; ignores quest-log button flag 0x11 (direct commit). Live drills.
- `agent/livebot`: managed mode live without setcap, death and IsDead fixes, soak findings. Live.
- `agent/integrate`: resource policy, task layer and quests wired into the bot; a live 31 minute campaign run to clvl 6. Live.
- `agent/infra`: the d2input helper died after about 2000 commands (callback table overflow); fixed, with an exit and crash log and a self-healing client; `tools/helper.sh` is per-instance.
- `agent/ci`: CI workflow, hygiene checks, release tarball script.
- `agent/leakhunt`, `agent/soakfix`: fd study (no real leak), game recycler, soak warm-up and fd plateau.
- `agent/state`, `agent/event`, `agent/obs`: state and event packages, metrics registry, soak harness.

### Classes and combat
- `agent/combat`, `agent/combat2`: fighter, targeting, skills and hotkeys, death detection; verified death data, unreachable targets, chicken. Live.
- `agent/combatclass`, `agent/levelerclass`: data-driven class combat, skill binder, aura upkeep, Paladin melee loop, class builds, skill-id read-back, gear score, weapon swap.
- `agent/classes`, `agent/classpal`, `agent/classnec`, `agent/classama`: class plans and ranking; Paladin, Necromancer and Amazon smoke test findings, `cmd/classsmoke`. Live smoke runs.
- `agent/leveler`: stat and skill spending, equipment, level-run planner.

### Town, loot and items
- `agent/town`, `agent/town2`: town chores, rebuy, loot pickit, valuables, live buying and portals, seed-independent town. Live.
- `agent/townfix`: Akara Den claim fixed, full-pack purchase fix. Live-verified.

### Navigation
- `agent/map`, `agent/nav`, `agent/nav2`, `agent/navfix`, `agent/navfix2`: world-map reader (town verified live), A* navigation and exploration, seed independence, waypoints, dungeon transitions verified 20 consecutive times, re-plan on level change. Live.

### Quests and acts
- `agent/quests`, `agent/memquest`, `agent/liveq`: quest state reader, NPC dialog primitive, act hops; in-memory quest and waypoint source; live verification of quest, waypoint, act and difficulty save state. Live.
- `agent/cheatq`: quest, waypoint, act and difficulty save edits via the logged cheat tool. Live.
- `agent/act2`: Normal Act 2 campaign, cube contract, Radament, Summoner and Duriel scripts.
- `agent/tasklayer`: task layer (step graph, planner, executor, simulation).

### Bosses
- `agent/bossruns`: Act 1 boss script framework and scripts for Blood Raven, Countess, Smith, Andariel (live results arrive 2026-10-29).

### Resource and poverty
- `agent/resource`: pure mana and health resource policy package.

### Fleet, container and setup
- `agent/fleet`, `agent/serverfix`, `agent/server`, `agent/wire`, `agent/dash`: fleet supervisor and party orchestrator, unified instance-scoped HTTP server, dashboard UI, `d2dash` and `d2sentinel` on it.
- `agent/container`, `agent/containergame`: headless container spike, then the real game and bot run headless in the container under plain Wine. Live in the container.
- `agent/setup`, `agent/wizardui`: setup wizard backend, `d2doctor`, wizard front-end (mock-tested).
- `agent/multi`, `agent/launch`, `agent/lifecycle`, `agent/memfd`, `agent/oog`, `agent/partylive`: per-instance prefixes and Xvfb displays, subreaper launcher, game lifecycle in Go, fd passing under Yama, verified out-of-game flows, LAN host and join over loopback. Live.
- `agent/input`, `agent/mem`, `agent/data`, `agent/saves`, `agent/replay`, `agent/data2`: helper protocol v2, typed memory readers, MPQ reader and tables, save snapshots, fixture and canary tooling (live or real tables).
- `agent/paths`: data-path migration with legacy fallbacks.

### Soak and tooling
- `agent/cheat`: logged test cheats, cheat ledger, `d2cheat` (throwaway characters only).
- `agent/watch`, `agent/viewport`, `agent/display`: `d2watch` live viewer, capture and stream library, status hub. Live display.
- `agent/moveboard`, agent status board and hub, `live.sh --bg`, `jobs.sh`, `devhealth.sh`, mute watcher (tooling).
- Stream: `agent/streamctl`, `streamha`, `streamfx`, `selsync`, `scenepage`, `scenestream`, `streamdesk`, `streamgames`, `kicktitle`, `kickapi`: stream control service and scheduler, Home Assistant package, overlay, scene page, tray app, Kick title library. The Kick scene ran live; the OAuth title flow did not. Direct commits add the Kick streamer (Radeon h264_vaapi) and an ingest URL fix noted as verified live.
- Public page: `agent/publish`, `publish2`, `publish3`: public status page generator with fail-closed sanitizer, D.idle name and footer, gated GitHub Pages push, live-now link.
- `agent/milestones`, `agent/milestones2`: alpha, beta and good-bot definitions and tracker (docs).

### Docs and research
- `agent/plan`, `agent/goalcat`, `agent/goals`: leveling strategy, goal catalog, goal manager, API and embedded dashboard (simulation runner).
- `agent/progdesign`, `agent/prior`, `agent/priorart2`, `agent/net`, `agent/review`, `agent/hacheck`: progression design, prior-art specs, multiplayer feasibility, plan review with gotchas and roadmap, rebuilt HA stream dashboard.
- Project skeleton (88d22fb): memory-read probe, input helper, memory reference, plan and agent rules. Agent rules cover cheats, muting, disk policy, commit-often and the four project agent types.
- feeds2 (live): CI job `notify-failure` posts failed non-PR runs to `#progress` (secret `DISCORD_PROGRESS_WEBHOOK`, `workflow_dispatch` input `feed_test`); the repo webhook no longer sends `workflow_run`; `#d2d-upstream` webhook added and `feed.py upstream`/`pr-status` post there (agent/feeds2).
- palprobe: profile id string (LookupProfile errors for an unknown class or id), explicit per-class gear preferences, classRow error for an unknown class; Paladin probe tests and docs/classes/paladin-probe.md
