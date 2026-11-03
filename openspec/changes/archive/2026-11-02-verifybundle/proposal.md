# verifybundle

## Why

The maintainer asked for all our upstream work in one PR under the review limits. Bundle 1 (`local/bundle-1`, ed087f88 on upstream main 48908176) joins five components; the ground component (standing footprint, one burst roll) had no independent verification, and nobody had checked the five together on the bundle head.

## What Changes

- docs/research/pr-bundle-1-verifier.md: own game.exe oracle and own d2d probe for the ground claims (base vs head counts), integration on the bundle head, CI parity, commit hygiene and credit, the PR body against the counts, size against the budget, merge with current upstream main, a verdict with exact changes.
- docs/research/pr-bundle-1-verifier/: the harness scripts and the raw count tables.

## Impact

Docs only. No code, spec, ADR or glossary change; nothing pushed or posted.
