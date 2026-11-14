#!/bin/sh
# Pictures of the static mockup (design phase). Runs ON the worker host through labrun, from the repo root:
#   labrun <worker> --pull docs/ui/redesign-2026-11-14/mockups/shots -- sh docs/ui/redesign-2026-11-14/mockups/shots.sh
# The mockup file has no doctype of its own (the Artifact page adds one), so a wrapped copy is rendered.
set -u
d=docs/ui/redesign-2026-11-14/mockups
chrome=$(command -v google-chrome-stable || command -v chromium || command -v chromium-browser) || { echo "no Chrome" >&2; exit 2; }
tmp=$(mktemp -d); trap 'rm -rf "${tmp:?}"' EXIT
{ printf '<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"></head><body>'; cat "$d/watch-orders.html"; printf '</body></html>'; } >"$tmp/m.html"
mkdir -p "$d/shots"
for s in "1440 1100" "900 1500" "390 1700"; do
	set -- $s
	timeout 40 nice -n 19 "$chrome" --headless=new --no-sandbox --disable-gpu --hide-scrollbars --no-first-run --user-data-dir="$tmp/p$1" \
		--force-device-scale-factor=1 --window-size="$1,$2" --virtual-time-budget=1500 --screenshot="$d/shots/mockup-$1.png" "file://$tmp/m.html" >/dev/null 2>&1
done
ls -la "$d/shots"
