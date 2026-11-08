# The Pact system: a design note (not a decision)

> status: design note of 2026-11-07, written at the user's go ("design note + cards, build later"). NOT a decision: `docs/decisions.md` and the ADRs are untouched; proposed decision lines are in the closing section. The cards are Proposed, milestone m-4, and none of them is on the two-week playable path. Nothing here has run in a game.

Tags: V verified (a file in this tree was read, or a page was fetched), S secondary (a search snippet or second-hand), U unverified (veteran memory or a design guess). The veteran numbers come from a d2-sme report (all EXP, none looked up); the prior-art notes come from a research pass (18 sources, 2026-11-07). Every numeric REWARD value below is U: a starting point to tune in play.

## 1. Purpose, the user's words and decisions

The user's words (dictated): "Hardcore Pact is a really good idea, actually how should we think about a Pact system, modify the game difficulty (more elites, more damage, etc...) for buffed rewards. Unlockable/upgradable system"; "Unlock pacts after beating the game for the first time, unlocks the Pact of Nightmares (nightmare difficulty)".

Decisions taken in popups the same day:

- The first Pacts unlock when ANDARIEL is killed for the first time.
- The Pact of Nightmares unlocks when BAAL is killed in Normal. This is D2's own Nightmare unlock for 1.14d LoD. The user first said Mephisto, then chose Baal after being told Baal is the LoD final boss.
- The Pact of Nightmares moves THE SAME HERO into Nightmare with a Pact bonus. It is not a new hero.

Words the user has not defined (asked in section 9, not guessed): whether "Pact of Nightmares" is one Pact or a family, and whether "Hardcore Pact" is a heat-costed Pact or a mode flag.

Why: the loop (ADR 0018) pays the player in tally, firsts and the Hall. A Pact is the opt-in way to make the same run harder for a larger payout, so the second and third hero of a player have a reason to differ from the first.

## 2. The model

### 2.1 Principle

1. A Pact changes bot policy, or opts into difficulty that D2 itself offers (Hardcore, Nightmare, Hell, `/players N`), or (later) a mod-data overlay. It never edits a reward directly.
2. Rewards sit on the meta layer: tally multiplier, legacy charm tier, Hall titles, cosmetics. Vanilla XP and loot never change except through D2's own difficulty settings. Reason: the project's fidelity rule, and reward inflation (research caution 1).
3. Opt-in only. No Pact is mandatory and none gates a first unlock behind grinding (research caution 5).

### 2.2 Slots, ranks, heat

| part | rule | tag |
|---|---|---|
| Slots | 2 at the Andariel unlock; a third from a Hall or tally milestone. Tally buys only extra slots, never ranks | U |
| Ranks | 1 to 5 per Pact (the SME ladders are 3 steps; section 3 gives 5 to fit the brief). Rank r of a Pact costs `heat_per_rank * r` | U |
| Heat | each Pact has 1 to 3 heat per rank. Costlier heat sits on the levers that kill the hero (Hades pattern: speed and choice cost 3 per rank, V source 1) | U |
| Cap | 12 heat at the start; each mastery (a Pact cleared at its top rank) raises it one step toward 20. Total available heat exceeds the cap so the player must choose | U |
| Reward | linear in heat up to the cap, flat above it; above the cap only a title or frame (Hades: bounty stops at 20, V source 2) | U |
| Change | only at run start or in town with no expedition out, never mid-expedition (Dead Cells, V source 8) | U |

Heat cap 12 matches the veteran estimate of "about 12 rank-weighted points per Act 1 for clvl 1 to 18" (EXP).

### 2.3 Unlock events and upgrades

| event | what unlocks | note |
|---|---|---|
| Andariel killed, first time (any hero on the account) | the first Pacts (the bot-side catalogue of section 3.1), two slots, the Hardcore mode flag | user decision |
| Baal killed in Normal | the Pact of Nightmares | user decision; same hero; D2 unlocks the Nightmare difficulty at this point (S, D2 rule) |
| Baal killed in Nightmare | the Pact of Hell, same structure | proposed, not a user decision |
| a Pact cleared at rank r | rank r+1 of that Pact | Ascension / Dead Cells pattern (V sources 5, 8); "cleared" = the hero killed Andariel with it active (U: the definition belongs to the unlock ladder card, TASK-153) |
| a Pact cleared at rank 5 | one mastery: cap step, Hall title | U |
| hall or tally milestone | third slot | U |

Upgrade means a rank unlock earned by a clear, not a purchase. This stops the player from buying their way into a heat they never survived.

### 2.4 Nightmare and Hell as special Pacts

The Pact of Nightmares is not a rank ladder. It is one Pact with a fixed heat chunk (proposed 8 of 12, U), so it leaves little heat for the others. It moves the same hero into Nightmare once Baal is dead in Normal, with a Pact bonus: a tally multiplier while the hero plays in Nightmare, and a Hall title on the first Nightmare Andariel kill. The legacy charm tier is granted on retire after Nightmare, not on entry. Rule-restriction Pacts may combine with it. Hell is the same shape one step later. The veteran's view (EXP): Hell is not usable by a bot at clvl 1 to 18, so Normal is the arena for early Pacts.

### 2.5 Hardcore

Hardcore works offline (EXP, U until the live check). Proposed default, which the user has not ruled: it is a mode flag with its own hard confirmation, not a heat-costed Pact. Reason: permadeath on a consent sheet next to the other Pacts is the pattern research caution 4 warns against, and a flag can carry its own separate Hall row (D2 and Path of Exile both keep Hardcore ladders apart, V sources 14, 15). The flag must be chosen at hero creation (D2 cannot turn it on later; EXP).

### 2.6 Legacy interplay

The retire-after-Andariel loop (TASK-135) and the legacy charm (TASK-136, parked) stay separate and optional. A Pact adds to them in two ways only: the charm tier gets +1 for each three Pacts held at rank 3 or higher at retire (SME proposal, U), and the Hall row for a retired hero lists the Pacts it carried and the highest heat. A dead Hardcore hero enters the Hall with its Pacts and cause of death.

## 3. The catalogue

Lever families: **bot** (changes a policy-sheet row or knob; offline, the cheapest), **native** (D2 itself: Hardcore, `/players N`, Nightmare), **overlay** (table or spawn changes through the d2d overlay; game.exe needs a mod-data layer and the user's go). Names are the veteran's suggestions (U). Ranks give the effect at 1, 3 and 5; ranks 2 and 4 interpolate. Reward columns are tally multiplier deltas at ranks 1/3/5 and are U. Heat per rank in brackets.

### 3.1 Bot-side Pacts (first wave)

| Pact | family | rank 1 / 3 / 5 | heat | tally at 1/3/5 | policy-sheet rows reused |
|---|---|---|---|---|---|
| Blood Raven's Tithe | bot | no belt potions below clvl 8 / below clvl 14 / never buy or drink belt potions | 3 | +6 / +30 / +90 % | `belt` (`town.beltFillPct` 0 at rank 5 would be a new value: the shipped options stop at 25), `potionmix`, `potiontier`; the drink rule is a new `t0` answer (gap) |
| Andariel's Shadow | bot | death limit 3 / 2 / 1 per expedition before the hero parks in town and waits for the player | 2 | +5 / +25 / +75 % | none today: a death counter over `EvDeath` events plus an order "park" (gap, section 8) |
| Rakanishu's Bargain | bot | no town portal after the Den quest / never / never and no waypoints | 2 | +6 / +30 / +85 % | `tp` (`walk`), `tpreserve` (`one`); the waypoint ban is new |
| Griswold's Debt | bot | repair costs 2x (bot pays to stay un-repaired till 50 %) / 4x / no repair | 1 | +5 / +20 / +45 % | `repair` (`low`, `never`) ; the cost multipliers are cosmetic, treat ranks 1 and 2 as `low` and 3 to 5 as `never` (U) |
| Cain's Silence | bot | sell unidentified from rank 1 | 1 | +5 / +10 / +25 % | `identify` (`none`), `sell` |
| Pact of the Cursed Gold | bot | gold lost on death 10 % / 30 % / all carried gold | 2 | +6 / +30 / +90 % | none: a gold sink on `EvDeath` (a ledger entry only, since a bot cannot lose game gold without a memory write; U: this is a tally penalty, not a game change) |
| The Hungry Dark | bot | only the skills of one chosen tree / one skill only (mono) | 3 | +8 / +35 / +100 % | none: plan table filter in `internal/plandata` (gap); data, not a Go class check |
| The Stranger's Toll | bot | hire no merc until clvl 10 / until Andariel / never | 2 | +5 / +25 / +60 % | none today; `internal/merc` hire knobs (`HireGoldMult`) could gate it (gap, U) |
| Warriv's Long Road | bot | walk everywhere: no waypoints after the first / no waypoints / no waypoints and no TP | 2 | +5 / +28 / +80 % | `tp` (`walk`); waypoint ban is new. Overlaps Rakanishu's Bargain at rank 5 (incompatible, section 3.4) |

The following four are also bot-side and low heat (the veteran's "chores Pacts"): no gambling (1 heat), no stash (1), no repair (1), sell-unidentified (1, listed above). They exist to teach the system; their tally is small (max +10 %).

### 3.2 Native Pacts

| Pact | family | rank 1 / 3 / 5 | heat | tally at 1/3/5 | status |
|---|---|---|---|---|---|
| Hardcore | native flag | one death ends the run (flag, not a ranked Pact) | n/a (flag) | x1.5 and a separate Hall row (U) | needs a live check: creating a Hardcore hero offline on 1.14d (U) and what the bot does after the "dead" screen (U) |
| Pact of Rotting Bone (`/players N`) | native | `/players 2` / `/players 4` / `/players 6` | 2 | +6 / +30 / +85 % | UNVERIFIED offline on 1.14d: the veteran is "fairly confident", range 1 to 8, monster life about +50 % of base per extra player (UNSURE), damage, to-hit and resists do not scale, drops improve, XP effect UNSURE, champion and unique frequency does not change. Card `pactnative` assigns a live check |
| Pact of Nightmares | native (special) | fixed chunk, same hero into Nightmare (section 2.4) | 8 fixed (U) | per section 2.4 | blocked until the bot plays past Act 1 (STATUS: nothing past Act 1 has run live, V) |
| Pact of Hell | native (special) | as above one step later | 12 fixed (U) | per section 2.4 | not a user decision; blocked |

Per the veteran, `/players` at rank 5 on a clvl 3 Sorceress in the Blood Moor means Fallen and Zombies take about 4.5x the hits and Fire Bolt cannot keep up (EXP). The user should decide if `/players` Pacts are wanted at all (section 9).

### 3.3 Overlay Pacts

| Pact | family | rank 1 / 3 / 5 | heat | tally at 1/3/5 | status |
|---|---|---|---|---|---|
| Bishibosh's Wager | overlay | elite groups +10 % / +30 % / +60 % | 3 | +8 / +35 / +100 % | which table drives elite chance is UNSURE (candidates: `Levels.txt` MonUMin, MonUMax per difficulty, MonDen, and `SuperUniques.txt` for named uniques); card `pactoverlay` researches it; d2d first |
| Pact of the Wandering Deep | overlay | pack size +1 / +2 / +4 | 3 | +8 / +35 / +100 % | `MonStats.txt` MinGrp, MaxGrp (U); same card |

Ambush spawns (a champion pack that appears behind the hero) are probably a d2d overlay spawn, not a table (EXP, U). Game.exe would need the mod-data layer and the user's go per hook set (AGENTS.md, memory-write rule: injected code is INSTRUMENTATION only).

### 3.4 Incompatibilities and the heat cap

An explicit table, in data (our own addition, U). Reasons are shown to the player.

| pair or set | why |
|---|---|
| Blood Raven's Tithe (rank 3+) with Pact of Rotting Bone (rank 3+, that is `/players 5+`) | no potions plus a 4x-life monster is not a fight, it is a coin flip |
| Hardcore with Andariel's Shadow, Blood Raven's Tithe and Rakanishu's Bargain together | three safety losses on a one-life run (EXP: "hardcore death limit + no potions + no TP") |
| The Stranger's Toll with Blood Raven's Tithe | no merc and no potions |
| any two Pacts that both remove a safety net at rank 3+ | the "no safety" tag in data; the check is a tag count, not an id list |
| Rakanishu's Bargain with Warriv's Long Road | the same lever twice |
| Pact of Rotting Bone for Necromancer and Druid heroes | minions die; veteran: unfun for summoners (class scope, not an incompatibility) |

Cap: the heat of the active set may not exceed the cap (12 at the start). The UI shows the sum and refuses an over-cap set; it does not refuse poor or odd choices below the cap (the same "your call" stance as the policy sheet, V `docs/policy-sheet.md`).

### 3.5 Per-class scope

From the veteran (EXP): melee (Barbarian, Paladin) take elite-density, `/players` and no-potion Pacts. Casters (Sorceress) take death-limit and mono-skill with care (fragile early, mana-bound). Summoners (Necromancer, Druid) tolerate pack size but not `/players`. Amazon and Assassin sit between. In the catalogue this is the `class_scope` field of each Pact: a list of allowed classes, or `all`, resolved by name against the class table; an unknown class is a load error that names it (AGENTS.md, build flexibility; never a silent Sorceress fallback). The default test class is the Paladin (decisions F11).

## 4. The catalogue data schema (proposed)

One file, `data/pacts/pacts.json`, embedded and validated at load (the pattern of `internal/decide/policy_chores.json`, V: `docs/policy-sheet.md`, "The rows are data"). No Pact lives in Go.

| field | meaning |
|---|---|
| `id`, `name` | stable id and display name |
| `lever_family` | `bot`, `native` or `overlay` |
| `kind` | `ranked` or `fixed` (Nightmare, Hell) |
| `ranks[]` | one entry per rank: `rank`, `params` (family-specific: for `bot`, a policy-sheet `row` and `option` and/or `knobs`; for `native`, `players` or `difficulty`; for `overlay`, table key and delta), `heat`, `reward`, `effect_line` |
| `effect` | generated one-line text for the current rank (from `effect_line`, never hand-copied) |
| `incompatible_with[]` | `{id or tag, reason}` |
| `reward_per_rank` | `{tally_pct, charm_points, title, cosmetic}` (folded into `ranks[].reward`; shown here for the reader) |
| `unlock_trigger` | `andariel_first_kill`, `baal_normal`, `baal_nightmare`, `hall_count:N`, `tally:N` |
| `class_scope` | `all` or a class list |
| `away_safety` | `safe`, `needs_consent` or `present_only` |
| `learn_text` | the copy of the before/after card: what this does and what it teaches |
| `tags` | e.g. `no_safety`, used by the incompatibility rule |

Loader rules (the `pactcatalogue` card): unknown family, unknown policy row or option, a knob that is not in `docs/knobs.md`, a rank with no heat, a duplicate id, an incompatibility naming a missing id, a class that the table does not know: each is a load error naming the cause (V pattern: `TestPolicyTableErrors`).

Worked example:

```json
{
  "id": "blood_ravens_tithe",
  "name": "Blood Raven's Tithe",
  "lever_family": "bot",
  "kind": "ranked",
  "unlock_trigger": "andariel_first_kill",
  "class_scope": "all",
  "away_safety": "needs_consent",
  "tags": ["no_safety"],
  "incompatible_with": [
    {"id": "pact_of_rotting_bone", "reason": "No potions plus 4x enemy life is not a fight."},
    {"id": "strangers_toll", "reason": "No merc and no potions."}
  ],
  "ranks": [
    {"rank": 1, "heat": 3, "params": {"row": "belt", "option": "quarter", "min_clvl_potions": 8},
     "reward": {"tally_pct": 6},
     "effect_line": "No belt potions below clvl 8; tally x1.06"},
    {"rank": 3, "heat": 9, "params": {"row": "belt", "option": "none", "min_clvl_potions": 14},
     "reward": {"tally_pct": 30, "charm_points": 1},
     "effect_line": "No belt potions below clvl 14; tally x1.30"},
    {"rank": 5, "heat": 15, "params": {"row": "belt", "option": "none", "min_clvl_potions": 99},
     "reward": {"tally_pct": 90, "charm_points": 1, "title": "Dry Belt"},
     "effect_line": "Never uses belt potions; tally x1.90"}
  ],
  "learn_text": "Potions are the cheapest safety there is. This shows how far the hero gets without them."
}
```

Note: rank 5 of this example costs 15 heat and cannot be taken alone under a cap of 12. The heat per rank for the ranked Pacts is therefore tuned so that rank 5 of a 3-heat Pact needs a mastery cap raise. This is on purpose (the top ranks are earned) and U.

## 5. Away safety

The bot runs while the player is away (the whole point of the loop). A Pact can kill a hero or hold it back while nobody is watching, so the design is conservative. Prior-art support for the pattern: Hades shows the exact per-rank effect (V source 1); Dead Cells allows change at run start only (V source 8).

1. A Pact is active only while a human is connected, or after an explicit per-Pact away-consent: a typed, dated record, stored in the decide store, and shown on the return screen (`pactaway`, `pactui`).
2. Auto-pause at a low-life floor (default 35 % life, U) and pause in town as the safe zone. A paused hero waits for the player and says so on the return screen; it never drops the Pact silently.
3. Before the player goes away, a plain-language list of what can be lost, with numbers ("up to 2 deaths; each costs about X gold of repairs and about N minutes", numbers computed from the active Pacts, not typed).
4. Hardcore (permadeath) needs its own hard confirmation on its own screen, typed (the hero's name), never in the same sheet as the other Pacts. Away play with Hardcore is a separate consent and is OFF by default (proposed, U).
5. Nightmare entry. D2's death penalty is harsher there: XP loss on death (-5 % in Nightmare, -10 % in Hell) and a resist penalty (-40 in Nightmare, -100 in Hell). These numbers are the veteran's memory and UNVERIFIED: verify against the game before showing any of them to the player (card `pactnative` lists the check). The Pact offer warns once, and the first away-consent after the move warns again; the UI flags under-geared resists but does not block. Away default after the move is OFF and consent is renewed per difficulty.
6. Away consent expires when a Pact is changed, when the hero changes difficulty, or after 7 days (U).

## 6. Effect lines, before/after cards, results display

- Every rank shows one generated line, for example "Enemies deal +20 % damage; tally x1.05", from the data (`effect_line`), same discipline as the policy sheet's effect lines.
- The offer (`pactui`) shows the current policy and the proposed one side by side, the heat sum against the cap, the reward delta and the reasons for any incompatibility. A bot-side Pact shows the same before/after cards as a policy change: counters taken from the journal over equal windows (V pattern: `PolicySection`, `docs/policy-sheet.md`).
- The return screen (TASK-127) shows, per active Pact, what it cost and what it paid since the last visit: deaths, potions not used, tally gained over the base. The point is the player learns which conditions matter (Hades pattern).
- The Hall (TASK-135) lists the Pacts and ranks a hero carried, its highest heat and any title earned.
- Counters whose link to a Pact is a guess carry `(U)`, as in the policy sheet. The text says what happened while the Pact was active, never that the Pact caused it.

## 7. The first hours: safe zones and frustration traps

Safe zones where Pacts are off or capped (EXP, U): the first 10 minutes or clvl 5, whichever is later; the Den of Evil until its quest is done; town and the Rogue Encampment. Andariel keeps the hit-point and damage caps but not the no-TP rule.

Watch moments the Pacts create (what the player will want to see): the Den clear at 3 % life, Blood Raven with a champion pack, Andariel at rank 3, a Hardcore near-death with the retreat triggering, a named unique in the Cold Plains.

Traps to design against: dying at clvl 3 to Fallen with no potions; dying in the first 10 minutes; losing a long run to a random first-pack ambush; big heat jumps that skip the lesser rewards (Hades advice: raise heat one point at a time, V source 1); cheap Pacts that become the default pick (fix: unequal heat, the cap, and the known bad pairs); a reward curve that is flat (Diablo 4 Pit finding, S source 12: players stop when higher tiers pay no more).

## 8. Feasibility, by reading the code

All V unless marked: the files were read in this tree on 2026-11-07. None of it has run against a live bot.

| exists | where | note |
|---|---|---|
| the policy sheet, 14 rows, chore knobs by option | `internal/decide/policy_chores.json`, `internal/watch/policysheet.go`, `docs/policy-sheet.md` | rows relevant here: `repair` (default, low, always, never), `tp` (default, buy, walk), `tpreserve`, `retreat` (default, early, bold; `part`), `potionmix`, `potiontier` (`record`), `belt` (default, half 50, quarter 25), `whites`, `sell`, `identify`, `stash` (`record`) |
| the tuner and knob store | `bot.Tuner.Edit`, `docs/knobs.md` | a Pact can set knobs through the same all-or-nothing path |
| the decide store with a `policy` op and replay | `Store.SetPolicy`, `Store.ask` | a Pact could be a second op next to it; the file version stays |
| the death event | `goals.EvDeath` | a death counter can be derived from the journal |
| the tally strip and events mode | `internal/watch/tallystrip.go`, `docs/tally-strip.md` | credit by event with idempotency keys: a multiplier is a new field on the credit, not a new ledger |
| the return screen and its sections | TASK-127, `PolicySection` | a Pact section follows the same shape |
| merc hire knobs | `internal/merc/knobs.go` (`HireGoldMult`) | a hire gate is a new check, not a knob (U) |

What is missing, per lever family:

| family | missing | size |
|---|---|---|
| bot: catalogue and loader | the data file, validation, the heat and incompatibility rules, the apply path to the policy sheet and tuner | S to M (`pactcatalogue`) |
| bot: new answers | no belt potions below clvl X (the shipped belt options stop at 25 %), a death limit with a park order, a waypoint ban, a mono-skill filter in `plandata`, a merc hire gate, a gold-loss ledger entry | M for the first three Pacts (`pactbotside`); each further one S to M |
| bot: tally multiplier | a multiplier on credit and a place to show it | M, in `pactui` / tally |
| native: `/players` | a way to type the command, the live check of the real scaling | S research, live (`pactnative`), then S |
| native: Hardcore | creating a Hardcore hero (the setup wizard and the OOG tool), handling the dead screen | M (not carded: depends on the live check) |
| native: Nightmare | the bot playing past Act 1: nothing past Act 1 has run live; class and act coverage cards | L, parked (`pactnightmare`) |
| overlay | which table drives elite density and pack size, a d2d overlay write path for them, a game.exe mod-data layer | L, parked (`pactoverlay`) |
| away safety | per-Pact consent record, pause hook, low-life floor | M (`pactaway`) |

d2d versus game.exe: bot-side Pacts are identical on both (they only change what the bot decides). Native `/players` and Hardcore are game features and work on both if they work offline (U). Overlay Pacts work on d2d only until a game.exe mod-data layer exists, which needs the user's go (U). Anything touching the d2d fork is committed and pushed to the fork's private remote the same day (AGENTS.md).

## 9. Open questions for the user

1. "Pact of Nightmares": one Pact (Nightmare, then Hell as a sibling) or a family of Pacts that all start Nightmare with different rules? Proposed: one Pact, Hell a second one.
2. Hardcore: a mode flag with its own confirmation, or a heat-costed Pact? Proposed: a flag.
3. Nightmare heat: a fixed chunk (proposed 8 of 12) or scaled by build and gear? Proposed: fixed.
4. Are `/players` Pacts wanted at all? The veteran says they are unfun for summoners and very hard for a clvl 3 caster; the live check may show the scaling is weak enough to skip them.
5. Which Pact to prototype first? Proposed: "Blood Raven's Tithe" (no belt potions) and "Andariel's Shadow" (death limit), the cheapest bot-side ones.

## 10. Caveats

Every numeric reward value (tally percent, charm points, heat per rank, cap 12 to 20, 8 heat for Nightmare, the 35 % floor, the 7 days) is U: tune in play. The veteran's native numbers (`/players` life scaling, XP effect, Nightmare death penalty, Nightmare and Hell resists) are EXP and UNVERIFIED. The unlock event "Baal killed in Normal" is D2's own rule for Nightmare (S) and cannot be reached by the current bot: nothing past Act 1 has run live (V, `docs/plan/STATUS.md`). Everything in section 8 was read, not run.

## 11. Cards

Proposed, milestone m-4, label `watch` (component), priority low, assignee credit-runner. None is on the two-week playable path. Task ids follow.

- `pactcatalogue` (TASK-166): data file, loader, validation, incompatibility and heat rules. S to M.
- `pactbotside` (TASK-167): the first three bot-side Pacts through the policy sheet (Blood Raven's Tithe, Andariel's Shadow, Rakanishu's Bargain). M. Depends on `pactcatalogue`, TASK-128.
- `pactui` (TASK-168): offer UI, effect lines, before/after cards, away-consent record. M. Depends on `pactcatalogue`, TASK-127.
- `pactaway` (TASK-169): consent, pause, low-life floor, the Hardcore confirmation. M.
- `pactnightmare` (TASK-173): the Pact of Nightmares. L, parked. Needs the bot past Act 1.
- `pactnative` (TASK-170): live check of `/players`, Hardcore and the Nightmare penalties. S research, live.
- `pactoverlay` (TASK-172): elite density and pack size through the d2d overlay. L, parked.
- `pacthall` (TASK-171): Hall titles, legacy charm points. Depends on TASK-135, TASK-136.

## 12. Proposed decision lines (for the user's session; not written to `docs/decisions.md`)

- Pacts are opt-in challenge rules; they change bot policy or D2's own difficulty settings, and their rewards live on the meta layer (tally, Hall titles, legacy charm tier, cosmetics). Vanilla XP and loot never change except through D2's own difficulty.
- The first Pacts unlock when Andariel is first killed; the Pact of Nightmares when Baal is killed in Normal (same hero, Nightmare, Pact bonus).
- Hardcore is a mode flag with its own confirmation, not a heat-costed Pact (pending the user).
- A Pact never runs away from home without a per-Pact, typed, dated consent.
