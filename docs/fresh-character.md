# Fresh-character soak path (plan-v2 M1)

Goal: soak a character from clvl 1 with a scripted, verified creation. `d2bot` itself never creates characters (it only re-enters an
existing one, `docs/plan/code-reality.md` s9); creation is a separate step by OOG tooling. The user approved creating throwaway
characters for soak testing on a designated instance (2026-10-30). Status of everything below: written and tested OFFLINE with fakes;
NOT run live (tags: V = verified live, S = secondary, U = unverified).

## Pieces
| piece | where | state |
|---|---|---|
| Create flow (class click read back from the header label, name, softcore, expansion, Normal) | `launch.Flow.CreateCharacter`, `fillCreate` | V live for all 7 class coordinates (docs/oog.md); bounded class retries (4) |
| `CreateVerified`: name rules, up to 2 create attempts only for failures that cannot have made the character, then verify in game, then Save and Exit | `internal/launch/fresh.go` | offline tests with `fakeWorld` for all 7 classes, retry, name clash final, mismatch |
| `oog fresh NAME CLASS` | `cmd/oog/main.go` | builds; reads name/class/level from game memory (`d2.Player`, V reader) |
| `saves.CheckFresh` + `d2savecheck NAME CLASS` | `internal/saves/fresh.go`, `cmd/d2savecheck` | offline tests on a hand-built v96 file; status bit layout V (docs/saves.md) |
| `tools/soak/fresh.sh <iN> <class> <Name> [--dry-run]` | refusal logic in `freshcheck.sh`, list in `fresh-names.txt` | offline tests `bash tools/soak/fresh_test.sh`; live path U |
| Cheat-ledger WARN in `soak24.sh preflight <iN> [NAME]` and `start` | `ledgercheck.sh` | offline test; remote path simulated locally |

## Procedure (for the main agent or a d2-live agent holding the instance)
1. Pick a free instance and a name from `tools/soak/fresh-names.txt` (Fresha..Freshg). Never i0. Never an instance a soak or agent uses.
2. `tools/soak/fresh.sh i4 sorceress Fresha --dry-run` (refusals and the soak line, starts nothing).
3. `tools/soak/fresh.sh i4 sorceress Fresha`. It refuses unless the instance is free (no d2bot, no soak unit, locks free, no restart notice,
   no Game.exe carrying the instance marker, no save of that name). Then it starts that game itself (`game.sh start` + `helper`), runs
   `oog "waitphase menu 60000" "fresh Fresha sorceress"` under the instance lock, stops the game it started, and checks the file with
   `d2savecheck Fresha sorceress` (name, class, level 1, exp 0, softcore, non-ladder, expansion). It never stops a game it did not start.
4. It prints `tools/soak/soak24.sh start i4 --gate --char Fresha`. Run that yourself; the script does not start a soak.
5. No cheats on the character (docs/cheats.md "Policy for fresh test characters").

Failure handling: name taken, popup, wrong class after all retries, verify mismatch or a missing file exit non-zero with the reason; the game
it started is stopped by an EXIT trap. A half-made character (created but file check failed) keeps its name: pick the next name; do not
retry the same one. Class names: amazon assassin barbarian druid necromancer paladin sorceress. Only Sorceress has an unattended managed
run (code-reality s9): other classes can be created but are smoke-test material, not gate material.

## Name rules (U)
`launch.ValidCharName`: 2 to 15 letters, at most one inner `-` or `_`. Stricter than needed on purpose; whether the game accepts digits is U.

## LIVE CHECKLIST for a d2-live agent (nothing here has been run)
Instance and name given in the brief, nothing else running there. Ask the main agent to approve the first creation.
1. `tools/soak/fresh_test.sh` and `go test -count=1 -p 1 ./internal/launch ./internal/saves` pass.
2. `tools/devhealth.sh`; `tools/game.sh status iN` shows no game. `tools/soak/fresh.sh iN sorceress Fresha --dry-run` prints `is free`.
3. Real run. Watch: (a) `game.sh start` ends with exactly one Game.exe pid for iN; (b) `waitphase menu` reached (U: phase value at the
   main menu right after start); (c) the log `flow:` lines show class picked (header label), name typed, OK clicked, difficulty clicked if the
   screen appears (U for a fresh Normal-only character), in game; (d) `fresh: verified in game: Fresha Sorceress level 1`; (e) Save and Exit
   reaches the main menu; (f) `FRESH OK` from d2savecheck. Record time per step.
4. Record in docs/oog.md: whether the name box needs the retries, popups seen, and whether the `.d2s` exists right after creation or only
   after Save and Exit (U: this script always exits, so it does not matter for the flow, only for the docs).
5. One class other than Sorceress (e.g. paladin, Freshb) to confirm the class coordinates under the new verification.
6. Then `soak24.sh preflight iN Fresha` (expect the ledger line "no entry") and the 2 h gate from the printed line.
7. Cleanup: `fresh.sh` already stopped its game; confirm with `tools/game.sh status iN` and `tools/jobs.sh list`. Bring back later with
   `soak24.sh start` (the managed bot launches the game).
