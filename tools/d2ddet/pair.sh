#!/bin/bash
# Divergence finder: run the full bot twice on the hidden d2d with the SAME seed and a FRESH character, journal every
# command (D2DCTL_JOURNAL), then compare the two journals (cmd/d2ddet). docs/d2d-determinism.md.
#
#   usage (on the worker, in ~/d2d-lab/det): pair.sh TAG BOTBIN SEED TICKS [RUNFOR] [N]
#     TAG     run name; journals land in TAG-A.jnl, TAG-B.jnl (N > 2: TAG-C ...), bot logs in bot-TAG-X.log
#     BOTBIN  d2bot binary in this dir (built by tools/d2ddet/build.sh)
#     SEED    map seed (--seed)
#     TICKS   virtual ticks per run (25 per game second; 5000 = 200 s, 50000 = 2000 s)
#     RUNFOR  wall cap per run (default 12m)
#     N       runs per set (default 2); every run is compared with the first
#
# Each run holds /tmp/d2d-lab.lock only while its d2d is up (KILLGAP and the lab probes queue on the same lock).
# The d2d is the fork's lab6 (D2D_BIN, default lab6; PAR=n runs n at a time, slot pool), lockstep, managed by the bot; the character is a new Sorceress "Dety" in a
# run-private data dir (no save survives between runs: the start state is the same for A and B).
set -eu
cd "$(dirname "$0")"
# whole script (lock waits included) runs as one registered process group: tools/agentrun.sh, reap with tools/agentreap.sh
if [ -z "${AGENTRUN_ON:-}" ] && [ -x "$HOME/d2d-lab/bin/agentrun.sh" ]; then
  exec "$HOME/d2d-lab/bin/agentrun.sh" "${D2D_AGENT:-det}" -- env AGENTRUN_ON=1 "$PWD/${0##*/}" "$@"
fi
T=$1; BIN=$2; SEED=$3; TICKS=$4; RUNFOR=${5:-12m}; N=${6:-2}
HOURS=$(python3 -c "print($TICKS/90000)")
export D2BOT_INSTANCE=det D2DSOAK_HOURS=$HOURS D2DCTL_JOURNAL_EVERY=${D2DCTL_JOURNAL_EVERY:-250}
run() {
  X=$1
  rm -rf "data-$T-$X"; mkdir -p "data-$T-$X"
  D2DCTL_JOURNAL="$PWD/$T-$X.jnl" \
  $HOME/d2d-lab/d2dslot.sh run -w 3000 -- env D2D_BIN="${D2D_BIN:-$HOME/d2d-lab/fork/bin/d2d.lab6}" D2D_DATA="$HOME/d2d-lab/data" \
    nice -n 10 "./$BIN" --backend d2d --managed --d2d-class 4 --char Dety --seed "$SEED" \
    --toggle autoloot=off,autoparty=off --d2d-args "--lockstep" --d2d-linkwait 60s \
    --gamedir "$HOME/d2d-lab/data" --data "$PWD/data-$T-$X" --d2d-sock "$PWD/d2d-$T-$X.sock" \
    --addr 127.0.0.1:8479 --goals campaign --run-for "$RUNFOR" > "bot-$T-$X.log" 2>&1 || true
  echo "exit $?" >> "bot-$T-$X.log"
}
letters=(A B C D E F G H)
# PAR runs at once (each takes its own slot, at most 3 for the shared box); same seed and binaries, so parallel is as valid as serial.
PAR=${PAR:-1}
for ((i = 0; i < N; i++)); do
  run "${letters[$i]}" &
  if (( (i + 1) % PAR == 0 )); then wait; fi
done
wait
for ((i = 1; i < N; i++)); do
  echo "== $T-A vs $T-${letters[$i]}"
  ./d2ddet compare "$T-A.jnl" "$T-${letters[$i]}.jnl" || true
done
