# Sourced by soak24.sh and soak.sh: where one soak lives, per instance, so two soaks on two instances never share a path.
#   new layout (V: tools/live.sh puts the binary at ~/d2lod-live/<agent>/<pkg> and the job pid/log in <agent>/runs/):
#     agent soak24-iN -> ~/d2lod-live/soak24-iN/d2bot, ~/d2lod-live/soak24-iN/runs/<ts>-d2bot.{pid,log}
#   old layout (until 2026-10-30, still running for i4/G2): agent soak24 -> ~/d2lod-live/soak24/d2bot, .../soak24/runs/
# Env: D2_SOAK_ROOT overrides ~/d2lod-live (tests only).
soak_root() { echo "${D2_SOAK_ROOT:-$HOME/d2lod-live}"; }
soak_agent() { echo "soak24-$1"; }
soak_new_dir() { echo "$(soak_root)/$(soak_agent "$1")"; }
soak_old_dir() { echo "$(soak_root)/soak24"; }
# One bot HTTP port per instance. i4 keeps 8420 (the running G2 bot and its harness use it); the others get 8440+N
# (8421 is crashdrill's, 8433 is taken on the worker: V ss -ltn 2026-10-30). D2_BOT_PORT overrides.
soak_port() { if [ -n "${D2_BOT_PORT:-}" ]; then echo "$D2_BOT_PORT"; elif [ "$1" = i4 ]; then echo 8420; else echo $((8440 + ${1#i})); fi; }
# soak_botlogdir_cmd iN: remote shell text that prints the dir whose newest *.log is the bot log of iN: the per-instance
# runs dir when it has logs, else the old shared one (a soak started before the split).
soak_botlogdir_cmd() { local n; n="$(soak_new_dir "$1")/runs"
  echo "if ls $n/*.log >/dev/null 2>&1; then echo $n; else echo $(soak_old_dir)/runs; fi"; }
# soak_bot_pat iN LAYOUT: pgrep/pkill -f pattern for the bot binary of that layout; the old binary is shared by every
# instance, so the pattern also pins --instance.
soak_bot_pat() {
  case "$2" in
    new) echo "^$(soak_new_dir "$1")/d2bot " ;;
    old) echo "^$(soak_old_dir)/d2bot .*--instance $1( |\$)" ;;
  esac
}
