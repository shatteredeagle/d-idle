#!/bin/bash
# Sourced by soak24.sh. hung_mounts [mounts-file]: network mounts (cifs/smb3/smbfs/nfs/nfs4) that do not answer a 3 s stat,
# one per line. A hung one stalls Wine at startup: Game.exe starts and never opens a window (V: G1 attempts 2 and 3, the NAS
# rebooting, /mnt/media 'Host is down'). It runs where the games run (the worker): soak24.sh sends this file over ssh.
hung_mounts() {
  awk '$3=="cifs"||$3=="smb3"||$3=="smbfs"||$3=="nfs"||$3=="nfs4"{print $2}' "${1:-/proc/mounts}" 2>/dev/null |
    while read -r m; do
      m="$(printf '%b' "$m")" # /proc/mounts octal escapes (\040 = space)
      timeout 3 stat -- "$m" >/dev/null 2>&1 || echo "$m"
    done
}
