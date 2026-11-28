#!/bin/sh
# tester-report.sh: one scrubbed feedback bundle the friend can send (TASK-27, beta exit criterion 6).
# Runs INSIDE the tester container; the image has no python, so this is POSIX sh + sed + tar only.
#   docker cp tester-report.sh d2dbot:/tmp/ && docker exec d2dbot sh /tmp/tester-report.sh
#   docker cp d2dbot:/data/reports/<name>.tar.gz .        (the last line printed is the path)
# Bundle: bot.log, d2d.log, minutes.txt, deaths.jsonl, ladder/*, wizard.json, wizard.env, d2dping-state.txt, README.txt.
# Everything text goes through scrub(); the README text at the end lists what it replaces and what it does not.
# Left out on purpose: watch.token, access.json (hash), saves, keys. Read the files before you send them.
# Env: D2BOT_DATA (/data), D2D_SOCK (/run/d2d/d2d.sock), REPORT_OUT (/data/reports), REPORT_TAIL (lines per log, 20000),
#   REPORT_SCRUB_ONLY=1 (filter stdin to stdout and stop: used by the test), REPORT_NAMES (extra words to hide, | separated),
#   REPORT_SCAN_ONLY=1 (print the kinds of known secret shapes found on stdin; exit 3 when there is one).
# Second line of defence: after scrubbing, leftover() looks at every bundled file and at the file names with its own
# short list of shapes; one hit and no tarball is written (exit 3). The bundle is private to its owner (umask 077).
set -eu
umask 077

data=${D2BOT_DATA:-/data}
tail_n=${REPORT_TAIL:-20000}

# secret names as whole words inside an identifier: watch_token, apiKey, passphrase, cdkey, ...
words='token|secret|passw(or)?d|passphrase|pass[_-]?code|pwd|pin|cookie|credentials?|auth(orization)?|api[_-]?key|cd[_-]?key|cdkey|private[_-]?key|license[_-]?key|access[_-]?key|session[_-]?id|watch[_-]?key|user(name)?|hostname|computername|logname'

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
# whole-word host names of that list; its two other words are game words (an Act 1 area, a quest item) and stay
hosts='ven''us|on''yx'
h4='[0-9a-fA-F]{1,4}'
# a JSON line keeps its shape: a hidden value becomes a JSON string, an object or array value is left to the shape rules
json='/^[[:space:]]*\{/'
notjson='/^[[:space:]]*\{/!'
val="\"([^\"\\\\]|\\\\.)*\"|'[^']*'|[^[:space:],}\"'{[<][^[:space:],}\"']*"
# words whose value is a phrase: unquoted it runs to the end of the line, and the word alone labels what follows it
pass='pass[_-]?code|passphrase|passw(or)?d|pwd|pin'
passw='pass[_-]?code|passphrase|passw(or)?d'
# labels of a build identity: a hex id after one of these is kept (it is how a report names the build)
ident='commit|sha256|sha1|rev|revision|build'
key='[A-Za-z0-9]{4,6}'
nb='(^|[^0-9A-Za-z:.])'

scrub() {
	# order matters: structured secrets first, then shapes, then places and names.
	# <k:..> marks a 16+ run of letters and digits; it stays only with both a letter and a digit and is not a run stamp.
	# <LB> and <ANY> park 127.x and 0.0.0.0 while every other dotted quad becomes <ip> (twice: neighbours share a separator).
	# NUL holes and control bytes (terminal escapes) are dropped first; tab and newline stay.
	tr -d '\000-\010\013-\037\177' | sed -E "$lit
s#-----BEGIN [A-Z ]*PRIVATE KEY-----.*#<key-block>#
s#([Bb]earer|[Bb]asic)[[:space:]]+[A-Za-z0-9._~+/=-]{6,}#\\1 <token>#g
s#([:=][[:space:]]*)(true|false|null)([^A-Za-z0-9_]|\$)#\\1<B:\\2>\\3#g
$notjson s#((^|[^A-Za-z0-9])([A-Za-z0-9._-]*[_.-])?($pass)[[:space:]]*[:=][[:space:]]*)[^\"'[:space:]<].*#\\1<redacted>#I
$notjson s#((^|[^A-Za-z0-9_.-])($passw)[[:space:]]+)[^[:space:]<:=\"'].*#\\1<redacted>#I
$json s#((^|[^A-Za-z0-9_.-])($passw)[[:space:]]+)[^[:space:]<:=\"'\\\\][^\"\\\\]*#\\1<redacted>#Ig
$json s#((^|[^A-Za-z0-9])(([A-Za-z0-9._-]*[_.-])?($words)([_.-][A-Za-z0-9._-]*)?)[\"']?[[:space:]]*[:=][[:space:]]*)($val)#\\1\"<redacted>\"#Ig
$notjson s#((^|[^A-Za-z0-9])(([A-Za-z0-9._-]*[_.-])?($words)([_.-][A-Za-z0-9._-]*)?)[\"']?[[:space:]]*[:=][[:space:]]*)($val)#\\1<redacted>#Ig
s#([?&](token|t|key|auth|sig|code|access_token|pass|password|passcode)=)[^&[:space:]\"'<]+#\\1<token>#Ig
s#(://)[^/[:space:]@\"']+@#\\1<cred>@#g
s#\b[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\b#<uuid>#g
s#\b$key(-$key){2,4}\b#<d:&>#g
s#\b[A-Z0-9]{4,6}( [A-Z0-9]{4,6}){2,4}\b#<d:&>#g
s#((cd[ _-]?key|key|serial|licen[sc]e)[[:space:]:=]+)($key([ -]$key){2,4})\b#\\1<d:\\3>#Ig
s#<d:([^0-9>]*)>#\\1#g
s#<d:([0-9 ]*)>#\\1#g
s#<d:[^>]*>#<cdkey>#g
s#\b[A-Z0-9]{26}\b#<cdkey>#g
s#\b[A-Za-z0-9_-]{32,}\b#<tok:&>#g
s#<tok:([^0-9>]*)>#\\1#g
s#(($ident)[\"']?[[:space:]]*[:=]?[[:space:]]*[\"']?)<tok:([0-9a-f]{7,64})>#\\1\\3#Ig
s#<tok:[^>]*>#<token>#g
s#(^|[^A-Za-z0-9/_.+-])([A-Za-z0-9+]{16,}={0,2})#\\1<k:\\2>#g
s#<k:([^0-9>]*)>#\\1#g
s#<k:([^A-Za-z>]*)>#\\1#g
s#<k:([0-9]{8}T[0-9]{6}Z)>#\\1#g
s#(($ident)[\"']?[[:space:]]*[:=]?[[:space:]]*[\"']?)<k:([0-9a-f]{7,64})>#\\1\\3#Ig
s#<k:[^>]*>#<key>#g
s#[A-Za-z0-9._%+-]+@[A-Za-z0-9-]+(\.[A-Za-z0-9-]+)+#<email>#g
s#/Users[/]+[^/[:space:]\"'<:;)]+( [^/[:space:]\"'<:;)]+){1,2}/#<home>/#g
s#[A-Za-z]:(\\\\+|/)(Users|Documents and Settings)(\\\\+|/)+[^\\\\/\"'<:;)[:space:]]+( [^\\\\/\"'<:;)[:space:]]+){1,2}(\\\\|/|\$)#<home>\\5#g
s#(/home|/Users)[/\\\\]+[^/\\\\[:space:]\"'<:;)]+#<home>#g
s#[A-Za-z]:(\\\\+|/)(Users|home|Documents and Settings)(\\\\+|/)+[^\\\\/\"'<:;)[:space:]]+#<home>#g
s#(^|[^A-Za-z0-9._-])/(mnt|media|root|Volumes)(/[^[:space:]\"'<;)]*|([^A-Za-z0-9._/-]|\$))#\\1<path>\\4#g
s#~/(d2d-lab|labrun|d2lod-wt|d2lod-ops)#<lab>#g
s#(version[\"']?[ =:]+[\"']?v?[0-9]{1,3})\.([0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3})([^0-9.]|\$)#\\1<VD>\\2\\3#Ig
s#(^|[^0-9.])127\.([0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3})([^0-9.]|\$)#\\1<LB>\\2\\3#g
s#(^|[^0-9.])0\.0\.0\.0([^0-9.]|\$)#\\1<ANY>\\2#g
s#(^|[^0-9.])[0-9]{1,3}(\.[0-9]{1,3}){3}([^0-9.]|\$)#\\1<ip>\\3#g
s#(^|[^0-9.])[0-9]{1,3}(\.[0-9]{1,3}){3}([^0-9.]|\$)#\\1<ip>\\3#g
s#(^|[^0-9A-Za-z:])([0-9a-fA-F]{2}[:-]){5}[0-9a-fA-F]{2}([^0-9A-Za-z:]|\$)#\\1<mac>\\3#g
s#$nb($h4:){7}$h4#\\1<ip6>#g
s#$nb($h4:){1,7}(:$h4){1,7}#\\1<ip6>#g
s#$nb($h4:){1,7}:#\\1<ip6>#g
s#$nb::($h4:){0,6}$h4#\\1<ip6>#g
s#://localhost#://<LH>#g
s#(https?://|wss?://)[^/:[:space:]\"'<>?]+#\\1<host>#g
s#://<LH>#://localhost#g
s#[A-Za-z0-9-]+(\.[A-Za-z0-9-]+)*\.(lan|local|internal|home\.arpa|${dom})\b#<host>#Ig
s#\b(DESKTOP|LAPTOP|WIN)-[A-Z0-9]{5,}\b#<host>#g
s#\b($hosts)\b#<private>#Ig
s#($names${REPORT_NAMES:+|$REPORT_NAMES})#<private>#Ig
s#<LB>#127.#g
s#<ANY>#0.0.0.0#g
s#<VD>#.#g
s#<B:([a-z]+)>#\\1#g"
}

# leftover: the kinds of known secret shapes on stdin, one per line. Its patterns are written apart from scrub's on
# purpose (a rule that scrub gets wrong is not copied here). A version number is not an address.
leftover() {
	t=$(mktemp)
	sed -E "s#version[\"']?[ =:]+[\"']?v?[0-9.]+##Ig" >"$t"
	while IFS='	' read -r kind rx; do
		if grep -E -i -q -e "$rx" "$t"; then echo "$kind"; fi
	done <<RULES
private key block	-----BEGIN [A-Z ]*PRIVATE KEY
bearer token	(bearer|basic) +[a-z0-9._~+/=-]{6,}
token in a link	[?&](t|token|key|auth|passcode|password)=[a-z0-9]
passcode	(passcode|pass_code|passphrase|password|passwd)["']?[[:space:]]*[:=][[:space:]]*["']?[^<"'[:space:]]
e-mail address	[a-z0-9._%+-]+@[a-z0-9-]+\.[a-z]{2,}
home folder	(/home|/Users|[a-z]:[\\/]+Users)[\\/]+[a-z0-9]
private address	(^|[^0-9.])(10(\.[0-9]{1,3}){3}|192\.168(\.[0-9]{1,3}){2}|172\.(1[6-9]|2[0-9]|3[01])(\.[0-9]{1,3}){2}|100\.(6[4-9]|[7-9][0-9]|1[01][0-9]|12[0-7])(\.[0-9]{1,3}){2})([^0-9.]|\$)
lab name	$names|$dom
RULES
	for f in "$data/watch.token"; do
		[ -s "$f" ] || continue
		v=$(head -n 1 "$f" | tr -d '\r\n')
		[ ${#v} -ge 8 ] || continue
		if grep -F -q -e "$v" "$t"; then echo "watch token"; fi
	done
	if [ -n "${D2DVIEW_TOKEN:-}" ] && grep -F -q -e "$D2DVIEW_TOKEN" "$t"; then echo "watch token"; fi
	rm -f "$t"
}

if [ "${REPORT_SCRUB_ONLY:-0}" = 1 ]; then scrub; exit 0; fi
if [ "${REPORT_SCAN_ONLY:-0}" = 1 ]; then
	found=$(leftover)
	[ -z "$found" ] && exit 0
	printf '%s\n' "$found"
	exit 3
fi

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
answers (name, class, goals) and the game's own state line.
Replaced by a filter of shapes: values labelled token, secret, password, passcode, passphrase, key, auth, cookie, user
or hostname (a passcode or password also when it is several words); Bearer tokens; key-shaped text (16 or more
letters and digits mixed, game keys with dashes or spaces); control characters; e-mail addresses;
home folders (/home, /Users, C:\\Users); paths under /mnt, /media and /root; IPv4 (except 127.x), IPv6 and MAC
addresses; host names in links and names ending in .lan or .local.
NOT replaced: your hero's name (it is in every line on purpose), a bare computer or person name the filter cannot
know, other Windows folders, a short code with no label. A filter of shapes can miss things. A second check looked
at every file for a few known shapes; had it found one, this bundle would not exist.
Not copied at all: the watch token file, the passcode file, save files and keys.
Please open the files and read them before you send anything. If you see something private that was missed, delete that
line and tell us which kind it was.
What to send: this .tar.gz, plus three or four sentences: what you did (class, how long you played), what felt wrong or
boring, and the clock time when something went wrong. If the hero died, say what you saw on the screen. Section F of
docs/first-session.md has the three questions about teaching, orders and watching.
TXT

# refuse to write the tarball when a known shape survived (file text and file names); the text itself is not printed
bad=0
for f in $(find "$b" -type f ! -name README.txt | sort); do
	found=$(leftover <"$f")
	[ -z "$found" ] && continue
	bad=1
	printf '%s\n' "$found" | while IFS= read -r k; do echo "tester-report: NOT written: $k left in ${f#"$b"/}" >&2; done
done
found=$(find "$b" -type f | sed "s#^$b/##" | leftover)
if [ -n "$found" ]; then
	bad=1
	printf '%s\n' "$found" | while IFS= read -r k; do echo "tester-report: NOT written: $k in a file name" >&2; done
fi
if [ "$bad" = 1 ]; then
	echo "tester-report: no bundle was made. Tell us the line above (not the file); nothing left the machine." >&2
	exit 3
fi

tar -C "$work" -czf "$out/report-$stamp.tar.gz" "report-$stamp"
echo "$out/report-$stamp.tar.gz"
