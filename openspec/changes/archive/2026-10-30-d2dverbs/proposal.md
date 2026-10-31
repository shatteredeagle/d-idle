## Why

The town agents asked for fork verbs (docs/d2d-town.md, "Fork verb requests"): pack-to-belt moves, interacting with objects by id, panel rectangles so the bot stops guessing pixels, and waypoint activation. Some already exist upstream under other names; the rest go into a new fork branch `lab3`.

## What Changes

- build `lab3` (= `lab2` plus new patches) in the private fork, binary `bin/d2d.lab3`
- document the verbs for our side (docs/d2d-lab.md section lab3, docs/d2d-fork.md verb reference)
- nothing of the fork code enters this repo (GPL separation)
