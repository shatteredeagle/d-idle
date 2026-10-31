## Why

Work package B of docs/process/public-split.md: the public tree must not hardcode a private host, URL, domain or home path. Hosts come from the role variables of tools/env.sh (D2_WORKER_HOST, D2_WORKER_USER, D2_NAS_HOST, D2_BOARD_URL, D2_FORGE_URL, D2_COLD_DIR), real values live in the gitignored local.env.

## What Changes

- Shell scripts source tools/env.sh and build ssh targets from the role variables; `labrun <worker>` uses the first label of D2_WORKER_HOST.
- Go code and tests: no real hostnames or IPs in fixtures (worker.example, 192.0.2.x), no /home/<user> literals, D2_COLD_DIR for the legacy backup folder, worker/NAS host words kept out of the public d2dview feed via env.
- Comments say "the worker", "the NAS", "the head box" instead of machine names.
- leakcheck: game phrases (Forgotten Tower, Jade Figurine) are not machine names; allow-list entries for public placeholders and Authorization header handling.
- Docs are package C and are not touched, except one-word renames that code needs (d2dfarm node type lab-slots).
