# bundlefinal

## Why

The independent verifier (card verifybundle) answered READY WITH CHANGES for bundle 1. Three text and process changes are needed before anything is pushed: commit bodies of at most three lines, credit blocks on the bundle's cards, and a PR body with counted differentials and honest limits.

## What Changes

- docs/research/pr-bundle-1-body.md: the final PR body (title, summary, one section per component with counted differentials, limits, Credits section from `tools/credit/credit.py credits`, checks, `Agent: Claude Code`).
- `credit:` blocks on the archived cards of the bundle's components and verifiers (edited in place).
- Private fork (not in this repo): branches `local/bundle-2` (upstream main + the 10 commits, messages shortened) and `stage/bundle-2` (the same on the fork's base for a staging PR), pushed to the private remote only.

## Impact

Docs and card headers only. Nothing pushed to GitHub, no PR opened.
