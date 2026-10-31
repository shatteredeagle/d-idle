## Why

Under d2d lockstep the bot's time sources (minute ticker, no-progress detector, ladder timers, heartbeat) must all agree on virtual time, and the campaign never leaves clvl 1.

## What Changes

- audit and unify time sources on d2d via env.Clock / Pacer
- fix minute-line kill/death counters on d2d
- live sessions on the private fork, findings in docs/d2d-pace.md
