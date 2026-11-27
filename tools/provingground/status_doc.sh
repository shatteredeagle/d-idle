#!/bin/bash
# Regenerates docs/process/proving-ground-status.md (and optionally the board's /pg page data) from the results ledger.
# Called by the nightly job on the board host, where the ledger lives (docs/process/pg-ledger-ops.md); never edit the doc by hand.
#   status_doc.sh LEDGER.jsonl [PAGE.json]
# Env: PG_RELEASES (releases JSON array, optional), PG_STATUS_BRANCH (default main), PG_RELEASE_COMMIT (release view in
# PAGE.json), D2SCENARIO (a built d2scenario; default `go run ./cmd/d2scenario`).
set -euo pipefail
here=$(cd "$(dirname "$0")" && pwd); repo=$(cd "$here/../.." && pwd); cd "$repo"
[ $# -ge 1 ] || { echo "usage: status_doc.sh LEDGER.jsonl [PAGE.json]" >&2; exit 2; }
ledger=$1; page=${2:-}
[ -f "$ledger" ] || { echo "status_doc: no ledger at $ledger" >&2; exit 2; }
if [ -n "${D2SCENARIO:-}" ]; then run=("$D2SCENARIO"); else run=(go run ./cmd/d2scenario); fi
args=(-in "$ledger" -branch "${PG_STATUS_BRANCH-main}" -tiers tools/provingground/scenarios/tiers.json -releases "${PG_RELEASES:-}")
"${run[@]}" ledger check -in "$ledger" >/dev/null
"${run[@]}" ledger status-doc "${args[@]}" -out docs/process/proving-ground-status.md
[ -z "$page" ] || "${run[@]}" ledger page-json "${args[@]}" -commit "${PG_RELEASE_COMMIT:-}" -out "$page"
echo "status_doc: docs/process/proving-ground-status.md${page:+ and $page} regenerated"
