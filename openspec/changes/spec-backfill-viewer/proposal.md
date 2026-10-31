## Why

Card streamfix (2026-10-31) changed the public data of the viewer (/api/pub backend and engine fields) and the stream page (status row, image reconnect) under skip_specs, so no spec states what a stream consumer may rely on. The spec-sme review listed it as behaviour-bearing without a spec.

## What Changes

- New capability `dashboard-pub` (delta ADDED, applied to openspec/specs/dashboard-pub/spec.md at merge): the /api/pub backend and engine fields, the status row text, the image reconnect rules with their timings.
- Specs only; no code is touched. Browser timings are tagged UNVERIFIED.
