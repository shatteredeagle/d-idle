## Why

internal/gear/data.json and armor.json (170 KB of Blizzard table rows) were committed. THIRD_PARTY.md forbids committing tables and the repo goes public (user decision 2026-10-30): extract them at setup from the user's own MPQs instead.

## What Changes

- cmd/d2tables and gear.ExtractTables: Go port of gen.py / gen_armor.py, writes gear.json, armor.json and a hash stamp into a cache dir (default ~/.cache/d2lod/tables, D2_TABLES_DIR)
- internal/gear loads from the cache at first use (auto-extracts when D2_DATA_DIR is set), error "run d2tables first" otherwise
- the committed JSON, generators and the real-row test excerpt are removed; tests use a made-up fixture; tests asserting real values skip without tables (geartest.Need)
- THIRD_PARTY.md, quickstart, gear.md updated
