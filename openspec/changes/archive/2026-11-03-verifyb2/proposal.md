# verifyb2: independent check of upstream batch 2 (minmana floor, Teleport D1 + D2)

## Why

Batch 2 (docs/research/pr-batch2-draft.md, card prbuildb2) is two fork branches ready for an upstream PR. The PR standard asks for an independent verifier before anything is pushed.

## What Changes

Read game.exe's functions first, build an own model and harness, run game.exe's code natively with own seeds, run the real d2d builds (base vs head), then compare with the author's counts. Check commit hygiene, CI parity, the suites and a throwaway merge with upstream pull/178. Deliverable: docs/research/verify-batch2.md and the harness files tools/emu/vb2_*.

Offline only: no game, no d2d slot, nothing pushed or posted, the fork checkout only read (own clone under the lab dir for builds).
