# Charm management: carry, pack budget, build wants, Vault and Inventory UI

> status: design note, docs only (2026-11-27, agent charms). Nothing here is built. Tags: V = read in this tree at commit 388dcc3da, or a page I fetched (URL given); S = secondary or snippet; U = unverified or from memory. Target: classic 1.14d, not D2R (D2R-only facts are flagged). Consumers: the four cards in section 7.

## 1. Game facts (classic LoD 1.14d)

| Fact | Tag | Source |
|---|---|---|
| A charm works only while it is in the BACKPACK. Not in the stash, not in the Horadric Cube, even when the Cube sits in the pack. | V | Arreat Summit "Charms", Wayback 2012 snapshot: https://web.archive.org/web/20120925053438/http://classic.battle.net/diablo2exp/items/charms.shtml |
| Charms are Magic or Unique only. There are no Rare, Set or Crafted charms. | V | same page, and the prefix pages (`.../items/magic/smallcharms.shtml`, `mediumcharms.shtml`, `largecharms.shtml` under the same Wayback prefix) |
| Three sizes. Arreat calls them Small, Medium, Large; in game they are Small, Large, Grand (the Medium page says "known as Large within the game", the Large page "Grand"). Repo codes: `cm1`/`cm2`/`cm3`, types `scha`/`mcha`/`lcha`. | V names (Arreat), V codes (`internal/data/invfile_test.go`, `internal/data/invfile_real_test.go`) | as above |
| Footprints 1x1, 1x2, 1x3 cells | U (memory; read `invwidth`/`invheight` of `misc.txt` via `db.ItemByCode("cm1".."cm3")` before coding) | - |
| Pack is 10x4 = 40 cells, stash 6x8 = 48, Cube 2x2 = 4 (in the pack). | V | `docs/inventory.md` line 4; `internal/data/real_test.go:90` (10x4 pack) |
| Stats come from affixes by size. Small (V): life 5-20 (Life 5-10, Sustenance 11-15, Vita 16-20), single resist up to 10-11, all-res 3-5, FHR 5%, MF 3-7% ("Fortune", "Good Luck"), +1-2 Str/Dex, Run/Walk 3%, elemental/poison damage, defense, AR. Large (in game; Arreat "Medium"): life up to 26-35 (Vita), single resist up to 13-15, all-res 3-8, FHR 8%, MF 1-6%, Str/Dex 2-5, mana. Grand (Arreat "Large"): single resist up to 26-30, all-res 3-15, FHR 12%, MF 1-12%, Str 3-6, life from +5 up (top tier not read: the snippet cut), and ONLY on Grand the 21 class-skill-tree skillers (+1 to one tab, e.g. Burning = +1 Sorceress Fire Skills, Lion Branded = +1 Paladin Combat). | V (Arreat prefix/suffix pages; numbers copied from them) | the three URLs above |
| No charm affix list I read has Faster Cast Rate; no +all skills on a magic charm. (Skillers are +1 to a skill TAB, not all skills.) | V for "not in the lists fetched", U for "not anywhere" (Arreat is the 2012 snapshot, patch 1.13-era) | same |
| Small charm "Skillers" do not exist in classic. The "charm skiller" is the Grand +1 tab charm; a Grand 1x3 costs three cells for +1 to a tab. | V | Large page above |
| Unique charms (Arreat): Annihilus (Small, clvl 70: +1 all skills, +10-20 all stats, +10-20 all res, +5-10% XP), Hellfire Torch (Large in game, clvl 75: +3 to a random class's skills, +10-20 all stats, +10-20 all res, 5% Firestorm on strike, level 30 Hydra charges, +8 light), Gheed's Fortune (Grand, clvl 62: 80-160% gold, -10-15% vendor prices, 20-40% MF). You can hold only one of each in backpack, cube and stash together. Annihilus and Torch come from the Uber bosses (Annihilus: Uber Diablo; Torch: the Uber Tristram/Pandemonium keys flow; the Arreat page lists them at 1.10 and 1.13). | V stats and one-of-each rule (https://web.archive.org/web/20120925053438/http://classic.battle.net/diablo2exp/items/normal/ucharms.shtml); S/U for how the offline single-player bot would obtain them (Pandemonium keys are out of the bot's reach today, `docs/capabilities.md`) | - |
| Arreat tip: put Life and Resist charms in the stash and carry them only for tough bosses. Arreat also says charms are best chosen sparingly because they eat pack space. | V | charms page |
| Charm drop rate in this repo's model: roughly 0.1 per area clear in the early areas. | V (as data), the model's own caveats apply | `internal/plandata/dropmodel-normal.json` `per_clear.charm` |
| D2R: charms became a tab-expandable pack; the 1.14d bot is not D2R. D2R-only: Sunder charms, the bigger stash, the "charm inventory" changes. | S, flagged D2R | `docs/builds-endgame.md:79` |

### 1.1 Which charms for which build (the usual logic; S/U, SME-class judgement, not measured)
- The universal pick is the small charm with Life (+20 life Vita, tier 16-20) plus any all-res or single resist small charm: 1 cell each, so best value per cell. Resist charms are decisive on the way to Hell: they patch a missing resist. Value per cell: small >> large >> grand for flat stats; grand is only good for a skiller or a big resist (26-30).
- Casters (Sorceress, Necromancer, Druid; `d2.ClassTraits.Caster`): FHR and life; a build-matched skiller on a Grand (Burning, Sparking, Chilling for Sorc; Graverobber's or Fungal for Necromancer; Natural, Spiritual for Druid) is worth 3 cells when the tab holds the main skill. Melee: life, resist, +Str/Dex only when a requirement gap needs it.
- Magic Find is a pure farming charm; the Arreat Grand MF goes to 12%. For a bot that does not farm for drops it is worth little; flagged as a build-data weight, not a constant.
- Typical committed endgame budget: Annihilus + Torch + Gheed (when available) + 5 to 10 small charms. 10 smalls = 25% of the pack, the figure to quote when discussing cost. U (player lore).
- Gheed's Fortune is a gold/MF charm and not worth 3 cells for a gold-starved early bot; keep it stashed unless the plan farms (U).

## 2. What the bot does today with charms (V, read in this tree)

| Area | What exists | Where |
|---|---|---|
| Ground pickup (Starter) | unique charm: stash, S tier, prio 95. Small charm magic/rare, identified, with life>=20 OR any resist>=5 OR FHR>=5 OR FCR>=1 OR MF>=1 OR any skill stat: stash, A, prio 70. Large/grand: same test, only while `needed: charmroom` (3 free pack cells). Unidentified small: keep + `needId`; unidentified big: keep + `needId` with charmroom. | `internal/lootrules/starter.json` (rules `g-unique-charm`, `g-small-charm`, `g-big-charm`, `g-small-charm-id`, `g-big-charm-id`); legacy mirror `internal/loot/pickit_starter.json` lines 8, 18-25; room check `internal/loot/loot.go:650` |
| Pack chore | every charm in the pack is moved to the stash (`stash`, reason "charm"). Nothing wears or keeps a charm in the pack. | `internal/lootrules/starter.json` `t-charms`; `internal/tidy/policy.json` row "charms"; `internal/tidy` |
| Sell | charms are "valuable": plain identified small charms sell, valuable charms are never sold or dropped unless a rule says so; unidentified charms in a full pack are stashed. | `internal/lootrules/schema.go:97` invariant; `internal/town/stash.go:288`; `internal/town/shop.go:712`; test `internal/loot/charms_test.go` (`TestCharmsRunesGemsOnSellVisit`) |
| Vault preset | unidentified magic/rare charms go to the vault like other magic items (the gold reveal opens them there); Keep to stash or pack exists; the Vault page has a "Charms" kind filter. | `internal/lootrules/vaultpreset.go:8-14`; `internal/watch/web/vault.json:265,301`; `docs/vault.md` Keep section |
| Vault equip | refuses charms: "Charms and belt or backpack items are not equipped here". | `internal/watch/web/vault.json:108,113`; `internal/vaultkeep/equip.go` (not wearable: a charm) |
| Scoring / leveler | `leveler/judge.go`: charms get no merc slot; docs say charms and respec are "Not done". No code reads charm stats into a build score or into a "do I have enough resists" gate. | `internal/leveler/judge.go:105`; `docs/leveler.md:93` |
| Watch UI | an item `kind` word "charm" and the "C" fallback letter; no per-build view, no carry state. | `internal/status/inventory.go:26`; `docs/inventory-ui.md:96` |

Gaps found while reading:
1. **No carry concept at all.** Every kept charm is stashed or sold; a charm is never kept in the pack to work. The game gives the bonus only from the pack (V), so today the bot collects charms whose effect it never gets. Worse, the `t-charms` pack chore and the tidy row "charms" would move a charm the player Kept to the pack straight back to the stash (S by reading the rules; U live). Any carry feature must protect carried charms from those two rules first.
2. **Rare charms are a dead class.** The rules match `quality in [magic, rare]`; charms cannot be rare (V Arreat). Harmless, but rare is dead data and misleading.
3. **Two dead predicates for small charms.** `stat:fcr >= 1` cannot match: I found no FCR affix on any charm size (V for the three lists; the stat is a wasted clause). `stat:mf` on a small charm is only 3-7% from two affixes.
4. **Value filter is not build aware.** One stat-or-list for every class: a Sorcerer's Burning grand is `stat:skilltab` yes, but a Barbarian-class tab charm also passes (all tabs pass `skilltab >= 1`), so the pack fills with other classes' skillers. `classskills` and `skilltab` are class-blind here. U: confirm that the predicate does not look at the class.
5. **Pack cost is counted for room (charmroom = 3 free cells) but not against the kit** (potions, tomes, scrolls, keys, Cube, gems for the cube). Nothing decides how many cells charms may take.
6. **No charm-vs-charm comparison**: a Vita small is kept next to a newly found better Vita; `spareSurplus` is not charm-aware.

## 3. The packing problem

Pack 10x4 = 40 cells (V). Claimants:
- Fixed kit: TP tome, ID tome (U sizes; `ibk`/`tbk` in `docs/items.md:183`), keys (1 cell), the Horadric Cube (2x2 = 4), 1x1 gems for the Cube (a stack of flawless; `GemKeep`, `internal/loot/loot.go` gem case), potions that did not fit the belt (`loot.potionBuffer`), loot in transit to the stash chore, and a pickup reserve ("keep some space for one or two large items" per Arreat).
- Charms: S/M/L = 1/2/3 cells (U sizes).
- Today's trip trigger: `town.TripPackUsedPct` leaves town runs when the pack is about X% used (`internal/env/tuning_town.go`); a carried-charm budget lowers the usable cells and so brings the trip earlier unless the budget is subtracted from the denominator (a card decision, section 6).

Proposed budget model (a dark knob, not a constant):
- `charms.packCells`: how many pack cells charms may occupy (default 0 = no change). A sensible live default for an endgame hero: 8 to 12 (U).
- `charms.reserveCells`: cells that must remain free for pickups beyond the kit (default 4, U).
- Carry set = the best subset that fits `packCells`, chosen by build value per cell (knapsack over 1/2/3-cell items; the candidate set is at most tens of items, so exact DP is cheap). Everything else is a stash charm (kept, not sold).
- Keep vs sell: a magic charm with a build-value of 0 under every known build is sold (as today for plain ones); any charm that is in the best set of some class or is a unique is stashed forever. Never sell by "worse than carried" alone: it may be the only resist patch for a later build.
- Cube interplay (V the Arreat rule): charms in the Cube do not count; the Cube is a pack item, so the sell-chore must not stash a charm into the Cube to "save" it. Gem upgrades and the Cube recipes use the same pack cells; the Cube is only open in town and is cleared after use.
- Stash: 6x8 = 48 cells (V). A Grand is 3 cells; stashing every ID-ed charm hits the stash limit (`tidy.stashMaxPct`) fast, so charm stash needs a cap with a drop policy (a card item).

## 4. Builds as data (no class literals)

Charms are build wants, so they belong in the build data, not in Go (AGENTS.md "Build flexibility").
- Per-build charm profile, in the same table family as the plan table (`internal/plandata/build-*.json`, `build-catalog.json`) or the combat profile (`internal/combat/classdata/profiles.json`), a new key `charms`: `{ "weights": {"life": 1.0, "allres": 4.0, "res_fire": 2.5, ..., "fhr": 2.0, "skill_tab:<tab name>": 12.0, "all_skills": 20, "mf": 0.1, "str": 0.3, "dex": 0.3}, "targets": {"fireres": 75, "lightres": 75 ...}, "maxCells": 12, "carryUniques": ["Annihilus","Hellfire Torch"] }`. Weights default from a class-neutral table; a build only overrides what differs (hammerdin: fhr and FCR-free; Sorc: fhr and a tab).
- The class-blind stat list in the Starter becomes `group charms` plus a new rule field `value` (build value per cell) read from the active build, so the Starter rule says "keep when value per cell >= knob", not a stat list. This replaces `g-small-charm`/`g-big-charm` stat clauses (they stay as the offline fallback until the new field has live proof).
- Skill tab words come from the class tables by name (`skills.json` tab names), never an id. A tab not of the hero's class has weight 0 by the same lookup, which also fixes gap 4.
- Resist need is derived: `needRes = target - currentRes` per element from the character's resists and the difficulty penalty (`docs/difficulty.md` resist gate), so a charm is worth more while a resist is short and worth little once capped (value falls to the threshold, not zero, for the mercenary swap case). This needs a `resistGap` reader that the leveler gate already approximates (U: find the existing reader before building another).
- `d2.ClassTraits` (`internal/d2/classtraits.json`) gets at most one new bool/int trait if needed, e.g. `charmFHRMatters`; most data goes in the per-build profile, not the class table.
- Two-build test (AGENTS.md rule): Sorceress (default) and Paladin hammerdin; a Burning grand scores for the first and 0 for the second.

## 5. Vault and Inventory UI

Ideas, in the Vault's own verbs (V `docs/vault.md`: Keep, Reveal, Sell, Pin, Discard, Sell all with two-tap confirm; equip refuses charms V):
- **Carry** replaces Equip for a charm. The Keep sends it to the pack (`target: inv` exists V); Carry adds the protected flag `carried` (like the `keep` mark in `docs/inventory-ui.md`: "never sold or dropped by the town; allowed in any container") plus an exemption from the pack-to-stash chore (gap 1). A Keep to the pack of a charm that is not Carried gets a visible warning "the pack chore will stash it".
- **Compare by build value**: each charm card shows the active build's value, the cells it costs, and value per cell, with the delta against the weakest carried charm it would replace (`+3 all res, -20 life`). The value engine is the same function the pickit uses (one source: section 4).
- **Suggest swap**: a single action "Suggest best carry set" that lists the proposed pack (with the total value, cells used, cells free after the kit) and Apply queues Keep or Stash orders in order, one town visit, with the existing in-flight recovery of Keep (V `docs/vault.md` crash recovery). The player may edit before apply. Never automatic until a dark knob `charms.autoCarry` is flipped (ladder rung rule, ADR 0016).
- **Inventory page**: the pack canvas marks carried charms and shows "charms: 9 / 12 cells" with a bar next to the kit, a one-line effect sum (life, resists, FHR, skills) so the player sees what the pack gives.
- **Sell-all safety (hard rules)**: Vault Sell all and the inventory Sell-now never include (a) a carried charm, (b) any unique charm, (c) any charm whose build value per cell is within the top set of any known build, (d) a charm marked `keep`. Sell all shows the excluded count and why ("3 charms held back: carried or wanted"). Charms sell only by explicit single-item Sell with the two-tap confirm. A unit test per exclusion, in two builds.
- Cheat-free: all of this only reads items and places them; no memory writes (CRUTCH RULE). Everything is offline-tested; nothing is claimed live (U).

## 6. Risks and unknowns
- Sizes of tomes and the 3 charm bases are U until read from `misc.txt`.
- Whether `d2d` reads charm bonuses into the stat sheet the way Game.exe does: `docs/forkdbg.md:65` says the gear sum counts "inventory charms" (S), so carried charms probably affect the character stats; confirm with a read-only probe.
- Whether `skilltab`/`classskills` predicates are class-aware (gap 4): read `internal/lootrules` stat table before the value card.
- Unique charms in offline play: Annihilus, Torch and Gheed's availability in single player is not verified (Arreat lists Realms/ladder notes); the bot cannot reach Pandemonium keys now.

## 7. Staged proposal (in the order the player would see value)
Each card: new `agent/<name>` branch, one owner of shared files, dark knob default 0 with a `data/darklaunch.json` row.

1. **charm-carry-protect** (smallest, visible first). Owns: `internal/tidy/*`, `internal/lootrules/starter.json` (pack scope rules only), `internal/lootrules/vaultpreset.go` if needed. ACs: (a) a charm marked carried (`keep` mark) survives the tidy and `t-charms` chore in both builds' tests; (b) Sell all and Sell now exclude carried and unique charms; (c) the excluded count is returned in the API answer. Knob: `charms.carryProtect` (default 0, `[dark-launch]`).
2. **charm-value** (data, then UI). Owns: `internal/plandata/charmprofile.go` plus `build-*.json` `charms` keys (shared files: one owner per wave), `internal/lootrules/` stat predicates. ACs: value function per build and per size; dead FCR and rare clauses removed with a note; a Sorceress build scores a Burning grand, a Paladin build scores 0 for it; resist gap reader. Knob: `charms.valueRule` (default 0 = old stat list).
3. **charm-vault-ui**. Owns: `internal/watch/web/vault*.js`, `internal/watch/web/vault.json`, `internal/vaultkeep/carry.go`. ACs: Carry verb for a charm (Keep to pack plus mark), build value, cells and delta on the card, Sell all held-back line. Knob: `loot.vaultCarry` (default 0). V offline tests only.
4. **charm-pack-budget** (the knapsack). Owns: `internal/charms/` (new), `internal/town/` hooks for charmroom replacement, `internal/env/tuning_town.go` knobs `charms.packCells`, `charms.reserveCells`. ACs: best carry set under the budget; trip trigger counts the budget; stash cap for charms. Knob default 0 (no change).
5. **charm-suggest-swap** (the action). Owns: `internal/charms/suggest.go`, watch route `POST /api/charms/suggest`, `docs/charms.md`. ACs: suggestion listing and Apply through the Keep queue; autoCarry stays 0. Live drill only after the user's go, with writes disabled.

## 8. Decisions needed from the user
1. Does the bot ever carry charms (cell budget and the default 8 to 12 cells), or is "Carry" a player-only action for now?
2. Should the pickit keep dropping rare-cap rules and grand charms of other classes (stop collecting foreign skillers), or keep everything valuable for a future build change?
3. Do unique charms (Annihilus, Torch, Gheed) need a special plan card (Uber runs) now, or only the carry rule when one appears?
