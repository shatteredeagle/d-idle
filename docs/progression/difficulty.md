# Difficulty: selection screen, unlock, gate, the end-of-difficulty hop

Sources: `docs/archive/prior-art/08-oog.md` sections 2-3 (Kolbot control geometry, SP difficulty dialog), `01-leveling.md` section 4 (SoloPlay gate and resist rule),
`docs/goals/planner.md` (gate predicates), `docs/cheats.md` (save layout), `docs/goals/tracking.md` (difficulty byte). Offline single player only; we never create
characters (AGENTS.md), so only *selecting* an existing difficulty is designed.

## 1. Facts and unknowns

| item | status |
|---|---|
| In-game difficulty byte at RVA 0x3A060C | V location; Normal = 0 only seen; 1 NM, 2 Hell are U (tracking.md item 8) |
| Save difficulty bytes 0xA8..0xAA, one per difficulty, bit 7 = active, low bits = act (cheats.md layout) | V layout; semantic of "which one is the *last played*" U (V-08) |
| Save `progression` byte 0x25 | V offset, editable by `d2cheat save progression=N`; its meaning as "unlocked difficulty" U (cheats.md says unverified) |
| SP difficulty dialog after choosing a character: Normal 264,297,272,35; NM 264,340,272,35; Hell 264,383,272,35 (800x600 control rects, Kolbot) | S/U for our client; dialog existence per start is U (08-oog section 3) |
| A locked difficulty's button does nothing when clicked | S (Kolbot fall-through "try highest first") |
| Control list walk from FirstControl identifies screens by (type,x,y,w,h) | V mechanism for FirstControl (PLAN), U for this dialog |
| `cmd/oog` exists | V (cmd/oog) and `launch.Flow` has a Cycler wired and "UNVERIFIED end to end" (bot.md) except one verified fresh-game cycle (bot.md run 10: new game in 1.5 s) |

## 2. Selection procedure (`launch.Flow.StartGame(char, diff)`)

1. Menu-state detector says `SelectDifficultySP` (control list signature: three wide buttons at the Kolbot rects, U) or `InGame` (player unit valid, V).
2. If the dialog is up: click the button of the *requested* difficulty. Do **not** copy Kolbot's "highest first" fall-through blindly: a goal that wants Normal must
   not land in Hell. Verified click = the dialog disappears within 5 s.
3. If the dialog is still up after 1 s (button disabled because locked): the requested difficulty is not unlocked. Return `ErrDifficultyLocked` (fault Kind
   `KindPrereq`), take no fallback click. The planner decides to level elsewhere.
4. After entering the game, read the difficulty byte and compare with the request (V byte, V mapping 0 only). Mismatch => leave the game (Cycler) and report; never
   play on in the wrong difficulty (Hell resistances kill a Normal build).
5. If no dialog appears (U: the dialog may only show for new games or when more than one difficulty is unlocked), the character enters with the last-played
   difficulty (stored in the save, bytes above). The verification in step 4 then fixes it with one more leave/enter only if a way to pick the difficulty exists;
   otherwise change the active difficulty through the save-cheat path on a *throwaway* character only (cheats.md: allowed for Botty/Multi/..., never otherwise).
   The product path for real characters is the dialog; if there is none, it is a documented limit.

Goal API: `goals.Params.Difficulty` (exists? tracking.md proposes `diff.enter`); the Runner passes `Difficulty` to `Cycler.Start`. No new runner state: the Cycler already
leaves and re-enters.

## 3. Unlock and gate

Unlock (game rule, S): killing Baal in Expansion on a difficulty unlocks the next difficulty for that character. The unlock writes `progression` (U) and/or the
quest bit "Eve of Destruction" (index 40, quests.md); either T1 signal. Classic: Diablo (not our case, offline expansion chars; check the status flag, V-02).

Gate to enter the next difficulty (SoloPlay `Check.nextDifficulty`, prior-art 01, S as a reference, our policy U):
- (G1) current difficulty complete (Eve of Destruction bit for this difficulty, T1/T2).
- (G2) clvl >= the cap: SP softcore Normal 33, Nightmare 64 (classic 60), Hell 100 (S). Our planner floors: progression-nightmare/hell.json `enter_clvl` (R) are
  lower; the **gate for the move is G2 or "over-levelled by 2 (Normal) / 5 (others)"** (S).
- (G3) not broke (gold >= the "broke" formula, prior-art 01).
- (G4) all three of fire/cold/lightning resist, **after** the penalty, > 0: Nightmare -40, Hell -100 in expansion, -20/-50 classic (S, "standard D2 values",
  U in this repo). Read current resists from stats 39/41/43 (V ids, values U) with the *current* difficulty's penalty already applied; the new penalty delta is
  `(pen(next) - pen(current))`. Poison (45) is not in the SP rule. Magic/physical not considered.
- Hardcore: out of scope (planner.md).

When G1 holds but G2/G4 fail: planner generates `level.reach` in the hardest area still sane, or `item.equip` / farm for resists, never the hop.

## 4. The hop (`Step: EnterDifficulty(next)`)

1. Finish Baal: loot picked, in town (TP) or in the Worldstone Chamber.
2. Save: leave game via the Cycler (Esc menu -> Save and Exit, 08-oog section 4 with the 3-tick stability protocol; D2R coordinates do not transfer, so the
   button position is calibrated by V-08 for the classic menu).
3. Backup the save here (WP-SAVES `Snapshot` at a menu, F3): this is the safest moment of the whole game, the character just finished a difficulty.
4. Re-read T1 (quests.md) to prove G1 for the *next* run (the unlock showed up).
5. `StartGame(char, next)` per section 2. The character appears in the Act 1 town of the new difficulty (S standard).
6. Verify: difficulty byte, act = 1, T1/T2 quest block for the new difficulty all zero except what the game preseeds, waypoints: only the town (tracking.md item 7).
7. Run the hop's first chore: `ActivateTownWP`, buy potions, then the planner takes over (a new campaign `nightmare_playthrough` etc. from planner.md).

## 5. Penalties the Fighter and planner must know on arrival

(S unless noted)
- Monster level and resist scale per difficulty: read monstats.txt through `internal/data` rather than tables (docs/plan/leveling.md already says so).
- Player resist penalty applies at once on entering NM/Hell; the first potions run should stop at the town if any resist < 0 (a "do not leave town below X"
  rule: `safe` family in the catalog, planner rule).
- Nightmare/Hell pack danger affixes (danger-table.json) are enabled by difficulty: the Fighter reads `difficulty` from the Observation and loads the matching rows.
- Act bosses are not re-played in a new difficulty until the quest chain reaches them: the planner starts at "level.reach:`enter_clvl` of `progression-<diff>.json`" and uses the playthrough template.
- Cow level and Normal-only rewards (Lam Esen, Den): the planner skips them in NM/Hell (quests.md section 4).

## 6. Open questions (checked in verification.md V-08)

1. Does the SP difficulty dialog appear on every start, only for characters with more than one unlocked difficulty, or only on the first start?
2. Is the active difficulty of a loaded character the *highest unlocked* or the *last played*?
3. Values of the difficulty byte for NM and Hell (1 and 2 expected).
4. Does `d2cheat save ... progression=N` or `difficulty=D` change what the dialog shows? (cheats.md: unverified.)
5. The classic Save-and-Exit button geometry.
