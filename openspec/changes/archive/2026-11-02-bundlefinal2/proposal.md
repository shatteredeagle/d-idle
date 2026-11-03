# bundlefinal2

## Why
The maintainer merged upstream PR 176 (batch 9). The earlier bundle was built on 82ea6db4; the owner approved the Smite finding with a credit for its tracer and said to open one upstream PR when ready. This card assembles the final bundle on the newest upstream main and writes its PR text.

## What Changes
- Private fork branch `local/bundle-final` (pushed to the private remote only): the ten commits of `local/bundle-4` plus the Smite commit (`local/smite-block-v1-final`) on upstream main bdc1eb02, one conflict resolved (`repair_cost` in `components/rules/rules.hpp`), CodeRabbit's newest comment folded into the ground-roll test commit, `Traced-by:` on the Smite commit.
- `docs/research/pr-bundle-final-body.md`: the final PR body (Smite component added, Credits regenerated with `tools/credit/credit.py credits`, counts, honest limits).

Exclusive: fork-branch:local/bundle-final. Nothing is pushed to GitHub and nothing is posted.

## Impact
Docs and card headers in this repo; the code lives only on the private fork remote (GPL, never in this repo). No spec, ADR, glossary or decision change.

Overlap (declared): `docs/research/pr-bundle-1-body.md` (archived cards bundlefinal, crfixbundle) is the text this card copies and updates; it is left as it is.
