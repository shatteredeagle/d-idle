## Why

Container work on a d2d image restarts under ADR 0009: the d2d container is the v1 product. Steps 1 to 4 of docs/d2d-container/risks.md (vendor bundle, first g++ build on the worker host, optional Sentry patch, runtime image with notices) give a smoke-tested image; no image leaves the NAS until the GPL question (ADR 0003, grill 0002) is decided.

## What Changes

- vendor bundle script and context assembler (risks step 1)
- first d2d stage build on the worker host, g++ 15 or 16 (step 2)
- optional fork patch: Sentry optional, launcher skipped (step 3; fork work pushed to <owner>/d2d-fork the same day)
- runtime image: notices, SOURCE.txt, non-root, tini, healthcheck; smoke with MPQs (step 4)
