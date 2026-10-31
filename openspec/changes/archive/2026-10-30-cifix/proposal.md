## Why

ci.yml has run about 390 times on the worker host runner: 266 failed, 120 cancelled, 1 success, always at the Scripts step, on every branch push. It loads the worker host for nothing.

## What Changes

- find and fix the failing step (environment assumptions, not weaker checks)
- run on push to master, pull_request, workflow_dispatch, tags; concurrency cancel; light `checks` job always, heavy `test` job niced and only for Go changes
- update docs/ci.md
