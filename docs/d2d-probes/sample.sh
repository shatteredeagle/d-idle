#!/bin/bash
# sample.sh OUT SECONDS [INTERVAL]: log "t_s rss_kb cpu_pct_of_core threads" of the d2d process every INTERVAL s (default 30)
out=$1; dur=$2; iv=${3:-30}; pid=$(pgrep -x d2d | head -1); [ -z "$pid" ] && exit 1
: > "$out"; t0=$(date +%s)
prev=$(awk '{print $14+$15}' /proc/$pid/stat); pt=$t0
while [ $(( $(date +%s) - t0 )) -lt "$dur" ]; do
  sleep "$iv"; now=$(date +%s); cur=$(awk '{print $14+$15}' /proc/$pid/stat 2>/dev/null) || break
  rss=$(awk '/VmRSS/{print $2}' /proc/$pid/status)
  echo "$((now-t0)) $rss $(( (cur-prev)*100/(now-pt)/$(getconf CLK_TCK) )) load=$(cut -d' ' -f1 /proc/loadavg)" >> "$out"
  prev=$cur; pt=$now
done
