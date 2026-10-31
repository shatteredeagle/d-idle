## Why

Code review 2026-10-30 section 5 found gaps in the public page generator: the sanitizer lets real-person names, home paths, odd boundaries and token shapes through, a failed ssh fetch keeps showing "Live now" from the cache, and the leak test greps a few substrings on one fixture.

## What Changes

- sanitize.py: names (the maintainer, a housemate, home-path forms), letter-only boundaries, UUID/base64url/22-char/Basic/octal-hex IP rules. Character names stay public by user decision (no name allow-list).
- runs.py, scene.py, pages.py: stale cache is honest (running false after the liveness window, "last update N min ago" shown); junk ssh replies count as a failed fetch; cache chmod 600 and never under the output dir; host regex rejects a leading dash.
- live.py, publish.py: no redirects with the Bearer header; JSON Infinity fails closed; stats without ts are not live.
- tests: sanitize.scan_file over generated index.html, scene.html, scene.json for live, between-runs, stale-cache, d2d and original fixtures.
