# Evidence pack: drlg-dump opens the folder's 1.14d patch (Lut Gholein two-room seed lag)

> status: for the user's go / no-go (agent LUTPROOF, 2026-10-31). Standard: docs/process/upstream-pr-standard.md. Proof: docs/research/lutgholein-proof.md. Draft body: docs/research/pr-lut-gholein-draft.md. Rebased on upstream main 4778a47c (UPSYNC section at the end). Nothing pushed, posted or filed.

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

**Residual risks.** For the maintainer (installer at `diff_drlg.py`'s default path) the change is a no-op; without that file upstream's default run failed outright before ("MPQ open failed") and works after, and the 0/20 "before" is the tool run with the variable unset. The d2d game's own Lut Gholein was not re-measured (its loader reads the patch; S). Harrogath's tile change shows the patch layer carries data files too: anything upstream measured with `drlg-dump` and no installer may have been measured against CD-era files. A user who had the installer sees no change. `presubmit` warns on the throwing `fs::exists` overloads (same as the existing line; answered in the body).

**Verifier (independent, adversarial, read-only; model fable).** Report quoted verbatim below.

> Verdict: HOLDS WITH LIMITS. The mechanism, the trace and the differential results check out; the "0/20 before" number is specific to an environment with no installer, and the fix is a no-op for the maintainer's own setup.
>
> What I ran (worker host, no builds, no emulator): `git show d59c3e5f`; read `open_game_data`, `find_file`, `Stack::push_installer`, `Archive::installer`, `units.hpp`, `diff_drlg.py`, `emu.py`, `lut_trace.py`, the trace, the disassembly, `AGENTS.md` s6, `CONTRIBUTING.md`; `pr_guard` and `presubmit.py --quick`; spot-run seed 7 level 40 three ways; `mpq-cat` on both folders.
>
> Findings by attack:
>
> 1. Baseline fairness. `diff_drlg.py` does `env.setdefault("D2_PATCH_INSTALLER", ~/Downloads/Diablo II + LoD/patch/LODPatch_114d.exe)` (line 41), and `emu.py` uses the same default for the emulator's own MPQ stack. On the maintainer's box that file exists (PLAN.md:704 records level 40 tiles 20/20, units 10/10), so the maintainer's gate already had the patch: with env set and present the after-stack is identical to before. Limit: the PR fixes every other user, not upstream's own gate. On a box without that file, upstream's default is worse than 0/20: the before binary exits 1 ("MPQ open failed: /nonexistent/...") and diff_drlg reports "drlg-dump failed"; the after binary warns and falls back, and matches the ref. The wrappers' `unset` was needed to get a before number at all. State both in the PR body.
>
> 2. Trace. 54 writes: 1 at 0x681f4f (memset), 2 at 0x650e40, 2 at 0x667744, 49 at 0x66b451; zero call events (steps are inlined). Disassembly: 0x667744 is `mov [esi],ecx` on the `*0xaaaaaaab` %3 path reached only via jump-table entry 0; index bytes decode to 0 for 0xcc, 0xcd (0x6677dc-de) and 0x173, 0x174 (0x667883-84 = `00 00`), 1 (keep) elsewhere. The trace records no unit ids, so "those two are 204/205" rests on the table plus the 20/20 result, not on the trace alone. `lut_trace.py` hooks nothing out; `drlg.boot` only re-enables the string lookup. Note `lut_trace.py`/`lut_seeds.py` are untracked, not in the commit.
>
> 3. Installer users: same stack. Level 109 (Harrogath) before: `path.dt1:5` vs `buildingses.dt1:82`, a different DS1/DT1 from the patch; game.exe opens patch first, so reading it is right. `layout` mode now also reads the patch; not gated.
>
> 4. presubmit --quick: 0 FAIL, 3 WARNs on `fs::exists` (lines 76, 77, 80; line 80 is the pre-existing d2exp pattern), pre-answer them in the PR body. `fs::exists(patch)` is called twice. Minor divergence: the game wraps `push_patch` in try/catch and falls through on a corrupt patch; the fix throws out of main.
>
> 5. See 1.
>
> 6. pr_guard 0 FAIL/0 WARN; author is the user, 1-line body (very long line), no trailer needed. Commit is fine.
>
> Spot checks: seed 7 level 40 before != ref (line 3), after == ref; after on the lab data folder (only `Patch_D2.mpq`, capital) == ref, so the case-insensitive lookup works. Note mpq-cat/MpqCache still use a case-sensitive `patch_d2.mpq` (that is why the lowercase symlink folder exists); not this PR's scope but worth a line.
>
> Nothing started, nothing left running; scratch dir removed.

Acted on (after the report): `fs::exists(patch)` computed once, the commit body wrapped to three lines; amended as 632d5cc2 (author the user, no trailer), rebuilt, plain dumps re-checked (level 40 20/20, the other five 10/10) and the level 40 gate re-run (`~/d2d-lab/pr/lut-diff/after-40-v2.txt`). Not acted on, listed as limits: the game's try/catch around a corrupt patch (the tool throws out of main, as it did for every archive before); mpq-cat / MpqCache's case-sensitive `patch_d2.mpq` (candidate follow-up, not this claim); the trace names addresses, not unit ids (the ids come from `monpreset.bin` plus the before / after `units` dumps).

**PROOF: complete** for the claim as narrowed (tool-side), subject to the verifier's findings above. The original card's premise ("d2d generator bug") is withdrawn: PROOF complete that it is not one.

**Recommendation.** Go, as a small tool PR (one claim: drlg-dump reads the folder's patch like the game), spaced >= 1 h after #135; the maintainer may prefer to fold it into `diff_drlg.py` or the README instead, which the body invites.

**Push and PR commands (NOT RUN; the user runs or approves each).**

```sh
cd ~/d2d-lab/pr/lut
git log --format='%h %an <%ae>%n%s%n%b' -1          # 632d5cc2 shatteredeagle, no trailer
git push origin fix/lut-gholein-preset-rng
gh pr create --repo psi29a/d2decomped --base main --head shatteredeagle:fix/lut-gholein-preset-rng \
  --title "drlg-dump: open the folder's 1.14d patch like the game does" \
  --body-file ~/code/d2lod-bot/docs/research/pr-lut-gholein-draft.md     # body = the part under the --- line
# then attach tools/emu/lut_trace.py to the PR conversation (not committed), and watch: gh pr checks <n> --watch
```

## Rebased onto upstream main (UPSYNC, 2026-10-31, local only)

- Base: upstream/main **4778a47c**. New head: **4bbf7f37** on `fix/lut-gholein-preset-rng` (`~/d2d-lab/pr/lut`), one commit (was 632d5cc2 on 32768ddb; backup branch `fix/lut-gholein-preset-rng-pre-upstream-merge`). No conflicts. Not superseded: upstream's `drlg-dump` still opens only `$D2_PATCH_INSTALLER` (grep of upstream/main), never the folder's patch. Nothing pushed.
- Differential re-run with both binaries rebuilt (before = upstream/main, after = the patch; both under the `unset D2_PATCH_INSTALLER` wrapper, `D2_MPQ_DIR=/tmp/ug-emu`): level 40 seeds 1-20 tiles / units / seeds: before 0/20 each, **after 20/20 each**. Regression levels seeds 1-10 tiles / units, before -> after: level 1 10/10, 0/10 -> 10/10, 10/10; level 2 10/10, 6/10 -> 10/10, 10/10; level 75 10/10, 0/10 -> 10/10, 10/10; level 103 10/10, 0/10 -> 10/10, 10/10; level 109 0/10, 0/10 -> 10/10, 10/10.
- Does upstream's own gate still fail on current upstream main, unpatched? It depends on the installer. With no installer at `diff_drlg.py`'s default path (this worker host: absent), the gate fails outright (`drlg-dump tiles failed (1)`, MPQ open failed). With `D2_PATCH_INSTALLER` pointing at the installer (which is what the maintainer's default path does on his box), unpatched upstream main passes 20/20 on tiles, units and seeds. So the claim stands as the verifier worded it: the fix helps everyone without the installer env, and is a no-op for a box that has it; the PR body must not say the maintainer's gate fails.
- Checks: `pr_guard.py upstream/main` 0 FAIL 0 WARN; `presubmit.py --quick` 0 FAIL, 3 WARN (the `fs::exists` error-path ones, answered in the body).

Commands to open (NOT RUN; need the user's explicit go; base main):

```sh
ssh worker.example 'cd ~/d2d-lab/pr/lut && git push origin fix/lut-gholein-preset-rng'
ssh worker.example 'cd ~/d2d-lab/pr/lut && gh pr create --repo psi29a/d2decomped --base main --head shatteredeagle:fix/lut-gholein-preset-rng --title "drlg-dump: open the folder'"'"'s 1.14d patch like the game does" --body-file PR_DESCRIPTION.md'   # body = the part of pr-lut-gholein-draft.md under the --- line
```

> Wording note (2026-10-31): the PR text must not claim upstream's gate fails in general; it fails or mismatches only with no installer env, and passes 20/20 unpatched with `D2_PATCH_INSTALLER` set. The draft body now says so.
