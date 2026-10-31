#!/bin/bash
# Runs ON THE WORKER inside a d2dslot slot (env D2D_SOCK, D2D_SLOT_DIR set by d2dslot.sh).
# usage: d2dslot.sh run -- runone.sh TAG BIN CBLEVEL CLVL SEED OUTDIR
# One lockstep d2d (BIN) per run, then ~/d2d-lab/probes/d2dcasts2 exactly as docs/probes/run-d2d.sh does.
set -u
TAG=$1; BIN=$2; L=$3; C=$4; S=$5; OUT=$6
mkdir -p "$OUT"; SOCK=$D2D_SOCK; rm -f "$SOCK"
SDL_VIDEODRIVER=dummy nice -n 10 "$BIN" --data $HOME/d2d-lab/data --headless \
  --devctl "$SOCK" --no-video --lockstep --no-save --toggle autoloot=off,autoparty=off --seed "$S" --start-screen ingame \
  --start-class 4 --start-name Probey > "$OUT/d2d-$TAG-$L-$S.log" 2>&1 &
PID=$!
for i in $(seq 120); do [ -S "$SOCK" ] && break; sleep 0.5; done
sleep 2
nice -n 10 $HOME/d2d-lab/probes/d2dcasts2 -sock "$SOCK" -seed "$S" -tables $HOME/d2lod-live/combat/data -area 2 -cb "$L" -clvl "$C" -castgap 450 -virtual 15m \
  > "$OUT/$TAG-$L-$S.out" 2> "$OUT/$TAG-$L-$S.err"
echo "exit $?" >> "$OUT/$TAG-$L-$S.err"
kill $PID 2>/dev/null; sleep 1; kill -9 $PID 2>/dev/null; wait $PID 2>/dev/null
