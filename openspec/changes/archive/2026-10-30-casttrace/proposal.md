## Why

PRSCAN found that upstream (psi29a/d2decomped) treats game.exe 1.14d as the ground truth and other sources (D2MOO, community tables) as confirmation only. Our cast PR admitted "nothing traced in game.exe" and bundled three claims. Trace the cast timing in game.exe, run the oracle, and split the work into one claim per branch, ready for the user to review. Nothing is posted.

## What Changes

- trace the cast rate, the end test and the sequence path in game.exe 1.14d (addresses and explanations, in our words)
- three upstream-shaped branches on the worker host (`~/d2d-lab/pr/{cast-ticks,lightning-sequence,skills-doc-cast}`), PR descriptions and the unrun `gh pr create` commands
- record the trace and the conflict inventory in docs/research/cast-trace-gameexe.md
