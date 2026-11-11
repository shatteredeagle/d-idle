#!/bin/sh
# Watch UI screenshot suite (docs/ui-regression-plan.md section 4). Runs ON the worker host (through labrun):
# a frozen fixture viewer (go run ./tools/uishots serve), headless google-chrome-stable on a private Xvfb display at
# nice 19 (never the user's desktop), one PNG and one DOM dump per page, width and theme, then the DOM contract check
# and a pixel diff against the golden set kept on the worker host (not in git: pictures change with every UI PR).
#   tools/uishots.sh [--update] [--out DIR] [--only NAME]   --update makes this run the golden set
# Exit: 0 ok, 1 a check failed, 2 a tool is missing. Every process has a hard limit; one run per port at a time (flock).
# UISHOTS_PORT and UISHOTS_DISPLAY pick another port and private display; UISHOTS_GOLDEN another golden directory.
set -u
update=0 out=${TMPDIR:-/tmp}/uishots-$(id -u)/run only=
while [ $# -gt 0 ]; do
	case $1 in
	--update) update=1; shift ;;
	--out) out=$2; shift 2 ;;
	--only) only=$2; shift 2 ;;
	*) echo "uishots: unknown argument $1" >&2; exit 2 ;;
	esac
done
golden=${UISHOTS_GOLDEN:-$HOME/.cache/didle-uishots/golden}
chrome=$(command -v google-chrome-stable || command -v chromium || command -v chromium-browser) || { echo "uishots: no Chrome or Chromium on this host" >&2; exit 2; }
command -v Xvfb >/dev/null || { echo "uishots: no Xvfb on this host" >&2; exit 2; }
port=${UISHOTS_PORT:-18941} disp=${UISHOTS_DISPLAY:-:241}
exec 9>/tmp/uishots-$(id -u)-$port.lock
flock -w 600 9 || { echo "uishots: another run holds the lock" >&2; exit 2; }
cd "$(dirname "$0")/.." || exit 2
bin=$(mktemp -d)
trap 'kill $sp $xp 2>/dev/null; rm -rf "${bin:?}"' EXIT INT TERM
go build -o "$bin/uishots" ./tools/uishots || exit 2
timeout 600 nice -n 19 "$bin/uishots" serve -addr 127.0.0.1:$port >"$bin/serve.log" 2>&1 &
sp=$!
timeout 600 Xvfb $disp -screen 0 1920x1200x24 -nolisten tcp >/dev/null 2>&1 &
xp=$!
sleep 2
mkdir -p "$out" && rm -f "$out"/*.png "$out"/*.html
# name width height path?query
shots="watch-390-dark 390 844 watch?theme=dark
watch-390-light 390 844 watch?theme=light
watch-768-dark 768 1024 watch?theme=dark
watch-1280-dark 1280 800 watch?theme=dark
watch-1440-dark 1440 900 watch?theme=dark
watch-1440-light 1440 900 watch?theme=light
watch-1440full-dark 1440 2600 watch?theme=dark
watch-1920-dark 1920 1080 watch?theme=dark
watch-1920full-dark 1920 2400 watch?theme=dark
watch-1280full-dark 1280 2800 watch?theme=dark
watch-768full-dark 768 5000 watch?theme=dark
watch-1440-nolayout 1440 2600 watch?theme=dark&layout=none
watch-1920-mockup 1920 2400 watch?theme=dark&layout=mockup
watch-1920-live 1920 2800 watch?theme=dark&layout=live
watch-1920-command 1920 2600 watch?theme=dark&layout=command
overlay-1920-dark 1920 160 watch?theme=dark&mode=overlay
inventory-390-dark 390 844 watch/inventory?theme=dark
inventory-390full-dark 390 2400 watch/inventory?theme=dark
inventory-1440-dark 1440 900 watch/inventory?theme=dark
inventory-1440-light 1440 900 watch/inventory?theme=light
policy-390-dark 390 1600 watch/policy?theme=dark
policy-1440-dark 1440 900 watch/policy?theme=dark
return-390-dark 390 1600 watch/return?theme=dark
return-1440-dark 1440 900 watch/return?theme=dark"
echo "$shots" | while read -r name w h q; do
	[ -n "$only" ] && [ "$only" != "$name" ] && continue
	url="http://127.0.0.1:$port/$q"
	prof=$(mktemp -d)
	DISPLAY=$disp timeout 40 nice -n 19 "$chrome" --headless=new --no-sandbox --disable-gpu --hide-scrollbars --no-first-run \
		--user-data-dir="$prof" --force-device-scale-factor=1 --force-prefers-reduced-motion --disable-lcd-text \
		--window-size="$w,$h" --virtual-time-budget=8000 --timeout=20000 --screenshot="$out/$name.png" "$url" >/dev/null 2>&1
	DISPLAY=$disp timeout 40 nice -n 19 "$chrome" --headless=new --no-sandbox --disable-gpu --no-first-run \
		--user-data-dir="$prof" --window-size="$w,$h" --virtual-time-budget=8000 --timeout=20000 --dump-dom "$url" >"$out/$name.html" 2>/dev/null
	rm -rf "${prof:?}"
done
rc=0
"$bin/uishots" check "$out" || rc=1
if [ "$update" = 1 ]; then
	mkdir -p "$golden" && cp "$out"/*.png "$golden"/ && echo "uishots: golden set updated in $golden"
elif [ -d "$golden" ]; then
	"$bin/uishots" diff -golden "$golden" -got "$out" || rc=1
else
	echo "uishots: no golden set yet (run with --update once)"
fi
echo "uishots: pictures in $out"
exit $rc
