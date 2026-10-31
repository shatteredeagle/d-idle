# Cast timing traced in game.exe 1.14d (upstream PR prep)

> status: research, 2026-10-30, agent CASTTRACE. Nothing was posted, pushed or commented on GitHub. Supersedes the "not traced" caveat in `docs/d2d-upstream-cast.md` (card d2dcast). Tags: V = read in game.exe or run in the unicorn oracle by this agent; D = data read from the patched 1.14d MPQs; U = unverified.

Why: upstream (psi29a/d2decomped, AGENTS.md section 5) takes game.exe 1.14d as the ground truth; D2MOO and community tables can only confirm a reading. Our first cast PR (branch `fix/cast-timing`, three claims, no trace) was not postable (docs/research/upstream-pr-scan.md). This note is the trace behind the three replacement branches.

## How it was checked

- game.exe: the 1.14d `Game.exe` of our install (md5 2b3600c1..., read-only copy under /tmp, never run). Disassembly with upstream's `tools/emu/disasm.py` (capstone + pefile + unicorn in a venv under /tmp; `uv` is not installed on the worker host, pip in a venv did the same).
- Oracle: `tools/emu/cast_rates.py` (new, in the cast-ticks branch) runs game.exe's own code under unicorn: the cast branch of `FUN_00623f50` (0x623fcb..0x624012, with the stat getter `FUN_00625500` replaced by the FCR) and the end test and step (`FUN_006217c0`, `FUN_00623e00`) in the order the client tick calls them. d2d's `test_combat --cast-rates` prints the same lines; the diff is empty (2,807 lines on the plain branch, 1,203 on the sequence branch).
- Data: Skills.txt and animdata.d2 from the PATCHED stack (d2data + d2exp + `patch_d2.mpq` symlinked in lowercase; without it Skills.txt has no `seqtrans`, so the trap in mpq-data-reading.md is real). Rows read with `mpq-cat`.
- Upstream rules re-checked at 507e3c06: `python3 tools/pr_guard.py` (0 FAIL, 0 WARN on each branch: every address is inside .text 0x401000..0x6cb5a1) and `python3 tools/presubmit.py --quick`.

## What game.exe does (addresses; explanations in our words)

| Address | What it does | Tag |
|---|---|---|
| `00623f50` | sets a unit's per-tick animation rate (+0x4c, and +0x3c for casts). First test: `006216e0` says the mode is a "cast" mode; if so, the cast branch (below), else the attack, get-hit, block, walk branches the maintainer's char-select.md already lists | V |
| `006216e0` | cast-mode test: a player's SC (mode 10); a player's SQ (0x12) whose skill record byte +0x12 (seqtrans) equals 0xa (SC) (the record comes from `00620250` -> `00643ce0` -> `0045c4b0`); a monster's mode 7 (SC) | V |
| `00623fcb`..`00624012` | cast branch: `rate = min(100 + FUN_00621740(unit, 2), 175)`; `speed = floor(AnimData record speed (+0xc) * rate / 100)`, clamped 0..0x7fff; stored at +0x4c and +0x3c | V |
| `00621740` | stat to effective value through the 12-byte rows at `006e8e24`: row 2 (`006e8e3c`) = stat 105 (faster cast rate), flag 1 (diminishing), factor 120; it returns `factor * v / (factor + v)` (signed integer divide) when v is not 0. Rows: 0 = IAS (93, 120), 1 = FHR (99, 120), 2 = FCR (105, 120), 3 = FBR (102, 120), 4 = FRW (96, 150). FCR 200 gives effective 75, so rate 175 and speed 448: more FCR changes nothing | V |
| `00624390` | sets up the animation state when a mode starts (called from the mode setter `00624690`). Current position (+0x44) = start frame x 256 (`00623b10` returns 0 for BL, SC, TH, KK, S1 and S2; A1, A2, S3, S4 use a table). SQ (0x12): `00621260`, then `00623f50`. Other player modes: +0x30 = 0, `00623f50`, then +0x48 = AnimData frame count (+8) x 256 (store at `00624473`) | V |
| `00621260` | for SQ: +0x30 = the sequence (`006633b0`), +0x34 = its length, +0x38 = 0, +0x3c = 0x100 (the cast branch then overwrites it with the speed), +0x48 = the sequence record's second count (`006633f0`) x 256 (store at `0062129a`). The record is found by `00663310` (the maintainer's sequences.hpp: table `007483b8` + weapon index x 12) | V |
| `006217c0` | "is the animation over?": with +0x30 empty (plain): `+0x44 + speed(+0x4c) >= +0x48`; with a sequence: `+0x48 <= 0` | V |
| `00623e00` | one step: plain: `+0x44 += speed`; sequence: position `+0x38 += +0x3c`, countdown `+0x48 -= +0x3c`. It also records an event byte (+0x4e) from the AnimData flags / the sequence | V |
| `00463390` | the client's player tick. A mode of class 1 (jump table `00463710`; the class column is at `00711e04`, 12-byte rows by mode) runs `006217c0`; if not over, `00623e00`; and then (row value 3) `006217c0` again: yes ends the mode in the same tick | V |

So the "one tick early" of plain casts is: the over-test looks one step ahead (`current + speed >= end`) while a sequence's test is on the countdown after the step. With start 0, speed s, N = frames x 256, a plain mode lasts the smallest k >= 1 with (k + 1) s >= N, i.e. `ceil(N / s) - 1`; a sequence lasts the smallest k with N - k s <= 0, i.e. `ceil(N / s)`. Whether the first of those ticks is the tick after the mode was set is the same for both, so the plain-versus-sequence difference (14 frames: 13 ticks; 19 steps: 19 ticks at 0 FCR) holds whatever the tick count's origin.

## Claims: confirmed or contradicted

| Claim of the first PR | Binary | Result |
|---|---|---|
| EFCR = floor(120 FCR / (120 + FCR)), speed = floor(AnimData speed (100 + EFCR) / 100) | `00621740`, `00623fe6..00623ff3` | CONFIRMED, plus a cap the PR missed: rate at most 175 (`00623fda`: cmp 0xaf). Effective at FCR >= 200 only; our old code would have kept speeding up |
| a plain cast ends a tick early: `ceil(256 f / s) - 1` | `006217c0` + `00463390` order | CONFIRMED (oracle: Sorc 13/12/11/10/9/8/7 at 0/9/20/37/63/105/200; Amazon 19/18; Assassin 16/15..9; Druid 18..10) |
| a sequence has no minus one; Lightning 19 ticks at 0 FCR | `006217c0`, `00621260` | CONFIRMED (oracle: 19, 18 from 7, ... 11 from 194) |
| the FCR rate only applies to a sequence whose seqtrans is SC | `006216e0` byte +0x12 == 0xa | CONFIRMED (read from the record, not from `seqnum == 12`; the new code reads `seqtrans` from Skills.txt) |
| Inferno / Arctic Blast are not sped up by FCR | same test: seqtrans SQ -> not a cast mode | CONFIRMED for the cast branch; what rate they do get is U (other branches of `00623f50`) |
| Druid SC speed 208 | animdata.d2 DZSCHTH = 15 frames, speed 208 (D); `00623fe6` multiplies the record's speed | CONFIRMED |
| Charged Bolt / Nova / Thunder Storm are plain `cast` (SC) in Skills.txt, the maintainer's "S1" is wrong | Skills.txt patched (D): anim SC, seqtrans SC | CONFIRMED (data, not code) |
| SQ plays at speed 256 | no `SOSQ*` record in animdata.d2 (D); the maintainer's note says a missing record falls back to speed 256 | CONFIRMED by data + their note; the fallback code itself not re-read (`0066a8f0`) |
| missile leaves at the sequence's event-1 step spread over the cast | not traced (`00621210` / `00663410` look up the event by position) | U: dropped from the PR, `ponytail:` note |
| Nothing contradicted: no claim had to be corrected except the missing 175 cap (our `cast_ticks` had none) and the hard-coded `seqnum == 12` |  |  |

Not traced (listed in the PRs): the code that turns a skill's `anim` (record +0x10) into the player's mode; a monster's cast path (client or server); negative FCR (the formula runs on negative values, d2d keeps them unchanged; FCR is never negative in play); the server-side callers of the end test (we used the client tick `00463390`; `004b12d6`, `005a7372` etc. also call `006217c0`).

## D2MOO `Units.cpp` (MIT), where it matches or differs

The first PR cited it as: table `0x006E8E24`, cap 175, `speed = rate x AnimData speed / 100`, `0x006216E0` for the SQ-with-seqtrans-SC rule, `0x006217C0` for the two end tests. Every one of those statements matches what the binary does above (same addresses, same behaviour). It was not re-fetched in this run (no network step was needed: the binary settles each point); it stays a "confirms", never a source, in the PR text. No difference found.

## The three branches (the worker host, `~/d2d-lab/pr/`, worktrees of `~/d2d-lab/src`, base upstream 507e3c06)

Author and committer `shatteredeagle <26069301+shatteredeagle@users.noreply.github.com>`, no Co-Authored-By. Each has `PR_DESCRIPTION.md` (template, `Agent: Claude Code (Claude Sonnet 5.5), working for the maintainer`; untracked on purpose). Built with `nice -n 10 ninja -j4`, `-DD2D_WERROR=OFF`, own `build/` per branch; `test_combat` and `test_game` pass (test_game SKIPs its data checks: no d2data.mpq in the d2d default folder).

| Branch | Commit | One claim | Test / oracle |
|---|---|---|---|
| `fix/cast-ticks` | 1edc5f20 | cast speed (cap 175, AnimData speed) and a plain cast's `ceil - 1` ticks; replaces `speed_frames` for SC | `test_combat` asserts + `cast_rates.py` vs `test_combat --cast-rates` (empty diff) |
| `fix/lightning-sequence` | 2036fd3f | Lightning / Chain Lightning (anim SQ, seqtrans SC) take their 19 steps at the cast rate; `seqtrans` loaded from Skills.txt | `test_combat` (19 steps, event 1 on the 8th, for all 14 weapon classes; ticks 19..11) + the same oracle with the sequence lines (empty diff) |
| `fix/skills-doc-cast` | e310a1c9 | docs only: Charged Bolt / Nova / Thunder Storm are SC, the Druid row at speed 208 | pr_guard; evidence is data + `00623fe6` |

`pr_guard.py`: 0 FAIL, 0 WARN on all three. `presubmit.py --quick`: only WARNs (ruff not installed; `build` skipped by `--quick`; doc-claims "every" on the doc branch, answered in its PR text; "siblings" on lightning, answered). The old branch `fix/cast-timing` (commit 17100b7, with the trailer) stays in `~/d2d-lab/pr/cast` as history only; do not post it.

The two code branches both add the identical `cast_speed` and `tools/emu/cast_rates.py` hunks and touch the same `else if` in `Fight::set_pmode`: whichever merges second needs a small rebase (the identical hunks drop out). The doc branch is independent of both (it edits the Druid row and the S1 sentence; the cast-ticks branch edits the neighbouring "Not traced" paragraph and the Port line, so the second one may need a trivial merge in skills.md).

### `gh pr create` (NOT run; each needs the user's go and a push of the branch to the user's fork first)

```sh
cd ~/d2d-lab/pr/cast-ticks         && gh pr create --repo psi29a/d2decomped --base main --head shatteredeagle:fix/cast-ticks --title "Cast speed and length from game.exe: the 175 cap, a tick less per plain cast" --body-file PR_DESCRIPTION.md
cd ~/d2d-lab/pr/lightning-sequence && gh pr create --repo psi29a/d2decomped --base main --head shatteredeagle:fix/lightning-sequence --title "Lightning and Chain Lightning play their 19-step sequence at the cast rate" --body-file PR_DESCRIPTION.md
cd ~/d2d-lab/pr/skills-doc-cast    && gh pr create --repo psi29a/d2decomped --base main --head shatteredeagle:fix/skills-doc-cast --title "skills.md cast rate: Charged Bolt, Nova and Thunder Storm are SC; Druid speed 208" --body-file PR_DESCRIPTION.md
```

(`shatteredeagle` as the fork owner is an assumption from the commit identity; adjust `--head`. Per AGENTS.md the agent asks the human before every push, issue and PR.)

## Open upstream PRs that could conflict (read-only `gh pr view`, 2026-10-30)

| PR | Files near ours | Risk |
|---|---|---|
| #125 (maintainer, Throw level) | `components/game/fight.cpp` (hunks near `need_mana` ~685 and `cast_missile` ~2637), `fight.hpp` (~366, ~510-533), `docs/research/re/skills.md` (~540) | low: none touches `set_pmode` (~455) or our skills.md regions (~1100); fight.hpp comment near `set_pmode` is 3 lines |
| #120 (Codex, Andariel/Diablo shake) | `tools/emu/README.md` (adds a table row at line ~32), `tools/README.md`, `tests/README.md`, new `shake.py` | README table row: adjacent-line conflict with our `cast_rates.py` row, trivial |
| #119 (Codex, fork PR #6 port) | `apps/d2d/*`, `tests/test_game.cpp`, `tools/mpq-cat/mpq-cat.cpp`, objects.md | none on cast/fight; `test_game.cpp` is not touched by our branches |

## What the maintainer might object to

- The PRs add a Python oracle (`cast_rates.py`) that mirrors `walk_rates.py`; it runs game.exe's tick functions with stubbed unit memory and two hooks (`00625500`, `00621210`), not the whole mode setter. He may ask for a run on his own copy (that is the intended re-check).
- The tick count is the client tick's (`00463390`); the server-side tick path was not read. The plain-versus-sequence offset does not depend on it.
- "Ticks a cast lasts" is counted from the first tick after the mode starts, through the tick in which the end test says yes; the in-game period measured earlier (mean mana-drop period 13.0 / 19.0 on d2d) is d2d's, not the original's. No original-game measurement exists.
- The Druid row is rewritten in the file's old `frames_shown` unit (ticks + 1) to stay consistent with the other rows; he may prefer ticks.
- `seqtrans` is a new `Skill` field read from Skills.txt; Inferno and Arctic Blast keep the old timing under a `ponytail:` (the rate branch they use is not traced).
- Lightning's missile still leaves at SC's action frame (ponytail), so the visible effect is the cast length, not the release moment.
