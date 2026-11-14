#!/bin/bash
# Run ONE script of this worktree's tools/emu on the worker host, under agentrun with a ttl. Audited entry point for the
# API-credit runner (it allows exactly this path, not arbitrary `labrun <host> -- bash ...`). Card apicookbook.
#   tools/emu/run-on-worker.sh [--ttl SECS] SCRIPT.sh|SCRIPT.py [ARGS...]
# - SCRIPT is a bare file name in tools/emu (this worktree is mirrored to the worker host by labrun, so a script you just wrote there
#   is already visible). .sh runs with bash, .py with the emulator venv python (PYTHONPATH tools/emu + upstream tools/emu).
# - The script text is refused if it names sudo, kill, curl, ssh, rsync, wine, docker, systemctl, rm -r/-f, chmod, key or claude paths.
# - ARGS may only use [A-Za-z0-9 _.,:=@%+/-]; no "..", and an absolute path must sit under /tmp/ug-emu, /tmp/casttrace (old names), ~/d2d-lab (the emulator image lives in ~/d2d-lab/emu) or the
#   agent's scratch /tmp/d2agent-AGENT (exported to the script as D2_SCRATCH, created first).
# - Runs `timeout TTL` on the worker host and registers the local ssh under tools/agentrun.sh AGENT --ttl TTL+30 (reap: tools/agentreap.sh AGENT).
# - AGENT = $D2D_JOB (set by the API-credit runner to the job name), else the worktree directory name. Default ttl 900, max 3000. Exit code is the script's (124 = ttl).
# Test hooks: WRAP_LABRUN, WRAP_AGENTRUN replace the two programs (tools/emu/run-on-worker-test.sh).
set -u
die() { echo "run-on-worker: $*" >&2; exit 2; }
ttl=900
while [ $# -gt 0 ]; do
	case $1 in
	--ttl) ttl=${2:-}; shift 2 || die "--ttl needs a value" ;;
	*) break ;;
	esac
done
case $ttl in ''|*[!0-9]*) die "bad ttl" ;; esac
[ "$ttl" -ge 5 ] && [ "$ttl" -le 3000 ] || die "ttl must be 5..3000"
[ $# -ge 1 ] || die "usage: run-on-worker.sh [--ttl SECS] SCRIPT [ARGS...]"
script=$1; shift
case $script in ''|*[!A-Za-z0-9_.-]*|.*) die "SCRIPT must be a bare file name in tools/emu" ;; esac
case $script in *.sh|*.py) ;; *) die "SCRIPT must end in .sh or .py" ;; esac
root=$(cd "$(dirname "$0")/../.." && pwd -P) || die "no repo root"
f=$root/tools/emu/$script
[ -f "$f" ] && [ ! -L "$f" ] || die "tools/emu/$script is not a regular file"
agent=${D2D_JOB:-$(basename "$root")}
case $agent in ''|*[!A-Za-z0-9._-]*) die "bad agent name '$agent'" ;; esac
scratch=/tmp/d2agent-$agent

# Same family of words the runner denies for a bare labrun line; here applied to the script itself.
if grep -Eiq 'sudo|pkill|killall|\bkill\b|\bcurl\b|\bwget\b|\bssh\b|\bscp\b|rsync|systemctl|docker|\bclaude\b|\bwine|d2lod-live|rm +-[a-zA-Z]*[rf]|chmod|api-key|anthropic|\.claude|steam|nohup|setsid' "$f"; then
	die "tools/emu/$script names a forbidden command or path (sudo/kill/net/wine/rm -r/chmod/key): remove it"
fi
for a in "$@"; do
	case $a in *[!A-Za-z0-9\ _.,:=@%+/-]*) die "argument '$a' has a character outside [A-Za-z0-9 _.,:=@%+/-]" ;; esac
	case $a in *..*) die "argument '$a' contains .." ;; esac
	case $a in
	/tmp/ug-emu|/tmp/ug-emu/*|/tmp/casttrace|/tmp/casttrace/*|"$scratch"|"$scratch"/*|"$HOME"/d2d-lab/*) ;;
	/*) die "absolute path '$a' is outside /tmp/ug-emu, /tmp/casttrace (old names), ~/d2d-lab (the emulator image lives in ~/d2d-lab/emu) and $scratch" ;;
	esac
done

LABRUN=${WRAP_LABRUN:-labrun}
if [ -z "${D2_LABRUN_HOST:-}" ] && [ -f "$root/tools/env.sh" ]; then . "$root/tools/env.sh"; fi
[ -n "${D2_LABRUN_HOST:-}" ] || die "D2_LABRUN_HOST is not set (labrun machine name; see local.env.example)"
AGENTRUN=${WRAP_AGENTRUN:-$root/tools/agentrun.sh}
cd "$root" || die "cd"
if [ "${script##*.}" = py ]; then
	prog=(bash tools/emu/run-py.sh "$script")
else
	prog=(bash "tools/emu/$script")
fi
exec "$AGENTRUN" "$agent" --ttl $((ttl + 30)) -- "$LABRUN" "$D2_LABRUN_HOST" -- \
	env "D2_SCRATCH=$scratch" \
	sh -c 'mkdir -p "$D2_SCRATCH" && exec "$@"' sh timeout "$ttl" "${prog[@]}" "$@"
