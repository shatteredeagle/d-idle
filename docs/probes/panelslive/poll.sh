#!/bin/bash
# poll.sh TAG PORT SECS: every SECS wall seconds save the watch GET endpoints of the viewer (token from the data dir).
# Output poll-TAG/NNN.EP.json plus poll-TAG/timing.tsv (n, endpoint, http code, bytes, seconds). Runs ON THE WORKER, nice 19.
T=$1 P=$2 N=$3; D=$HOME/d2d-lab/panelslive; mkdir -p $D/poll-$T; i=0
for _ in $(seq 60); do [ -s $D/data-$T/watch.token ] && break; sleep 2; done
TOK=$(cat $D/data-$T/watch.token 2>/dev/null) || exit 1
while true; do
  i=$((i+1)); n=$(printf %03d $i)
  for ep in senses bestiary risk orderqueue hall pacts unlocks alerts codex tally return proposals; do
    curl -s -m 8 -H "Cookie: d2watch=$TOK" -o $D/poll-$T/$n.$ep.json -w "$i\t$ep\t%{http_code}\t%{size_download}\t%{time_total}\n" "http://127.0.0.1:$P/api/$ep?t=$TOK" >> $D/poll-$T/timing.tsv
  done
  sleep $N
done
