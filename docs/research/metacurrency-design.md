# Metacurrency design: per-character and account-wide currencies from custom items

> status: approved by the user 2026-11-06 (all six recommendations; recorded in docs/decisions.md and the ADR 0016 Amendment 2026-11-06). Design, 2026-11-06, card `metacurrency` (TASK-61; docs and tasks only: no Go change, no live run, no fork change). Answers the user's 2026-11-06 reply to docs/research/custom-items.md: custom items are for a METACURRENCY, "both, one that is per character and one that is account wide, we may end up adding multiple". The study's other open questions (power budget, which boss first, the Blood Raven unique) are n/a: the first custom item is a currency token, not gear.
> Tags: **V** read this session in this repo; **S** secondary (a repo doc, the d2-sme agent's veteran memory, tagged EXP); **U** unverified or from memory. Target classic 1.14d. D2R facts are flagged D2R. No Blizzard table text is copied; only column names and our own proposed rows appear.

## Answer in eight lines

1. **Currencies are data.** A registry row: `id, scope (per_character | account), name, source, sink, stackable, display, converts_to`. A third currency is a new row, not code (section 1.1).
2. **Two currencies to start (working ids, names are open question 1):** `mark`, a per-character stackable ITEM in the hero's pack; `tally`, an account-wide LEDGER balance in the app. `mark` converts to `tally` when banked in town (section 1.1).
3. **Per-character storage = a real stackable Misc item** from the overlay (new code, key art reused, D2 EXP: the key is the best donor, stack 12). **Account storage = an app ledger** (append-only, hash-chained file under the profile) credited only from OBSERVED pack loss at a bank action. Runner-up: a per-character counter in the app only (no item).
4. **The original game has no shared stash** (V docs/research/d2-landscape.md: PlugY is the only account-wide mechanism, a DLL; S d2-sme confirms). The fork does not model one (U: no `.d2x` in any save seen, V docs/research/character-factory.md). So an in-game account store is not an option; the ledger is.
5. **Sources:** treasure-class rows (overlay), a first-kill bonus per boss, tier unlock grants, quest grants. Flat per-kill chance, no magic-find coupling (section 2).
6. **Sinks:** ladder sacrifice upgrades first (ADR 0016), a little per-hero convenience through the cube second, gear power never (section 3).
7. **Safety:** vanilla mode untouched, d.idle mode stamps saves, custom rows KNOWN as not 1:1 by design, no effect on Gates (section 4).
8. **First slice, about 38 h without TASK-59:** one token, one early source, one cube bank recipe, one ledger credit, one balance line, one kill drill (section 5). Needs TASK-59 (overlay).

## 1. The model

### 1.1 Registry (data, not code)

One JSON table in the repo, read at start, validated (unknown scope or missing field is an error that names the currency; no silent default, AGENTS.md "build flexibility" spirit). Proposed file `internal/currency/registry.json`:

| field | meaning |
|---|---|
| `id` | stable key (`mark`, `tally`); never reused |
| `scope` | `per_character` (lives with one hero) or `account` (lives with the player profile) |
| `name`, `blurb` | display strings, plain language |
| `storage` | `item` (overlay Misc code) or `ledger` |
| `item` | for `item`: overlay code, stack cap, art donor (data only) |
| `sources` | list of source ids (section 2), each with its rate as data |
| `sinks` | list of sink ids (section 3) |
| `stackable`, `cap` | stack cap for items; soft cap per day for the ledger (runaway guard, section 2.4) |
| `converts_to` | `{currency, ratio, via}`: `mark -> tally, 1:1, via cube bank` |
| `display` | where it shows (watch header, spend sheet), icon name, order |

Worked rows (working names):

- `mark`: scope per_character, storage item, converts_to `tally` 1:1, sinks `bank`, `cube_boon`.
- `tally`: scope account, storage ledger, sinks `ladder_upgrade`.
- Later additions the same table allows without code: a per-character `boon` item that is only spent in that hero's life; an account `ember` earned from milestones only; a "per difficulty" mark (Normal/Nightmare/Hell tokens as three rows).

Fit with ADR 0016 (V docs/adr/0016-automation-ladder.md items 15 to 17): the ADR says "Anything can be sacrificed, under one worth rule with no second currency." That text conflicts with a user wish for several currencies. Reconciliation, for the user to confirm (open question 2): there is still ONE worth unit. `tally` is "banked worth": the sum of worth points of what was sacrificed or banked. `mark` has a worth number in the one worth table like any item; banking turns `mark x n` into `n x worth(mark)` tally. Currencies are different WAYS to hold worth (in a pack, in the bank), not different units. If the user wants truly separate units, ADR 0016 item 15 needs an amendment (proposed decision line in section 7).

### 1.2 Storage evaluation

Per-character currency (`mark`):

| option | persistence | death / reset | vanilla-mode save | tamper | races | verdict |
|---|---|---|---|---|---|---|
| (a1) real stackable Misc item in the pack/stash | in the `.d2s` (V docs/snaplib.md: the whole file set is snapshotted) | stays on the corpse (softcore single-player: recoverable); lost on hardcore death (hardcore is unbuilt, V ADR 0016 item 9) | a vanilla load meets an unknown code (U: refuse by offline scan, see 4) | visible in-game, so the player sees it; a write is a `d2cheat` entry (V AGENTS.md ledger) | none (one hero, one file) | **recommended** |
| (a2) app counter per character | app store | survives death and loss (too safe) | none | file edit | easy | runner-up; no risk/carry feel, and the design loses "the drop is visible" |

Account currency (`tally`):

| option | persistence | loss on death/reset | vanilla compat | tamper | multi-instance | tester `/data` | d2d vs game.exe |
|---|---|---|---|---|---|---|---|
| (i) shared stash | none in plain 1.14d; PlugY (S EXP, V d2-landscape) | n/a | PlugY is a DLL; not in d2d | n/a | n/a | n/a | **not available**: d2d has no `.d2x` model (U; checked only the repo docs, the fork was not grepped this session) |
| (ii) app ledger, credited on pickup | file in the profile | none | no item needed | hash chain, but pickup observation is easy to double-credit (drop, re-pick) | needs a lock | `/data` volume (V docs/research/tester-build-plan.md) | works the same on both backends |
| (iii) hybrid: items drop on the hero, a bank action converts pack loss to a ledger credit | file in the profile; items on the hero in between | unbanked marks can be lost with a hardcore hero; banked ones never | vanilla mode: no marks exist, ledger untouched | credit is an observed pack LOSS at a bot-run town action, with a write-ahead entry; cheated stacks refused | flock + one append per line | same | d2d: observe pack via memory (V mem reads are always allowed) |

**Recommendation: (iii) hybrid.** `mark` item in the pack, a bank action in town that removes the stack, and a ledger credit equal to the observed loss. It keeps the item visible, gives carry-versus-bank a tiny bit of risk (EXP: veterans like the carried-versus-banked gold split; the SME also advised that "the ledger should be the real currency and the game item only visible proof of a drop"; S), and credit never depends on a pickup event. **Runner-up: (ii)** with a credit on pickup: simpler, but pickup events replay (a dropped and re-picked stack, a game restart that re-reports the same unit) and so need a unit-guid dedupe that is harder to prove than a pack-loss diff.

### 1.3 The bank action (before the altar exists)

The altar is parked: a private fork object, out of measured binaries, GPL reviewer not named, built after the store and the Thoughts feed (V ADR 0016 items 17 and gates). Until then the bank action uses what the game has:

- **Cube bank recipe** (a CubeMain overlay row, our own key): `n x mark` in the Horadric Cube, transmute, the stack is consumed. The cube planner already exists (V internal/cubejobs). U: whether an output can be empty or must be a placeholder item (e.g. a junk scroll the pickit already ignores); the first task probes it. U: a recipe with an input stack of quantity n, how the game counts quantity in a recipe; fallback is a fixed recipe "1 mark, 1 per transmute".
- **Fallback if the cube cannot take it:** the bot drops the stack on the town floor and does not re-pick (pickit rule `mark in town: ignore`); pack loss is still the observation.
- **Later:** the altar replaces the cube; same observation contract.

### 1.4 Observation rules (no double credit, no cheat credit)

1. **Write-ahead.** Before the bank action the bot appends `bank_intent{char, run, seq, pack_count_before}`. After it, it re-reads the pack, appends `bank_commit{seq, pack_count_after, credited = before - after}`. Credit is the OBSERVED loss, never the intended number.
2. **Idempotent.** An entry key is `(character id, seq)`; replaying a log never credits twice. A crash between intent and commit leaves an unmatched intent: on the next start the bot reads the pack and settles it (credit `before - now`, capped at `before`), or voids it if the count is unchanged.
3. **Hash chain.** Each line carries the previous line's hash. Detects truncation and editing by accident, not by an adversary with file access (the threat is bot bugs and silent corruption; the player owns the machine).
4. **Cheat provenance.** The cheat ledger (V AGENTS.md: every write goes through `internal/cheat`) is consulted: a stack whose character has any setup write of the token code since the last bank is credited as `cheated` (kept in the ledger, spendable only if the profile flag allows, never counted in a gate or A/B), same rule as the CRUTCH RULE. Run records carry `writes after start: N` next to the ledger head hash.
5. **Multi-instance.** One append per line under `flock` on the ledger file (Linux, the tester container). Each line carries the instance id. Balance = fold over the log, cached in a `store.File` snapshot (V internal/store: File and Log exist; Log is append-only JSONL with rotation, so the ledger must NOT rotate away entries: use `MaxBytes` 0, or fold rotated files).
6. **Spending** writes `spend{upgrade id, cost, level_before, level_after}`; the balance cannot go below zero (the spend is refused, not clamped).

### 1.5 Where it lives

`<data>/profile/<profile>/currency/ledger.jsonl` plus `balance.json` (U path: it follows the ladder store convention `<data>/instances/<id>/state/ladder`, V docs/t0-slice.md; the rules store is per profile "player account", V decision-point-store-spec.md line 157, so the ledger uses the same profile scope). "Account" therefore means PER PROFILE (per install), not a Battle.net-style name (open question 3). Hardcore rebirth keeps it (a rebirth resets the hero, V ADR 0016 item 9: quality survives, "sacrificed goods are lost"; the ledger holds banked worth, which behaves like quality: survives). Open question 4 asks whether it should.

## 2. Sources

### 2.1 Treasure-class rows (overlay, d.idle mode only)

Our own TreasureClassEx lines (V custom-items.md section 1c, 3): a private class `DI Mark` with one entry (the token code) and pointer entries added to existing classes by key. The rows belong to the overlay (TASK-59); the numeric rates below are data in the registry and the overlay, tuned from soak logs.

Per-kill, flat chance, NOT tied to magic find (S EXP: keeps the economy predictable):

| source | rate (starting guess, U) | note |
|---|---|---|
| common monsters in the first field | about 1 mark per 40 kills | early, so a first session earns |
| champions/uniques/minions | 3x the common rate | |
| Blood Raven, Countess, Andariel | 2 marks the first time per profile (first-kill grant), then about 5 per 100 kills | boss visits are rare |
| chests | about 2 per 100 | |
| quest completion | one-off 5 to 10 | |
| difficulty scale | x1, x2, x4 for Normal, Nightmare, Hell | |

Stack quantity: a stackable Misc dropped from a treasure class gets a random quantity (S EXP likely; U exact rule: `spawnstack` or a `minstack`..`maxstack` roll). A drop of one at a time is easiest to reason about: set `minstack` 1 and `maxstack` 12 on the row but check the roll with the offline Monte-Carlo before relying on it (TASK-63).

### 2.2 Grants

| grant | currency | rule |
|---|---|---|
| tier unlock T0..T5 | `tally` directly | a one-off, listed in the ledger as `grant{reason}`, by tier (size U) |
| first rule learned | `tally` small | teaches the loop; a sacrifice can skip one repetition (V ADR 0016 item 12) |
| order completed | none | orders are cheap and repeatable; a reward would invite order spam |

### 2.3 Economy sketch (U, from measured speeds)

Speeds (V docs/research/gate2-analysis-20261102.md and the Gate 2 g2f record): an 8-hour seed has about 12 thousand kills (user figure 2026-11-06, S) so about 1,500 kills/hour; XP/h about 9 thousand (S). Common monsters give 1,500/40 about 37 marks/hour; boss visits perhaps 1 per game, fresh game every ~4 minutes (V docs/g1-log.md, a different run), so assume 0 to 5 boss marks/hour in the early field. About 40 `mark` per hour in Normal, 80 in Nightmare, 160 in Hell by the difficulty scale. The SME target is "about one token every 5 to 10 minutes early, rare big ones about once an hour" (S EXP), which matches 20 to 60 per hour.

### 2.4 Caps against runaway

- Stack cap on the item (12 for a key donor).
- Ledger soft cap per profile-day (tunable; U 2,000): credits beyond it still book but are flagged `over_cap` and display "banked, bonus sealed"; the cap is data.
- Costs rise geometrically (section 3.2), so a linear source cannot buy everything at once.
- Rates are flat; a farm goal (T5 Farmer) can raise the time spent in a source but not the per-kill rate.

## 3. Sinks

### 3.1 What it buys

1. **Ladder upgrades ("sacrifice upgrades", V ADR 0016 items 14 to 16; V automation-ladder.md section 4: an upgrade is data: knob, value, cost, using the Tuning metadata, V docs/knobs.md).** This is the main sink. Candidate first buys, each a knob row, not code:
   - fewer needless retreats (T0), larger belt rule or refill threshold (T1), shorter town visits (T1), a faster thought fade for answering (T0), merc revive priority (T4), loot filter precision (T1/T5).
2. **Per-hero convenience via the cube** (mild; the `mark` sink `cube_boon`): e.g. marks for potions or scrolls through a CubeMain overlay row. Stay mild (EXP: anything beyond two quality tiers makes Andariel trivial, V custom-items.md section 6).
3. **Not a sink:** gear power. No unique, no rune words, no stats in exchange for currency. Avoids the item-power inflation the study warned about, keeps the clvl 17 Andariel acceptance meaningful (V ADR 0016 gate text).

### 3.2 Worth rule and pricing

ONE worth table (V ADR 0016 item 15). A `mark` is a row with a worth number set low enough that banking a mark never beats cubing it (the ADR's own guard). Cost of a level: priced "in farming time of the tier below" (V ADR 0016 item 16). With about 40 marks/hour, a sketch (U): level 1 about 30 tally (under an hour), level 2 about 75, level 3 about 180 (ratio 2.4); the tier above takes longer per level. Costs are data in the upgrade table.

### 3.3 The altar (parked)

Before it exists: the worth table and quality levels in the bot (in this repo, V ADR 0016 item 17), the private-fork object (shows the altar and removes the item only), the store and Thoughts feed done, a named GPL reviewer for shipping (V ADR 0016 gates). The cube bank is a stand-in that needs none of these.

### 3.4 What the player sees and does

- **Balance:** one strip in the watch header (V docs/research/watch-screen-design.md section 1): `Marks 12 in pack · Tally 340 banked`. The data contract gets a `currency` block (list of `{id, scope, name, amount, cap}`) so a third currency appears with no UI change (the screen already uses a versioned JSON contract, V section 2 there).
- **Spend sheet:** a card listing each upgrade: name, plain effect, cost, level, "buy" with a confirm. Irreversible choices default to doing nothing (V ADR 0016 item 11), so no upgrade is bought by a timeout. A bank action is automatic (T1 Tidy does it when the pack holds marks and the hero is in town) and journals a line ("Banked 7 marks: tally 340").
- **Onboarding (3 plain sentences, V onboarding-ux-design.md style):** "Your hero finds marks while it fights. In town it turns them into tally, which stays with you even if the hero is lost. You spend tally on upgrades that teach the hero to play a little better."

## 4. Fidelity and safety

- **Vanilla mode untouched.** No overlay, no tokens, no ledger entries. A d.idle-mode character is stamped; vanilla refuses a stamped save with an offline scan of item codes before load (V custom-items.md section 4 policy). U: how d2d and game.exe react to an unknown Misc code; test in TASK-63 first.
- **Fidelity probe.** Custom rows are registered in `internal/fidprobe/known.json` as `custom row: not 1:1 by design` (the fidprobe is on its own branch, commit 8318fb99, V custom-items.md section 4; the file may not exist on main; confirm when merged). They never count as MISMATCH.
- **Gates.** Gates and A/B stay on vanilla (V custom-items.md section 4). A d.idle run record carries `mode: d.idle (overlay <name> <hash>)`; a strategy change that spends currency needs the standing A/B (V docs/backlog-howto.md).
- **Legal.** Overlay rows are our own: a new Misc code with our key prefix, our TC lines as diffs, our CubeMain row. The recipe of the key art donor refers to the user's own game files at run time; no Blizzard text or art is stored (GPL, V docs/licensing.md, ADR 0010). Patches stay in the private fork (AGENTS.md).
- **Cheat rule.** Setup writes of tokens are allowed on throwaway characters (SETUP purpose, V AGENTS.md) and are marked in the ledger as cheated.

## 5. First slice (the smallest end-to-end proof)

**Content:** ONE per-character stackable (`mark`, key art, d.idle mode only); ONE early source (a common-monster class of the first field the bot already farms; the exact MonStats TC row is read in TASK-63: U) at a high test rate (1 in 10) with a production rate in data; ONE cube bank recipe; ONE ledger credit; ONE balance line on the watch screen; ONE drill.

**Rows and code (all ours):**

| piece | where | hours (U) |
|---|---|---|
| Misc row (new code, key invfile, stackable, stack 12), string, TC private class and one pointer, CubeMain bank row | overlay fragments `overlay/di-currency/` (TASK-63) | 4 |
| Probe: does the fork accept a new Misc code in a `.d2s` read and write; stack quantity roll; cube with empty output | fork, private branch, push same day (AGENTS.md); TASK-63 | 6 to 10 |
| Registry + ledger + write-ahead + observer + cheat provenance + tests (two currencies) | `internal/currency/` (TASK-62) | 14 |
| Bank step in town (cube), pickit "keep mark, ignore in town", watch `currency` block + header strip, journal line | `internal/currencyflow/` (TASK-64) plus a small named change in `internal/watch` (report: needs an owner) | 10 |
| Drill: kill N, pack count versus binomial interval, credit equals pack loss, `writes after start: 0`, vanilla control gets 0 | `docs/probes/currency-drill.md` (TASK-64), live slot with the user's go | 6 |
| docs and review | | 2 |

Total about 42 h (range 36 to 50), on top of TASK-59 (16 h overlay) which it depends on. Drill N: at 1 in 10 and N=300 expect 30 with a 95% binomial interval of about 21 to 41; offline Monte-Carlo of the rows first.

**Dependencies:** TASK-59 (overlay hook, Go loader and stamp). TASK-60 is reworked (below) and becomes the umbrella card for the slice: its acceptance is now the token row and the kill drill. TASK-62 (ledger) is independent of TASK-59 and can start now; TASK-63 needs the overlay spec; TASK-64 needs both.

**Tasks:**

- Created TASK-61 (this design). Proposed under label T5: TASK-62 `currency-ledger` (owns `internal/currency/`, `docs/currency.md`), TASK-63 `currency-rows` (owns `overlay/di-currency/`, `docs/research/currency-rows.md`), TASK-64 `currency-slice` (owns `internal/currencyflow/`, `docs/probes/currency-drill.md`). Owns do not overlap.
- Edited TASK-60: the Raven's Cowl unique is no longer wanted. Title now "custom-item-1: stackable currency token from one early source"; its two old acceptance criteria were replaced by the token-row and kill-drill criteria. Its `owns` (`docs/research/custom-item-1.md`) is unchanged; it is the umbrella that TASK-62 to 64 close into.

## 6. Veteran notes merged (d2-sme, S EXP, unchecked against tables)

- Stackable Misc: keys (stack 12, likely), TP/ID scrolls (20), arrows and bolts (weapon class, quiver; bad donors). Best donor: the key, 1x1, harmless offline. Keep the 12 cap and bank often.
- Runes are the de facto currency online (Ist as anchor) but offline there is no trade; essences are quest items (four cube into a Token of Absolution).
- A currency feels good with small frequent drops, rare big ones, a boss pity drop, banking on pickup so death cannot lose them, and a cap or sink that pushes spending. Our design banks by observed loss instead of on pickup; same result with fewer replay risks.
- Original knobs worth copying: carried gold cap 10,000 x clvl, stash gold cap 2,500,000, softcore death drops part of carried gold, Gold Find only affects gold (certain per the SME). The carried-versus-banked split is the model for `mark` versus `tally`.
- The SME doc set has no `minstack/maxstack/spawnstack` notes: TASK-63 reads the extracted Misc and TreasureClassEx tables on a worker host.

## 7. Open questions for the user (max 6, with recommendations)

1. **Names and fiction of the first two currencies.** Recommend: `mark` is "Raven's Mark" (a token the fallen leave; the Rogue Encampment's dead), `tally` is "Tally" (what the hero's patron keeps). Plain, stackable, tied to Act 1. Rename later is a data change.
2. **One worth unit or truly separate currencies** (ADR 0016 item 15 says "no second currency")? Recommend: one worth unit; `tally` is banked worth; amend the ADR wording to "one worth unit, held as items or in the bank".
3. **Account = per profile (install) or per Battle.net-style account name?** Recommend: per profile, same as the rules store; there is no Battle.net in an offline tool (V AGENTS.md).
4. **Can currencies be lost on death?** Recommend: carried marks only, never `tally`; softcore corpse keeps them; hardcore (unbuilt) loses unbanked marks; the ledger survives rebirth.
5. **First upgrade they buy.** Recommend: T0 "faster answer fade" or "retreat precision level 1"; cheap, visible, and exercises the spend sheet before any T1 work exists.
6. **Allow a short live probe for a new Misc code** (one throwaway character, d2d only)? Recommend: yes, under TASK-63, because the whole slice rests on the fork reading and saving a custom Misc code (U); if it fails, fall back to reusing the real key code with TC lines only (no new Misc row; every key in the bot's hand then counts, so the bot's vendor key buying must be off).

Proposed glossary terms (not written): **Mark** (per-character currency item), **Tally** (account-wide banked worth), **Bank** (convert marks to tally in town), **Registry** (the currency table). Proposed decision line for the coordinator: "2026-11-06: custom items are a metacurrency, per character (a stackable item) and per profile (a ledger of banked worth); one worth unit; first slice TASK-60 to 64."

## Dead ends and not verified

- The fork was not read this session; the d2s reader's acceptance of a new Misc code and the stack-roll rule are U.
- Whether CubeMain accepts an empty output, and how it counts a stack as an input, is U.
- Monster and TC names for the first source are U (table text is the user's install).
- The rates in section 2 are guesses from the SME and the user's kill figure; tune from a soak. The hour estimates are U.
- fidprobe's `known.json` may not be on main.
