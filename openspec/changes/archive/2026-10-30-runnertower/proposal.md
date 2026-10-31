## Why

Forge CI had one Linux runner (the worker host, busy with soaks). The user approved a default Linux runner on the NAS as a UI-visible NAS container (host executor, no docker.sock).

## What Changes

- container `forge-runner` on the NAS from image `d2-forge-runner:13.2.0-go1.27.3`, user-level runner `nas` for the maintainer
- Dockerfile, config.yml and the NAS template under tools/ci/runner/, rollback in docs/process/ci-runners-plan.md
