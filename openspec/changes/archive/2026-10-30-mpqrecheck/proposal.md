## Why

VERIFY-0003 found that our `mpq-cat` silently serves the old 1.10 tables when the 1.14d patch archive is named `Patch_D2.mpq` (it looks for lowercase `patch_d2.mpq`). Two prepared upstream PRs (price, cast) and several docs read data with it, so every such read needs a recheck and the trap needs a written fix.

## What Changes

- Recheck the PR data claims against the patched 1.14d tables (read-only, scratch dir on the worker host).
- New `docs/process/mpq-data-reading.md` (recipe, symptom, one-line sanity check); a short "Data reading trap" note in `docs/d2d-lab.md`.
