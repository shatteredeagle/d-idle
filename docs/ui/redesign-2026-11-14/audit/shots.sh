#!/bin/sh
# Audit pictures for docs/ui/redesign-2026-11-14 (design phase, not a test). Runs ON the worker host through labrun,
# from the repo root: the frozen fixture viewer of tools/uishots, headless Chrome on a private Xvfb display at nice 19.
#   labrun <worker> --pull docs/ui/redesign-2026-11-14/audit/shots -- sh docs/ui/redesign-2026-11-14/audit/shots.sh
# Same method as tools/uishots.sh, other widths (1440, 900, 390) and its own port and display so both can run.
set -u
out=docs/ui/redesign-2026-11-14/audit/shots
chrome=$(command -v google-chrome-stable || command -v chromium || command -v chromium-browser) || { echo "no Chrome" >&2; exit 2; }
command -v Xvfb >/dev/null || { echo "no Xvfb" >&2; exit 2; }
port=18947 disp=:247
bin=$(mktemp -d)
trap 'kill $sp $xp 2>/dev/null; rm -rf "${bin:?}"' EXIT INT TERM
go build -o "$bin/uishots" ./tools/uishots || exit 2
timeout 500 nice -n 19 "$bin/uishots" serve -addr 127.0.0.1:$port >"$bin/serve.log" 2>&1 &
sp=$!
timeout 500 Xvfb $disp -screen 0 1920x1200x24 -nolisten tcp >/dev/null 2>&1 &
xp=$!
sleep 2
mkdir -p "$out" && rm -f "$out"/*.png "$out"/*.jpg
shots="watch-1440-fold 1440 900 watch?theme=dark
watch-1440-full 1440 3400 watch?theme=dark
watch-900-fold 900 900 watch?theme=dark
watch-900-full 900 5200 watch?theme=dark
watch-390-fold 390 844 watch?theme=dark
watch-390-full 390 9000 watch?theme=dark
watch-1440-edit 1440 3400 watch?theme=dark&edit=1
inventory-1440 1440 1500 watch/inventory?theme=dark
skills-1440 1440 1200 watch/skills?theme=dark"
echo "$shots" | while read -r name w h q; do
	prof=$(mktemp -d)
	DISPLAY=$disp timeout 40 nice -n 19 "$chrome" --headless=new --no-sandbox --disable-gpu --hide-scrollbars --no-first-run \
		--user-data-dir="$prof" --force-device-scale-factor=1 --force-prefers-reduced-motion --disable-lcd-text \
		--window-size="$w,$h" --virtual-time-budget=8000 --timeout=20000 --screenshot="$out/$name.png" "http://127.0.0.1:$port/$q" >/dev/null 2>&1
	rm -rf "${prof:?}"
done
ls -la "$out"
