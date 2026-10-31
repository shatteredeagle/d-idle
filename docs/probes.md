# Live probes (cmd/probes)

Prebuilt, never run live yet (everything here is U until a d2-live agent runs it; this file says what each probe
would prove). All probes print one JSON object per line on stdout with the envelope `probe`, `kind`, `t` (RFC 3339)
and then fields. They read memory only; `runewords cube -go` is the single probe that presses buttons
(`Town.Transmute`), no cheats are used by any probe. A failed memory read prints a `{"kind":"error"}` line and exits 1.

Run (agent name picks the instance, AGENTS.md; runs under the instance lock, keep it under 3 minutes):

    tools/live.sh <agent> probes <sub> [flags]
    tools/live.sh --bg --max 300 <agent> probes monflags -sec 150     # long ones in the background, then tools/jobs.sh tail <job>

Common flag `-sec N` (1..170, default 20). Bad flags fail before the game is touched. Offline check:
`go test -p 1 -count=1 ./cmd/probes` (fake snapshots, argument parsing, output shape).

## 1. runewords (docs/items.md Q1: do ladder-flagged things work offline?)

Two subs. `check` is read-only, `cube` runs a recipe.

Runewords are made by socketing runes by hand in 1.14d, not in the cube (docs/items.md 3.1, no code does it yet), so
the runeword half is manual and the probe brackets it. Setup (d2-live agent, throwaway character, Cheats via
`cmd/d2cheat` only if a rune or base has to be made, each with a ledger reason):

1. Pack: a white socketed base for Spirit (sword with 4 sockets; shield with 4 sockets) and Tal, Thul, Ort, Amn in
   order (Insight: Ral, Tir, Tal, Sol on a polearm/staff with 4 sockets). Spirit and Insight are both ladder-flagged
   in runes.txt (V list in docs/items.md 3.1).
2. `tools/live.sh <agent> probes runewords check` -> baseline. Expect `summary.candidates` to name "Spirit" with
   `"ladder": true` once the 4 runes are in the pack (this only proves the rune multiset, not the order or base).
3. Insert the runes in order by hand (pick up the rune, click the base in the inventory; screenshot with
   `tools/shot.sh`). Run `runewords check` again.
4. Verdict by eye + data: the base's item line now has `"runeword": true` (flag 0x04000000, V layout) and the
   runes are gone from the pack. If the item shows `"runeword": false` but `socketed: true` with all runes consumed,
   the insert did not make a runeword: ladder flag blocks it (check the in-game item name colour with the screenshot).
   `runeword: true` = Q1 answered YES for that runeword; change `RunewordAvailableOffline` to ignore `Ladder`.

`cube`: tests the other ladder-flagged piece, the Dol -> Hel recipe (cubemain ladder=true V, docs/items.md 4.2):

    tools/live.sh <agent> probes runewords cube              # dry run: prints the plan and what is missing
    tools/live.sh <agent> probes runewords cube -go          # 3 Dol + 1 Chipped Emerald through Town.Transmute
    tools/live.sh <agent> probes runewords cube -go -recipe custom -inputs r07,r07,r07,gcr   # any inputs by base code

Preconditions: in town, Horadric Cube in the main pack (Act 2 quest item) and the inputs in the pack or stash-free
inventory (Transmute only reads the pack; Dol = r14, Hel = r15 V cubejobs; Chipped Emerald `gcg` is S, same pattern as
the V codes gcy/gcv/gcw). Expect `kind=transmute` with `outputs:["r15"]`, then `verdict.result` =
`ladder-recipe-works-offline`. `no-result` / `error` (ErrCubeNoResult) = the game refused the recipe: ladder flag blocks
Hel+ upgrades offline (Q1: NO). The run closes all panels and prints `summary_after`.

## 2. monflags (docs/difficulty.md: flag byte and enchant array, both U)

    tools/live.sh <agent> probes monflags -sec 60 -radius 40 -ms 1000

Preconditions: outside town, with monsters around, ideally a champion pack, a unique (e.g. Corpsefire, a Rare) and
its minions in range; stand still or fight normally. Output: a `monster` line the first time a unit is seen (and
again whenever its flag byte or enchant bytes change) with raw `flags`, `flag_bits`, `mon_kind` (what
`MonsterInfo.Kind` says), `enchants_raw`, `enchant_ids`, `enchants_trusted`, `unique_no`, `unique_name`, life, states.
Town NPCs, the merc and corpses are skipped. `summary` holds histograms and a `verdict` hint.

Interpret: bits are 2 normal, 4 champion, 8 unique, 16 minion (D2BS, U). Compare each `monster` line with the game
(`tools/shot.sh`: blue name = rare/unique, etc.). If every flag is `0x00` the +0x16 offset is wrong; if normal
monsters read 0x02 and a known champion reads 0x04 the bits are V and `monflags.go` can drop its U notes. For enchants,
the ids are monumod.txt rows (max 40, V); a champion should show none or its mods, a pack should share the leader's.
Record the evidence in docs/difficulty.md.

## 3. mercdump (merc and NPC dialog)

    tools/live.sh <agent> probes mercdump -sec 60

Preconditions: a hired merc (Rogue Scout is the only class seen live, 271; the others U). While it runs open an NPC
dialog (click Kashya/Greiz) and the merc panel. Output once a second: `kind=merc` with `present, alive, class, id, x, y,
life_pct, level, name`, plus `ui_open` (every UI flag word that is set, `0x08=1` is the NPC menu, V) and `merc_panel`.
While the NPC menu flag is up a `kind=dialog` line adds hover id/type, the NPC class and position, and `lines_expected`.
Limits: the dialog TEXT is not in any known memory path (U), so `lines_expected` is the `town.MenuOrder` label list
(V only for Gheed 147 and Akara 150, default Talk/Trade/Cancel otherwise); take `tools/shot.sh` for the real text and
compare. Questions it answers: does `level` (stat 12) and `life_pct` of a damaged merc read plausibly (docs
merc-farm.md 1.4, U), does a dead merc stay `present` with `alive:false`, which UI flags show with the revive dialog.

## 4. missiledump (docs/difficulty.md "Wiring (MISSILES)")

    tools/live.sh <agent> probes missiledump -sec 60 -ms 250

Preconditions: in a fight where things shoot or cast (Fire Wall, Lightning, Hydra, Chaos), or cast your own skills;
for curses fight Amplify Damage/Decrepify/Lower Resist/Weaken/Iron Maiden casters (Dark Shamans, Gloams, Fallen
Shamans...). Output: `kind=player` every tick with `curses`, the ids used (`curse_ids`, resolved by name from
states.txt: V once resolved) and `states_on` (every set state bit with its states.txt name); `kind=missiles` lines
with class, `name` (missiles.txt), `kind` (MissileKindByName) and owner ids, only when something is in flight.
`summary` lists every missile name seen with count and the kind it was given, `classified_other` (names that matched
none of the U substrings: fix `MissileKindByName` from these), `states_seen`, and a verdict.

Interpret: `ok:false` on every sample means the client missile table (rvaClientUnits) is wrong. Missiles with plausible
x/y near the player confirm it (V). A curse cast on the player must flip its bit and name in `states_on`; if the name
differs from the S default ids (9/56/61/19/55) the bit order assumption (StateSet bit n = state n, U) is off.

## 5. wpdump (docs/act3.md X4: Act 3-5 waypoint data)

    tools/live.sh <agent> probes wpdump -sec 40

Preconditions: a character that has been to Acts 3-5 towns (Navi's i1 is the natural one). Stand in the town next to the
waypoint with the panel open, once per town (Kurast Docks 75, Pandemonium Fortress 103, Harrogath 109), and for the tab
and row geometry also open it in a second act. Output: three `kind=progress` lines (normal, nightmare, hell) with the
raw mask (`0x...`) and `acts3to5` rows (bit, level, name, act, active) decoded with the S saves table order; a `state`
line at start, when the panel flag (UI 0x14) flips and every 5 s while a waypoint object is loaded, with `waypoint_objects`
(class, x, y, mode) and the viewport. Waypoint object classes are the `d2map` list (S).

Use it for: (a) the `x, y` of the town waypoint = value for `nav.knownWaypointPos[75|103|109]` (per seed: note the
seed/char); (b) confirming bit -> level against what the game lets you click (an activated waypoint shows as `active`);
(c) `panel_open` proves the flag. What it does NOT give: the panel's tab x for acts 4/5 and the icon rows (7, 8): those
are screen geometry, not memory. Measure them with `tools/shot.sh` while the panel is open and click with the existing
navlive tooling, then update `wpPanel` in internal/nav (not this tool).

## Dead ends and limits (all U)
- NPC dialog text, waypoint panel text and the item runeword NAME are not read from memory by any known path.
- `Item.FileIndex` (+0x28) is U, printed raw as `file_index` in `runewords check` in case it carries the runeword id.
- The probes assume `data.Open` finds the install at the usual proton path (same as cmd/townlive).
