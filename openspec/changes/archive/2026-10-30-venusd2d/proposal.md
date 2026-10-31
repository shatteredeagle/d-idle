## Why

The user approved a Windows worker (his Windows 11 PC) as an extra d2d worker, unobtrusively. d2d needs Linux, so we must know whether WSL2 or a container runtime is already there before anything is set up.

## What Changes

- read-only inspection of a Windows worker over ssh
- docs/process/d2d-worker-host.md: findings, user steps, dispatch rule, resource limits, undo
