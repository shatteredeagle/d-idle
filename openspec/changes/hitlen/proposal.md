## Why

DECOMPREC (docs/research/movement-and-recovery-trace.md section 3) and HITREC (docs/research/pr-get-hit-rule-proof.md, PR #134) left a second, separate claim open: the LENGTH of the hit recovery. game.exe 1.14d's get-hit animation rate is AnimData speed x (50 + effective FHR) / 100 (the Sorceress 15 / 14 / 13 ... / 5 FHR table), while d2d uses 100 + FHR for the player (`speed_frames`) and the plain AnimData length for monsters, so d2d's staggers are about half as long. The user's PR standard (memory d2-pr-proof-standard, docs/process/upstream-pr-standard.md) needs a differential against game.exe's own code with zero mismatches after the patch, the U items (server-side monster mode end, record +0x64, frozen / stunned exemptions) closed or the claim narrowed around them, and an independent adversarial check before anything is pushed.

## What Changes

- The GH length chain is traced in game.exe (the hit-recovery branch of FUN_00623f50, the animation length formula, the mode end test for players and monsters) and tagged V / S / U.
- `tools/emu/hit_len.py` (oracle: game.exe's code under unicorn) gives the length in ticks per (unit class / monster, AnimData speed and frames, FHR, speed modifiers) over >= 100,000 cases; before / after mismatch counts against d2d's own code.
- A minimal patch on a fresh clone of the user's fork (`~/d2d-lab/pr/hitlen`), upstream tests, pr_guard; nothing pushed, posted or filed.
- Three docs: proof, draft PR body, evidence pack with PROOF complete / incomplete and the push commands (not run).
