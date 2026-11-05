# Det state hashes and per-commit checks for the combined Frozen Armor and freeze-resume PR (BUNDLEDET)

> status: archived 2026-11-05 (cleanup B): finished or superseded, kept for the record. Old path: `docs/research/pr-bundle-armor-resume-det.md` (now a stub). Earlier status: DONE, 2026-11-01 (card bundledet). Result: 14 of 14 DETHASH2 scenarios identical for the resume claim and for both claims together. Nothing pushed or posted. PR body: [pr-bundle-armor-resume-body.md](../../research/pr-bundle-armor-resume-body.md) (owned by the bundlearmor card; see the overlap in the card).

## Squash

On the worker host, PR checkout `bundlear`, new branch `fix/frozen-armor-and-resume-2c` from upstream main 72132b99; the four-commit branch `fix/frozen-armor-and-resume` is untouched.

| commit | content | built from |
|---|---|---|
| `f9e38b15` | armor claim plus armor tooling | 6b5f0267 + a2a16a87 |
| `9a4e726a` | resume claim plus resume tooling | 0ab7b5e7 + a4ad2099 |

Both keep the numeric-ID noreply author and committer, a body of three lines, `Agent: Claude Code`, no Claude trailer. `git diff --stat fix/frozen-armor-and-resume fix/frozen-armor-and-resume-2c` is empty (the trees are identical). `tools/pr_guard.py` 0 FAIL / 0 WARN against 72132b99 for commit 1 and for the range, and against the first commit for commit 2.

Per commit, each in its own detached worktree, Release build with `-Wshadow -Wunused -Wsign-compare`, d2d side against the stored game.exe side of the armor seed 1 (0x41524d48) and the resume seed 1 (0xF2E50001):

| commit | build | armor test (108,506 cases) | resume test |
|---|---|---|---|
| `f9e38b15` | `test_armor_hit` builds | mismatches 0, calc1 differs 0, RESFAIL 0 | not present |
| `9a4e726a` | `test_armor_hit` and `test_freeze_resume` build | mismatches 0, calc1 differs 0, RESFAIL 0 | frozen in game.exe: 12,548 cases, 0 differ |

The not-frozen rows of the resume compare (specials, 25,228 cases) differ as before: they are the specials residual of the body, not claimed.

## Det state hashes

Method as in [pr-armorhit-verifier.md](../../research/pr-armorhit-verifier.md) section 5: the DETHASH2 lab tree (upstream 72132b99 plus the lab commits, 5e0a51bc) as base, and the same tree plus the claim, one hidden slot (`d2dslot.sh run`, under `agentrun.sh bundledet`), capacity checked first (idle 49 percent plus nice 3 percent), the DETHASH2 driver unchanged (`det_scenarios_run.py run`), a fresh d2d per run, seed 3, lockstep. Tags: `base` (sha256 70c7b3694173, the binary of the earlier runs), `base2` (rerun of the base binary, the repeat check), `resume` (lab tree plus 0ab7b5e7, 9878f19ee171), `both` (lab tree plus 6b5f0267 and 0ab7b5e7, ce3e4932d177; the cherry-picks merged cleanly onto the lab tree). The tooling files add no d2d code and are not in the det trees. 56 runs.

Result: **all 14 scenarios identical** between base, base2, resume and both, in the monster hash and in the state hash. The base column reproduces the earlier DETHASH2 base values.

| scenario | mon | state |
|---|---|---|
| passive | d1ca6c463f45 | 84942b4a0e26 |
| fight | 5f90630302ee | 70fa7405daed |
| cold | bba8362ccde9 | c8197bb82c7c |
| static | 34a1b9ef9f44 | cad5671002e0 |
| cold2 | bba8362ccde9 | c8197bb82c7c |
| hit_plain | 81b48ba4bdb0 | d75c22798784 |
| hit_champion | 9b7c25f25f6a | 26c01e11198b |
| hit_unique | 04f91fcf36ff | 865168e8ee37 |
| hit_boss | d49d11125ccc | 0bcc4196a10e |
| hit_immune | 2cf37f51eea2 | 65c6bbceea5d |
| hit_minion | 27714ca3de68 | 598e277f04e4 |
| merc | 1ecd7abc1cd7 | f993d75658e4 |
| pet | 99d912dcc2e9 | ac7ac7a0a49f |
| follow | 9f51307f5f85 | a583fd8df5d8 |

No scenario differs, so there is no freeze scene list to give. Coverage limit: no scenario wears Frozen Armor. The hit scenes show stun deadlines (get-hit and chill ones, for example a plain target at stun=10240) and the hashes do not move, but these snapshots do not show a freeze thawing, so the freeze and the thaw stay covered by the differentials of the body, not by these hashes. What the hashes do show: neither claim adds an RNG draw or moves any unit in the 14 scenes.

Logs on the worker host: `~/d2d-lab/pr/bundledet-out/det/results.txt` and the per-run `det8_*` files in that directory.

## Cleanup

`agentreap.sh bundledet` found nothing registered and `agentreap.sh --list` showed none of ours; the two per-commit worktrees (`bundledet-c1`, `bundledet-c2`) were removed; the det trees `dethash2/resume` and `dethash2/both` and their binaries stay for a rerun.

## Shared-file lines for the merger

- docs/research/pr-bundle-queue.md (not edited here), append: `- BUNDLEDET (2026-11-01, docs/research/pr-bundle-armor-resume-det.md): the combined PR is two commits, fix/frozen-armor-and-resume-2c (f9e38b15 armor, 9a4e726a resume), tree identical to the four-commit branch; det state hashes 14 of 14 identical for resume and for both claims; body updated, still READY, held for the go`
