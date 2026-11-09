#!/bin/sh
# Stands in for Chromium in smoke.sh: logs its arguments; when given a local shell URL it plays the page
# (saves the address on first run, polls /status until the game is up).
echo "ARGS: $*" >> "$SMOKE_LOG"
for a in "$@"; do case "$a" in --app=*) url="${a#--app=}";; esac; done
case "$url" in
http://127.0.0.1:*/t/*/)
  if [ -n "$SMOKE_SAVE" ]; then
    curl -s -X POST -d "address=$SMOKE_SAVE" "${url}save" >> "$SMOKE_LOG"; echo >> "$SMOKE_LOG"
  fi
  i=0
  while [ $i -lt 20 ]; do
    s=$(curl -s "${url}status"); echo "STATUS: $s" >> "$SMOKE_LOG"
    case "$s" in *'"state":"up"'*) break;; esac
    i=$((i+1)); sleep 1
  done;;
esac
