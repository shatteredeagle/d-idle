# Test characters created by agents (provenance log)

> status: current. One line per character an agent created on a lab slot or in a container (PROVENANCE RULE of the private rules). Tags: V verified live.

| date (EST) | character | class | created by | path | writes (setup) | where it lives |
|---|---|---|---|---|---|---|
| 2026-11-07 | Pally-pa1, Pally-pa2, Pally-pa3 | Paladin | paladinsmoke (TASK-142) | `d2bot play --backend d2d --managed --d2d-class 2` on a lab slot (d2d `--start-class 2`, lockstep, no `--lab-verbs`, no `--no-save`) | 0 (V: lab verbs are off, so every setup verb answers `err lab verbs off`; no cheat tool was started) | worker `~/d2d-lab/paladinsmoke/data-pa*/` (save under the instance dir); pa1 and pa2 were stopped within minutes, pa3 ran about 20 wall minutes |
| 2026-11-07 | Palsmoke | Paladin | paladinsmoke (TASK-142) | container image built from the public fork tag, `D2D_CLASS=2` through a hand-written `wizard.json`/`wizard.env` in the data volume (the wizard refuses the Paladin, defect TASK-145) | 0 (V: no lab verbs, no memory write) | docker volume `d2pal-data`, container `d2pal-smoke` (removed at the end of the run) |
