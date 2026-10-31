## Why

The point of the d2d backend is fast replicated soaks, but the whole bot under lockstep ran at about 0.5 virtual seconds per wall second (D2DPACE): every devctl verb costs one d2d frame and a snapshot is eight verbs.

## What Changes

- profile verbs per tick and wall ms per verb (trace in d2dctl, `D2DCTL_TRACE`)
- pipelined snapshot read (`Client.Prefetch`), one write per observation
- fork branch `lab4`: lockstep idle frames no longer wait 16 ms and no longer draw every frame
- inventory flag in d2dsrc (ui.rects, key hint on lab2), d2d.out rotated per start, channel diagnosis
