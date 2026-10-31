#!/usr/bin/env bash
# Build the runtime image d2idle:dev on a lab worker with docker:  labrun <worker> -- tools/container-build.sh
# Cross-compiles d2container, d2bot, d2fleet, d2doctor, d2soak (linux/amd64) and d2input.exe (windows/386) into deploy/container/bin
# (git-ignored build context only, removed afterwards). No Blizzard file goes into the image.
set -eu
cd "$(dirname "$0")/.."
GO=${GO:-$(command -v go || echo ~/.local/bin/go)}
ENGINE=${ENGINE:-docker}
IMG=${IMG:-d2idle:dev}
CTX=deploy/container
mkdir -p $CTX/bin
CGO_ENABLED=0 $GO build -o $CTX/bin/d2container ./deploy/container/entrypoint
CGO_ENABLED=0 $GO build -o $CTX/bin/d2bot ./cmd/d2bot
CGO_ENABLED=0 $GO build -o $CTX/bin/d2fleet ./cmd/d2fleet   # web assets are embedded (web.FS)
CGO_ENABLED=0 $GO build -o $CTX/bin/d2doctor ./cmd/d2doctor
CGO_ENABLED=0 $GO build -o $CTX/bin/d2soak ./cmd/d2soak     # the D2FLEET_SOAK=1 sidecar and `docker exec d2idle d2soak report`
GOOS=windows GOARCH=386 CGO_ENABLED=0 $GO build -o $CTX/bin/d2input.exe ./cmd/d2input
GOOS=windows GOARCH=386 CGO_ENABLED=0 $GO build -o $CTX/bin/winprobe.exe ./tools/subreaperproof/winprobe
file $CTX/bin/d2input.exe 2>/dev/null || true
t0=$(date +%s)
nice -n 10 $ENGINE build --label d2lod.container=1 -t "$IMG" $CTX
echo "build seconds: $(( $(date +%s) - t0 ))"
$ENGINE image inspect -f 'image size: {{.Size}} bytes' "$IMG"
rm -rf $CTX/bin
