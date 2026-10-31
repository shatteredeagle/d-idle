## Why

User rule 2026-10-30 (memory d2-pr-proof-standard): every upstream PR of ours must carry verifiable proof that it moves d2d toward 1:1 with game.exe 1.14d. The pending candidate fix/missile-damage-fine (branch on the worker host, draft docs/research/pr-missile-damage-fine-draft.md) rested on a 3000-case roll oracle plus noisy casts-per-kill statistics, and its `life_frac` model was never shown equivalent to game.exe's 256ths life for hit/heal sequences. This job writes the standard and applies it to that candidate before the user is asked for a go.

## What Changes

- docs/process/upstream-pr-standard.md: the bar (differential proof, V/S/U chain, statistics as corroboration only, one claim, adversarial verification, limits, checklist, evidence pack, PR spacing and disclosure).
- An end-to-end differential (game.exe FUN_0057c6c0 chain in the emulator vs the real d2d `missile_blow` + `hurt` + `heal`) on a new branch fix/missile-damage-fine-v2 in the worker host clone, improving the patch until mismatches are zero.
- docs/research/pr-missile-damage-fine-proof.md, -draft-v2.md (their template), -evidence-pack.md (verdict and recommendation for the user). Nothing pushed or posted.
