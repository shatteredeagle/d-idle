# Decision-point store: spec (DRAFT, no code)

> status: DRAFT spec for the learn-by-doing mechanism of [automation-ladder.md](automation-ladder.md) (decisions of 2026-11-05, section 7 there). No card exists (card cap 12 reached); if built, the natural owner is the T1/T2 lane (task-11 trip loop is the nearest open card) or an unowned new card once a slot frees. Nothing here is built.

Tags: **V** = read in this repo (file cited), **S** = veteran-player answer from the d2-sme agent in the ladder doc (memory, unverified), **D** = user decision 2026-11-05, **U** = unverified or my design choice. Classic 1.14d.

## 1. Purpose and rules (D)

A decision point is a named place where the hero may ask. The store holds the player's answers, counts consecutive identical ones per context, and turns them into learned rules. Rules (D):

- Same answer in the same context N times IN A ROW teaches a rule. A different answer resets the streak to 1 with the new answer.
- N by risk class: safe 2, normal 3, risky 5. N is data (tunable), not code.
- Rules are revocable from a list; after a revoke the hero asks again and the streak restarts at 0.
- A sacrifice can skip one repetition (the streak counts one extra).
- A new act or difficulty is a new context: the context key includes both, so nothing carries over.
- Deterministic: no ML, no fuzzy matching. The same answer log gives the same rules (U, needed for comparable soaks).

## 2. Data model (U)

```
DecisionPoint   (static catalogue entry, shipped as data)
  id            string  stable, "t1.keep.rune"
  tier          0..5    unlock order; only offered when tier-1 is unlocked
  question      text    plain language for the card
  answers       []Answer {id, label}      small closed set (2 to 5)
  risk          safe | normal | risky     picks N
  context_keys  []string  which state fields form the context (see 2.1)

Context         (runtime, built from the snapshot)
  point id + values of context_keys, e.g. {act:1, diff:normal, itemKind:rune}
  key = point id + sorted "k=v" of context_keys     (a string; bucketed values only)

Record          (persisted per key)
  key, last_answer, streak, answered_at (virtual time), skips_banked
  rule: {answer, learned_at, revoked bool, revoked_at}   absent until learned

Config
  n_by_risk {safe:2, normal:3, risky:5}      (D defaults)
  sacrifice_skip {enabled, max_skips_per_point}   (U: 1)
```

### 2.1 Context keys

State must be bucketed so that identical situations map to one key. Allowed fields (U): act, difficulty, area class (town, field, boss), item kind or rune rank class, life band (the retreat decision), gold band, free-rows band, run kind. Continuous values are bucketed in data, never raw (a raw life percent would never repeat). The bucket table is part of the catalogue so a bucket change is visible in review. Class and build are NOT in the key unless the point says so (build-defining points are keyed by the plan id; U, the build-flexibility rule in V AGENTS.md).

### 2.2 Streak algorithm (U)

```
Answer(key, a):
  r = load(key)
  if r.rule exists and not revoked: return     # already automatic; answers are not asked
  if r.last_answer == a: r.streak += 1 (plus banked skips) else r.last_answer = a; r.streak = 1
  if r.streak >= N(risk): r.rule = {answer a}; emit RuleLearned
  save(r)
Revoke(key): r.rule.revoked = true; r.streak = 0; r.last_answer = ""
Sacrifice(point): r.skips_banked += 1 (bounded)
```

A learned rule applies for every context that produces the same key. Revoked rules are kept (history) but not applied.

## 3. Where it lives (U)

- **Rules and streaks: per profile (account), not per character.** Hardcore rebirth resets the character and keeps the rules (D). A per-character file would be wiped with the character. In the repo's terms the profile is the per-instance data directory beside the goals state file (V `internal/goals/store.go` writes the goals state atomically with `.bak` and a `FileVersion`; the store should reuse that pattern: `writeAtomic`, version number, refuse a newer file).
- **Altar quality levels** (not part of this store) live next to the rules in the same profile file so that rebirth keeps them (assumed; the user said unlocks survive, the altar quality case is still open in the ladder doc).
- **Answer log:** append-only, one line per answer and per revoke, with virtual time. The rules are a pure function of the log, so a soak can replay a fixed log (U).
- Bot reads the store at start and on a `RuleLearned` event; it never writes game memory (no crutch problem: the store is bot-side state, not a game write; V AGENTS.md crutch rule concerns game state).

## 4. Event flow (U)

1. The bot reaches a decision point (a call in the tier's code path, see 6) with a Context.
2. `store.Lookup(ctx)`: a live rule returns its answer and the bot continues automatically. A capability whose tier is not unlocked cannot run past its decision point at all.
3. No rule: the bot emits `needs-input{pointId, ctx, question, answers}` and enters the waiting state.
4. Waiting state: the hero goes to a safe place (town by town portal or waypoint, or stands still in town if already there), no panels open, not idle-failing the watchdog (heartbeat stays live; the existing blocked wait in town already does sliced sleeps, V internal/bot/task.go near line 972).
5. The dashboard shows a question card (cmd/d2dash and the real dashboard server); the player answers.
6. `store.Answer` updates the streak. If the threshold is met the rule is learned and a `RuleLearned` event is emitted (card says "the hero now does X on its own; revoke").
7. The bot resumes with the answer. From the next time at the same context the capability runs automatically.
8. A timeout rule for unattended soaks (U): after T seconds with no answer the point resolves to its declared default and the incident is logged; defaults never count toward a streak.

## 5. First catalogue, T0 to T2 (S for the situations, U for N class and context)

Source of the situations is the SME list in the ladder doc section 3 (S: memory, no lookup).

| id | tier | question (short) | answers | risk | context |
|---|---|---|---|---|---|
| t0.retreat.drink | 0 | life low: drink, run or portal | drink / run / portal | normal | life band, area class, pack danger class |
| t0.retreat.lethal | 0 | a pack type that killed or nearly killed us | avoid / fight | risky | pack type |
| t0.area.toohard | 0 | area far above the hero | go / skip / lower area | risky | area, level gap band |
| t0.retreat.repeat | 0 | two retreats in a row in one area | stay / leave | normal | area |
| t1.restock | 1 | restock belt potions now? | yes / no | safe | gold band, belt state |
| t1.return | 1 | return to the field after town | yes / no | safe | area class |
| t1.keep.kind | 1 | keep this item kind? | keep / sell | normal | item kind (rune rank class, gem class, charm, jewel, rare/unique tier) |
| t1.town.trigger | 1 | go to town now (free rows low) | go / continue | safe | free-rows band |
| t1.tp.lost | 1 | no town portal scroll | buy / walk | normal | gold band |
| t1.stash.full | 1 | stash full | sell / drop / pause | normal | stash band |
| t2.repeat | 2 | repeat the same run | again / stop | safe | run kind |
| t2.failstreak | 2 | stop after repeated deaths | stop / continue | risky | run kind, death streak band |
| t2.slowrun | 2 | yield or time worse than expected | continue / switch | normal | run kind |
| t2.bossunsafe | 2 | boss unsafe at current gear | skip / go | risky | boss, gear band |

Always-ask points (S: build-defining; N effectively never reached, rule is not offered): skill and stat allocation, respec, unique or set swaps. They belong to T3 and are out of the first cut. The "pick area" point of the user's list is `t0.area.toohard` plus `t2.repeat` in this cut (U); a "choose which area next" point is T3.

## 6. Interfaces to existing code (V unless marked)

- **Goals and runner.** `internal/goals` owns the goal types (`Runs` goal in `internal/goals/runs.go`, state file in `store.go`) and `internal/bot/runner.go` `type Runner` is the real `goals.Runner`, a tick loop on a snapshot. The store is called from the bot code that already decides these things (retreat in the combat loop, town trip in `internal/town`, run repeat in the Runs goal). A new `Runner` status value (`waiting for input`, U) is needed so the goal manager and watchdog know the wait is deliberate; the goals `Event` channel (V runner.go `ch chan goals.Event`) and the event bus (`bus *event.Bus`) carry `needs-input` and `RuleLearned` (U).
- **The existing "blocked: needs input" line.** `internal/bot/task.go` (around line 972) logs `leveling: blocked: needs input` when fresh games gain no XP and probes one restart per `route.freshBlockedSec` while waiting in town (V; added under TASK-C, see backlog/tasks/task-18). That is today a log line with no card and no way to answer. Under this spec it becomes `t2.blocked.leveling` (id U) with answers like `keep waiting / change area / stop`, and the wait in town is the standard waiting state of section 4. Migrate it last; do not change its numbers (V route knobs in `internal/env/tuning_route.go`).
- **Tuning knobs.** `api.Tuning` (V internal/api/tuning.go: `Snapshot`, `Edit`, `Unlock(id)`, errors `ErrInvalidTune`, `ErrUnknownUpgrade`) already has an `Unlock(id)` for upgrades, and `internal/env` knobs have min, max and a better direction (V docs/knobs.md). The store reuses that: a learned rule or a sacrifice level writes a knob override through the same path; N per risk class are config values, not knobs named after skills (build-flexibility rule, V AGENTS.md).
- **Dashboard.** `cmd/d2dash` serves the dashboard with the simulated runner (V cmd/d2dash/main.go: manager, `internal/server`, `web`), so the question card, the rules list and the revoke button can be built and tried without a game (U; this is the intended place). Needs an API: `GET /decisions` (open cards and rules), `POST /decisions/answer`, `POST /decisions/revoke` (U names), following the style of the knobs routes (V `KnobsHandler`).
- **Sacrifice.** The altar is a d2d custom object (D) and out of scope here; the store only exposes `Sacrifice(point)` as the skip hook.

## 7. Test plan (deterministic, no ML)

- Unit (offline, `go test` per package with `-p 1`): streak table tests for the algorithm (N reached; reset on a different answer; revoke; skip; risk class N; new context key means a new streak; a default-by-timeout never counts); log replay gives identical rules; atomic write and refused newer version, like the goals store tests (V internal/goals/store_test.go).
- Two builds (V AGENTS.md "Build flexibility"): run the same catalogue for the default Sorceress and a second class or build (Paladin is the other offline build, V docs/capabilities.md) and assert that no point id or context bucket names a class; a build-keyed point must fail with an error naming the plan when the plan is unknown, never fall back to the Sorceress.
- Integration with `mem.Fake` / the goals `sim` package (V internal/goals/sim): a scripted run reaches `t1.restock`, emits needs-input, the test answers N times, the rule is learned, the next trip runs with no card.
- Gate wiring (U): soaks and gates run with a fixed answer script and report `rules learned: K, answers given: M` next to `writes after start: N`.

## 8. Out of scope

The altar object and its costs; the worth rule numbers; the T3 to T5 catalogue; the question-card visual design; the order UI; hardcore rebirth handling beyond "rules live in the profile"; any change to game memory; any ML or fuzzy context matching; changing existing knob defaults or gate numbers.

## 9. Risks (U)

1. **Context key design.** Too fine and nothing is ever learned (every key is new); too coarse and the hero learns a wrong rule. The bucket table needs real play data; start coarse and split by evidence.
2. **Soak comparability and testing.** Learned rules make behaviour depend on the answer log; hardcore rebirth adds a state (rules present, character fresh). Fixed scripted logs and a recorded rule set per gate run are required.
3. **Unattended waiting.** A wait in town can idle the hero for hours (uptime drops, watchdog and the 60 min live limits); the default-on-timeout rule keeps soaks finishing but must never teach rules.
