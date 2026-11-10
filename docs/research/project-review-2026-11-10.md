# Project review and true-up, 2026-11-10: direction, the week since the last review, backlog, research fit, beta path, efficiency

> status: review for the owner (branch `agent/fable-review`, Fable, docs only; the user asked for this review on 2026-11-10). Nothing here is decided: section 7 lists the questions, appendix A the proposed text for the guard files. Previous review: [project-review-20261103.md](project-review-20261103.md); the direction and beta reviews in between: [direction-review.md](direction-review.md), [beta-readiness-review.md](beta-readiness-review.md). No game was run, nothing was deployed, nothing was posted.

Tags: **V** read in this tree or produced by a command named here; **D** computed here from `git log`, the task files or a tool run (command given); **S** session fact or another doc's claim, not re-checked; **I** my judgement; **U** unverified. Window: `d4fcd366` (the 2026-11-03 review) to `e1515b1e` (main at the start of this run).

## 0. Verdict in ten lines

1. The direction is right and better defined than a week ago: fun-first idle game, Expedition Ledger loop, class-generic data, Paladin then Sorceress, friends through the container (ADR 0017 to 0019, V). The research behind it (22 incremental games, player power, UI) is good and mostly carded.
2. The plan's **measurement steps were skipped while its feature steps were overbuilt.** Of the eight steps of the two-week plan (direction-review section 7), the ones that produce a verdict (H4/H1 baseline TASK-107, 4-hour away run TASK-110, H3 TASK-112, design brief TASK-114) are all Proposed with 0 boxes ticked (V), the smoke that was to be the regression floor has never run on a game (V `docs/smoke.md`, 0 records), and 81 m-4 cards exist of which 66 are merged (45 Done, 21 merged but not closed: D), nearly all "offline only, NOT verified live" (V task results).
3. **The fun killers are bot early-game defects, not missing UI** (beta review, V from the sandbox log): clvl 10 to 12 in 11.7 h, gold under 300 most of the day, the cave shuttle, mana back-offs, 67 game exits a day. Fixes for five of them landed this week (packfight, poverty, areathrash, walksmooth, questgivers) but not one is verified live (V results).
4. The user is the only person who has seen the loop, and the sandbox hero is a long-lived clvl 12 Paladin, not a fresh friend hero (S beta review). No session record with the H numbers exists (V TASK-107 0/3).
5. **Process cost is still the largest controllable waste**: 2498 commits and 826 merges in seven days (D), of which 252 of 440 agent-branch merges changed no Go (D); five shared files are rewritten on nearly every merge (STATUS.md 337 commits, BACKLOG.md 272, CHANGELOG.md 257, OWNERS 232, INDEX.md 219: D); the backlog drifts (59 merged tasks not Done, V `drift.py`); STATUS.md's hand-written part still describes the 2026-11-03 goals (V).
6. Every agent is told to read about 136 KB of standing docs before its task (D, section 6.1): at roughly 450 agent runs a week that is the single largest avoidable token cost we control.
7. Recommendation, ranked: (1) a one-week freeze on new `/watch` panels and new families (pacts, hall, senses tiers, readiness rows) while the three sessions and the 4-hour away run are recorded and the design brief v1 is written; (2) the headline metric becomes "fresh Paladin, 60 realtime minutes at 1x" (clvl, kills, gold, chickens, zone entries), measured daily on the sandbox with the smoke tool; (3) the backlog true-up of section 3 (close 40, park 20, create 12); (4) the process changes of section 6 (generate the shared files, auto-Done on, standing-context diet); (5) the beta exit criteria of section 5 before any friend copy.
8. Parity work is correctly paused; the decomp filler kept producing docs (decomp5 to decomp8 closed 14 fid rows, V) that nobody consumes now. Keep it as idle filler at one job a day or stop it until the brief.
9. Risks nobody owns: the licence one-pager before a friend copy (direction-review section 8, still undecided), the Blizzard art on a streamed page (paperdoll-art section 6), and "measured runs" that have no measured-run slot while the gates are paused.
10. The project is one person's taste until a second person plays. Everything in sections 4 and 5 is about getting a second pair of eyes on a good first hour.

## 1. Executive verdict and steering recommendations (ranked)

| # | Recommendation | Why (evidence) | Cost (U) | What the user sees |
|---|---|---|---|---|
| R1 | **Run the plan's measurement steps now, freeze new families for one week.** Sessions H4 and H1 (TASK-107), the 4-hour away run plus return-screen read (TASK-110), H3 (TASK-112), then the design brief v1 (TASK-114). New `/watch` panels, new pact/hall/senses/readiness rows wait; fixes to defects the sessions show do not. | ADR 0017 D1 made these the point of the fortnight (V); none has a ticked box (V); 66 m-4 cards merged meanwhile (D) | 0 agent-hours for the sessions, about 4 for the brief; 1 week of restraint | three numbers and three sentences per session; a brief that replaces the ranking paragraph |
| R2 | **One headline metric: the fresh first hour.** `tools/smoke.sh --class paladin --minutes 60 --record` on a fresh hero at 1x realtime (U whether the tool runs at 1x: it is written for virtual minutes; if not, a 60-minute `d2bot play` with the session record), once a day on the sandbox after the day's deploys, logged as a table in `docs/probes/firsthour.md`: clvl@60, kills, gold, chickens, zone entries, restarts. | fun killers 3 to 7 all show in this hour (V beta review); XP/h and Gate 2 are paused and measure T4 behaviour (V ADR 0017) | 1 host-hour a day, 0.5 agent-hour | a line per day that goes up or does not |
| R3 | **True up the board** (section 3): close the 40 tasks whose work is merged and whose only open boxes are live checks, with those checks moved to one live-check list; park 20; create the 12 missing cards; rewrite STATUS.md's hand part (TASK-46). | 59 drift rows (V), 6 all-ticked-open (V), STATUS hand part dated 2026-11-03 describing G-A..G-F (V) | 3 agent-hours (Haiku for the mechanical part) | a board that matches main |
| R4 | **Cut the per-merge and per-agent overhead** (section 6): generate STATUS block, BACKLOG and CHANGELOG at merge instead of committing them per branch; switch auto-Done on; cut the mandatory reading to START-HERE plus the task plus the area doc. | five shared files on nearly every merge (D), 136 KB standing reading (D), auto-Done off by default (V `merging.md`) | 6 agent-hours once | fewer conflicts, faster merges, cheaper agents |
| R5 | **Beta exit criteria before a friend copy** (section 5): a fresh Paladin reaches a stated clvl in the first hour with no shuttle and no poverty stall, the return screen after a 4-hour absence makes the user set 3 or more policies (the H2 bar), a one-command install with the version line and a feedback box, the licence one-pager read by the user. | fun killer 1 and 2 are install and explanation (V); GPL offer and EULA note undecided (V licensing.md, direction-review section 8) | 2 to 4 agent-days after R1 | a go/no-go checklist |
| R6 | **Verify live before building the next in a family.** Rule for the ladder, senses, pacts, readiness, repair, inventory: the next card in a family starts only after one person has seen the previous one on a real hero (a line in the task result: "seen live by <role> on <date>"). | 10 of the 16 rows of `first-session.md` section C say "no" or "partly" under "verified live" (V) | 0 | less work that is later found wrong |
| R7 | **Decomp filler at one job a day or none.** | 14 fid rows closed this week with no consumer (V), parity paused (V ADR 0017) | saves about 1 Sonnet agent a day | nothing visible, which is the point |
| R8 | **Record the pending legal notes** as the user's call: a one-page risk note before the first outside copy, art off on any streamed view by default. | raised twice, decided never (V direction-review section 8, paperdoll-art section 6) | 2 agent-hours | one page to read |

What I would not change: the fun-first ranking (D1), the idle reading (D2), the container as the friends route (D3), Paladin first (F11), data-not-code (F10), the merge queue and the checks, the published fork.

## 2. Review of changes since the last review (2026-11-03 to 2026-11-10)

### 2.1 Shape of the week (D)

Commands: `git rev-list --count d4fcd366..HEAD` (2498 commits, 826 merges); `git log --merges --format=%s d4fcd366..HEAD | grep -c 'Merge agent/'` (440 agent branches); for each agent merge `git diff --name-only $h^1 $h | grep -q '\.go$'` (252 of 440 changed no Go file); `git log --no-merges --numstat --format= d4fcd366..HEAD` summed by path.

| Area | Lines added | What landed (merge subjects, V) | Verified live? |
|---|---:|---|---|
| `internal/watch` (+web) | 36,164 | watch shell and panel registry, design tokens and 12 components, layout editor, return screen, policy sheet, tally strip, whyline, inventory page and paper doll, senses minimap and tiers, unlock ladder with 10 features, pact pages, Hall of Fame, readiness API, compatibility goldens, screenshot suite, scroll fix | partly: the user saw the sandbox pages; `first-session.md` C says 10 of 16 rows "no" or "partly" |
| `docs/research` | 15,560 | direction review, fun ideas, incremental ARPG (two passes), player power, UI design, beta readiness, paperdoll art, pact system, info unlocks, unlockables, class early game, fork publication scan, g2g/g2h analyses, 16 fid oracle docs | n/a |
| `tools/emu`, `docs/probes` | 29,200 | decomp batches 2 to 4 (fid oracles), classpacing, pactnative, recoverypace, panelslive | oracle runs V; nothing on the bot |
| `backlog/tasks` | 12,056 | the Backlog.md board itself (OpenSpec removed 2026-11-04): 242 task files added in the window, 238 exist (D) | n/a |
| `internal/decide`, `internal/ladder`, `cmd/d2bot` | 16,300 | policy sheet rows, question catalogue, decision dataset, session record, zone proposals, unlock wiring, pacts through the tuner, Hall mount | fixtures only (V results) |
| `internal/bot`, `town`, `combat`, `nav`, `resource` | about 9,000 | walksmooth 1 to 3, approachfast, packfight 1 and 2, poverty band, areathrash breaker, quest givers, stance row, mouthfix, tpfirst, mercfix, equipfix, cpnav, repair gate | walksmooth after-numbers live (V `docs/walksmooth.md`); the rest "offline only" |
| `internal/pact`, `currency`, `selfupdate`, `cmd/didleboot`, `deploy` | 10,000 | pacts, tally ledger, update shim with rollback, signing key, inert tester-image workflow, dev-update loop | dev-update used daily on the sandbox (S memory); the rest offline |
| `internal/readiness`, `senses`, `unlock`, `art` | 3,500 | area table and evaluator, senses data, unlock engine, DC6 decoder (doll-art A to C) | offline |

### 2.2 What is good

- **The design base exists and is coherent.** ADR 0017 to 0019 plus fun-ideas, incremental-arpg, unlockables, info-unlocks and pact-system form one design with sources and anti-patterns (V). The previous review asked for exactly this ("decide what done means"); it is done at the product level.
- **Class-generic is real in new code.** Every new package I opened (readiness, repair, quest givers, stance, inventory) runs its tests on Paladin and Sorceress and errors on an unknown class (V the docs' test lists; I did not run them).
- **Honesty held.** Every feature doc carries a "not run live" line and a live checklist (V, all eight docs read). The beta review is the most useful document of the week because it read the log instead of the docs.
- **The dev loop on the sandbox** (dev-update, didleboot rollback, 5 to 20 s restarts) is the right tool for a design iteration and the user approved it (S memory).
- **The UI compatibility promise** (versioned manifests, additive panels, goldens) came as a reaction to "EXTREMELY messy" (S memory, V `uicompat`) but it is the right reaction and it is tested.
- **The doll-art plan** is the best-shaped work package of the week: six PRs, A to D invisible, E turns it on, F guards; each PR one package; licence stated plainly (V `paperdoll-art.md`).
- **The decomp oracles keep finding 0-diff rows** (xp award, monster velocity, freeze rule) that lower the parity risk list (V fid docs) at a steady cost.

### 2.3 What is risky

- **No verdict exists for the loop** (section 0 item 2). The two-week clock started 2026-11-07; three days in, the only measured thing is a long-lived sandbox hero, by one reviewer reading the log (V). The plan said day 2 is the H4/H1 baseline.
- **Features that cannot fire yet were built anyway**: pacts unlock at the first Andariel kill, the Hall needs Andariel, senses tiers 5 to 7 carry no data, the readiness model matters in Nightmare; no hero has killed Andariel live, ever (V STATUS G-B, V `first-session.md` C). Roughly 20 to 25 cards of this week (pacts 166 to 173, Hall 135/163/171, senses tiers, readiness 214 to 224, legacy 136) are content for a game state nobody has reached (I).
- **Measured-run slots are gone.** Pacts, stance starts, repair gate, quest givers all switch off under `D2BOT_RUN` (V the docs), which is the right rule, but no measured run has been scheduled since D1, so the "gates stay comparable" branch is untested while the player branch diverges from it weekly (I).
- **Sandbox evidence is contaminated by deploys**: 18 release swaps in 11.7 h (V beta review); every log analysis has to exclude them. Batch deploys to two windows a day and stamp the session record with the build (section 6.5).
- **Shared JSON registries are conflict magnets**: `data/unlocks/ladder.json`, `internal/watch/funwire.go`, `cmd/d2bot/ladder.go` (13, 28, 28 commits; 6 merges say "union ladder rows" or similar: D). Ten unlock cards ran in parallel and each merged main 3 to 5 times (D).
- **The clvl-17 floor** before the Burial Grounds reads as the level wall ADR 0018 dropped (V beta review Q1, still open) and it is the "Level up to 17" stall the user sees (S).
- **Legal posture drifts without a decision** (section 0 item 9).

### 2.4 Unfinished and drift

- STATUS.md hand part: dated 2026-11-03, lists goals G-A..G-F, "next three days: S1 class check" (V). TASK-46 exists, Proposed. The generated block below it is current. A reader of START-HERE's "where we are" link gets the wrong direction.
- Drift (V `python3 -I tools/backlog/drift.py`): 59 tasks merged on main but not Done, 6 with every box ticked but not Done. Cause (V `merging.md`, `auto_done.py`): auto-Done is off by default and requires every box ticked and a `result:`; most cards keep an open "verified live" box, so they never close.
- D5 (docs diet) decided 2026-11-07: `docs/cards.md` is now generated (done), TASK-117 archive and TASK-118 merge-limit rule are Proposed 0/2 (V); 141 research docs were added since the last review (D), 376 markdown files under docs overall (D).
- The glossary has none of the loop's terms (Expedition, Report card, Charter, Policy sheet, Codex, Pact, Senses, Unlock ladder; V `grep` on GLOSSARY.md; only Tally is there). ADR 0018 lists them as candidates for the grill.
- In Progress tasks older than three days with no commit on their owns: TASK-53 abmerc (11-06), TASK-59 custom-overlay (11-06, the slice is paused by D4), TASK-28 testerlicence (11-07) (D `git log -1 -- <owns>`).
- Fork: published as `d2d-didle` (V merge subject); docs still say "the GitHub fork is created after the coordinator reads the scan" in the Current truth bullet (V decisions.md line 10): stale sentence.

## 3. Backlog true-up

Board today (V `grep status: backlog/tasks/*.md`): 238 tasks; 105 Done, 78 Proposed, 47 Deferred, 8 In Progress. Milestones: m-4 81, m-2 39, m-1 31, m-0 17, m-3 7. I did not edit any task file (not my `owns:`); this section is the list for the coordinator.

### 3.1 Close now (merged on main; open boxes are live checks or "seen by eye"): 40

Rule proposed: a card closes when its code and docs are merged and its offline acceptance is proven; its live-check lines move to `docs/live-checks.md` (one table: card, check, owner role, date seen) so the live debt stays visible without 59 half-open cards. Candidates (V drift D rows plus the task's `result:`):

- Fun loop and watch: 129 tallystrip, 133 skillunlocks, 134 playsession2 (re-create the session as a fresh card under R1), 147 classpacing, 149 sensesunlock, 157 unlockidreveal, 158 unlockbestiary, 159 unlockrisk, 160 unlockatlas, 161 unlockorderqueue, 162 unlockalerts, 186 sensestarget, 195 mapreveal (card only), 206 whyline, 212 invui, 213 upgradeadvisor (hook only; re-card the advisor if wanted), 215 panelgrid, 225 panelgrid2, 226 repairmanual, 227 desktopapp, 230 uishell1 (all ticked), 238 betareview (all ticked).
- Bot defects: 188 walksmooth, 233 packfight, 239 packfight2, 240 poverty (all ticked), 241 killloop (all ticked), 242 areathrash (all ticked), 243 questgivers (all ticked), 164 beltplan (AC 5 to the live list), 170 pactnative (verified live), 179 ptrclick (fold into the fid list).
- Fidelity docs (Deferred but finished as docs): 75, 76, 77, 78, 80, 82, 83, 85, 87, 95, 102, 203 (mark Done with the fid doc as result; the open "live read" boxes go to the live list).
- Process: 111 smokebot (built; the first live run is R2's job), 113 friendsfb (re-card after R5), 120 ghcrpublish (prep done; the publish needs the user: keep as a one-line "needs go" card or close and reopen), 123 updboot, 124 updkey (user steps open: same), 126 learnedbrain, 193 originfix, 22 decisions-mirror, 53 abmerc, 61 metacurrency (design done; the slice is paused).

### 3.2 Park (Deferred, label `parked`) until a live hero reaches the state they need: 20

- Pacts: 171 pacthall, 172 pactoverlay, 173 pactnightmare (first Pacts need Andariel killed live).
- Hall: 163 unlockhall (0 boxes defined), 136 legacycharm (already Deferred).
- Readiness: 219 ready-wire-gates, 220 ready-meter, 221 ready-whyline, 222 ready-bossrows, 223 ready-policy, 224 ready-calibrate (Normal Act 1 does not need them: V beta review section 8); keep 216 ready-audit as a five-minute live read when a Nightmare character exists.
- Ladder mechanism: 19 decision-point-store and 21 t0-slice (In Progress since 2026-11-05; ADR 0017 D2 replaced the fade with the queue and the catalogue's chore points moved to the policy sheet: re-scope to "question queue for the return screen" or park), 36 t0fight.
- Metacurrency: 59 custom-overlay (In Progress, paused by D4): Deferred.
- Classes other than Paladin and Sorceress: 204 amazonammo (per the class order, S memory).
- 45 upstream-next stays Proposed low (one batch at a time).

### 3.3 Duplicates and overlaps to merge

- 43 playsession (LAN command) and 134 playsession2 (container session) and 110 run4h: one card "sessions" with three runs (H4/H1, H3, 4-hour away) and one record format (`docs/session-record.md`).
- 206 whyline (merged) and 221 ready-whyline: the second is a template set, fold into 219 when un-parked.
- 159 unlockrisk (merged) and 220 ready-meter: same meter; one card.
- 108 qdensity and 115 qqueue (Deferred) are folded into the return screen by ADR 0018; close them as superseded rather than keep them Deferred.
- 26 testerdocs, 27 testerreport, 29 testerpack, 48 onboardtour, 49 onboardctl: five m-1 cards that the beta path (section 5) re-cuts into three; keep the three, close the rest as superseded.
- 107 h1baseline and 112 h3verdict and 114 designbrief: keep, but they are the only m-4 cards that matter this week (R1).

### 3.4 Missing cards (none created here)

| Card | Why | Size (U) | Source |
|---|---|---|---|
| `firsthour` | the R2 metric: fresh Paladin, 60 realtime minutes, daily table | S | section 1 |
| `sessions` | the H4/H1, H3 and 4-hour away runs as one card with one record format (replaces 43, 110, 134) | S | section 3.3 |
| `firstminutes` | welcome card and a friend's first-30-minutes text on `/watch` | M | beta review card 2 |
| `awaydrill` | the 4-hour away run and the return-screen read (merge 110 into it) | M | beta review card 8, direction-review step 4 |
| `reconnectux` | exits and chickens shown on `/watch`; the 67 exits a day | S | beta review card 7 |
| `actplan` | the clvl-17 floor as a price or a per-class data row, not a wall; "leave a zone six levels under the hero, take the next story step"; wizard marks unproven classes | M | beta review Q1, SME (section 4.4 items 1 and 2a) |
| `palprofile` | Paladin profile data: plain Attack under Might and Sacrifice before Zeal, Prayer off while fighting, no mana potions before Zeal; fix `docs/sme/class-paladin.md` (Prayer costs mana) and `class-early-game.md` (Zeal at 12, not 6); test on Paladin and Sorceress | S | SME (section 4.4 items 3 and 6) |
| `smepace` | a `docs/sme/` page with the Act 1 pace table (clvl by beat), the mana formulas and the kit prices, so SME consults stop re-fetching the wiki | S | section 4.4 cost note |
| `livechecks` | `docs/live-checks.md` and the closing rule of 3.1 | S | section 3.1 |
| `sharedgen` | generate STATUS block, BACKLOG and CHANGELOG at merge; auto-Done on | M | section 6.3 |
| `contextdiet` | the agent reading order cut to START-HERE, task, area doc; one two-page digest of memory facts and glossary terms per area | S | section 6.1 |
| `legalnote` | the one-page risk note and the art-off default on streamed views | S | section 0 item 9 |

### 3.5 Order for the next two weeks (what I would launch, in order)

1. `firsthour` (R2) and the sessions card (R1, the user plays). 2. Live checks for the five bot fixes already merged, on the sandbox, one fresh hero each (areathrash, poverty, packfight2, questgivers, walksmooth). 3. `palprofile` and `actplan` (the SME's two early-game changes, both data). 4. `reconnectux`, `firstminutes`. 5. True-up (3.1 to 3.3) and `sharedgen`, `contextdiet` (Haiku and Sonnet). 6. `awaydrill`. 7. Design brief v1. 8. Beta checklist (section 5). Doll-art D to F finish as planned (they are invisible until E). Everything else waits for the brief.

## 4. Direction against the idle, incremental and ARPG research

### 4.1 What the research says and what we built

| Research point (source) | Status in the product | Gap |
|---|---|---|
| Return screen first, as a diary with 3 to 6 true lines then numbers (incremental-arpg #4, Godville V) | built (TASK-140, returnscreen); 20-minute landing not exercised live (V first-session C) | **never read after a real absence**; the 4-hour run is the test |
| Policy sheet as the real inventory UI; strictness presets plus dry-run (#1, PoE and Last Epoch V) | sheet built, dry-run built (TASK-205), stance row, repair row | presets (Lenient/Standard/Strict) not built; several rows only store a choice the bot never asks (V first-session C) |
| Price, never a wall (ADR 0018 F2, Clicker Heroes V) | charters priced in tally | the clvl-17 floor is a wall in effect (beta review Q1); the design says the hero warns and goes |
| Visible short beat inside a long trip (incremental-arpg section 7) | expedition cards grouped by town trip | no 10 to 20 minute beat shown live; the first expedition card target (fun-ideas section 7) unmeasured |
| Failure priced and shown (Loop Hero, Warframe) | death report in the log is rich (V beta review) | not on the return screen as "what the retreat cost" |
| Visible reasoning (Claude Plays Pokemon, V) | whyline built | one line; the "why each question exists" half is open |
| Persona and nicknames (Neuro-sama study, Twitch Plays Pokemon) | hero voice and diary on the ladder (TASK-155) | fixed persona as data not written; nicknames not stored |
| Nudge budget (Godville godpower) | not built; left as a user decision (incremental-arpg section 6) | correct to wait; decide after H3 |
| Anti-pattern: notification spam, modal on routine events | in-page alert centre, senders off (V) | good |
| Anti-pattern: grind as the only content; something new every third loop | Codex firsts, unlock ladder, bestiary | the first-40-minutes schedule (unlockables section 3) is a draft at 1x, unmeasured |
| Onboarding: a meaningful decision within 5 minutes (fun-ideas section 7) | wizard (class, goal), policy sheet at creation | nothing explains the game or what the first hour will look like (beta review killer 2) |

### 4.2 What we are missing

1. **The away experience.** The product's claim is "away for hours, then a 5 to 20 minute return" (F5). Not one return after a real absence has been observed (V). This is the H2 bar and it is cheap: a 4-hour run on the sandbox while the user does something else, then one read.
2. **The first hour as a designed experience.** Everything from the fun killers (poverty, shuttle, mana, exits) happens in the hour a friend judges the game by. The pacing targets of fun-ideas section 7 are good and unmeasured at 1x (U everywhere). The SME's view of the natural beats is in 4.4.
3. **A second player.** Until then "fun" is one person's taste (direction-review section 8, still true).
4. **Failure on the return screen** and **the visible short beat** (table above).

### 4.3 What we are overbuilding

- Content for states no hero reaches live: pacts, Hall, senses tiers 5 to 7, readiness beyond Normal, legacy charm (section 2.3). The incremental research supports all of them as later content; nothing supports building them before the first hour works.
- Watch surface: 66 `/api` paths, 35 routes, 18 web files, 103 Go files in `internal/watch` (D); the user's own word for it was "EXTREMELY messy" before the shell migration (S). The shell and the compatibility promise fix the shape; the count still means every change touches a large surface.
- Process artefacts: 141 research docs in a week (D), most with one reader (the coordinator) and no consumer card; the D5 diet is decided and not done.
- Decomp oracles while parity is paused (R7).

### 4.4 SME consult (d2-sme, 2026-11-10, read-only; tags SRC/EXP/UNSURE are the SME's)

One consult, six questions, answer under 450 words (the SME cited `docs/sme/`, the Amazon Basin wiki, the Arreat Summit mirror and a D2R Paladin leveling guide; its clvl numbers are EXP unless marked SRC; nothing here was checked against game data by me).

1. **The clvl-17 floor.** Fair only as the no-merc Andariel floor; as a hold before the Burial Grounds it is a wall. A human Paladin with the Rogue kills Andariel at clvl 14 to 18 in hour 2.5 to 4.5 (EXP); fast routes 13 to 17 (SRC). Not-overcautious floors (EXP): Catacombs 13 with the merc, 14 without; Andariel 14 with the merc, 16 without, with 8 healing and 2 antidote potions. ADR 0011's 15/17 is within a level of that. The arithmetic that makes it a wall: about 185 k XP to clvl 17 is roughly 77 hours at the sandbox's 2.4 k XP/h against 2 to 5 hours at a human 40 to 100 k/h (SRC the SME's pace tables; U cells).
2. **Why a human is never poor by clvl 8 to 12** (the three habits the bot most lacks, ranked): (a) farm where the monster level is at least clvl minus 5, since pile size and drop level follow monster level; Blood Moor and Cave Level 1 at clvl 10 to 12 pay 5 to 43 percent of normal XP (SRC); (b) buy kit only: Minor Healing 30 gold, a town portal scroll about 100 (SRC), no mana potions before Zeal, no gear, no gambling; pick up every pile, sell every magic and rare; (c) Akara is the potion: free life, mana and stamina refill (SRC), go home by waypoint near half life, carry 4 to 6 healing and 1 to 2 portal scrolls (EXP).
3. **Paladin mana.** 28 mana at clvl 10 with 0 Energy is right (SRC: 15 plus 1.5 a level; the guide takes Energy only from clvl 12). Running dry and backing off is not expected: veterans use the 0-mana attacks, plain Attack under Might and Sacrifice (+180 percent damage, costs 8 percent of damage dealt as life); Smite is a 2-mana shield attack kept as the Charge prerequisite; Zeal (2 mana) starts at clvl 12 (SRC). Our profile ranks Smite over Attack and rests on Prayer, which drains mana and stops regeneration while it heals (SRC; whether that is the cause of the sandbox back-offs is UNSURE). This is a profile-data fix, not a stat fix.
4. **Beats a veteran remembers, in order (clvl, EXP):** Cold Plains waypoint 3 to 5; Den cleared and Corpsefire, the skill point 4 to 6; Blood Raven and the Rogue 8 to 10; first unique or set drop 8 to 15 (random); Cain rescued 9 to 13; the Countess and the first runes 11 to 14; Andariel and Act 2 at 15 to 18. These are the return-screen headlines and the Codex firsts, and they match the loop's world layer (F10).
5. **Codex skill unlocks:** fine; D2 already pays quests in skill points (SRC). Risk: an unlock event the hero never fires (Cain skipped, a Blood Raven bailout) strands the skill; give every unlock row a `clvl:N` fallback.
6. **Second class:** the Sorceress is the right second class because it differs on every axis (ranged, mana-bound, buff-first; EXP). Its early trap is the clvl 1 to 8 mana wall: 35 mana, Charged Bolt 3 mana plus 0.5 a point, 40 life (SRC); empty at melee range kills.

SME-found doc defects (to fix as data, V the SME's reading, U mine): `docs/sme/class-paladin.md` says auras cost no mana (Prayer does); `docs/research/class-early-game.md` puts Zeal and Charge at clvl 6 (the tables say 12); there is no measured clvl-by-hour table anywhere.

The SME's one change that would make a veteran say "that looks like a real player": leave any zone six or more levels under the hero and take the next story step (Blood Raven and the Rogue, Dark Wood, the Tower, the Catacombs), never a level floor. That is the `actplan` card, and with 2.(a) it is also the poverty fix.

Cost note for section 6: this consult used about 344 k tokens and 77 tool calls for 450 words of answer (V harness usage), most of it web fetching; the user's rule "small specific questions beat bulk research" (S memory) held: six narrow questions, but each one sent the SME to the wiki. A `docs/sme/` page with the pace table and the mana formulas would make the next consult a tenth of the price.

### 4.5 Fun-first loop, pacing, offline experience, onboarding, unlock ladder: my reading

- **Loop:** right shape (expedition, report card, sheet, tally). The one thing to check before adding anything is whether a report card after a real trip makes the user change a policy. If it does not, no amount of unlock content fixes it.
- **Pacing:** the target table (fun-ideas section 7) is the design's own acceptance test; log it at 1x on a fresh hero and the sessions become comparable.
- **Offline and away:** idle (D2) means the return screen is the product; make it land after 20 minutes and test the "answer once for all" grouping with real missed questions.
- **Onboarding:** a 200-word welcome card and a "what the first hour looks like" line beat any tour (beta review killer 2; onboarding slices B and C stay paused).
- **Unlock ladder:** correct in principle (Godville, Melvor, Hades), never silent and basics never gated (F14 holds). Risk: unlocks that fire in the first 40 minutes come from a draft schedule; measure, then tune the table, which is data (V `data/unlocks/ladder.json`).

## 5. Beta readiness and the path to a friends beta

The user's stance (S brief): friends beta planned, held until play feedback. I agree, and propose the exit criteria so "held" has an end.

### 5.1 Exit criteria (proposal; numbers U until the first-hour table exists)

1. **First hour:** three consecutive daily fresh-Paladin hours on the sandbox with clvl at 60 minutes at or above the class pace table's value (TASK-147 measured 90 virtual minutes for seven tables; use its Paladin row), no cave shuttle (zone entries under a stated cap), no poverty stall over 10 minutes, no repeated game-exit loop.
2. **Away:** one 4-hour unattended run; the return screen read in under 3 minutes and the user sets 3 or more policies from it (direction-review H2).
3. **Install:** one command for a Docker-literate friend (compose plus the game folder), a version line in the watch header, a feedback box that writes into the session record and an export the friend can send (beta review section 7).
4. **Explain:** the welcome card and the first-30-minutes text (`firstminutes`).
5. **Legal:** the user has read the one-page note (GPL source offer for the image, the EULA posture, art off on streamed views) and said go.
6. **Support:** the token recovery line and the "known broken" table of `first-session.md` are current.

### 5.2 Order

`firsthour` and the bot fixes verified (weeks 1 to 2) -> `awaydrill` -> `firstminutes`, `reconnectux` -> install path (TASK-120 publish or a tarball with the source to two friends; the user's call, question Q3) -> one friend, then two. Keep Paladin-only until the Sorceress normal30 table exists (ADR 0018 amendment; question Q9).

### 5.3 What not to do for the beta

A seeded day-one save (ADR 0016 item 8 forbids premade heroes, V); phone senders on by default; a public registry before the licence note; more unlock content.

## 6. Efficiency: how to work and chunk work for better token usage

### 6.1 Where tokens go (what can be seen)

- Usage ledger (`tools/usage-ledger.py scan; report`, run here read-only, V): 1625 agents and 190.6 M reported tokens since 2026-10-29; Sonnet 86 %, Opus 9 %, Fable 3 %, Haiku 2 %; merge agents 229 at 59 k tokens and 9.8 minutes each; the class "other" (unclassified) is 26 %. Caveat: the ledger scans only the coordinator transcripts of one project directory, and the Pro and credit runners stopped on 2026-11-05 (V their ledgers: 87 and 103 jobs); the window 11-03 to 11-10 shows 457 agents while 440 branches merged, and 11-09 shows 14 agents against 59 merges, so a large share of this week's runs is invisible. **There is no complete usage view** (finding, not a blame).
- **Standing reading per agent** (D `wc -c`): AGENTS.md 14.8 KB, the private overlay 7.1 KB, START-HERE 4.3 KB, backlog-howto 28.8 KB, contracts README 3.2 KB, d2-1.14d-memory 48.6 KB, GLOSSARY 29.4 KB = 136 KB, plus the agent definition (2 to 9 KB), the user's global instructions and memory index, and the brief. That is roughly 35 k tokens before the task starts (I: 4 bytes a token); against the ledger's 111 k average per Sonnet agent it is about a third of a typical agent, and at about 450 agent runs a week about 15 M tokens of orientation (I).
- **Per-merge overhead** (D): the five shared files rewritten per merge (section 0 item 5), the merge-lines file, the generated STATUS block, a CI run and a whole-repo gate on the worker host; 386 "merge main into branch" commits for 440 branches (D), six explicit conflict unions on the registries.
- **Docs-only merges**: 252 of 440 (D). Each is a worktree, a brief, an agent, a gate, a merge agent (59 k tokens) for a document that often has one reader.
- **Biggest single agents** (V ledger): an Opus UI migration agent at 598 k tokens (three PRs in one agent) and the Fable UI/UX research at 409 k. Above about 400 k an agent is in the context-rollover zone and the summary it hands back loses detail (I).

### 6.2 Agent sizing and brief template

- **One agent = one PR = one package family, under about 400 lines of Go plus tests, one doc, 60 to 90 minutes, report under 300 words.** The doll-art split (A to F) is the model; the ten parallel unlock cards on one registry file are the anti-model (3 to 5 main merges each).
- **Shared data files get one owner per wave**: when several cards add rows to `ladder.json` or `funwire.go`, one agent adds the rows for all of them first (data only), then the feature agents run in parallel without touching the registry.
- **Brief template** (proposed as `docs/process/brief-template.md`, 12 lines): goal in one sentence; the task id; owns and exclusive; the three files to read (task, area doc, one design doc); what is already ruled out; the acceptance in the task; "offline only" or the instance; report length; the trailer lines. Nothing else: the agent definition carries the standing rules.
- **Reading order diet**: AGENTS.md's mandatory list becomes START-HERE, the task, the area doc; `d2-1.14d-memory.md` and GLOSSARY become "when a term or offset is unclear"; backlog-howto gets a one-page "for agents" section at the top and the rest moves below a fold. Proposed text in appendix A3.

### 6.3 What to automate

1. **Generate, do not commit**: STATUS's generated block, BACKLOG.md and CHANGELOG.md from the merge summaries and task results at merge time (one commit per merge batch, by the merge worker), OWNERS from the task files (`check.py` already mirrors them). The `backlog/merge-lines/` files then disappear.
2. **Auto-Done on** (`MERGEQ_AUTO_DONE=1`) with the 3.1 rule: live-check boxes move to `docs/live-checks.md`, so merged cards close.
3. **Drift in CI** as a warning line (the tool exists; `drift.py` is read-only and exits 0).
4. **Smoke after every bot merge** on the worker host, nice 19, one class, recorded (the tool exists and has never run; 45 minutes per bot change was the D1 floor).
5. **Usage ledger across runners**: one scan that also reads the Pro and credit ledgers and other project directories, one line per day in the tick.
6. **Deploy stamp**: dev-update writes the build hash into the session record so every log analysis can exclude restarts in one filter.

### 6.4 Model fit

| Work | Model | Why |
|---|---|---|
| status sync, drift fixes, merge-lines, cards, archive stubs, doc link fixes | Haiku | mechanical, checked by tools |
| implementation cards, bot fixes, live checks, decomp oracles | Sonnet | the default; the ledger shows no quality gap for decomp after the 11-01 switch (V: avg tokens flat, Opus share 0) |
| reviews of a merged family, merge-risk reads, hard debugging after two Sonnet failures | Opus, singly | as today |
| direction and whole-project reviews, design partner rounds, when the user asks | Fable | as today (3 % of tokens, V) |
| SME questions | Sonnet d2-sme, small and specific | the user's rule (S memory) |

### 6.5 Fewer deploy collisions and merge-queue habits

- Two deploy windows a day on the sandbox unless the user asks for one now; one line to the user per window.
- Queue by family with `--after` when branches touch the same registry or the same `cmd/d2bot` wiring file (the dispatcher already says so; make it the default for `internal/watch`).
- Docs-only branches: batch three or more into one merge run (the efficiency rule (b) exists); or better, let research agents write to `docs/research/` on a shared `docs` branch merged once a day.
- Fewer research docs: a research card needs a consumer card named in its brief; otherwise the result is a section in an existing doc.

### 6.6 Doc sprawl

1880 files and 37 MB under docs, 378 research docs (D). The D5 decision covers it; the work is TASK-117 and TASK-118 plus `sharedgen`. Rule of thumb: one doc per card family, dated run records under `docs/probes/<family>/`, research older than seven days with no inbound link archived with a stub (`git mv`, never delete).

## 7. Open questions for the user

1. **Headline metric.** Accept "fresh Paladin, 60 realtime minutes at 1x: clvl, kills, gold, chickens, zone entries" as the one number for the next two weeks (XP/h and Gate 2 stay paused)?
2. **Freeze.** One week with no new `/watch` panel and no new pact, Hall, senses-tier or readiness row, fixes to seen defects allowed, until the three sessions and the 4-hour run are recorded?
3. **Beta route.** Publish the image (TASK-120, needs your go) or hand a tarball with the source to one or two friends first? And the exit criteria of 5.1: accept or edit?
4. **Park list.** Park pacts 171 to 173, Hall 163, readiness 219 to 224, ladder mechanism 19/21/36 and overlay 59 until a live hero reaches Andariel or Nightmare?
5. **Decomp filler.** One oracle job a day when the worker host is idle, or none until the design brief?
6. **Shared files.** Approve generating STATUS's block, BACKLOG.md and CHANGELOG.md at merge (no per-branch merge-lines) and switching auto-Done on with the live-check rule of 3.1?
7. **Reading diet.** Approve the AGENTS.md reading-order change of appendix A3 (memory doc and glossary on demand)?
8. **clvl-17.** Is the floor before the Burial Grounds a safety rule you want kept, or a placeholder to turn into a per-class data row with a warning (`actplan`)? The SME puts a human Paladin with the Rogue at Andariel at clvl 14 to 18 in hour 2.5 to 4.5 and calls 17 fair only as the no-merc Andariel floor (section 4.4).
8b. **Paladin profile.** May `palprofile` change the Paladin's early attack order to plain Attack under Might and Sacrifice before Zeal, with Prayer off while fighting (the SME's reading of the mana back-offs; UNSURE as the cause)? It is profile data with a Paladin and Sorceress test, no Go rule.
9. **Classes for friends.** Paladin-only wizard until the Sorceress normal30 table exists?
10. **Legal note.** Do you want the one-page risk note (GPL source offer, EULA posture, art on streamed views) before the first outside copy?
11. **Glossary.** May the grill session add the loop terms (appendix A4)?

## Appendix A. Proposed text for the guard files (not applied)

### A1. docs/decisions.md, Current truth bullet (replace the "Next two weeks" bullet when the brief lands; interim wording)
- `**Next two weeks (interim, review 2026-11-10):** the fun-first ranking of ADR 0017 stands; within it the measurement steps come first (sessions H4/H1/H3, the 4-hour away run, the design brief v1) and the headline metric is the fresh first hour (clvl, kills, gold, chickens, zone entries at 60 realtime minutes); new watch panels and new pact, Hall, senses-tier and readiness rows wait for the brief; a merged card closes when its offline acceptance is proven and its live checks are listed in docs/live-checks.md.`
- Fix the stale sentence in the Fork bullet: the GitHub fork `d2d-didle` exists (merge forkpub2, 2026-11-07).

### A2. docs/plan/STATUS.md hand part (one page; replaces the 2026-11-03 text)
> Direction: ADR 0016 to 0019 (fun-first idle game; Expedition Ledger; Paladin first). Loop built on fixtures, seen by the owner on the sandbox, no session record yet. Gates paused (D1); Gate 2 last indicative 3-seed result g2h, not passed. Fidelity paused except slices 1 and 2. Fork published. Beta held until the first-hour and away criteria (project-review-2026-11-10 section 5). Next: sessions, first-hour metric, true-up, design brief v1.

### A3. AGENTS.md reading order (first paragraph)
> Read in this order: `docs/START-HERE.md`, YOUR BACKLOG TASK, the doc of your area (named in the task or found in `docs/INDEX.md`). Open `docs/contracts/README.md` when your task names a contract, `GLOSSARY.md` when a term is unclear, `docs/d2-1.14d-memory.md` when you touch memory offsets or game structs. `docs/plan/STATUS.md` is the coordinator's page; agents read its generated block only when a brief says so.

### A4. GLOSSARY candidates (one sentence each; from ADR 0018/0019)
Expedition (one town-to-town trip of the hero, reported as a card); Report card (the expedition's summary: zone, length, cost, return, closest call, firsts, tally); Charter (a tally price for an above-range order; never a wall); Policy sheet (the player's standing chore and stance choices, data rows, changeable any time); Codex (the Act 1 firsts and their discovery questions); Pact (an opt-in challenge rule paid in the meta layer, off in measured runs); Senses (view-only information tiers on the side-panel minimap); Unlock ladder (the data-driven order in which watch features reveal themselves).

## Appendix B. Commands and files behind the numbers

- Commits and merges: `git rev-list --count d4fcd366..HEAD`; `git log --merges --format='%ad %s' --date=short d4fcd366..HEAD`.
- Docs-only share: loop over agent merges with `git diff --name-only $h^1 $h | grep -q '\.go$'`.
- Lines by path: `git log --no-merges --numstat --format= d4fcd366..HEAD` summed on the first two path parts.
- Most-touched files: `git log --no-merges --format= --name-only d4fcd366..HEAD | sort | uniq -c | sort -rn`.
- Board: `python3 -I tools/backlog/drift.py`; `grep -h '^status:' backlog/tasks/*.md | sort | uniq -c`; ticked boxes per task with `grep -c '^- \[x\]'`.
- Sizes: `wc -c` on the files named in 6.1; `find docs -type f | wc -l`; `du -sh docs`.
- Usage: `tools/usage-ledger.py scan; report` in the ops repo (read-only; writes only its own cache).
- Sandbox numbers are the beta review's (V there, S here); I did not read the container log.

## Appendix C. Not verified here

No game run, no deploy, no code read beyond what the docs name (the class-generic claims are the docs' test lists); the usage ledger's token semantics are unknown (its own caveat); all hour and size estimates are U; the SME's clvl numbers are EXP unless marked SRC.
