#!/bin/sh
# One-time (idempotent) volume setup, run as the image user: Wine prefix at /data/proton/pfx, game dir link
# to the read-only /game mount, D2 registry values, helper dir. Needs no network.
set -e
export WINEPREFIX=/data/proton/pfx WINEDEBUG=-all
mkdir -p /data/proton /data/logs /data/helper
if [ ! -f "$WINEPREFIX/system.reg" ]; then wineboot --init; wineserver -w; fi
PF="$WINEPREFIX/drive_c/Program Files (x86)"
mkdir -p "$PF"
[ -L "$PF/Diablo II" ] || ln -sfn /game "$PF/Diablo II"
U="$(ls "$WINEPREFIX/drive_c/users" | grep -v Public | head -1)"
mkdir -p "$WINEPREFIX/drive_c/users/$U/Saved Games/Diablo II"
cat > /tmp/d2.reg <<REG
Windows Registry Editor Version 5.00

[HKEY_CURRENT_USER\\Software\\Blizzard Entertainment\\Diablo II]
"CmdLine"="-w -ns -skiptobnet"
"UseCmdLine"=dword:00000000
"GamePath"="C:\\\\Program Files (x86)\\\\Diablo II\\\\Diablo II.exe"
"InstallPath"="C:\\\\Program Files (x86)\\\\Diablo II\\\\"
"NewSavePath"="C:\\\\users\\\\$U\\\\Saved Games\\\\Diablo II\\\\"
"Music Volume"=dword:00000000
"NPC Speech"=dword:00000002
"Contrast"=dword:00000064

[HKEY_CURRENT_USER\\Software\\Blizzard Entertainment\\Diablo II\\VideoConfig]
"Render"=dword:00000001
"DirectDrawDevice0"=dword:00000001

[HKEY_CURRENT_USER\\Software\\Wine\\Drivers]
"Audio"=""

[HKEY_LOCAL_MACHINE\\Software\\Wow6432Node\\Blizzard Entertainment\\Diablo II]
"GamePath"="C:\\\\Program Files (x86)\\\\Diablo II\\\\Diablo II.exe"
"InstallPath"="C:\\\\Program Files (x86)\\\\Diablo II\\\\"
REG
wine regedit /tmp/d2.reg
wine reg query "HKCU\\Software\\Blizzard Entertainment\\Diablo II" /v NewSavePath
wineserver -w
echo d2init done
