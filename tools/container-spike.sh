#!/usr/bin/env bash
# WP-CONTAINER spike: build the image and run the container checks on a lab worker.
# Run from the repo:  labrun auto -- tools/container-spike.sh        (needs docker or podman on the worker)
# Only creates items labelled d2lod.spike=1 and removes them at the end (KEEP=1 keeps the image).
set -u
cd "$(dirname "$0")/.."
GO=${GO:-$(command -v go || echo ~/.local/bin/go)}
ENGINE=${ENGINE:-$(command -v docker || command -v podman)}
[ -n "$ENGINE" ] || { echo "no docker/podman"; exit 1; }
IMG=d2lod-bot:spike
L="--label d2lod.spike=1"
CTX=deploy/container
mkdir -p $CTX/bin
CGO_ENABLED=0 $GO build -o $CTX/bin/d2container ./deploy/container/entrypoint || exit 1
GOOS=windows GOARCH=386 CGO_ENABLED=0 $GO build -o $CTX/bin/winprobe.exe ./tools/subreaperproof/winprobe || exit 1
CGO_ENABLED=0 $GO build -o $CTX/bin/d2bot ./cmd/d2bot || exit 1
GOOS=windows GOARCH=386 CGO_ENABLED=0 $GO build -o $CTX/bin/d2input.exe ./cmd/d2input || exit 1

echo "== build"
t0=$(date +%s)
$ENGINE build $L -t $IMG $CTX || exit 1
echo "build seconds: $(( $(date +%s) - t0 ))"
$ENGINE image inspect -f 'image size: {{.Size}} bytes' $IMG
echo "host ptrace_scope: $(cat /proc/sys/kernel/yama/ptrace_scope 2>/dev/null)"

RUN="$ENGINE run --rm $L --network none --security-opt no-new-privileges"
echo; echo "== A: Go entrypoint as PID 1, default caps"
$RUN $IMG spike | grep -E 'RESULT|FAIL'
echo; echo "== B1: same with --cap-add SYS_PTRACE (image user is uid 1000: caps stay out of CapEff)"
$RUN --cap-add SYS_PTRACE $IMG spike -idle 5s | grep -E 'RESULT (env|self|sibling)'
echo; echo "== B2: --user 0 without cap (control) and B3: --user 0 --cap-add SYS_PTRACE"
$RUN --user 0 -e WINEPREFIX=/tmp/pfx $IMG spike -idle 5s | grep -E 'RESULT (env|self|sibling)'
$RUN --user 0 -e WINEPREFIX=/tmp/pfx --cap-add SYS_PTRACE $IMG spike -idle 5s | grep -E 'RESULT (env|self|sibling)'
echo; echo "== C: same under docker --init (tini is PID 1, d2container is a child)"
$RUN --init $IMG spike -idle 5s | grep -E 'RESULT (pid1|self|sibling|reaping)'
echo; echo "== D: stop time (SIGTERM to PID 1)"
for mode in d2container sleep; do
  if [ $mode = sleep ]; then c=$($ENGINE run -d $L --entrypoint sleep $IMG 1000); else c=$($ENGINE run -d $L $IMG hold); fi
  sleep 3; s=$(date +%s.%N); $ENGINE stop $c >/dev/null; e=$(date +%s.%N)
  echo "$mode as PID 1: docker stop took $(echo "$e - $s" | bc 2>/dev/null || python3 -c "print(round($e-$s,2))")s"
  $ENGINE rm -f $c >/dev/null
done
echo; echo "== E: idle container, docker stats (hold: Xvfb only)"
c=$($ENGINE run -d $L $IMG hold); sleep 8; $ENGINE stats --no-stream --format 'Xvfb only: mem={{.MemUsage}} cpu={{.CPUPerc}}' $c; $ENGINE rm -f $c >/dev/null

echo; echo "== cleanup"
[ "${KEEP:-0}" = 1 ] || $ENGINE rmi $IMG >/dev/null
rm -rf $CTX/bin
$ENGINE ps -a --filter label=d2lod.spike=1 -q | wc -l | sed 's/^/leftover containers: /'
