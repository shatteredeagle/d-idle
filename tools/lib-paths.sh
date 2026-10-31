# Shell mirror of internal/paths (data-dir layout). POSIX sh; source it, or send its text in front of a remote
# command (live.sh, sentinel.sh do the latter, because ~/code/d2lod-bot on the worker may lag behind this checkout).
# Keep it in step with internal/paths/{paths,legacy}.go. Functions only; sourcing has no side effects.

d2_data_dir() {
  if [ -n "${D2BOT_DATA:-}" ]; then echo "$D2BOT_DATA"
  elif [ -n "${XDG_DATA_HOME:-}" ]; then echo "$XDG_DATA_HOME/d2bot"
  else echo "$HOME/.local/share/d2bot"; fi
}

# Runtime dir: locks, pids, sockets. $XDG_RUNTIME_DIR/d2bot, else <data>/run.
d2_run_dir() {
  if [ -n "${XDG_RUNTIME_DIR:-}" ]; then echo "$XDG_RUNTIME_DIR/d2bot"; else echo "$(d2_data_dir)/run"; fi
}

d2_lock_legacy() { if [ "$1" = i0 ]; then echo /tmp/d2lod-game.lock; else echo "/tmp/d2lod-game-$1.lock"; fi; }

# d2_lock_env ID: export D2LOCK_NEW (paths.Instance.GameLock) and D2LOCK_OLD (the legacy /tmp lock). A run takes both,
# in this order: flock -w S "$D2LOCK_NEW" flock -w S "$D2LOCK_OLD" CMD...  (the old one can go once no old
# binary is deployed; internal/launch.LockInstance takes the same two).
d2_lock_env() {
  mkdir -p "$(d2_run_dir)" && chmod 700 "$(d2_run_dir)" 2>/dev/null || true
  D2LOCK_NEW="$(d2_run_dir)/game-$1.lock"; D2LOCK_OLD="$(d2_lock_legacy "$1")"
  export D2LOCK_NEW D2LOCK_OLD
}

# d2_game_root ID: Wine-side root of an instance (paths.Layout.GameRoot): <data>/prefixes/<id>, or the dev-era
# ~/Games/d2lod[-inst/<id>] while that exists and the new i0 prefix does not. D2_ROOT (i0 only) wins.
d2_game_root() {
  if [ "$1" = i0 ] && [ -n "${D2_ROOT:-}" ]; then echo "$D2_ROOT"; return; fi
  if [ -d "$(d2_data_dir)/prefixes/i0" ] || [ ! -d "$HOME/Games/d2lod" ]; then echo "$(d2_data_dir)/prefixes/$1"
  elif [ "$1" = i0 ]; then echo "$HOME/Games/d2lod"
  else echo "$HOME/Games/d2lod-inst/$1"; fi
}

# d2_sentinel_log: the sentinel JSONL log; the old ~/.local/state/d2lod one while it is the only one.
d2_sentinel_log() {
  new="$(d2_data_dir)/logs/sentinel.jsonl"; old="$HOME/.local/state/d2lod/sentinel.jsonl"
  if [ ! -e "$new" ] && [ -e "$old" ]; then echo "$old"; else echo "$new"; fi
}

# d2_token_file NAME: <data>/tokens/NAME, or the old ~/.config/NAME/token while it is the only one.
d2_token_file() {
  new="$(d2_data_dir)/tokens/$1"; old="$HOME/.config/$1/token"
  if [ ! -e "$new" ] && [ -e "$old" ]; then echo "$old"; else echo "$new"; fi
}
