#!/bin/bash
# Pure refusal logic of tools/soak/fresh.sh, sourced by it and by fresh_test.sh. No ssh, no side effects.
FRESH_CLASSES="amazon assassin barbarian druid necromancer paladin sorceress"

# fresh_check_args INST CLASS NAME NAMES_FILE: prints the reason and returns 1 when the request must be refused.
fresh_check_args() {
  local inst="$1" class="$2" name="$3" names="$4"
  case "$inst" in i[1-9]) ;; *) echo "instance must be i1..i9 (never i0, the user's own game): '$inst'"; return 1 ;; esac
  case " $FRESH_CLASSES " in *" $class "*) ;; *) echo "unknown class '$class' (one of: $FRESH_CLASSES)"; return 1 ;; esac
  # same rules as launch.ValidCharName: 2-15 letters, one inner - or _
  if ! printf '%s' "$name" | grep -Eq '^[A-Za-z][A-Za-z]*([-_][A-Za-z]+)?$' || [ "${#name}" -lt 2 ] || [ "${#name}" -gt 15 ]; then
    echo "invalid character name '$name'"; return 1
  fi
  [ -r "$names" ] || { echo "throwaway list $names not readable"; return 1; }
  if ! grep -v '^[[:space:]]*#' "$names" | tr -d '\r' | grep -qix -- "$name"; then
    echo "'$name' is not on the throwaway list ($names): refusing to create it"; return 1
  fi
  return 0
}

# fresh_busy PROBE: PROBE is the token lines the remote probe printed. Prints one reason per busy token, returns 1 if any.
fresh_busy() {
  local res="$1" t rc=0
  for t in NOTICE:"/tmp/d2lod-live.notice exists (a game restart is in progress)" \
           SOAKUNIT:"a d2soak unit is running for this instance" BOT:"a d2bot is running for this instance" \
           LOCKNEW:"the instance lock is held" LOCKOLD:"the legacy instance lock is held" \
           GAME:"a Game.exe of this instance is already running (not ours to stop)" \
           SAVE:"a character file with this name already exists (never overwritten)"; do
    if printf '%s\n' "$res" | grep -qx "${t%%:*}"; then echo "${t#*:}"; rc=1; fi
  done
  return $rc
}

# fresh_soak_line INST NAME: the line to run next (the script never starts the soak).
fresh_soak_line() { echo "tools/soak/soak24.sh start $1 --gate --char $2"; }
