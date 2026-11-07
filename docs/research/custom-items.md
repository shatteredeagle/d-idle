# Custom items: can we add our own items and drop rows? (feasibility study)

> status: research, 2026-11-06, card `customitems` (docs and tasks only; no Go change, no live run, no fork change). Answers the user's question "Can we make our own items and add them to the drop tables?" (user chose: study first, no build yet).
> Tags: **V** read this session in this repo or in the d2d fork (read-only, local branch `local/batch2-B-r2`, file named); **S** secondary (a repo doc, the SME KB, the d2-sme agent's veteran memory); **U** unverified or from memory. Target is classic 1.14d. D2R-only facts are flagged D2R. Table contents are Blizzard data and are not copied here; only column names and our own proposed rows appear.

## Answer in eight lines

1. **Yes for rows, with one fork change.** The d2d fork reads every game table through one call, `Stack::try_read("data\global\excel\X.txt")` (V components/game/gamedata/gamedata_load.cpp, components/mpq/mpq.hpp). That is a single choke point where a flag-gated row overlay can be merged, so a new unique, set piece, rune word or affix needs no change to the item generator (V: the generator walks the loaded tables, components/rules/rules.hpp).
2. **d2d has no loose-file path today** (V): `Stack` holds MPQs only (StormLib). It does, however, already let a whole replacement `patch_d2.mpq` or an explicit `patch =` layer win over everything (V gamedata_load.cpp open_game_data), and `check_patch_layer` already warns on a modded row count "a mod, or another version" (V). A whole-table replacement would need Blizzard's table text, so it is the wrong shape for us (GPL, see section 4).
3. **Overlay = rows only.** A mod directory of our own row fragments (TSV, same column names) merged into the table bytes inside `Stack::try_read`, append or replace-by-key, off by default. Estimate 12 to 18 h in the fork (section 3).
4. **New art and new strings are the hard cases.** A unique on an existing base reuses its art and its name needs only a string. d2d shows an unknown string key as the key text or empty (U, not read in item_text; check in the first task). New base types with new art are out of scope (section 1e).
5. **Two modes.** Vanilla (default): every fidelity check, probe, oracle and gate. d.idle mode: overlay on; custom rows are "not 1:1 by design" and excluded from comparisons (section 4).
6. **Bot impact is small for uniques on existing bases** (V): the starter pickit already stashes any unique or set item by quality, with no per-item knowledge (section 5).
7. **First item:** one unique cap on an existing base, added to Blood Raven's drop, d.idle mode only, about 30 h end to end across two repos (section 7).
8. **Product question:** what are custom items FOR? Sacrifice target, tier reward or farm goal? Recommendation in section 8.

## 1. How the game defines items and drops

All tables are tab-separated text in `data\global\excel\` (V docs/data.md: the 1.14d `Patch_D2.mpq` carries plain `.txt` for every table; the game also has `.bin`, see section 2). Strings are in `.tbl` files under `data\local\lng\eng\` (V docs/data.md).

### Tables involved (S unless marked; columns V where the fork reads them)

| table | role | columns the fork reads (V gamedata_load.cpp) |
|---|---|---|
| ItemTypes | type tree (Equiv1/2), body locations, Rarity, Treasure class flag | Rarity, sparsePopulate, Sk lvl columns |
| Armor / Weapons / Misc | base items (code, level, qlvl, cost, art, sockets, stats) | code, rarity, cost, art columns |
| UniqueItems | one row per unique | index, code, lvl, lvl req, rarity, enabled, ladder, nolimit, prop1..12 with par/min/max, cost mult/add, chrtransform, invtransform, sound columns |
| SetItems and Sets | set piece rows and set bonuses | index, item, set, add func, aprop1a..5b |
| Runes | rune word rows | Rune Name, Rune1..6, server (ladder flag) |
| MagicPrefix / MagicSuffix / AutoMagic | affixes | Name, levelreq, multiply/add, transformcolor, class |
| TreasureClassEx | drop tables | Treasure Class, Picks, NoDrop, group, level, Unique/Set/Rare/Magic, Item1..10, Prob1..10 |
| ItemRatio | quality odds | Unique/UniqueDivisor/UniqueMin etc. |
| Properties, ItemStatCost | what a mod can do | referenced by property code; a new stat needs code (U) |
| CubeMain, Gems | recipes, socket effects | optional |
| MonStats, SuperUniques | which TC a monster/boss uses | TreasureClass columns (V internal/data/monsters.go TreasureClass[diff][0..3]); SuperUniques TC, TC(N), TC(H) (V internal/data/monsters.go) |

### Minimal row sets

- **(a) New unique, existing base, existing art.** One UniqueItems row: `index` (string key), `code` of the base, `lvl`, `lvl req`, `rarity`, `enabled=1`, props. Plus one string-table entry for `index` (name). Plus one drop route: either nothing (it competes in the normal unique roll for that base, diluted by the base's other uniques) or a TreasureClassEx line for a boss (section 1c). Art comes from the base's `invfile` unless the row sets `invfile` to another existing DC6 (S).
- **(b) New set piece.** A SetItems row (`item` code, `set`, props, aprop bonuses) plus a Sets row if it is a new set, plus names. A set piece drops only through set-quality rolls on its base (V rules.hpp special pick: same code and ilvl rule).
- **(c) New rune word.** One Runes row (name key, runes `Rune1..6`, `itype1..`, props). It is made by socketing, no drop route. Ladder-only words need `-direct -txt` in the real game (V sme AG, builds-open-questions.md line 55).
- **(d) New magic affix.** One MagicPrefix or MagicSuffix row (name key, levelreq, itypes it applies to, props). Affix ids are row numbers in the saved item (V components/game/item_text.hpp), so appended rows get new ids.
- **(e) New base type with new art.** Armor/Weapons/Misc row (new `code`), an ItemTypes row only if a new type, strings, inventory DC6, in-world "flippy" DC6 and often a `.dt1/.cof` for worn appearance. Needs binary art assets that we would have to author. **Out of scope** for now (U: hours depend on art; the fork's d2s reader keys items by 4-letter code, so a new code also needs the save code path to accept it).

### Drop flow (V rules.hpp, drops.hpp)

Monster dies -> its TreasureClass (monster row, or superunique row for a named boss) -> `roll_drops` walks TreasureClassEx: `Picks` rolls, each roll is `NoDrop` against the weights of Item1..10; an entry that names another class recurses, otherwise it names an item or a `weapN/armoN` auto class -> an item base is chosen -> quality rolled from ItemRatio and the class's Unique/Set/Rare/Magic modifiers at the item level -> if unique or set, `special()` picks among the rows whose `code` matches the base, `enabled`, not `ladder`, and `lvl <= ilvl`, weighted by `rarity` -> affix rolls (`apply_mod` over props). A unique that was already found fails and the item turns rare (V rules.hpp comment: one-per-game list, `nolimit` bypasses). Also V: the fork's TreasureClassEx loader drops entries with `Prob` under 1 and does not implement the unique/set flag entries of TreasureClassEx (V drops.hpp "ponytail": Cow King's classes only).

**Adding to a boss or monster class.** Edit (overlay) the boss's TreasureClassEx row: add or reweight an `ItemN`/`ProbN` slot with a base code (or a new private class `CI_Bloodraven_X` that names the unique's base and add that class as an entry), or change the monster row to point at a new class. The dilution fact (S, d2-sme): the base's other uniques share the rarity weight, so the chance is `P(entry) x P(unique quality) x rarity_i / sum(rarity of eligible uniques on that base)`. Use a base no vanilla unique uses at that level, or a high `rarity` (U exact formula; it is computable from the loaded rows, see the drill in section 7). Which TC name Blood Raven uses is not recorded in the repo (U): read it from SuperUniques/MonStats on the install in the first task.

## 2. The original game's modding path (game.exe 1.14d)

- V (docs/sme/builds-open-questions.md line 55, source tag AG): in 1.14d single player, `-direct -txt` plus a custom `data\global\excel` folder is how changed tables are accepted (ladder-only runewords only form that way). The file does not say what the game does with a loose `data\local\lng` string folder.
- S (community knowledge, not fetched): `-direct` makes game.exe read `data\` from disk before the MPQs, `-txt` makes it parse the `.txt` tables instead of the compiled `.bin`; loose DC6, DT1 and `.tbl` files in the same `data\` tree are picked up the same way. Treat the `.tbl` and DC6 parts as U until a short live probe (one afternoon, one throwaway character, user go) confirms them.
- V (docs/data.md): the install has both `.bin` and `.txt` for the tables, so the `.txt` route has a vanilla baseline to diff against.
- A shared consequence (S): a saved unique stores a row id; with different tables the same id means another item. That is why a character belongs to one mode (section 4).

Conclusion: for game.exe, loose `.txt` tables are supported by design (V as documented in the SME file, no run of ours); `.tbl` and art overlays are U. We only need game.exe in vanilla mode for oracles, so we do **not** need to run custom rows in game.exe, except one optional A:B of the drop math (section 4).

## 3. d2d: how tables load and what an overlay looks like

Read in the fork (V, local branch `local/batch2-B-r2`; the fork checkout was not touched).

- **Load path.** `open_game_data` builds an `mpq::Stack`: optional explicit patch (`patch =` in the config or `D2_PATCH_INSTALLER`), else `patch_d2.mpq`, else `LODPatch_114d.exe`; then `d2exp.mpq`, `d2data.mpq`, `d2char.mpq`, sound/video MPQs (V gamedata_load.cpp ~1660-1710). Priority is push order, first archive that has the file wins (V mpq.hpp Stack::find). Loaders call `mpqs.try_read(R"(data\global\excel\)" + name + ".txt")` and `txt::Table` (V, repeated in gamedata_load.cpp; apps/d2d/load.cpp reads the UI DC6s the same way).
- **Whole-file replacement works today.** A `.mpq` pushed as the explicit patch outranks `patch_d2.mpq` (V). A full replacement `UniqueItems.txt` would therefore load with zero code changes, and `check_patch_layer` would log a warning that the row count differs from 401 (V; "warns, doesn't refuse"). This is the cheap prototype path for a private test, but it ships Blizzard's table text and is not for release.
- **Row-merge overlay (proposal).** In `Stack::try_read`, if an overlay directory is configured and `name` ends with `.txt` under `data\global\excel\`, parse the base bytes, apply the fragment `mod/excel/<table>.txt` (same header names; keyed by `index` / `Treasure Class` / `Name`: key exists -> replace, else append; an empty cell keeps the base cell), re-serialise and return. One place, no per-loader change. For strings, the same hook on `data\local\lng\eng\*.tbl` would merge a small key list; if the string path is awkward, a first slice can use `index` keys that already display sensibly (U).
- **Row ids shift nothing if we append.** Appended UniqueItems/SetItems/affix rows get ids after the vanilla rows, so existing saves keep their meaning (V item_text.hpp: ids are the row without separators; S that the saved field is wide enough for 401 + a few, U exact width: check in `d2s_items`).
- **Inventory and flippy graphics.** Both come from MPQ paths read by `try_read` (V). An overlay directory of DC6 would load through the same hook, so the mechanism covers art; authoring the art is the cost (section 1e).
- **Item generator needs no change** for (a) to (d) (V: generation reads `tables.uniques`, `tables.treasure`, built from the loaders). The Go bot side has its own table reader: `internal/data` extracts from the same MPQs into a cache with a stamp of file sizes and a `Patch_D2.mpq` hash (V internal/data/extract.go). The overlay must be part of that stamp and applied in the Go loader too, or the bot would price and name items from vanilla rows. Both sides read one shared fragment directory (our rows only).
- **Estimate (U):** fork overlay hook, config switch, tests: 10 to 14 h; Go data loader overlay + stamp: 3 to 4 h; doc and probe: 1 h. About 14 to 19 h; call it 16.

## 4. The two-mode rule

**Rule.** Vanilla mode is the default and the only mode in which fidelity claims are made. d.idle mode (overlay on) is the product mode. A character, a save and a run record each carry the mode.

- **Fidelity matrix, probe and oracles** (V docs/research/fidelity-matrix.md; fidprobe on branch commit 8318fb99 has result classes EQUAL, KNOWN, MISMATCH, UNREADABLE with `known.json` for explained differences): run them in vanilla mode. In d.idle mode, a custom row is a new class "NOT-1:1-BY-DESIGN": a custom unique id or drop route is listed in `known.json` with reason `custom row`, never counted as MISMATCH. Oracle and emulator cases are generated from vanilla tables and never see the overlay (game.exe cannot run our rows without the `-txt` path anyway, section 2).
- **Gates and contracts** (docs/contracts/fidelity-gate.md, soak gates): gates run on vanilla unless a gate says otherwise; a d.idle-mode run record carries `mode: d.idle (overlay <name> <hash>)` and does not count toward a Gate A fidelity verdict. A/B between modes is a strategy question and needs the standing A/B protocol (V docs/backlog-howto.md "Standing rule").
- **Shipping (docs/licensing.md, ADR 0010).** The overlay holds only our own rows (names, stat numbers we chose, a few strings). It is GPL like the repo. It is merged at run time with the user's own game files; no Blizzard table text is stored in the overlay or the image. `leakcheck` should get a rule that the overlay directory contains only rows whose `index` carries our prefix, plus replaced TC rows kept as diffs (key + changed cells), not full rows (U: a replaced TC row would repeat Blizzard cells; prefer adding a new private TC and changing only one pointer cell).
- **Saves.** A character that holds a custom item and is loaded in vanilla mode: the saved item has quality unique and a row id past the vanilla range (V that ids are rows, S/U the width). Expected (U, needs a test): d2d's lookup is `name_at(names.unique, id)` and `level_at(...)`, which return empty or zero for an id out of range (U, not traced), so the item would show without a name or as a base item; game.exe may reject the item or the whole character (S, known mod behaviour: "corrupt character" risk, U). Policy: a save is stamped with the mode in the bot's character record; vanilla refuses a d.idle character (offline check on the item ids in the `.d2s` before load) rather than relying on tolerance.
- **Tester build and container.** The tester ships the fork with the overlay hook default off and the d.idle config naming the overlay directory; the vanilla build is byte-identical in behaviour with no directory configured (test: no config -> row counts and a hash of loaded tables equal to today's).

## 5. Bot impact

- **Unknown item today (V).** `internal/loot` is rule-based on quality, type and code, with a final `default: drop` (V pickit_starter.json). A custom unique on an existing base has quality unique and a known base code, so it hits the rule "unique and set" (stash, tier A, prio 85) with no change. Wear logic uses the base (`WearScore`, `WearSlot`, V internal/loot/wear.go), not the unique row, so it would be judged as its base plus stats.
- **Price (V internal/data/price.go).** Selling or valuing a unique uses `unique_cost` pairs from the loaded UniqueItems row (cost mult/add); an overlay row supplies them if the Go loader applies the overlay (section 3). Without it the item would price as the base (U).
- **Name lookups (V internal/data/db.go).** `UniqueByName` and the `uniqByName` index come from UniqueItems; strings come from the `.tbl` parser. A custom row needs its string in the overlay or the name is empty/the key.
- **What each custom item needs (builds-are-data):** a keep rule only if we want it kept above the generic unique rule (a `codes`+`qualities` row in the pickit JSON); a value from the cost columns; and, if a build should wear it, a row in the plan table (`internal/plandata`), never a Go literal (AGENTS.md "Build flexibility"). For the first item: no Go code, one optional pickit row, one plan-table row for a "wear this cap" preference.
- **Risk (U):** a unique with a stat the bot's evaluator does not understand (a custom property) would be rated by base only; use existing properties for the first slices.

## 6. Product fit

ADR 0016 (V docs/adr/0016-automation-ladder.md) makes automation the progression: tiers T0 to T5, currency is anything sacrificed under one worth rule, T5 Farmer picks runs from drop tables, the altar is parked. docs/research/automation-ladder.md lists the T5 gaps (V): drop tables beyond Act 1 and 2 Normal, run-choice planner, goal-item stash policy, sacrifice altar.

Where custom items would serve (S/U, ideas, not decisions):
- **Farm goals for T5:** a named custom unique on a boss gives the Farmer a goal item with a known probability, and the planner a concrete drop table to reason about (also a clean test: drop rate from the computed probability).
- **Tier rewards:** a custom item as the reward for unlocking a tier or for a sacrifice level (needs the altar, which is parked).
- **Sacrifice targets:** a custom item can be given a worth in the one worth rule; low value as a first step.
- **Risks:** power creep (SME: anything beyond about two quality-tier upgrades makes Andariel trivial; keep to a sidegrade or mild chase item), breaking the clvl 17 Andariel acceptance (V ADR 0016 / Gate 2: acceptance is judged on the bot, so a custom item would change what the gate measures; keep vanilla for gates), balance work becoming a second job, and a split between "real D2" and "d.idle D2" in public communication.

## 7. First-item proposal

**Item (S design from d2-sme, numbers are a starting point):** "Raven's Cowl", a unique on the Cap base (existing art), `lvl req` 8, `lvl` 6, `rarity` high (e.g. 20, to beat dilution on that base), mods: +15 to 25 life, +10 to 15 all resistances, +5 to 10 percent magic find (existing properties only), +1 defense per level is dropped to keep it simple. Roughly a mild Rockstopper-lite (SME, S). No new stat codes, no new art. Names/strings: our own.

**Exact rows (all ours):**
1. `UniqueItems` overlay: one appended row `index=cowl_raven_x` (prefixed with our private key), `code=cap`, `lvl=6`, `lvl req=8`, `rarity=20`, `enabled=1`, prop1..4 as above.
2. `TreasureClassEx` overlay: a new class `DI Bloodraven Cowl` with one entry (`cap`, forced via a high Unique modifier) and a pointer entry added to the boss's own class at a weight that gives about 1 in 100 kills (S SME: 1 in 50 to 1 in 150; at 1/100 the 50 percent point is about 70 kills and 95 percent about 300). Exact weight computed from the loaded rows (U until the TC name is read).
3. String entry for `cowl_raven_x`.
4. Pickit row (optional) and plan-table wear row.

**Tests:**
- Offline row validation (public repo, Go): overlay loads, every row has its columns, keys unique, prefix rule, no row text copied from the base table (a leakcheck-style test).
- Fork unit test (C++): with the overlay on, `tables.uniques` has 402 entries and the TC resolves to the new class; with it off, table hashes equal the vanilla hashes.
- Drop-probability test (offline, fork): compute the exact probability from the loaded rows and compare with a Monte-Carlo of `roll_drops` with fixed seeds (millions of cases, minutes).
- Lockstep drill (live, user go per AGENTS.md): kill Blood Raven N times (N about 300 with the existing `internal/boss` scripts, one throwaway character), count drops of the item, compare to the computed probability with a binomial interval; `writes after start: 0`. D.idle mode only; vanilla run in the same drill must show 0 drops (control).
- Bot keeps it: offline loot test that the item (base `cap`, quality 7, our unique id) gets action stash under the existing rule; live: it ends in the stash.

**Repos:** overlay hook, config, fork tests -> the d.idle fork (private; push the same day, AGENTS.md); overlay data, Go loader support, tests, docs -> the public bot repo. Patches never go into the public repo.

**Estimate (U):** fork overlay 12 to 14 h, Go loader and stamp 3 to 4 h, item rows and string 1 h, offline tests 4 h, drill 5 to 6 h, docs and review 3 h: about 30 h (28 to 34), of which the live drill needs a game slot.

## 8. Open questions for the user (max 5, with recommendations)

1. **What are custom items FOR** (farm goal for the Farmer, tier reward, sacrifice target, or just flavour)? Recommend: farm goal first (one item, section 7), because it exercises T5 and costs the least.
2. **May the d.idle mode diverge from vanilla permanently** (gates and fidelity stay vanilla, d.idle runs are separate)? Recommend: yes, with the mode stamped on saves and run records.
3. **Power budget:** sidegrade/mild chase item only (SME), or a real game-changer? Recommend: mild, so the clvl 17 Andariel acceptance stays meaningful.
4. **Which boss first:** Blood Raven (weak, reachable at clvl 8 to 17) or Andariel (strongest placement, "must run")? Recommend: Blood Raven first, Andariel as a second-tier item later; keep the Countess for runes.
5. **Allow a short live probe** (one throwaway character, game.exe with `-direct -txt` and a loose `.tbl`) to settle section 2? Recommend: not now; only if we want game.exe to cross-check custom drops, which we do not need.

## Dead ends and not verified

- The TC names of Blood Raven, the Countess and Andariel are U (not read; the table text is the user's install). The first task reads them on a worker host.
- No game.exe run with `-direct -txt` was made; section 2 is the SME file's statement plus community knowledge.
- d2d's behaviour for an out-of-range unique id and for a missing string key was not traced.
- Nothing here was run; no fork file was changed.
