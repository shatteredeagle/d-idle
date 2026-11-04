# Plan for card m3-andariel: the throwaway Andariel fight

> status: plan, offline, nothing verified live. Tags: V = read in this tree or counted from a recorded log (file named), S = secondary (another doc), I = inference, U = unverified. Written for the user decision 2026-11-04 (option c of docs/andarielgate.md). Numbers with an owner (knobs, floors, gates) are cited, never owned here: docs/knobs.md, openspec/specs/merc-policy, soak-gates.

## 1. What the fight must prove

Quotes (V):
- merc-policy, requirement "Andariel floor": "No unattended attempt before the throwaway fight of card m3-andariel has won once: `boss.andarielHold` 1 makes the campaign level instead, 0 lifts it." (openspec/specs/merc-policy/spec.md line 60). Same text in the knob description (internal/env/tuning_boss.go: "Set 0 after that win. Attended runs (bossrun) are not affected. ADR 0011 gate 3.").
- Card gates (openspec/changes/m3-andariel/tasks.md): "3 of 3 fights at clvl boss.andarielMinLevel with the kit and a living Rogue: Andariel dead, character alive, antidote drunk on poison, `boss andariel` log lines with life minimum and potions used (raw in docs/probes/andariel)"; "1 fight at the same level without a merc reported (pass or fail, with the cause)"; `go test -p 1 ./internal/boss` including `TestAndarielFightOrder`.
- Delta spec (openspec/changes/m3-andariel/specs/act1-campaign/spec.md): kit 6 healing, 2 antidotes, 3 mana (caster), tome with 2 or more charges; antidote within 2 s of poison; "open with Static Field to the Normal boss cap, keep range and retreat by portal at the chicken line"; a thin kit at the door starts no attempt and inserts a town trip.

What that means in one sentence (I): the fight proves that the existing script, kit gate and merc survive her poison nova and cold bolts at the lowest floor the campaign may use, so that an unattended attempt is no more risky than the death rule (spec death-rule) allows. It does not prove the campaign chain (Blood Raven, road, Warriv hop): those are other cards.

Two conflicts inside the card text, to settle before step 4 (V):
- The card asks for Static Field "to the boss cap", but internal/boss/andariel_floor_test.go `TestAndarielScriptHasNoStaticFieldOpener` (card fixstatic) pins a script with kite range 10 and NO opener (act1.go says "no script opens with Static Field"). One of the two must change; I suggest the test and the script follow the fight data (the drill decides, step 6), and the card text is relaxed to "Static Field only if the drill shows it helps".
- The card says the fight level is `boss.andarielMinLevel` (17, with a merc 15 = `boss.andarielMinLevelMerc`). "Same level without a merc" in the card means 17; with the merc 15. I propose: 3 fights with merc at the merc floor (15), 1 without at the no-merc floor (17), because those are the two levels the campaign will actually attempt at.

## 2. Current state

Exists (V unless noted):
- Route: `andariel()` in internal/boss/act1.go: waypoint Catacombs 2, route Catacombs 3 then 4 (area 37), arena Catacombs 4, anchor hint, two stand points, `Kite{Range:10, Every:1.5s, Away:12}`, hazards `NoPotionLowLife(35)`, `PoisonStatus()`, `PoisonCloud(6,10)`, `RaisedAdds`, `PreAntidoteBelow:75`, `MinPotions:3`, `ChickenPct:12`, `EscapePct:30`, timeout 12 min. Offline tests: internal/boss (floor knob, fake-based Run).
- Gate and kit: internal/task/gate.go `BossGate` for `QAndariel` (`Potions: minPotions, NeedTP, ManaPots:3, Antidotes:2, Andy, Route`), `andarielFloor`, `andarielHeld`, log lines `boss: andariel gate pass clvl=N kit=ok` and `boss: held for andariel (...)`.
- Wiring: d2d backend runs a boss through `bossKill` (cmd/d2bot/boss.go, logs `boss run boss=.. killed=..`) via cmd/d2bot/d2dwire.go. Attended: `cmd/bossrun` (game.exe path, `--dry`, `--god REASON` which MASKS the escape logic, so it can never count for the proof).
- Merc: Act 1 free Rogue claim, revive (internal/town/merc.go, internal/merc/policy.go `ShouldRevive`, cmd/d2bot/mercwire.go `ReviveStep`). Drill D8.a (docs/probes/merclive/2026-11-04-MC-revive.txt): revive through Kashya's menu PASSED but the price taken was 0 gold at clvl 1 (U whether that is a rule or a gap).
- Cheat tooling: `cmd/d2cheat` (`level N`, `xp`, `gold`, `runstart ID` with `--reason`), forkdbg verbs `debug mana full`, `debug mon set` (life, res, dmg), `debug cast`, `debug states` (docs/forkdbg.md), older `debug merc hire <type>|kill|revive|hurt`, `debug stat 14|15` for gold, `debug give`, `debug goto <level>` (docs/cheats.md).

Missing:
1. `TestAndarielFightOrder` and the `boss andariel` log line with life minimum and potions used: neither exists (V: grep of internal/ finds no such test; the only boss log is the generic `boss run` line). Size M, code.
2. The Static Field decision (section 1) and, if the opener is wanted, a `BeforeAttack` hook (script.go has the field, V). Boss cap of Static Field in Normal: U, a table read (tools/emu mpq-table skills / the monster tables).
3. A way to put a throwaway at Catacombs 4 near her without walking the road: `debug goto 37` (S: docs/cheats.md says `goto <level>`; whether it lands in a usable spot and spawns her is U). Walking the road at clvl 15 is itself the thing the route guard protects against, so it must not be the way in.
4. A merc scenario: a Rogue at clvl 15 equals a merc level that must be set; `debug merc hire 0` gives a Rogue (V in D8.a) but at which merc level and with which gear is U. The no-paid-Act-1-hire rule (merc-policy) means the merc is a claimed or cheat-given one, declared in the run record.
5. A repeatable kit setup: 6 healing, 2 antidotes, 3 mana, tome with 2 charges: `debug give` with potion codes and a tome is S, not drilled.
6. Evidence directory docs/probes/andariel and the result doc docs/andariel.md: not created.
7. Persistence of "won once": none. The hold is a knob, not a state (V: `andarielHeld` reads only `AndarielHold`, killed and quest). So lifting is a manual config step, see section 4.
8. Crutch rule fit (AGENTS.md): the fight is a SETUP-cheated character (level, gold, kit, merc) so it can never count toward a gate; that is fine for a throwaway, but the bot's play after `runstart` must write nothing (no `--god`, no `debug mon set` during the fight). `debug mon set` is allowed only for a separate diagnostic run (resist read, step 6).

## 3. Ordered steps

Owners: OW = offline worker (no game), D = live d2d (hidden slot, lab binary), G = needs game.exe (none planned), U = needs the user. Exclusive resources: `soak-slot:3` (card), a hidden slot, `d2d-lab-binary` while a probe runs, `fork-branch:*` only if a fork verb is needed (avoid).

| # | Step | Size | Owner | Exclusive | Pass condition |
|---|---|---|---|---|---|
| 1 | Settle the two card conflicts (Static Field, fight levels 15/17); user answer, section 6 | S | U | none | answers written in the card |
| 2 | Read Andariel and Static Field numbers from the tables (HP, resists, boss cap of Static Field in Normal); fill docs/andariel.md section tables | S | OW | none (emulator read-only) | table lines tagged V with the table name; Static Field cap no longer U |
| 3 | Write `TestAndarielFightOrder` first (antidote on poison within 2 s, opener only if step 1 says so, then Charged Bolt, retreat by portal at the chicken line) on the boss fake; add the `boss andariel` summary line (life minimum, potions used, antidotes drunk, outcome) in internal/boss or `bossKill` | M | OW | none | `go test -p 1 ./internal/boss ./cmd/d2bot` passes, old tests unchanged, line format tested |
| 4 | Dry-run on d2d with no fight: setup recipe as a script under docs/probes/andariel: new throwaway, `d2cheat --reason ... level 15`, gold, kit, merc claim/hire, `debug goto 37`, `runstart`; read the state (`dump stats`, `monsters`) and confirm Andariel is present and the arena is walkable | M | D | hidden slot, d2d-lab-binary | script runs twice with the same setup; ledger one entry per cheat with reason; `dump` shows kit and merc as planned; if `debug goto 37` does not work, stop and report the alternative (walk with `route.*` floors lowered, which needs the user, step 1b) |
| 5 | Diagnostic-only run (not counted): `debug mon set` to read her hit per poison nova and her damage on the character with and without the antidote; resists | M | D | hidden slot | numbers in docs/andariel.md tagged V; does not feed the gate |
| 6 | 3 fights with the merc at the merc floor, bot acts alone after `runstart`, writes after start: 0 | L | D | hidden slot, `soak-slot:3` | 3 of 3: boss dead (kill detect), character alive, antidote drunk on poison (log), `boss andariel` lines saved raw; ledger `writes after start: 0` |
| 7 | 1 fight without a merc at the no-merc floor | M | D | same | reported pass or fail with the cause (a fail does not block the lift only if the no-merc floor is raised, section 4) |
| 8 | Write docs/andariel.md (fight log, what hurt, Static Field result), fill card result, changecheck, leakcheck, validate, reap | S | OW | none | checks pass, `tools/agentreap.sh m3-andariel` empty |
| 9 | Decision to lift (section 4) | S | U | none | one line in the card result |

A failed fight in step 6 (death or no kill): stop the series, record the cause, change the script or the floor, restart the series from fight 1 (a win after a loss does not count as "3 of 3"). I: with a clvl 15 Sorceress and a Rogue that level, a loss is plausible; the most likely cure is a higher merc-floor (knob only, no code), not a script change.

## 4. Lifting the hold and the Gate 2 rerun

How (V for the mechanism, I for the policy): `boss.andarielHold` is a knob, default 1 (`internal/env/tuning_boss.go`). The hold is lifted by changing the default to 0 in that file plus the regenerated docs/knobs.md and the one spec line in merc-policy (`boss.andarielHold` text) as a delta spec of the card that does it; or, for one run only, a profile override (as in option b of docs/andarielgate.md). I recommend the default change in a separate small commit, made by the coordinator after the user reads the step 6 and 7 results, with the fight evidence path in the commit message. Do not lift it on a partial series, on a fight by `--god`, or on a character that has any gameplay write after `runstart`.

If the no-merc fight (step 7) lost, lift the hold but keep the no-merc floor high (`boss.andarielMinLevel` stays 17 or goes up; V that the knob range is 12..22), so the unattended attempt only happens with a living merc or at the higher level. The path through `andarielFloor` already prefers a living merc (V: `routeMercOK`).

Gate 2 rerun must then show (from docs/andarielgate.md section 5, V):
- a `boss: andariel gate pass clvl=N kit=ok` line at N >= 14, then a boss-run start line and a boss outcome (attempt event); the judge matches `boss:` and `route guard: andariel` lines, not `planner` text.
- merc alive (floor 15) or clvl >= 17 at the attempt; a clvl 14 to 16 character without a merc still reads floor 17, so the criterion "attempt at clvl 14 or higher" is only reachable with a living merc at 15 or 16, or at clvl 17. Gate wording (a) of docs/andarielgate.md is still worth keeping as the interim text; that is a user decision.
- `merc: revived` without runs of `revive skipped (gold)` at clvl 14+ (section 5 is the prerequisite).
- Not enough alone: lifting the hold does nothing for the Blood Raven, Rogue claim or road prerequisites in the chain (docs/research/andariel-chain-needs.md section 1), these must already pass in the rerun.

## 5. The merc revive-gold stall

Observation (V, docs/andarielgate.md section 2): seed 11, 366 `merc: revive skipped (gold)` lines, price 1920 against gold 1762 to 2265, merc level 16, the merc dead from about clvl 14 to the end.

Cause hypotheses (S = code reading, I = inference):
1. The test is not `gold >= price`. `ShouldRevive` needs `price*mult + Kit*KitReservePct/100` (S: internal/merc/policy.go `need`; defaults mult 1, reserve 150 percent of the full kit refill, internal/env/tuning_merc.go). With a kit refill around 300 gold (the test value is 310) the bar is about 2400, so a purse of 2265 was "just under" although it covered the price. I: the log prints only `price` and `gold`, which makes the stall look like a near-miss; the reserve is invisible.
2. `Ctx.Gold` includes stash (S: `mercCtx`), but the purse that actually pays is carried plus stash only if the revive pays from the stash (policy comment: "carried first, then stash, S"): U whether the real dialog uses the stash.
3. Gold farm switches itself off at clvl 16: `GoldFarm` needs `clvl < merc.goldFarmMaxLevel` (default 16, S: farm.go, mercwire `GoldFarm`). At clvl 16 with the merc dead the farm does nothing, so the character stays in the level plan and gold hovers (V: the gold range 1762 to 2265 moved with sales and spending).
4. Price formula `MercReviveCost = min(level*level/2*15, 50000)` (S: internal/resource/merc.go): merc level grows with the character, so every level raises the price (120 at merc level 4 and 1920 at 16, both V in the log). The price is the formula, not a dialog quote, in the check (`price` 0 passed to `ShouldRevive`; the dialog quote only exists after paying: `Quote()`), so the formula may be off from the game; U.
5. The Rogue died 3 times in 25 s once revived (V: log), which raises `DeathsLastHour` over `stopDeathsPerHour` 3 and could switch the multiple to `StopGoldMult` 5 (reason `stopped`, not `gold`); the log says `gold`, so not the case here (I).

Small fix proposal (offline, in the merc lane, needs the owner of m3-merc, not done here):
- Log the real bar: `LogReviveSkipped(... need=N)` and, once per change, a `merc: gold wait need=.. gold=..` line, so the stall is visible and countable.
- Make the gold farm stay on while a dead merc cannot be revived, up to `boss.andarielMinLevelMerc`+ or until clvl 17 (the first no-merc Andariel level), or raise `merc.goldFarmMaxLevel` from 16 to 18 (knob only, documented in docs/knobs.md). I prefer the knob first: smallest change, reversible.
- Optionally lower the reserve for a revive when the character is above the Andariel merc floor: the kit reserve of 150 percent of a 300 gold kit is 450, about a quarter of the price at merc level 16.

Test idea (offline, two builds per AGENTS.md): table test on `ShouldRevive` and `GoldFarm` with a Sorceress and one non-caster context: price 1920, kit 310, gold 2265 gives `no(gold)` and a log line that contains `need=2385`; with the farm cap raised, `GoldFarm(true, 2265, 2385, 16, k)` is true; at gold 2400 `ShouldRevive` is yes. Plus one fake-wire test that the stall line is logged once, not 366 times.

Live check (D): D8.b of the merc drills: read a real revive quote at a high merc level (`debug merc kill`, `debug stat 14` to set gold, Rogue level set) and compare it with the formula. This is also the answer to the "free at clvl 1" U of D8.a.

## 6. Risks and what needs the user

Risks:
- Fight results on a cheat-levelled, kit-given, merc-given character are a model of the unattended case (I): the real character arrives after a road that costs potions and the merc's life. The hold lift is only as good as that model.
- `debug goto 37` may not drop her where the script expects (the anchor hint is a game.exe coordinate pair, U on d2d); the first live run may only tune the setup.
- d2d versus game.exe: the card runs on d2d only. Her real poison and cold timings on game.exe are U (S: SME memory, docs/research/andariel-chain-needs.md section 2).
- Antidote and Static Field facts are U until step 2 and 5.
- 3 of 3 can be lost to a single bad roll; plan for 5 to 8 runs of a few minutes each (I).
- Hidden slot and lab binary are shared: check the worker host load before each run, one run at a time.

Needs the user:
1. Static Field: opener yes or no (conflict, section 1).
2. Fight levels: 15 with merc and 17 without, or the card's single "clvl boss.andarielMinLevel".
3. Whether the setup may use `debug goto 37` plus a given merc (cheats, declared) instead of the road.
4. Whether the hold lift is the default change or a per-profile override, and who decides after the series (section 4).
5. Whether Gate 2 wording (a) is applied meanwhile (docs/andarielgate.md).
6. The revive stall fix: knob first or code first (section 5).

Proposed text for guarded files: none (no ADR, GLOSSARY or decisions edit proposed). The glossary has no entry for "throwaway fight"; the term is used in merc-policy and the knob text only (I).
