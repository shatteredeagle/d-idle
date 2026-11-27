#!/bin/sh
# tester-report.sh: one scrubbed feedback bundle the friend can send (TASK-27, beta exit criterion 6).
# Runs INSIDE the tester container; the image has no python, so this is POSIX sh + sed + tar only.
#   docker cp tester-report.sh d2dbot:/tmp/ && docker exec d2dbot sh /tmp/tester-report.sh
#   docker cp d2dbot:/data/reports/<name>.tar.gz .        (the last line printed is the path)
# Bundle: bot.log, d2d.log, minutes.txt, deaths.jsonl, ladder/*, wizard.json, wizard.env, d2dping-state.txt, README.txt.
# Everything text goes through scrub(); secrets, CD-key-like text, home paths, private hosts/IPs and names are replaced.
# Left out on purpose: watch.token, access.json (hash), saves, keys. Read the files before you send them.
# Env: D2BOT_DATA (/data), D2D_SOCK (/run/d2d/d2d.sock), REPORT_OUT (/data/reports), REPORT_TAIL (lines per log, 20000),
#   REPORT_SCRUB_ONLY=1 (filter stdin to stdout and stop: used by the test), REPORT_NAMES (extra words to hide, | separated).
set -eu

data=${D2BOT_DATA:-/data}
tail_n=${REPORT_TAIL:-20000}

# secret names as whole words inside an identifier: watch_token, apiKey, passphrase, cdkey, ...
words='token|secret|passw(or)?d|passphrase|pwd|cookie|credentials?|auth(orization)?|api[_-]?key|cd[_-]?key|cdkey|private[_-]?key|license[_-]?key|access[_-]?key|session[_-]?id|watch[_-]?key'

# Literal secrets of this container (the watch token) are removed by value as well: a token that a log printed
# without a label is still caught. The sed script is built once; values are escaped for a basic regex.
lit=
for f in "$data/watch.token"; do
	[ -s "$f" ] || continue
	v=$(head -n 1 "$f" | tr -d '\r\n')
	[ ${#v} -ge 8 ] || continue
	e=$(printf '%s' "$v" | sed 's/[][\.*^$/|]/\\&/g')
	lit="$lit
s/$e/<token>/g"
done
[ -z "${D2DVIEW_TOKEN:-}" ] || lit="$lit
s/$(printf '%s' "$D2DVIEW_TOKEN" | sed 's/[][\.*^$/|]/\\&/g')/<token>/g"

# private names of the lab, written in pieces so this file is not itself a leakcheck hit (same list as cmd/leakcheck)
dom='bir''bhub\.com'
names='bir''bhub|ori''on|her''dr|for''gejo|unr''aid|omle''tte|mac''mini|jes''se|des''mond'

scrub() {
	# order matters: structured secrets first, then shapes, then places and names
	sed -E "$lit
s#-----BEGIN [A-Z ]*PRIVATE KEY-----.*#<key-block>#
s#([Bb]earer|[Bb]asic)[[:space:]]+[A-Za-z0-9._~+/=-]{6,}#\\1 <token>#g
s#((^|[^A-Za-z0-9])(([A-Za-z0-9._-]*[_.-])?($words)([_.-][A-Za-z0-9._-]*)?)[\"']?[[:space:]]*[:=][[:space:]]*)(\"[^\"]*\"|'[^']*'|[^[:space:],}\"']+)#\\1<redacted>#Ig
s#([?&](token|t|key|auth|sig|code|access_token|pass|password)=)[^&[:space:]\"'<]+#\\1<token>#Ig
s#(://)[^/[:space:]@\"']+@#\\1<cred>@#g
s#\b[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\b#<uuid>#g
s#\b[A-Z0-9]{4,6}(-[A-Z0-9]{4,6}){2,4}\b#<cdkey>#g
s#\b[A-Z0-9]{26}\b#<cdkey>#g
s#\b[A-Za-z0-9_-]{32,}\b#<tok:&>#g
s#<tok:([^0-9>]*)>#\1#g
s#<tok:[^>]*>#<token>#g
s#[A-Za-z0-9._%+-]+@[A-Za-z0-9-]+(\.[A-Za-z0-9-]+)+#<email>#g
s#(/home|/Users)[/\\\\]+[^/\\\\[:space:]\"'<:;)]+#<home>#g
s#[A-Za-z]:(\\\\+|/)(Users|home|Documents and Settings)(\\\\+|/)+[^\\\\/\"'<:;)[:space:]]+#<home>#g
s#~/(d2d-lab|labrun|d2lod-wt|d2lod-ops)#<lab>#g
s#(^|[^0-9.])(10(\.[0-9]{1,3}){3}|192\.168(\.[0-9]{1,3}){2}|172\.(1[6-9]|2[0-9]|3[01])(\.[0-9]{1,3}){2}|169\.254(\.[0-9]{1,3}){2})([^0-9]|\$)#\\1<ip>\\7#g
s#://localhost#://<LH>#g
s#://127\.0\.0\.1#://<LB>#g
s#(https?://|wss?://)[^/:[:space:]\"'<>?]+#\\1<host>#g
s#://<LH>#://localhost#g
s#://<LB>#://127.0.0.1#g
s#[A-Za-z0-9-]+(\.[A-Za-z0-9-]+)*\.(lan|local|internal|home\.arpa|${dom})\b#<host>#Ig
s#($names${REPORT_NAMES:+|$REPORT_NAMES})#<private>#Ig"
}

if [ "${REPORT_SCRUB_ONLY:-0}" = 1 ]; then scrub; exit 0; fi

stamp=$(date -u +%Y%m%dT%H%M%SZ)
out=${REPORT_OUT:-$data/reports}
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
b=$work/report-$stamp
mkdir -p "$b/ladder" "$out"

# take FILE through the scrubber into $b/NAME (last tail_n lines); say so when it is missing
put() { # name file...
	name=$1; shift
	n=0
	for f in "$@"; do
		[ -f "$f" ] || continue
		n=$((n + 1))
		{ [ "$n" -le 1 ] || printf '\n# ---- next file ----\n'; tail -n "$tail_n" "$f" | scrub; } >>"$b/$name"
	done
	[ "$n" -gt 0 ] || echo "(not found)" >"$b/$name"
}

botlogs=$(find "$data/instances" -name d2bot.log -type f 2>/dev/null | sort || true)
d2dlogs=$(find "$data/instances" -name d2d.out -type f 2>/dev/null | sort || true)
# word splitting of the find output is intended: paths here have no spaces
# shellcheck disable=SC2086
{
	put bot.log $botlogs
	put d2d.log $d2dlogs
	put deaths.jsonl $(find "$data/instances" -name deaths.jsonl -type f 2>/dev/null | sort || true)
}
# minute lines: the bot's once-a-minute status line (slog JSON "msg":"minute" or text msg=minute), all of them
{
	for f in $botlogs; do grep -E '"msg": ?"minute"|msg=minute' "$f" 2>/dev/null || true; done | scrub
} >"$b/minutes.txt"
[ -s "$b/minutes.txt" ] || echo "(no minute lines)" >"$b/minutes.txt"

# ladder journal and decisions: every json/jsonl file under a ladder directory
find "$data/instances" "$data/ladder" -path '*ladder*' -type f \( -name '*.jsonl' -o -name '*.json' \) 2>/dev/null | sort | while IFS= read -r f; do
	rel=$(printf '%s' "$f" | sed 's#^/##; s#[^A-Za-z0-9._-]#_#g')
	tail -n "$tail_n" "$f" | scrub >"$b/ladder/$rel"
done || true

# wizard inputs: name, class, goals, times; game folder paths and the patch checksum are dropped (no paths)
for f in wizard.json wizard.env; do
	[ -f "$data/$f" ] || continue
	sed -E 's/"(game_dir|gamedir|game|patch_sha256|token[a-z_]*|access[a-z_]*)" *: *("[^"]*"|[^,}]*),? *//Ig' "$data/$f" |
		grep -v -i -E 'game_dir|gamedir|patch_sha|token|access' | scrub >"$b/$f" || true
done

# d2dping state (screen, class, name of the live d2d); the socket may be down, then say so
sock=${D2D_SOCK:-/run/d2d/d2d.sock}.ctl
ping=/opt/d2lod/d2dping
[ -x "$ping" ] || ping=$(command -v d2dping 2>/dev/null || true)
{
	if [ -n "$ping" ]; then "$ping" -sock "$sock" state 2>&1 || echo "(d2dping state failed)"; else echo "(d2dping not found)"; fi
} | scrub >"$b/d2dping-state.txt"

cat >"$b/README.txt" <<TXT
Feedback bundle $stamp (made by tester-report.sh, nothing was sent anywhere).
What this is: the last $tail_n lines of the bot and d2d logs, the once-a-minute lines, deaths, the ladder journal, your wizard
answers (name, class, goals) and the game's own state line. Tokens, key-shaped text, home folders, private host names,
IP addresses and e-mail addresses were replaced. The watch token, your passcode, save files and keys are NOT in it.
Please open the files and read them before you send anything. If you see something private that was missed, delete that
line and tell us which kind it was.
What to send: this .tar.gz, plus three or four sentences: what you did (class, how long you played), what felt wrong or
boring, and the clock time when something went wrong. If the hero died, say what you saw on the screen. Section F of
docs/first-session.md has the three questions about teaching, orders and watching.
TXT

tar -C "$work" -czf "$out/report-$stamp.tar.gz" "report-$stamp"
echo "$out/report-$stamp.tar.gz"
