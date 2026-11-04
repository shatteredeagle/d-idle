## Why

The independent verifier confirmed the PALSKILL D2/D3 candidate with caveats, and upstream main moved (PRs 190 and 191). The candidate needs a third local rebase with the caveats applied and a body without references upstream cannot repeat.

## What Changes

- Local only: local/pal-d2d3-r3 in the private fork checkout (cherry-pick onto the new upstream main, unused include removed, a comment on the magic-only bonus and the double roll), checks and oracle re-run on the worker host, and the body docs/research/pr-palskill-body-r3.md. Nothing is pushed to GitHub or posted.

## Impact

Docs only.
