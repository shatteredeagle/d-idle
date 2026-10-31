## Why

User approved 2026-10-30 (dialog): "Review + doc hygiene". About sixty branches merged on master today (`git log --merges --oneline -60`) with no cross-cutting review: the KILLGAP hooks in the bot loop, the v2 soak gate maths, the d2dview tap (a viewer socket that must stay read-only), the public publish pipeline (index.html and scene.json are public), the slot and capacity shell tools, the Pacer/LateClock timing changes, and two unmerged branches (detloop, replay) the coordinator is about to merge. At the same time docs/plan/STATUS.md's hand-written part still says G2 is running (08:25 this morning) and several docs describe a Wine-era state that ADR 0009 (d2d primary) has superseded.

## What Changes

- docs/research/code-review-2026-10-30.md: ranked findings (file:line, failure scenario, fix), each confirmed by reading the code; clean areas listed; proposed bug-fix cards. No code is changed.
- docs/plan/STATUS.md hand-written part rewritten to today (<= 60 lines); generated block untouched.
- `> status:` banners on superseded docs; docs/BACKLOG.md deduped and ordered; docs/INDEX.md rewritten as a one-page map.
- docs/research/proposed-glossary-terms.md for the user's grill session (GLOSSARY.md, docs/adr/*, docs/decisions.md are not touched).
