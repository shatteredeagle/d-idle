# Evidence pack: drlg-dump opens the folder's 1.14d patch (Lut Gholein two-room seed lag)

> status: for the user's go / no-go (agent LUTPROOF, 2026-10-31). Standard: docs/process/upstream-pr-standard.md. Proof: docs/research/lutgholein-proof.md. Draft body: docs/research/pr-lut-gholein-draft.md. Nothing pushed, posted or filed.

**Claim (one sentence).** `tools/drlg-dump` never opened the MPQ folder's `patch_d2.mpq` (only `$D2_PATCH_INSTALLER`), so without the installer it had no `monpreset.bin`, mapped no DS1 monster id, skipped the Scan / Pops "roll to stay" steps game.exe takes in `FUN_00667620`, and seeded Lut Gholein's rooms two level-seed steps late; opening the MPQs as the game's `open_game_data` does makes its dumps equal game.exe's.

**What this is not.** Not a generator bug: `components/drlg` (`stays`, `generate_preset`) already does the roll; the d2d game's loader already opens the patch. LUTGHOLEIN's "real d2d bug" was the tool's data, so the claim is narrower than the card asked for: a tool fix (CODEOWNERS review of `tools/`), plus the fact that upstream's `diff_drlg.py` gate only held with the installer in `$D2_PATCH_INSTALLER`.

**Proof summary.** game.exe's own DRLG under upstream's emulator (`drlg.py`, `FUN_006194a0` server-side), with a memory-write watch on the level seed (`lut_trace.py`, nothing hooked out): 2 writes in `FUN_00667620` (the `% 3` roll, `0x667744`) then 49 in `FUN_0066b3e0`; the rolled units are the DS1's monsters 25 / 26 = MonStats 204 / 205 through `monpreset.bin` (patch only). d2d's tool, old and new binary, through upstream's own `diff_drlg.py`, same emulator, same folder, no installer in the environment.

| scenario | cases | mismatches before | after |
|---|---|---|---|
| level 40 tiles / units / seeds | 20 seeds each | 20 / 20 / 20 | 0 / 0 / 0 |
| level 40 plain dump (room seeds), `cmp` | 20 seeds | 20 (all equal with a two-room shift) | 0 |
| level 1 tiles / units | 10 | 0 / 10 | 0 / 0 |
| level 2 tiles / units (Cottages 2, Pops) | 10 | 0 / 4 | 0 / 0 |
| level 75 tiles / units | 10 | 0 / 10 | 0 / 0 |
| level 103 tiles / units | 10 | 0 / 10 | 0 / 0 |
| level 109 tiles / units | 10 | 10 / 10 (the patch's act 5 files) | 0 / 0 |

Coverage is 20 seeds for the claimed level and 10 for each other town, not 100k: a seed is one whole level (49 rooms, thousands of tiles), the generator is deterministic, and every case the gate produces was compared; more seeds are one command (`1-100`) if the user wants them before the push.

**Commands (worker host, run by this agent).**

```sh
cd ~/d2d-lab/pr/lut/tools/emu
D2_MPQ_DIR=/tmp/ug-emu /tmp/casttrace-venv/bin/python lut_trace.py 161464580 40          # the 54 level-seed writes
for k in before after; do D2_BUILD=build-$k D2_MPQ_DIR=/tmp/ug-emu /tmp/casttrace-venv/bin/python diff_drlg.py 1-20 40 tiles units seeds; done
for L in 1 2 75 103 109; do for k in after before; do D2_BUILD=build-$k D2_MPQ_DIR=/tmp/ug-emu /tmp/casttrace-venv/bin/python diff_drlg.py 1-10 $L tiles units; done; done
cd ~/d2d-lab/pr/lut && python3 tools/pr_guard.py upstream/main HEAD && python3 tools/presubmit.py --quick
D2_MPQ_DIR=/tmp/ug-emu build/tests/test_game; build/tests/test_combat
```

(`build-before` / `build-after` hold sh wrappers that unset `D2_PATCH_INSTALLER` and exec the upstream-main build or the fixed one; outputs in `~/d2d-lab/pr/lut-diff/`.)

**Residual risks.** The d2d game's own Lut Gholein was not re-measured (its loader reads the patch; S). Harrogath's tile change shows the patch layer carries data files too: anything upstream measured with `drlg-dump` and no installer may have been measured against CD-era files. A user who had the installer sees no change. `presubmit` warns on the throwing `fs::exists` overloads (same as the existing line; answered in the body).

**Verifier (independent, adversarial, read-only; model fable).** Report quoted verbatim below.

VERIFIER_REPORT_PLACEHOLDER

**PROOF: complete** for the claim as narrowed (tool-side), subject to the verifier's findings above. The original card's premise ("d2d generator bug") is withdrawn: PROOF complete that it is not one.

**Recommendation.** Go, as a small tool PR (one claim: drlg-dump reads the folder's patch like the game), spaced >= 1 h after #135; the maintainer may prefer to fold it into `diff_drlg.py` or the README instead, which the body invites.

**Push and PR commands (NOT RUN; the user runs or approves each).**

```sh
cd ~/d2d-lab/pr/lut
git log --format='%h %an <%ae>%n%s%n%b' -1          # d59c3e5f shatteredeagle, no trailer
git push origin fix/lut-gholein-preset-rng
gh pr create --repo psi29a/d2decomped --base main --head shatteredeagle:fix/lut-gholein-preset-rng \
  --title "drlg-dump: open the folder's 1.14d patch like the game does" \
  --body-file ~/code/d2lod-bot/docs/research/pr-lut-gholein-draft.md     # body = the part under the --- line
# then attach tools/emu/lut_trace.py to the PR conversation (not committed), and watch: gh pr checks <n> --watch
```
