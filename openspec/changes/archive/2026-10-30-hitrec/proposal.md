## Why

DECOMPMON and DECOMPREC found that d2d decides a flinch (get-hit) at a fixed twelfth of max life for the player and an eighth for a monster, while game.exe 1.14d's FUN_0057cb00 divides by a hit-class divisor and rolls 37.5 % / 75 % / sure bands (docs/research/monster-attack-trace.md, movement-and-recovery-trace.md). The user's PR standard (memory d2-pr-proof-standard, 2026-10-30) needs a differential against game.exe's own code with zero mismatches after the patch, the open U items closed or the claim narrowed around them, unchanged behaviour elsewhere, and an independent adversarial check before anything is pushed.

## What Changes

- The U items (record +0x28, the hit class source per damage source, the callers) are traced in game.exe (addresses only) and closed; the monster mode-end stays U and the length claim is kept out of this PR.
- `tools/emu/hit_rule.py` (oracle: game.exe's FUN_0057cee0 under unicorn) and `tools/emu/hit_rule_harness.cpp` (d2d's old rule) give the before / after mismatch counts over 100,000+ cases.
- A patch on a fresh clone of the user's fork (`~/d2d-lab/pr/hitrec`, branch `fix/get-hit-rule`), with upstream tests, pr_guard and presubmit run; nothing pushed.
- Three docs: proof, draft PR body, evidence pack with PROOF complete / incomplete and the push commands (not run).
