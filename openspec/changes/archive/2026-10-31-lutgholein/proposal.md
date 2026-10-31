## Why

LANORACLE (docs/d2d-lan-oracle.md) reproduced upstream's Rogue Encampment 35/35 room match between whisper `town` (original Game.exe host) and d2d `drlg-dump`, but Lut Gholein (level 40) differed in every loaded room ("tile-origin or ordering shift, U"). We need to know whether that is a d2d generator bug (upstream PR candidate) or an artifact of how the two sides are compared.

## What Changes

- investigation doc docs/research/lutgholein-trace.md: cause with evidence, the transform (if any) that makes the room lists equal, PR candidate status
- offline comparison of the recorded capture against d2d's generator under candidate orders; live re-capture on hidden i1 (Navi) to verify
- no code in internal/ or cmd/; no upstream posting
