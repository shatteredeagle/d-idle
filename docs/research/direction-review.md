# Direction review: d.idle, 2026-11-07

> status: review, 2026-11-07, branch agent/directionreview (reviewer agent, Fable), docs only; nothing run, no Go or game touched. Asked by the user: "give us a solid direction ... get to a solid place where we can iterate on designs, find the fun, and expand from there". This document challenges the current direction on purpose, including the parts the user already decided. Proposed decision text is in section 9; the guard files are not edited.
> Tags: **V** read in this tree or counted with `git`/shell in this worktree (file or command named), **S** secondary (a doc's claim not re-checked), **U** unverified or my judgment. Classic 1.14d throughout. Numbers that have an owner are cited, not copied as new facts.

## 1. Product in one paragraph

What I read: d.idle is a Go program that plays one Diablo II: Lord of Destruction 1.14d character on d2d (an open reimplementation), offline, on the player's own game files. The player never steers the hero. The hero plays Act 1 Normal by its built-in rules; at a few decision points it opens a question card ("Life is low, a pack of Fallen is close: drink, portal, or keep fighting?") that fades after 45 s, and three matching answers teach it a rule it then applies alone. The player can also tap an Act 1 map to send the hero somewhere, read a journal of what it did and why, and watch a video of the game window. Who plays it: today one person (the user) on a hidden lab instance; the planned second audience is one friend with Docker on Linux x86_64 (V `docs/research/tester-build-plan.md` section 1). What a session feels like: a browser tab with a slow 1x game, long quiet stretches, a card now and then, a hero that dies and gets stuck in the early game (V `docs/research/g2g-diagnosis.md`, `docs/probes/t0drill.md` Result 3).

Where the docs disagree about this (V, files named):
- **Three products.** `PLAN.md` (lines 10 to 11) is a goal-driven bot ("reach level 30", "farm Mephisto"; optimal first, handicapped by upgrades). ADR 0016 is a teaching game (questions, rules, tiers, an altar). `docs/research/mocks/friends-preview.html` and the onboarding Welcome screen sell an idle game ("ignore it for hours"). These pull in different directions: an idle player misses every 45 s card; a teaching player must watch; the measured product (gates) is a bot with "no live person in a measured test" (ADR 0016 item 18).
- **Done means** has moved four times in nine days (ADR 0009 item 1, ADR 0015 item 1, its 2026-11-04 amendment, ADR 0016 item 6) and the hand-written STATUS still ranks by G-A to G-F from 2026-11-03 (V `docs/plan/STATUS.md` line 1; its own rule is 3 days, ADR 0015 item 11).
- **The ladder has no milestone.** The four milestones are Gate 2 pass (16 cards), friends build (19), fidelity Gate A (32), metacurrency v1 (7); the ladder cards (decision-point-store, t0-slice, dpwire, t0fight, ladder-t1, ladder-t2) carry no milestone (V `grep milestone: backlog/tasks/*.md`). The product core is the one thing the board does not aim at.
- **Realtime is undesigned.** Every soak, gate and drill ran in lockstep at 3.5 to 5 virtual minutes per wall minute (V `docs/research/g2g-diagnosis.md` M1); a person plays at 1x. The tester plan says so (G6) and the realtime check (TASK-33 testerrt) is Proposed.

## 2. Foundation: is parity the base?

**Parity is an engine floor, not a design base.** The first live probe found a real deviation in one afternoon (d2d loads at full life and mana, game.exe keeps the saved value; V `docs/probes/fidprobe.md`), which shows the probe is cheap and useful as a regression tool. But the matrix lists 214 open hours, 29 new cards and a 40 h A:B (V `docs/research/fidelity-matrix.md` totals, `docs/research/fidelity-cards.md`), and none of the 13 NONE rows blocks a design question. The user's own bar already accepts 20 percent on XP/h and kills/h (decisions.md Current truth, "Done means"); a design that only works inside 20 percent of game.exe is not a design.

What in "find the fun" is blocked on parity (U, my split):
- **Blocked or coupled:** combat feel at the margin (hit recovery SK16, missile flight SK03, monster missiles MA10: the hero's deaths are the player's most felt event), drop economy if the metacurrency ever keys on real drops (IT05 is already OD), boss fights (MA13) when T4 content is built.
- **Not blocked:** the question catalogue, card wording, fade time, question density, the away-and-return loop, the orders map, pacing of the first hour, the journal, uptime, what the player sees on a phone, session length, the first-run flow. All of these run on today's lab binary and most of them on the simulated runner (`d2dview -watch`, V `docs/research/watch-screen-design.md` section 4) with no game at all.

**A minimum playable loop is needed in addition to parity.** Today nobody can tell whether the teaching loop is fun because the one real play session produced one sentence of feedback ("I didn't know why I was making the decisions or what prompted them", V `docs/research/watch-screen-design.md` section 9) and the four scripted drills learned 2 rules in 40 minutes with no fight-time question in a clean run (V `docs/probes/t0drill.md` Result 3). ADR 0016 item 16 names the metrics (questions per 10 minutes, time to first unlock, "the user judges the fun") and no record of them exists (V `grep -ri "questions per 10" docs/probes` finds the drill spec only).

**Concrete minimum definition of "a base we can iterate designs on"** (all U as a proposal; the pieces exist or are one card each):
1. **One command, two minutes:** `tools/play-session.sh` (TASK-43, Proposed) brings up a hidden realtime instance with `-ladder -watch -watch-video` and prints the LAN URL; the user opens it on a phone. No container, no wizard.
2. **Design as data, restart under 30 s:** the catalogue (`internal/decide` JSON), card texts (`internal/watch/cardtext.go` table), fade seconds, N per risk class and the fight-point thresholds (65 and 70 percent life, V `docs/t0-slice.md`) are editable without a Go change; a `--design-dir` override so a design variant is a directory, not a branch.
3. **A session record written by the bot:** questions opened, answered, faded, rules learned, deaths, uptime, orders given, per 10 minutes, plus a three-line note typed by the player at the end; one file per session under `docs/probes/sessions/` (the drill already records most of this per profile, V `docs/probes/t0drill.md` "Result files").
4. **A fixed start state as declared SETUP:** a saved clvl 6 Sorceress in Cold Plains with a tome and six potions (the fidprobe already has Freshe, V `docs/probes/fidprobe.md`), copied in before `d2cheat runstart`, so a design test starts where questions happen, not at clvl 1 in a quiet Blood Moor.
5. **A regression floor, not a ladder:** one Gate 1 smoke (45 wall minutes, soak-gates "Gate 0.5", V `docs/contracts/soak-gates.md` line 275) per merged bot change, on one seed. Nothing longer until a design needs hours.
6. **Parity floor:** fidprobe scenarios 1 to 4 green (fix the normaliser items in `docs/probes/fidprobe.md` "Next steps") and the known-deviation list; the rest of the matrix waits for a design hypothesis that needs the row.

## 3. Process: cost against value

Evidence counted in this worktree (V, commands in the text):
- **Merges:** 1,101 merge commits since 2026-10-24; 614 `Merge agent/*` since 2026-10-31, of which 438 (71 percent) changed no `.go` file (`git log --merges` plus `git diff --name-only` per merge). Per day: 158, 200, 238, 141, 199 (10-31 to 11-04), then 96, 48, 19 (11-05 to 11-07).
- **Merge queue:** 309 entries, 277 merged, 16 failed, 16 dropped; 7 of the 16 failures are conflicts in `docs/cards.md`, a generated file that is committed (`mergeq.sh list --all`; `python3 -I tools/backlog/cards_md.py --write`).
- **Code:** 205,665 Go lines in 66 `internal/` packages, 51,537 in 47 `cmd/`, 91,994 lines of tests. The product layer (ladder 1,636 + decide 4,022 + watch 6,795 + currency 1,587) is about 14,000 lines, under 7 percent; the bot/combat/town/nav core is 63,000 (`wc -l` per directory).
- **Docs:** 985 markdown files, 78,760 lines in `docs/` and one level below; 347 files in `docs/research/`; 18 contracts with 243 `### Requirement` blocks; GLOSSARY 144 terms.
- **Board:** 98 tasks: 46 Proposed, 28 Done, 13 In Progress, 11 Deferred; 32 of the 98 are the fidelity cards created in one pass on 2026-11-07.
- **Gates:** 11 multi-seed Gate 2 records (g2b to g2g2) plus 50 per-seed records in `docs/soaks/`; a wave is 3 slots x 8 wall hours, about 24 host-hours, plus a diagnosis doc of 100 to 160 lines each (`docs/research/g2e-diagnosis.md`, `g2g-diagnosis.md`). Seven waves ran after ADR 0016 (2026-11-05) said "Gate 2 re-runs stop for now" (ADR 0016 item 5; records dated 11-05 16:07 to 11-07 09:53).

What earns its keep (V for the evidence, U for the verdict):
- **The merge queue with the gate set** (leakcheck, contracts check, backlog check, CI): 16 failures out of 309 caught real problems (leakcheck hits, conflicts). Keep.
- **Honesty tags V/S/U and the crutch rule.** They are why this review could be written from the repo. Keep.
- **Diagnosis docs after a failed run.** g2e and g2g found the wall-clock merc cap, the mouth-gate loop and the scroll gap; each turned into a merged fix (CHANGELOG). This is the one place where the gate machinery produced bot quality. Keep the diagnosis, shorten the run.
- **The fidelity probe** (14 h, found a deviation on day one). Keep as a tool.

What should be simplified or stopped (evidence first):
- **Gate 2 waves at 24 host-hours each** judge a T4 behaviour (clvl 16, Andariel criterion, merc) that ADR 0016 demoted. 3 of 5 pass; the failing mechanisms recur (NOKILL from nav pockets and restarts). The next wave "must prove" three fixes (V `docs/research/g2g-five-seed.md` section 9); that proof costs another 24 host-hours and is not on the product path. Replace with the 45-minute smoke per bot change and run Gate 2 once when T4 is the tier being built.
- **Docs volume.** 71 percent of agent merges are docs only; 985 files for one developer and one friend. Reading cost lands on every agent (AGENTS.md reading order) and on the user. Rule proposal: a research doc that is not linked from a task result or a decision within 7 days is archived; new agents read START-HERE, STATUS, their task and at most one area doc.
- **Contracts at 243 requirements** (18 files) for a bot whose product layer is 7 percent of the code and has no contract of its own (`docs/contracts/` has no ladder, watch or session contract; V `ls docs/contracts`). The contract machinery protects the part of the system that is about to be demoted. Freeze new contract text except for the ladder.
- **Backlog:** 46 Proposed cards is a wish list, not a board; the 32 fidelity cards were created before any one of them was wanted. Move Proposed cards that have no milestone or whose milestone is paused into one `backlog/later.md` line each; keep the board at what is In Progress plus the next two per milestone.
- **`docs/cards.md` committed:** 7 queue failures. Generate it at merge (in the merger) or stop committing it.
- **Agent time is not recorded anywhere in the repo** (no field on tasks, no log). Every cost estimate in the plans is "U hours"; nothing closes the loop. Add `spent:` to the task result line.

## 4. Design hypotheses

Pieces on the table (V): teach-the-hero answers (9 of 23 points wired, `docs/t0-slice.md`), the T0 to T5 ladder (ADR 0016), the orders map (slice 1 built, `docs/research/orders-ux-design.md` section 11), Raven's Mark and Tally (registry and ledger offline, `CHANGELOG.md`), LAN play session (TASK-43 Proposed), custom items and the overlay (`docs/overlay.md`, TASK-59 built), the friends build (19 cards). Four hypotheses, cheapest experiment each, and what decides:

**H1: Teaching feels good when the hero asks often, in context, with a visible consequence.** This is the ADR 0016 bet and it is untested: in four clean 10-minute drills no fight-time question opened because a clvl 4 to 5 Sorceress in Blood Moor rarely drops under 70 percent life; the point that did fire was `t0.area.toohard`, 6 to 9 times per run, about the Forgotten Tower, from route planning (V `docs/probes/t0drill.md` Result 3, lines 147 to 155). Experiment: the user plays 30 minutes in realtime from the clvl 6 Cold Plains save (section 2 item 4) with two catalogue variants (today's thresholds; a "first time I meet X" variant that asks once per new monster kind, area and item kind, implemented as data where the call sites allow). Who: the user alone, then one friend over the LAN. Decides: at least 3 questions per 10 minutes in context, at least 1 rule learned, and the player's answer to "did you feel you taught it" is 4 of 5 or better. Below that, the ask mechanism is redesigned before any more point wiring (dpwire, ladder-t1, ladder-t2 wait). Serves: t0-slice, watch screen, playsession.

**H2: The come-back is the game.** An idle game's payoff is the return. Today a returning player gets `/watch` with no summary (V `docs/research/onboarding-ux-design.md` section 6 "Returning user: no tour, no banner") and a journal of hundreds of lines where every missed question is one line (ADR 0016 item 21). Experiment: a paper return screen (what happened while you were away: levels, deaths with cause, drops, areas cleared, questions missed grouped by kind with "answer once for all of them" and "make this a rule"), mocked in HTML like the orders mock, read after a real 4-hour unattended realtime run whose journal is exported. Who: the user. Decides: does the grouped-missed-questions screen produce 3 or more rules in under 5 minutes, and does the user want to leave the hero running tonight? Serves: journal, watch screen; needs no bot change for the mock.

**H3: Orders are the game; the hero is a unit, the player a commander.** The orders map is the most finished player-facing piece and the user's second-ranked shape (ADR 0016 context). Experiment: a 1-hour realtime session with questions switched off (catalogue empty) and only the map, the advice ranges and the journal; count orders given, time between orders, and whether the user stayed in the tab. Decides: more than 6 deliberate orders in the hour and a wish for "until level N" or a queue says the commander loop carries a session; fewer says the map is a utility, not the game. Serves: orders slice 1; slice 2 (until level, queue) only if H3 wins.

**H4 (control): Watching is enough.** One hour with video only, no cards, no orders. Decides: when the user closes the tab. This gives the baseline boredom time that H1 and H3 must beat. Costs nothing but the hour.

Not a hypothesis yet: the metacurrency and the altar. Their first sink is "a faster answer fade" (decisions.md 2026-11-06 answer 5), which is a fix for a design annoyance sold as progression. They cannot be tested for fun until H1 shows the questions are worth answering; keep the ledger code, stop the slice.

## 5. Player-experience pass (first-time player, on paper)

Sources: `docs/research/mocks/onboarding.html`, `orders.html`, `friends-preview.html` (read as text and JS strings), `docs/research/onboarding-ux-design.md`, `tester-build-plan.md`, `docs/t0-slice.md`, `docs/probes/t0drill.md`, the Gate 2 records for pace. A friend tester's likely words are my guesses (U).

**Install and onboarding.** Edit two compose lines, `docker compose up -d`, read the log for a URL with a token, then six setup screens (Welcome, Go-ahead with a tick box, Game files, Hero, Access with a passcode, Start with a 1 to 3 minute warm-up), then a five-step tour (V onboarding mock STEPS and TOUR arrays; the friends preview adds three "honest questions" tick boxes before that). That is eleven screens and a terminal before the hero appears. The copy is good and plain. The quantity is not: a friend who can run Docker needs two screens (files, name). Friend: "That was a lot of reading for a game that plays itself."

**First 10 minutes.** The hero is clvl 1 in the Rogue Encampment, walks to Blood Moor, kills Fallen. The Game panel moves. Thoughts is empty: the sample card from the tour is the only card they will see for a while, because fight-time points need life under 65 to 70 percent and a clvl 1 to 5 hero rarely gets there (V `docs/probes/t0drill.md` Result 3: "in e to h no fight-time point opened in 40 minutes"). The tester plan already tells the guide to say "questions can take a while; the hero is not broken" (V onboarding design section 7). Friend: "Is it working? What am I supposed to do?"

**The first question the hero asks.** By the live evidence it is `t0.area.toohard`: "the Forgotten Tower is too hard for you right now: go, skip, or lower?", asked from route planning at clvl 4 to 5 about a place the hero is nowhere near, repeated 6 to 9 times in ten minutes (V t0drill Result 3; `docs/t0-slice.md` line 35 flags the timing as U). Or a town card: "Repair now?" with durability at 90 percent. Neither is the moment the design promises (the Fallen pack at 30 percent life). Friend: "Why does it keep asking me about the Forgotten Tower? I said skip." Ranked first among the unclear moments.

**The watch screen.** Five panels, uptime, deaths, a fading bar. Uptime drops only on deaths today (V `docs/research/watch-screen-design.md` section 3 item 5). The question card itself is well designed (headline with real values, "if you do nothing", what each answer does, footnote). The Rules panel is empty for a long time. Friend, if they hit a real card: "Oh, that is nice." Then nothing for ten minutes.

**A 1-hour session.** At 1x a fresh Sorceress reaches about clvl 5 to 6 (the lockstep gates reach clvl 16 in 1,700 to 3,000 virtual minutes, V `docs/research/g2g-five-seed.md` table; the clvl 6 probe save is a real one-hour-class character). One or two deaths are likely (seed 13 dies deterministically at clvl 4, V `docs/research/g2e-diagnosis.md` E3), one or two restarts from a nav pocket (V g2g F1, F2). Cards: three to five town cards, the repeated Forgotten Tower question, maybe one retreat card. Rules learned: 0 to 2. Orders: Blood Moor, Cold Plains and the Den are the only sensible targets; the map shows 13 nodes most of which say "hard". Friend: "It died. Was that my fault? I never saw the question." Ranked second: death without a question.

**A day away.** At 1x the hero reaches about clvl 14 to 16 (half the lockstep distance; U), the merc is dead most of the time (alive share 4 to 30 percent in the gates), 25 to 88 fresh games have happened (V g2g table), and the journal has one "missed" line per faded thought. Nothing was learned while the player was away; the hero played at the untrained default the whole time, which ADR 0016 item 11 names as "the intended penalty". Friend: "So if I do not watch, it does not learn? Then it is not an idle game." Ranked third: the idle promise and the fade rule contradict.

**Coming back.** `/watch` opens on the live state with no summary (V onboarding design section 6). The Journal is the only account of the day. Friend: "I do not know what happened. I will leave it running, I guess." Then they do not come back, because nothing asked them to. Ranked fourth: no return hook.

Ranked list of unclear, boring or not-fun moments, with the file and screen:
1. The first question is out of context and repeats (`t0.area.toohard` from `fillRoute`; t0drill Result 3; watch Thoughts panel).
2. Deaths happen without a card and without a "what would you have done" afterwards (g2e E3; watch state bar).
3. Idle promise against 45 s fades: an away player learns nothing (ADR 0016 items 10 and 11; friends preview "ignore it for hours").
4. No return screen (onboarding design section 6).
5. Eleven screens before play (onboarding mock STEPS, TOUR; friends preview checkboxes).
6. Quiet first 10 minutes with an empty Thoughts panel (t0drill Result 3; tester plan G10).
7. Realtime pace: an hour buys five levels and one map node (gate records, U at 1x).
8. The orders map shows mostly "hard" nodes at clvl 1 to 6 (orders mock scenario A).

## 6. Stop list and keep list

Stop or pause now (one reason each):
- **Gate 2 waves and any Gate 3 or Gate A run:** they measure T4 behaviour the ladder demoted, at 24 host-hours a wave, and the user's words were "iterate on designs"; a 45-minute smoke per bot change is the regression floor.
- **Fidelity slices 3 to 6 and the 29 new fidelity cards (m-2):** keep the probe and its known list; no design question needs MA04 or the 40 h A:B this fortnight.
- **Metacurrency slice (currency-rows, currency-slice, custom-item-1):** the sink it buys fixes a design annoyance; it cannot be judged before H1.
- **Tester container work beyond what the LAN session needs (testerd2dimage, testerd2dwiz, testerlive, testerpack, testerlicence, testerreport, testerdocs):** the first friend can play over the LAN from the user's host; the container is a distribution problem, and distribution is what triggers the GPL and EULA questions in section 8.
- **Onboarding slices B and C, orders slices 2 and 3:** polish on a loop that has not been shown to be fun; H3 decides slice 2.
- **Unwired ladder points (dpwire, ladder-t1, ladder-t2):** wiring more points into a mechanism that asked the wrong question first makes more wrong questions.
- **New contracts, new research docs without a consumer, committing `docs/cards.md`:** process cost with no reader.
- **Merc work, Andariel, Act 1 clear (m3-*):** T4 content; parked already, keep it parked.

Keep (one reason each):
- **The lab binary, lockstep and the drill suite:** the only way to run many design variants in one afternoon.
- **fidprobe slices 1 and 2 as a tool:** cheap, found a deviation on day one.
- **The watch screen, question cards, orders map slice 1, journal:** the player-facing base; iterate on them.
- **The decision-point store and the 9 wired points:** the mechanism to test H1 against.
- **The merge queue, leakcheck, contracts check, backlog check:** they catch real mistakes.
- **One Gate 1 smoke per bot change:** regression floor.
- **The upstream PR path (one batch at a time):** protects the engine; costs little while a batch waits.
- **The published fork `d2d-didle`:** done; it is also the honest answer to the GPL source offer.

## 7. Sequenced plan: the next two weeks

Costs are agent-hours and worker-host hours, all U. "You see" is what the user sees at the end of the step.

1. **Day 1 to 2: play-session command and session record** (TASK-43 plus a session-record line in the bot; about 8 agent-hours, 1 host-hour). You see: one command, a phone URL, a realtime hero with video and cards, and a file per session with questions per 10 minutes. Invalidates the plan if: realtime play trips the lockstep-only clocks (tester plan G6 lists them) and the hero stalls; then step 1 grows by a day.
2. **Day 2: H4 and H1 baseline** (the user plays 1 hour watching, then 30 minutes from the clvl 6 save; 0 agent-hours, 2 host-hours). You see: your own boredom time and the real question count. Invalidates: if the count is already 3 per 10 minutes and it feels good, skip step 3.
3. **Day 3 to 5: question-density variant as data** (fight-point thresholds, "first time I meet X" points where a call site exists, fade 45 to 90 s, the Forgotten Tower question gated to "within two areas of the Forgotten Tower"; about 16 agent-hours). You see: a second 30-minute session with a different feel and a comparable record. Invalidates: if neither variant reaches the H1 bar, the ask mechanism is the problem and step 6 moves up.
4. **Day 4 to 6: return-screen mock and a 4-hour unattended realtime run** (HTML mock like the orders one, journal export; about 10 agent-hours, 4 host-hours). You see: the day-away story on one page with grouped missed questions. Invalidates: if the 4-hour run cannot finish without a restart loop, the smoke in step 5 is what fixes it, not more design.
5. **Day 5 onward: smoke per bot change** (reuse Gate 0.5; about 4 agent-hours to script, 45 host-minutes per change). You see: a one-line verdict on every bot merge instead of a 24-hour wave. Invalidates: a design change that needs hours of play to show; then one Gate 1 on one seed.
6. **Day 7 to 9: H3 orders session and the H1 verdict** (the user plays 1 hour with orders only; 0 agent-hours). You see: three numbers (questions, rules, orders) and three sentences you wrote, side by side for H1, H3, H4. Decision point: which loop gets the next week.
7. **Day 9 to 12: one friend over the LAN** (the user's host, an SSH tunnel or a LAN bind as documented in `docs/t0-slice.md`; the friends preview goes to them first; about 4 agent-hours for a one-page how-to). You see: a second person's three sentences. Invalidates: if the friend cannot reach the host, the container comes back as the next step, scoped to one person.
8. **Day 12 to 14: write the design brief v1** (one page: the loop that won, the three metrics with their target, what is cut; replaces the ranking paragraph of decisions.md Current truth through your grill; about 4 agent-hours). You see: a direction you can say in two sentences.

Total about 46 agent-hours and 8 host-hours, against one Gate 2 wave of 24 host-hours plus its diagnosis.

## 8. Risks the team may be blind to

- **Legal exposure grows with distribution, and distribution is in the plan.** NOT LEGAL ADVICE. Blizzard's end-user licence is understood to forbid bots and reverse engineering of its games (S, common knowledge, not fetched); the repo's own oracles describe game.exe functions by address and run them under an emulator (V the `FUN_` references throughout `docs/research/fidelity-matrix.md`), and the repo "is going public soon" (V `docs/licensing.md`). A private bot on your own machine is one risk class; a public repo that documents decompiled behaviour plus a shipped image is another. The docs name "Blizzard exposure" once as "outside what these docs can answer" (tester plan R6) and never return to it. Recommendation: before any copy leaves the host, a one-page risk note that a lawyer could read, and keep the decompile-derived research in the private ops repo (U whether that split is practical).
- **d2d's own provenance.** The engine is "a reimplementation with 1.14d as ground truth" (ADR 0009); the project name of the upstream contains "decomped". If upstream is ever taken down, the product has no engine. The fork publication (2026-11-07) ties d.idle's name to it. U: upstream's clean-room status; nothing in this repo records it.
- **Single-user dependency.** Every decision needs a popup, every run a go, every doc a user edit of the guard files; `docs/decisions.md` records 13 dated user decisions in the four days 2026-11-04 to 11-07 alone (V, count of dated entries), and the user is also the only player. If the user stops for a week the project stops; if the user is the only tester, "fun" is one person's taste. Recommendation: the friend in step 7 is also a hedge, and the session record makes a second opinion possible without the user present.
- **Lockstep results do not transfer to the product.** All stability evidence is at 3.5 to 5x; realtime clocks (hang detection, no-progress timers, fade) are different code paths (tester plan G6, `docs/d2d-wire.md` live result 5). The product mode has zero hours of evidence.
- **The hero is the game and the hero is bad at clvl 1 to 6.** Deterministic early deaths (seed 13), poverty loops, nav pockets: the player meets the bot's worst hour first. The ladder makes "the untrained default is poor" a feature (ADR 0016 item 11), which turns a bot defect into a design claim. If H1 fails it will be hard to tell whether the mechanism or the defaults failed.
- **Scope by accretion.** Four milestones, 98 cards, 243 requirements, a metacurrency, custom items, an altar, hardcore rebirth, five acts as expansions, a tester container, a published fork, a public repo. Each was approved one popup at a time. The user's sentence "we have been a bit aimless" is the sum of those popups.
- **The idle-versus-attention contradiction** (section 5 item 3) is a product risk, not a UX detail: if the answer is "it learns only while you watch", the game is a 30-minute-a-day companion app and the marketing, the fade rule and the return loop must say so.

## 9. Decisions for the user

Each is phrased as a popup with options and my recommendation. Proposed record text is in brackets for the coordinator or your grill session; agents do not write the guard files.

**D1. What do we optimise for the next two weeks?**
- (a) Finding the fun: play sessions and design variants, gates paused (recommended).
- (b) Gate 2 pass first, then design.
- (c) Both in parallel with the WIP limit.
Recommendation (a): the user asked for a design base; a Gate 2 pass changes nothing a player sees. [Record: "2026-11-xx (user): for two weeks work is ranked by the design-base steps of docs/research/direction-review.md section 7; Gate 2 and Gate 3 waves, fidelity slices 3 to 6 and the metacurrency slice are paused; a 45-minute smoke per bot change is the regression floor."]

**D2. Is d.idle an idle game or an attention game?**
- (a) Attention game: sessions of 20 to 60 minutes, the hero learns only when you watch, fades stay short (honest to today's design).
- (b) Idle game: questions queue while you are away and are answered in bulk on return; rules can be taught from the journal (recommended).
- (c) Decide after H1 and H2.
Recommendation (b), subject to H2: it is the only reading in which "ignore it for hours" and "teach the hero" are both true. [Record: "d.idle is an idle game: unanswered questions do not fade for the player; they fade for the hero (it acts on its default) and queue grouped by kind for the return screen, where one answer may teach a rule."]

**D3. Who tests first, and how?**
- (a) You, then one friend over the LAN from your host (recommended).
- (b) The container to one Docker-literate friend (today's plan, 19 cards).
- (c) Nobody outside until H1 passes.
Recommendation (a): same evidence, one tenth of the cost, no distribution, no GPL or EULA trigger. [Record: "the first outside tester plays over the LAN; the tester container waits for the design brief v1."]

**D4. Metacurrency and custom items now or after the fun verdict?**
- (a) Finish the first slice (about 42 h) as planned.
- (b) Keep the ledger and overlay code, stop the slice until H1 passes (recommended).
- (c) Drop custom items; sacrifice real items only (ADR 0016 item 15 original).
Recommendation (b). [Record: "metacurrency v1 is paused; its first sink is decided after the H1 verdict."]

**D5. Board and docs diet.**
- (a) Move Proposed cards without an active milestone to `backlog/later.md`; archive research docs with no consumer after 7 days; generate `docs/cards.md` at merge instead of committing it (recommended).
- (b) Keep the board as is; only fix cards.md.
- (c) No change.
Recommendation (a). [Record: "the board holds In Progress plus the next two cards per active milestone; everything else is one line in backlog/later.md."]

## Sources

`docs/START-HERE.md`, `docs/plan/STATUS.md`, `docs/decisions.md`, ADR 0003, 0009, 0015, 0016, `PLAN.md`, `GLOSSARY.md`, `docs/research/{fidelity-matrix,fidelity-probe-design,fidelity-cards,tester-build-plan,onboarding-ux-design,orders-ux-design,metacurrency-design,custom-items,g2g-five-seed,g2g-diagnosis,g2e-diagnosis,backlog-groom-2026-11-06,automation-ladder,watch-screen-design}.md`, `docs/probes/{fidprobe,t0drill}.md`, `docs/t0-slice.md`, `docs/capabilities.md`, `docs/licensing.md`, `docs/research/mocks/*.html`, `docs/soaks/gate2-multiseed-*.md`, `backlog/tasks/*.md`, `backlog/milestones/*.md`, `docs/contracts/soak-gates.md`, `CHANGELOG.md`, `git log` and `wc -l` in this worktree, `mergeq.sh list --all` of the ops tooling. Not read: the fork source, the ops repo beyond the queue list, any live game.
