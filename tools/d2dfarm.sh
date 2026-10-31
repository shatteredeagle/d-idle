#!/bin/bash
# d2dfarm: distributed hidden-d2d test nodes (spec: openspec/changes/d2dfarm/specs/d2d-farm, doc: docs/process/d2dfarm.md).
#   d2dfarm nodes                          nodes with labels, state (paused/idle-only), probe verdict, last seen, running jobs
#   d2dfarm node pause|resume NAME         stop/allow assigning new jobs (running jobs continue); persisted
#   d2dfarm probe [NAME...]                capacity probe per node: OK|WARN|NO + reason (cached, see D2_FARM_PROBE_TTL)
#   d2dfarm push-tools NAME                build d2dsoak/d2dreplay/d2dprobe and ship them to NAME's lab/farm/bin
#   d2dfarm submit [opts] -- ARGS          queue a job (opts: --recipe soak|replay|record|cmd --seeds S --hours H --split N
#                                          --labels a,b --host NAME --upload FILE --script FILE); soak ARGS = extra d2dsoak flags
#   d2dfarm dispatch                       assign queued jobs to the best free node (label, hours, pause, probe, capacity)
#   d2dfarm run [opts] -- ARGS             submit + dispatch
#   d2dfarm status                         refresh and list jobs (shuts a wsl node down when its queue is empty and collected)
#   d2dfarm collect JOB|BATCH [--keep]     copy the run dirs to $STATE/results/BATCH/JOB, remove them on the node
#   d2dfarm verdict BATCH                  ONE d2dsoak compare (-gate v2) over all collected seeds of a batch
#   d2dfarm stop JOB|BATCH                 TERM the recorded process group (never a name pattern)
#   d2dfarm det FILE.replay.gz [NODE...]   cross-host determinism check: play the same replay on each node, compare
#   d2dfarm reap                           wsl --shutdown on wsl nodes with nothing running or uncollected
# Registry: D2_FARM_HOSTS (inline, entries separated by ';') or D2_FARM_HOSTS_FILE (default tools/d2dfarm.hosts, else the
# example), both read from the environment or local.env. Line: NAME TYPE SSH MAX HOURS [key=value ...]
#   TYPE lab-slots|wsl; HOURS any|HH-HH (wraps midnight); keys: labels=a,b lab= distro= user= nice= prio= build= d2d= d2dbin=
# Transport: shell functions tx_<transport>_<verb> (probe start state collect stop put shutdown); only ssh exists.
# Env: D2_FARM_DIR (state), D2_FARM_TRANSPORT, D2_FARM_BUILD (d2d build name, default lab6), D2_FARM_DRY=1, D2_FARM_NOW_HOUR.
set -u
here=$(cd "$(dirname "$0")" && pwd)
repo=$(cd "$here/.." && pwd)
# the environment wins over local.env
[ -n "${D2_FARM_HOSTS:-}${D2_FARM_HOSTS_FILE:-}" ] || { [ -r "$repo/local.env" ] && . "$repo/local.env"; }
STATE=${D2_FARM_DIR:-$HOME/.local/state/d2dfarm}
TRANSPORT=${D2_FARM_TRANSPORT:-ssh}
BUILD=${D2_FARM_BUILD:-lab6}
PROBE_TTL=${D2_FARM_PROBE_TTL:-120}
PROBE_WINDOW=${D2_FARM_PROBE_WINDOW:-20}
mkdir -p "$STATE/jobs" "$STATE/nodes" "$STATE/results"
die() { echo "d2dfarm: $*" >&2; exit 2; }

# ---------- registry ----------
declare -A NT NS NMAX NHOURS NLAB NLABELS NDISTRO NUSER NNICE NPRIO NBUILD ND2D ND2DBIN
NODES=()
load_nodes() {
	local src line f k v rest
	if [ -n "${D2_FARM_HOSTS:-}" ]; then src=$(printf '%s\n' "$D2_FARM_HOSTS" | tr ';' '\n')
	else f=${D2_FARM_HOSTS_FILE:-$here/d2dfarm.hosts}; [ -r "$f" ] || f=$here/d2dfarm.hosts.example; src=$(cat "$f"); fi
	while read -r line; do
		case $line in ''|'#'*) continue ;; esac
		# shellcheck disable=SC2086
		set -- $line; [ $# -ge 5 ] || die "bad registry line: $line"
		local n=$1; NODES+=("$n"); NT[$n]=$2 NS[$n]=$3 NMAX[$n]=$4 NHOURS[$n]=$5; shift 5
		NLABELS[$n]="" NDISTRO[$n]=Debian NUSER[$n]=d2 NNICE[$n]=19 NBUILD[$n]=$BUILD ND2D[$n]="" ND2DBIN[$n]=""
		NLAB[$n]=; NPRIO[$n]=
		for f in "$@"; do k=${f%%=*} v=${f#*=}
			case $k in labels) NLABELS[$n]=$v ;; lab) NLAB[$n]=$v ;; distro) NDISTRO[$n]=$v ;; user) NUSER[$n]=$v ;;
			nice) NNICE[$n]=$v ;; prio) NPRIO[$n]=$v ;; build) NBUILD[$n]=$v ;; d2d) ND2D[$n]=$v ;; d2dbin) ND2DBIN[$n]=$v ;;
			*) die "unknown key $k for $n" ;; esac; done
		case ${NT[$n]} in
		lab-slots) : "${NLAB[$n]:=~/d2d-lab}" "${NPRIO[$n]:=10}"; : "${ND2D[$n]:=${NLAB[$n]}/fork/bin/d2d.${NBUILD[$n]}}"; ND2DBIN[$n]=${ND2DBIN[$n]:-${ND2D[$n]}} ;;
		wsl) : "${NLAB[$n]:=/home/${NUSER[$n]}/d2d-lab}" "${NPRIO[$n]:=50}"; : "${ND2D[$n]:=${NLAB[$n]}/bin/d2d.${NBUILD[$n]}.sh}"; ND2DBIN[$n]=${ND2DBIN[$n]:-${NLAB[$n]}/bin/d2d.${NBUILD[$n]}} ;;
		*) die "node $n: unknown type ${NT[$n]}" ;; esac
		[ "${NLAB[$n]#/}" != "${NLAB[$n]}" ] || [ "${NT[$n]}" != wsl ] || die "node $n: wsl lab must be absolute"
	done <<<"$src"
	[ ${#NODES[@]} -gt 0 ] || die "no nodes registered"
}
node_ok() { [ -n "${NT[$1]:-}" ] || die "unknown node $1"; }
labels_of() { echo "$1,${NT[$1]},${NLABELS[$1]}" | tr ',' '\n' | grep . | sort -u | tr '\n' ','; }
paused() { [ -e "$STATE/nodes/$1.paused" ]; }
in_hours() { # NAME
	local h=${NHOURS[$1]} now=${D2_FARM_NOW_HOUR:-$(date +%-H)} a b
	[ "$h" = any ] && return 0
	a=${h%-*} b=${h#*-}
	if [ "$a" -le "$b" ]; then [ "$now" -ge "$a" ] && [ "$now" -lt "$b" ]; else [ "$now" -ge "$a" ] || [ "$now" -lt "$b" ]; fi
}

# ---------- transport: ssh ----------
SSHO=(-o BatchMode=yes -o ConnectTimeout=10)
q() { local a o=""; for a in "$@"; do o="$o $(printf %q "$a")"; done; printf %s "$o"; }
tx_ssh_rexec() { # NODE VERB ARGS... (stdin passes through)
	local n=$1 line b lab=${NLAB[$1]}; shift
	line=$(q "$1" "$lab" "${@:2}"); b=$(printf %s "$line" | base64 -w0 | tr '+/=' '-_.')
	if [ "${NT[$n]}" = wsl ]; then ssh "${SSHO[@]}" "${NS[$n]}" "wsl -d ${NDISTRO[$n]} -u ${NUSER[$n]} -e sh $lab/farm/remote.sh b64:$b" | tr -d '\r'
	else ssh "${SSHO[@]}" "${NS[$n]}" "sh $lab/farm/remote.sh b64:$b"; fi
}
tx_ssh_install() { # NODE: ship the node-side script (small, idempotent)
	local n=$1 lab=${NLAB[$1]}
	if [ "${NT[$n]}" = wsl ]; then ssh "${SSHO[@]}" "${NS[$n]}" "wsl -d ${NDISTRO[$n]} -u ${NUSER[$n]} -e sh -c \"mkdir -p $lab/farm && cat > $lab/farm/remote.sh\"" <"$here/d2dfarm-remote.sh"
	else ssh "${SSHO[@]}" "${NS[$n]}" "mkdir -p $lab/farm && cat > $lab/farm/remote.sh" <"$here/d2dfarm-remote.sh"; fi
}
tx_ssh_put() { tx_ssh_rexec "$1" put "$2" <"$3"; } # NODE RELPATH LOCALFILE
tx_ssh_start() { tx_ssh_rexec "$1" start "$2" "${NNICE[$1]}" "$3"; } # NODE JOB CMD -> pid=N
tx_ssh_state() { tx_ssh_rexec "$1" state "$2"; }
tx_ssh_stop() { tx_ssh_rexec "$1" stop "$2"; }
tx_ssh_collect() { tx_ssh_install "$1"; tx_ssh_rexec "$1" tarb64 "$2" | tr -d "\r\n" | base64 -d | tar -x -C "$3" && tx_ssh_rexec "$1" rm "$2"; } # NODE JOB DESTDIR
tx_ssh_collect_keep() { tx_ssh_install "$1"; tx_ssh_rexec "$1" tarb64 "$2" | tr -d "\r\n" | base64 -d | tar -x -C "$3"; }
tx_ssh_shutdown() { [ "${NT[$1]}" = wsl ] && ssh "${SSHO[@]}" "${NS[$1]}" "wsl --shutdown"; return 0; }
tx_ssh_probe() { # NODE -> "VERDICT reason"
	local n=$1 out rc
	if [ "${NT[$n]}" = wsl ]; then
		local ps='$c=(Get-CimInstance Win32_Processor | Measure-Object LoadPercentage -Average).Average
$o=Get-CimInstance Win32_OperatingSystem
"load=$c"; "freemb=$([int]($o.FreePhysicalMemory/1024))"
"user=$((Get-CimInstance Win32_ComputerSystem).UserName)"
"explorer=$((Get-Process explorer -ErrorAction SilentlyContinue | Measure-Object).Count)"
"encoder=$((Get-Process ffmpeg -ErrorAction SilentlyContinue | Measure-Object).Count)"'
		out=$(timeout $((PROBE_WINDOW + 30)) ssh "${SSHO[@]}" "${NS[$n]}" "powershell -NoProfile -EncodedCommand $(printf %s "$ps" | iconv -f utf-8 -t utf-16le | base64 -w0)" 2>&1 | tr -d '\r'); rc=$?
		probe_wsl_verdict "$out"
	else
		out=$(timeout $((PROBE_WINDOW + 10)) ssh "${SSHO[@]}" "${NS[$n]}" "sh -s -- --add 1 --window $PROBE_WINDOW --samples 4" <"$here/capacity.sh" 2>/dev/null); rc=$?
		local why; why=$(printf '%s\n' "$out" | grep -E '^(add=|psi|iowait)' | tr '\n' ' ')
		case $rc in 0) echo "OK $why" ;; 1) echo "WARN $why" ;; 2) echo "NO $why" ;; *) echo "NO unreachable or probe failed (rc $rc)" ;; esac
	fi
}
probe_wsl_verdict() { # key=value lines; user logged in interactively (or explorer running) = busy: idle-only node
	local load freemb user expl enc
	load=$(sed -n 's/^load=//p' <<<"$1"); freemb=$(sed -n 's/^freemb=//p' <<<"$1"); user=$(sed -n 's/^user=//p' <<<"$1")
	expl=$(sed -n 's/^explorer=//p' <<<"$1"); enc=$(sed -n 's/^encoder=//p' <<<"$1")
	[ -n "$load$freemb" ] || { echo "NO unreachable or probe failed"; return; }
	if [ -n "$user" ] || [ "${expl:-0}" -gt 0 ]; then echo "NO user active (interactive session)"; return; fi
	if [ "${load:-0}" -ge 85 ] || [ "${freemb:-99999}" -lt 5000 ]; then echo "NO host busy: cpu ${load}% free ${freemb} MB"; return; fi
	if [ "${load:-0}" -ge 60 ] || [ "${freemb:-99999}" -lt 8000 ]; then echo "WARN cpu ${load}% free ${freemb} MB"; return; fi
	echo "OK cpu ${load}% free ${freemb} MB, no interactive user${enc:+, ${enc} encoder process(es)}"
}
tx() { local v=$1; shift; "tx_${TRANSPORT}_$v" "$@"; }

# ---------- probe cache / nodes ----------
probe_cached() { # NAME -> "VERDICT reason" (refreshes the cache and last-seen)
	local n=$1 f=$STATE/nodes/$1.probe now; now=$(date +%s)
	if [ -r "$f" ] && [ $((now - $(stat -c %Y "$f"))) -lt "$PROBE_TTL" ]; then cat "$f"; return; fi
	local r; r=$(tx probe "$n"); echo "$r" >"$f"
	case $r in NO\ unreachable*) ;; *) echo "$now" >"$STATE/nodes/$n.seen" ;; esac
	echo "$r"
}
running_on() { local c=0 j; for j in "$STATE"/jobs/*/job.env; do [ -r "$j" ] || continue; ( . "$j"; [ "$node" = "$1" ] && [ "$state" = running ] ) && c=$((c+1)); done; echo $c; }
cmd_nodes() {
	printf '%-12s %-12s %-6s %-9s %-7s %-34s %s\n' NODE TYPE JOBS STATE HOURS LABELS "PROBE (last seen)"
	local n st seen pr
	for n in "${NODES[@]}"; do
		st=ready; paused "$n" && st=paused; in_hours "$n" || st="$st/off-hours"
		seen=never; [ -r "$STATE/nodes/$n.seen" ] && seen=$(date -d "@$(cat "$STATE/nodes/$n.seen")" +%m-%d' '%H:%M)
		pr=$(cat "$STATE/nodes/$n.probe" 2>/dev/null || echo "not probed")
		printf '%-12s %-12s %-6s %-9s %-7s %-34s %s (%s)\n' "$n" "${NT[$n]}" "$(running_on "$n")/${NMAX[$n]}" "$st" "${NHOURS[$n]}" "$(labels_of "$n")" "$pr" "$seen"
	done
}
cmd_probe() {
	local ns=("$@") n; [ ${#ns[@]} -gt 0 ] || ns=("${NODES[@]}")
	for n in "${ns[@]}"; do node_ok "$n"; rm -f "$STATE/nodes/$n.probe"; ( probe_cached "$n" >"$STATE/nodes/$n.out" ) & done; wait
	for n in "${ns[@]}"; do printf '%-12s %s\n' "$n" "$(cat "$STATE/nodes/$n.out")"; done
}

# ---------- jobs ----------
jenv() { echo "$STATE/jobs/$1/job.env"; }
jsave() { # ID k=v... : rewrite the keys in job.env
	local id=$1 f; f=$(jenv "$id"); shift
	local kv k; for kv in "$@"; do k=${kv%%=*}; sed -i "/^$k=/d" "$f"; printf '%s=%q\n' "$k" "${kv#*=}" >>"$f"; done
}
jload() { # ID -> sets job vars
	id= batch= recipe= args= labels= node= pid= state= submitted= finished= exitc= collected= upload= host= hours= seeds=
	# shellcheck disable=SC1090
	. "$(jenv "$1")"
}
expand_seeds() { local x o=""; for x in $(echo "$1" | tr ',' ' '); do case $x in *-*) o="$o $(seq "${x%-*}" "${x#*-}")" ;; *) o="$o $x" ;; esac; done; echo $o; }
newid() { echo "$(date +%m%d-%H%M%S)-$RANDOM"; }
mkjob() { # recipe batch seeds hours labels host upload args
	local id; id=$(newid); mkdir -p "$STATE/jobs/$id"; : >"$(jenv "$id")"
	jsave "$id" id="$id" batch="$2" recipe="$1" seeds="$3" hours="$4" labels="$5" host="$6" upload="$7" args="$8" state=queued submitted="$(date +%s)"
	echo "$id"
}
cmd_submit() {
	local recipe=soak seeds=1-3 hours=2 split=1 labels="" host="" upload="" script="" batch
	while [ $# -gt 0 ]; do case $1 in
		--recipe) recipe=$2; shift 2 ;; --seeds) seeds=$2; shift 2 ;; --hours) hours=$2; shift 2 ;; --split) split=$2; shift 2 ;;
		--labels) labels=$2; shift 2 ;; --host) host=$2; shift 2 ;; --upload) upload=$2; shift 2 ;; --script) script=$2; shift 2 ;;
		--auto) shift ;; --) shift; break ;; *) die "submit: unknown option $1" ;; esac; done
	[ -z "$host" ] || node_ok "$host"
	case $recipe in soak|replay|record|cmd) ;; *) die "unknown recipe $recipe" ;; esac
	[ "$recipe" != replay ] || [ -r "$upload" ] || die "replay recipe needs --upload FILE.replay.gz"
	[ "$recipe" != record ] || { script=${script:-$here/d2dfarm.fight.txt}; upload=$script; [ -r "$script" ] || die "record: script not found"; }
	batch=$(newid)
	if [ "$recipe" = soak ] && [ "$split" -gt 1 ]; then
		local all=($(expand_seeds "$seeds")) i part
		for ((i = 0; i < split; i++)); do part=""; local k
			for ((k = i; k < ${#all[@]}; k += split)); do part="$part,${all[$k]}"; done
			[ -n "$part" ] && echo "job $(mkjob soak "$batch" "${part#,}" "$hours" "$labels" "$host" "$upload" "$*") batch $batch seeds ${part#,}"
		done
	else echo "job $(mkjob "$recipe" "$batch" "$seeds" "$hours" "$labels" "$host" "$upload" "$*") batch $batch"; fi
}
tools_dir() { echo "${NLAB[$1]}/farm/bin"; }
remote_cmd() { # NAME -> shell string for the node (placeholders resolved here, paths are the node's)
	local n=$1 t; t=$(tools_dir "$n")
	local d2d=${ND2D[$n]} data=${NLAB[$n]}/data home=${NLAB[$n]}/home
	case $recipe in
	soak) echo "$t/d2dsoak run -seeds $seeds -hours $hours -out \$PWD/out -d2d $d2d -data $data -home $home $args" ;;
	replay) echo "$t/d2dreplay play -run \$PWD/run -data $data -d2d-bin $d2d \$PWD/input.replay.gz $args" ;;
	record) echo "set -e; mkdir -p \$PWD/run; $t/d2dreplay record -launch -run \$PWD/run/rec -data $data -d2d-bin $d2d -seed $seeds -class 1 -name Det -d2d-args '--toggle autoloot=off,autoparty=off --lockstep --no-save' -listen \$PWD/run/tap.sock -out \$PWD/out/fight.replay.gz -every 250 -idle 20s -max 10m $args & rp=\$!; i=0; while [ ! -S \$PWD/run/tap.sock ] && [ \$i -lt 12000 ] && kill -0 \$rp 2>/dev/null; do sleep 0.25; i=\$((i+1)); done; $t/d2dprobe -sock \$PWD/run/tap.sock -total 5m -script \$PWD/input.txt; wait \$rp" ;;
	cmd) echo "$args" ;;
	esac
}
upload_name() { case $recipe in replay) echo input.replay.gz ;; record) echo input.txt ;; esac; }
start_job() { # ID NODE
	local id=$1 n=$2 r pid
	jload "$id"
	[ -z "${D2_FARM_DRY:-}" ] || { echo "dry: would start $id ($recipe) on $n"; return 0; }
	tx install "$n" || return 1
	[ -z "$upload" ] || { tx put "$n" "farm/$id/$(upload_name)" "$upload" || return 1; }
	r=$(tx start "$n" "$id" "$(remote_cmd "$n")" 9>&-) || return 1
	touch "$STATE/nodes/$n.up"; pid=${r#pid=}; jsave "$id" node="$n" pid="$pid" state=running started="$(date +%s)"
	echo "started $id on $n pid $pid"
}
cmd_dispatch() {
	exec 9>"$STATE/lock"; flock -n 9 || die "another dispatch is running"
	local j id n cand ranked pr verdict
	for j in "$STATE"/jobs/*/job.env; do [ -r "$j" ] || continue; id=$(basename "$(dirname "$j")"); jload "$id"; [ "$state" = queued ] || continue
		cand=()
		for n in "${NODES[@]}"; do
			[ -z "$host" ] || [ "$host" = "$n" ] || continue
			paused "$n" && continue
			in_hours "$n" || continue
			local want have; have=$(labels_of "$n"); want=${labels//,/ }
			local ok=1 w; for w in $want; do case $have in *"$w,"*) ;; *) ok=0 ;; esac; done; [ $ok = 1 ] || continue
			[ "$(running_on "$n")" -lt "${NMAX[$n]}" ] || continue
			cand+=("$n")
		done
		[ ${#cand[@]} -gt 0 ] || { echo "queued $id: no node (label/paused/hours/full)"; continue; }
		for n in "${cand[@]}"; do rm -f "$STATE/nodes/$n.probe"; ( probe_cached "$n" >"$STATE/nodes/$n.out" ) & done; wait
		ranked=""
		for n in "${cand[@]}"; do verdict=$(cut -d' ' -f1 "$STATE/nodes/$n.out"); case $verdict in OK) pr=0 ;; WARN) pr=1 ;; *) continue ;; esac
			ranked="$ranked$((pr * 1000 + NPRIO[$n] + $(running_on "$n"))) $n"$'\n'; done
		n=$(printf '%s' "$ranked" | sort -n | head -1 | cut -d' ' -f2)
		[ -n "$n" ] || { echo "queued $id: every candidate probes NO"; continue; }
		start_job "$id" "$n" || echo "start of $id on $n failed" >&2
	done
}
cmd_run() { local out; out=$(cmd_submit "$@") || exit $?; echo "$out"; cmd_dispatch; }
refresh() { # update running jobs from their nodes
	local j id r
	for j in "$STATE"/jobs/*/job.env; do [ -r "$j" ] || continue; id=$(basename "$(dirname "$j")"); jload "$id"; [ "$state" = running ] || continue
		r=$(tx state "$node" "$id" 2>/dev/null) || continue
		case $r in running) ;; "done 0") jsave "$id" state=done exitc=0 finished="$(date +%s)" ;;
		done*) jsave "$id" state=failed exitc="${r#done }" finished="$(date +%s)" ;; gone) jsave "$id" state=failed exitc=gone finished="$(date +%s)" ;; esac
	done
}
cmd_status() {
	refresh
	printf '%-18s %-9s %-7s %-12s %-5s %s\n' JOB STATE RECIPE NODE EXIT "BATCH seeds collected"
	local j id
	for j in "$STATE"/jobs/*/job.env; do [ -r "$j" ] || continue; id=$(basename "$(dirname "$j")"); jload "$id"
		printf '%-18s %-9s %-7s %-12s %-5s %s %s %s\n' "$id" "$state" "$recipe" "${node:--}" "${exitc:--}" "$batch" "$seeds" "${collected:+collected}"; done
	cmd_reap
}
sel() { # JOB|BATCH -> job ids
	local j id
	for j in "$STATE"/jobs/*/job.env; do [ -r "$j" ] || continue; id=$(basename "$(dirname "$j")"); jload "$id"; { [ "$id" = "$1" ] || [ "$batch" = "$1" ]; } && echo "$id"; done
}
cmd_collect() {
	local keep=0 id any=0 t; [ "${2:-}" != --keep ] || keep=1
	refresh
	for id in $(sel "$1"); do jload "$id"; any=1
		case $state in done|failed) ;; *) echo "$id: $state, not collected"; continue ;; esac
		[ -z "$collected" ] || { echo "$id: already collected"; continue; }
		t=$STATE/results/$batch; mkdir -p "$t"
		if [ $keep = 1 ]; then tx collect_keep "$node" "$id" "$t"; else tx collect "$node" "$id" "$t"; fi && { jsave "$id" collected=1; echo "$id: collected to $t/$id"; } || echo "$id: collect failed" >&2
	done
	[ $any = 1 ] || die "no such job or batch $1"
	cmd_reap
}
cmd_stop() {
	local id any=0 r; for id in $(sel "$1"); do jload "$id"; any=1
		case $state in queued) jsave "$id" state=stopped; echo "$id: dequeued" ;;
		running) r=$(tx stop "$node" "$id"); jsave "$id" state=stopped finished="$(date +%s)"; echo "$id: $r" ;; esac; done
	[ $any = 1 ] || die "no such job or batch $1"
}
cmd_reap() { # wsl node with no running job and nothing left to collect: shut the distro down
	local n j id busy
	for n in "${NODES[@]}"; do [ "${NT[$n]}" = wsl ] || continue; busy=0
		for j in "$STATE"/jobs/*/job.env; do [ -r "$j" ] || continue; id=$(basename "$(dirname "$j")"); jload "$id"
			[ "$node" = "$n" ] || continue
			case $state in running) busy=1 ;; done|failed) [ -n "$collected" ] || busy=1 ;; esac; done
		[ $busy = 0 ] || continue
		[ -e "$STATE/nodes/$n.up" ] || continue
		[ -n "${D2_FARM_DRY:-}" ] && { echo "dry: would shut $n down"; continue; }
		tx shutdown "$n" && rm -f "$STATE/nodes/$n.up" && echo "$n: wsl shut down (queue empty, results collected)"
	done
}
cmd_verdict() {
	local d=$STATE/results/$1 s=() sd
	[ -d "$d" ] || die "nothing collected for $1"
	for sd in "$d"/*/out/seed-*; do [ -d "$sd" ] && s+=("$sd"); done
	[ ${#s[@]} -gt 0 ] || die "no seed dirs under $d"
	local bin=${D2_FARM_SOAK_LOCAL:-$STATE/bin/d2dsoak}
	[ -x "$bin" ] || { mkdir -p "$STATE/bin"; (cd "$repo" && go build -o "$STATE/bin/d2dsoak" ./cmd/d2dsoak) || die "cannot build d2dsoak"; }
	echo "batch $1: ${#s[@]} seed dirs from $(ls "$d" | wc -l) job(s)"
	"$bin" compare -gate v2 -baselines "$repo/tools/d2dsoak/baselines.json" "${s[@]}"
}
cmd_push_tools() {
	local n=$1 t c; node_ok "$n"; t=$(tools_dir "$n"); tx install "$n"
	for c in d2dsoak d2dreplay d2dprobe; do
		(cd "$repo" && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o "$STATE/bin-$c" "./cmd/$c") || die "build $c"
		tx put "$n" "farm/bin/$c" "$STATE/bin-$c" || die "put $c"
	done
	touch "$STATE/nodes/$n.up"; echo "$n: tools in $t"
}
cmd_det() {
	local f=$1; shift; [ -r "$f" ] || die "det: replay file $f not found"
	local ns=("$@") n id ids=() batch="det-$(newid)"; [ ${#ns[@]} -gt 0 ] || ns=("${NODES[@]}")
	for n in "${ns[@]}"; do node_ok "$n"; echo "$n: sha256 $(tx_ssh_rexec "$n" sh "sha256sum ${ND2DBIN[$n]} | cut -d' ' -f1")"; done
	for n in "${ns[@]}"; do cmd_submit --recipe replay --host "$n" --upload "$f" >/dev/null; done
	cmd_dispatch
	echo "waiting; poll with: d2dfarm status, then d2dfarm collect JOB and read results/*/*/log"
}

# ---------- main ----------
load_nodes
case "${1:-}" in
nodes) cmd_nodes ;;
node) node_ok "${3:-}"; case ${2:-} in pause) touch "$STATE/nodes/$3.paused"; echo "$3 paused" ;; resume) rm -f "$STATE/nodes/$3.paused"; echo "$3 resumed" ;; *) die "node pause|resume NAME" ;; esac ;;
probe) shift; cmd_probe "$@" ;;
push-tools) cmd_push_tools "${2:?node}" ;;
submit) shift; cmd_submit "$@" ;;
dispatch) cmd_dispatch ;;
run) shift; cmd_run "$@" ;;
status) cmd_status ;;
collect) cmd_collect "${2:?job or batch}" "${3:-}" ;;
verdict) cmd_verdict "${2:?batch}" ;;
stop) cmd_stop "${2:?job or batch}" ;;
reap) cmd_reap ;;
det) shift; cmd_det "$@" ;;
*) sed -n 2,22p "$0"; exit 2 ;;
esac
