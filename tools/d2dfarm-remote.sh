#!/bin/sh
# Node side of d2dfarm (tools/d2dfarm.sh ships and runs this file; plain POSIX sh, runs on the node or inside the WSL distro).
#   d2dfarm-remote.sh b64:<base64 of one shell-quoted line of arguments>   (the line is decoded here: no quoting layers)
# Verbs (args: LAB = node lab dir, JOB = job id, a job dir is LAB/farm/JOB):
#   start LAB JOB NICE CMD   run CMD (one shell string) detached in its own process group, cwd = job dir; prints pid=N
#   state LAB JOB            prints running | done EXIT | gone (never started or already removed)
#   stop LAB JOB             TERM the recorded process group, only if /proc/PID/cmdline still carries the job id
#   tar LAB JOB              the job dir (minus big files) as a tar stream on stdout
#   tarb64 LAB JOB           same as tar, base64 text (survives ssh + PowerShell)
#   tail LAB JOB [N]         last N lines of the job log
#   rm LAB JOB               remove the job dir (after collect)
#   put LAB RELPATH          stdin to LAB/RELPATH (executable); sh LAB CMD   run CMD in LAB (read-only helpers like sha256sum)
#   procs LAB                list recorded pids of running jobs on this node ("JOB PID")
# Safety: kills only by the recorded pid (never a name pattern); never touches a game; the cmdline check stops a recycled pid.
case $1 in
b64:*) line=$(printf %s "${1#b64:}" | tr '_.-' '/=+' | base64 -d) || exit 2; eval "set -- $line" ;;
esac
verb=$1 lab=$2 job=${3:-}
case $lab in "~/"*) lab=$HOME/${lab#"~/"} ;; esac
case $job in *[!A-Za-z0-9._-]*|"") [ "$verb" = procs ] || [ "$verb" = put ] || [ "$verb" = sh ] || { echo "bad job id" >&2; exit 2; } ;; esac
dir=$lab/farm/$job
alive() { [ -r "$dir/pid" ] && p=$(cat "$dir/pid") && [ -r "/proc/$p/cmdline" ] && tr '\0' ' ' <"/proc/$p/cmdline" | grep -q -- "farm/$job"; }
case $verb in
start)
	nice=$4 cmd=$5
	mkdir -p "$dir/out" || exit 1
	[ -e "$dir/pid" ] && alive && { echo "already running" >&2; exit 1; }
	rm -f "$dir/exit"
	# the job path in the cmdline is what alive() checks; setsid makes the shell its own group leader (pgid = pid)
	( cd "$dir" && exec setsid nohup sh -c "nice -n $nice sh -c \"\$1\"; echo \$? >$dir/exit" "farm/$job" "$cmd" </dev/null >"$dir/log" 2>&1 ) &
	echo $! >"$dir/pid"
	sleep 1
	echo "pid=$(cat "$dir/pid")" ;;
state)
	if [ -r "$dir/exit" ]; then echo "done $(cat "$dir/exit")"
	elif alive; then echo running
	elif [ -r "$dir/pid" ]; then echo "done 255"   # process vanished without an exit file (killed, host restart)
	else echo gone; fi ;;
stop)
	if alive; then kill -TERM -- "-$p" 2>/dev/null || kill -TERM "$p"; echo "stopped $p"; else echo "not running"; fi ;;
tar) [ -d "$dir" ] || exit 1; tar -c -C "$lab/farm" --exclude='*.sock' --exclude='run/rec' "$job" ;;
tarb64) [ -d "$dir" ] || exit 1; tar -c -C "$lab/farm" --exclude='*.sock' --exclude='run/rec' "$job" | base64 -w0; echo ;;   # text-safe for the PowerShell hop
tail) tail -n "${4:-20}" "$dir/log" ;;
rm) alive && { echo "running" >&2; exit 1; }; rm -rf "$dir" ;;
put) mkdir -p "$(dirname "$lab/$3")" && cat >"$lab/$3" && chmod +x "$lab/$3" ;;   # put LAB RELPATH, stdin -> file
sh) cd "$lab" && eval "$3" ;;
procs) for f in "$lab"/farm/*/pid; do [ -r "$f" ] || continue; j=${f%/pid}; j=${j##*/}; job=$j dir=$lab/farm/$j; alive && echo "$j $p"; done; exit 0 ;;
*) echo "d2dfarm-remote: unknown verb $verb" >&2; exit 2 ;;
esac
