# Host configuration for public scripts: source it (`. "$(dirname "$0")/env.sh"`). POSIX sh, no side effects but the exports.
# Order: process environment, then the gitignored local.env at the repo root, then the generic defaults below
# (reserved .example names: a script that runs on a default does nothing useful, by design).
_d2_root=${D2_BOT_DIR:-$(git rev-parse --show-toplevel 2>/dev/null || pwd)}
if [ -f "$_d2_root/local.env" ]; then
	# keep values already set in the environment
	while IFS='=' read -r k v; do
		case $k in ''|\#*) continue ;; esac
		eval "[ -n \"\${$k:-}\" ] || export $k=\"\$v\""
	done <"$_d2_root/local.env"
fi
: "${D2_WORKER_HOST:=worker.example}" "${D2_WORKER_USER:=${USER:-$(id -un)}}" "${D2_NAS_HOST:=nas.example}"
: "${D2_BOARD_URL:=http://board.example:8897}" "${D2_FORGE_URL:=https://forge.example}" "${D2_COLD_DIR:=/mnt/cold}"
export D2_WORKER_HOST D2_WORKER_USER D2_NAS_HOST D2_BOARD_URL D2_FORGE_URL D2_COLD_DIR
unset _d2_root k v
