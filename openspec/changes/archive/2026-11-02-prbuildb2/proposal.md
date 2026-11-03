## Why

Bundle 1 (upstream #177) merged, so the next upstream batch needs its candidates built. The inventory (docs/research/pr-candidates-20261102.md, candidates 6 and 7) names the minmana floor and Teleport D1 to D3 as proven only against a line copy or by the game side alone; the standing rule wants a counted differential on a real d2d build for each change. Two private fork branches on upstream main after #177, kept easy to rebase onto #178, are built here and the evidence is written down.

## What Changes

- Private fork branches local/batch2-minmana and local/batch2-teleport (private remote only, nothing posted to GitHub).
- Two worker-host helpers, tools/emu/pb2_mana.sh and tools/emu/pb2_tele.sh, that run the existing game.exe oracles against base and head builds of the real d2d casts.
- docs/research/pr-batch2-draft.md: bases, heads, counts, parity results, what is left out and why.
