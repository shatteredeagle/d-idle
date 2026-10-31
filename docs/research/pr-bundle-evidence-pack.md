# Evidence pack: bundle PR (renderer get-hit frames + Lut Gholein tool fix)

> status: current (agent BUNDLEPR, 2026-10-31, card upstream-bundle). For the user's go / no-go. Body: pr-bundle-draft.md. Per-claim packs: pr-get-hit-render-evidence-pack.md (proof: pr-get-hit-render-proof.md), pr-lut-gholein-evidence-pack.md (proof: lutgholein-proof.md). Nothing pushed, posted or filed; the maintainer's bundle preference (psi29a, 2026-10-31) is in docs/process/upstream-pr-standard.md section 4.

posted:

## The bundle

Branch `fix/bundle-get-hit-render-lut` (upstream-clone worktree `~/d2d-lab/pr/bundle` on the worker host), two commits on upstream/main **afa0cdae** (upstream moved from 4778a47c: 57 files, none in the two claims' hunks; both cherry-picks applied with no conflict, messages and authorship kept):

| Commit | Claim | Source branch (untouched) |
|---|---|---|
| 4224a03d | monsters and the merc draw a get-hit once at its rate and hold the last frame | `fix/get-hit-render` 56fd98a7 (`~/d2d-lab/pr/render`) |
| e0731ebd | drlg-dump opens the folder's 1.14d patch like the game | `fix/lut-gholein-preset-rng` 4bbf7f37 (`~/d2d-lab/pr/lut`) |

Head: **e0731ebd**. Authors shatteredeagle (noreply), no trailer. The renderer commit's body is `Agent: Claude Code`; the Lut commit's three-line body has no Agent line (the PR body carries it; a commit-body Agent line is not required by pr_guard).

## Re-run on the bundle (worker host, build in `/tmp/bundle-build`, Release)

- `ninja test_combat test_game drlg-dump`: ok. `test_combat`: ok, rc 0. `test_game` with `D2_MPQ_DIR` (the lab data folder) and `D2_PATCH_INSTALLER` (the installer exe): rc 0, all scenarios OK.
- Renderer AFTER differential, saved cases in `/tmp/render-emu` (cases.tsv, game.tsv): `55089 cases, 1747580 ticks compared: 0 ticks in 0 cases differ; 523 ticks the client had ended; 0 cases with another rate or hold`; `after-bundle.tsv` is byte-identical to the earlier `after.tsv`. BEFORE (unchanged since the rebase on 4778a47c; the renderer lines it quotes are untouched by upstream's moves): 1,488,667 ticks in 49,641 cases.
- Lut differential with the after tool (wrapper unsets `D2_PATCH_INSTALLER`; `D2_MPQ_DIR` = a folder with only `Patch_D2.mpq`, d2exp, d2data): level 40 seeds 1-20 tiles / units / seeds **20/20, 20/20, 20/20**; levels 1, 2, 75, 103, 109 seeds 1-10 tiles / units **10/10 each**. BEFORE not re-run on afa0cdae (`git diff 4778a47c afa0cdae` is empty for `tools/drlg-dump`, `diff_drlg.py` and `emu.py`): 0/20 each on level 40, per the Lut pack.
- `python3 tools/pr_guard.py`: upstream/main..HEAD 0 FAIL 0 WARN; upstream/main..HEAD~1 (renderer alone) 0 FAIL 0 WARN; HEAD~1..HEAD (Lut alone) 0 FAIL 0 WARN.
- `python3 tools/presubmit.py --quick`: 0 FAIL, 6 WARN, all answered in the body (siblings, net roles, doc-claims "never loops" for combat.md:260, error-path `fs::exists` at drlg-dump.cpp:76 and :81).

## Verifier verdicts (independent, adversarial, read-only; quoted in the per-claim packs)

- Renderer: HOLDS WITH LIMITS (Fable; hand re-read of the chain plus 5,544 adversarial cases, 6,273,036 ticks at FHR >= 0, 0 differ; findings are the negative-FHR and frozen-monster limits). Disclosed limits: the player's frames, a joined game's merc, pets, rates at or above frames x 256.
- Lut Gholein: HOLDS WITH LIMITS (Fable; the 0/20 "before" is specific to an environment with no installer and the fix is a no-op for the maintainer's own setup; both stated in the body). Acted on: `fs::exists(patch)` computed once, commit body wrapped.

## Excluded

- #134 (get-hit rule): already open upstream, not a candidate. #135, #136: merged.
- Cast follow-ups (`fix/lightning-sequence`, `fix/skills-doc-cast`): held for the maintainer's response, no complete proof recorded.
- XP, spawn and monster candidate lists: "none ready" in their own docs.
- Nothing else in docs/BACKLOG.md or docs/research/README.md is marked ready to open.

## Overlap with other contributors' open PRs (GET-only check, 2026-10-31)

- #130 (mendescotta, "Frames at the display's refresh; units slide between ticks"): touches `apps/d2d/town.cpp` (`draw_view` blend hunk near :3306, plus :22, :443, :2215, :2424, :2934), `main.cpp`, `town.hpp`, `world_view.hpp` (:41), new `unit_slide.hpp`. It blends positions only; `unit_slide.hpp` has no mode / elapsed / frame handling, so the frame our rule picks (`get_hit_elapsed` in `world_view.cpp`, `get_hit_rate`) is unchanged. Textual overlap with our hunks: none in `town.cpp` (ours at :161, :163, :208; theirs elsewhere). `git apply --3way` of its diff conflicts in `town.hpp`, `world_view.hpp` and `tests/CMakeLists.txt` on BOTH upstream/main and our bundle, so those conflicts are #130's stale base, not ours.
- #119 ("Port reviewed fixes from fork PR #6 and correct animation data flow", same author): touches `apps/d2d/town.cpp:200` (`view_units`), `world_view.cpp:46` (`object_elapsed`, an object's FrameCnt), `components/game/replication.hpp` (:116 `monster_look`), `world.cpp`, `world.hpp:53`. Semantic overlap: none (objects vs get-hit; its replication hunk is another field). Textual overlap: yes, adjacent hunks. On upstream/main its diff conflicts only in `tests/test_game.cpp`; on our bundle also in `apps/d2d/town.cpp` and `apps/d2d/world_view.cpp` (our `get_hit_elapsed` sits beside `object_elapsed`, our `view_units` lines beside its :200 hunk). Resolution is trivial (keep both) for whichever lands second.
- Recommendation: bundle as is. Neither PR changes the draw-frame selection our rule relies on. If #119 merges first, rebase the renderer commit (two small adjacent-hunk conflicts, re-run the AFTER differential); if #130 merges first, no conflict with our commits.

## PROOF: complete for both claims (limits above and in the body). Recommendation: go, as one PR.

## Commands (NOT RUN; each needs the user's go)

```sh
ssh <the worker host> 'cd ~/d2d-lab/pr/bundle && git push origin fix/bundle-get-hit-render-lut'
# body = the text under the --- line of docs/research/pr-bundle-draft.md, saved to a file first
gh pr create --repo psi29a/d2decomped --base main --head shatteredeagle:fix/bundle-get-hit-render-lut \
  --title "Get-hit frames drawn once at the get-hit rate; drlg-dump opens the folder's patch" \
  --body-file <that file>
# then attach tools/emu/hit_render.py, hit_render_harness.cpp and lut_trace.py to the PR conversation (never game data); log each post in the posted: header above
```
