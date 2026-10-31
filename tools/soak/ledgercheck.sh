#!/bin/bash
# Cheat-ledger helpers of soak24.sh preflight (sourced; pure). A cheated character never counts toward a gate (docs/cheats.md).

# ledger_count LEDGER NAME: number of SUCCESSFUL ledger lines naming the character, case-insensitive. Refused attempts
# (success:false, e.g. the saveguard's god-mode restore on a character that is not on the allow-list) changed nothing and do not count.
ledger_count() {
  [ -r "$1" ] || { echo 0; return; }
  grep -iE "\"character\" *: *\"$2\"" "$1" | grep -ciE "\"success\" *: *true" || true
}
