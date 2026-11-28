// Package support builds, redacts, scans and sends the support bundle (docs/support-bundle.md). The bundle is an
// allowlist of named files; every text file goes through Redact and then Scan, and a bundle with a finding is never
// sent (fail closed). Scan has its own pattern set (scan.go), so it can catch what Redact misses.
package support

import (
	"net"
	"regexp"
	"strings"
	"unicode"
)

// rule is one redaction step. fix, when set, decides per match: the replacement and whether to change it at all.
type rule struct {
	class string
	rx    *regexp.Regexp
	repl  string
	fix   func(m string) (string, bool)
}

// secret names, as whole words inside identifiers (watch_token, apiKey, passPhrase, x-auth-header ...)
const secretWords = `(?:token|secret|passw(?:or)?d|passphrase|pass[_-]?code|pass|pwd|pw|pin|user(?:name)?|hostname|cookie|credentials?|auth(?:orization)?|api[_-]?key|cd[_-]?key|cdkey|private[_-]?key|license[_-]?key|access[_-]?key|session|key)`

var publicHosts = map[string]bool{"github.com": true, "api.github.com": true, "raw.githubusercontent.com": true, "ghcr.io": true, "docker.io": true,
	"localhost": true, "example.com": true, "example.org": true, "example.net": true}

var secretRx = regexp.MustCompile(`(?i)(["']?\b(?:[\w.-]*[_.-])?` + secretWords + `(?:[_.-][\w.-]*)?["']?\s*[:=]\s*)("(?:[^"\\\n]|\\.)*"|'[^'\n]*'|[^\n]*)`)

// the word alone labels a phrase ("passcode four random words"); it runs to the next quote or the end of the line
var barePassRx = regexp.MustCompile(`(?i)\b(pass[_-]?code|passphrase|password)(\s+)([^\s<:="'][^"\n]*)`)
var keywordRx = regexp.MustCompile(`(?i)\b((?:cd[ _-]?)?key\s+)([A-Za-z0-9-]{8,})\b`)

var rules = []rule{
	{"pem", regexp.MustCompile(`(?s)-----BEGIN [A-Z ]*PRIVATE KEY-----.*?(?:-----END [A-Z ]*PRIVATE KEY-----|$)`), "<key-block>", nil},
	{"bearer", regexp.MustCompile(`(?i)\b(?:bearer|basic)\s+[A-Za-z0-9._~+/=-]{6,}`), "Bearer <token>", nil},
	// name = value, name: value, "name": "value": a quoted value goes whole, an unquoted one to the end of the line
	// (passphrases have spaces); the name stays
	{"secret", secretRx, "", redactSecret},
	{"barepass", barePassRx, "${1}${2}<secret>", nil},
	{"keyword", keywordRx, "", keywordFix},
	{"urluser", regexp.MustCompile(`(?i)(://)[^/\s@"']+@`), "${1}<cred>@", nil},
	{"urltoken", regexp.MustCompile(`(?i)([?&](?:token|t|key|auth|sig|code|access_token|pass|password|passcode)=)[^&\s"'<]+`), `${1}<token>`, nil},
	{"urlpath", regexp.MustCompile(`/p/[A-Za-z0-9_-]{12,}`), `/p/<token>`, nil},
	{"uuid", regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`), "<uuid>", nil},
	{"cdkey", regexp.MustCompile(`\b[A-Z0-9]{4,6}(?:-[A-Z0-9]{4,6}){2,4}\b|\b[A-Z0-9]{26}\b|\b[A-Z0-9]{16}\b`), "<cdkey>", func(m string) (string, bool) {
		return "<cdkey>", strings.ContainsAny(m, "0123456789") && strings.ContainsAny(m, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	}},
	// any case with dashes, upper case with spaces; a digit and a letter are both needed (words and number columns stay)
	{"cdkey-loose", regexp.MustCompile(`\b[A-Za-z0-9]{4,6}(?:-[A-Za-z0-9]{4,6}){2,4}\b|\b[A-Z0-9]{4,6}(?: [A-Z0-9]{4,6}){2,4}\b`), "<cdkey>", func(m string) (string, bool) {
		return "<cdkey>", strings.ContainsAny(m, "0123456789") && strings.IndexFunc(m, unicode.IsLetter) >= 0
	}},
	{"keyfile", regexp.MustCompile(`(?i)[\w.-]*\.key\b`), "<keyfile>", nil},
	{"savefile", regexp.MustCompile(`(?i)[\w.-]+\.d2s\b`), "<save>.d2s", func(m string) (string, bool) { return "<save>.d2s", !strings.EqualFold(m, "<save>.d2s") }},
	{"email", regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9-]+(?:\.[A-Za-z0-9-]+)+`), "<email>", nil},
	// home directories: Linux, macOS, Windows (one or two backslashes, either slash), Wine's Z:\home, URL-encoded
	{"homepath", regexp.MustCompile(`(?i)(?:(?:/home|/Users)[/\\]+[^/\\\s"'<:;)]+|[A-Z]:(?:\\+|/)(?:Users|home|Documents and Settings)(?:\\+|/)+[^\\/"'<:;)\r\n]+|%2F(?:home|Users)%2F[^%&\s"'<]+)`), "<home>", nil},
	{"ip6", regexp.MustCompile(`(?:^|[^0-9A-Za-z:.])([0-9A-Fa-f:]*:[0-9A-Fa-f:.]*:[0-9A-Fa-f:.]*)`), "", func(m string) (string, bool) {
		i := strings.IndexFunc(m, func(r rune) bool {
			return r == ':' || (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
		})
		if i < 0 {
			return m, false
		}
		lead, c := m[:i], strings.TrimRight(m[i:], ".")
		if ip := net.ParseIP(c); ip != nil && strings.Contains(c, ":") && (strings.Contains(c, "::") || strings.Count(c, ":") >= 5) {
			return lead + "<ip6>", true
		}
		return m, false
	}},
	{"ip", regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`), "<ip>", func(m string) (string, bool) {
		for _, o := range strings.Split(m, ".") {
			if len(o) > 3 || (len(o) == 3 && o > "255") {
				return m, false
			}
		}
		return "<ip>", true
	}},
	// the host of any URL (single-label names like a PC name too), except a few public ones
	{"urlhost", regexp.MustCompile(`(?i)(https?://)([^/\s:"'?#<]+)`), "", func(m string) (string, bool) {
		i := strings.Index(m, "//") + 2
		if h := strings.ToLower(m[i:]); publicHosts[h] || strings.HasPrefix(h, "<") {
			return m, false
		}
		return m[:i] + "<host>", true
	}},
	{"host", regexp.MustCompile(`(?i)\b(?:[a-z0-9_-]+\.)+(?:lan|local|localdomain|internal|home|home\.arpa|box|ts\.net|com|net|org|io|dev|app|co|uk|de|fr|xyz|me|tv|gg|info|cloud|test|example|invalid)\b`), "<host>", func(m string) (string, bool) {
		return "<host>", !publicHosts[strings.ToLower(m)]
	}},
	{"hex", regexp.MustCompile(`\b[0-9a-fA-F]{32,}\b`), "<hex>", nil},
	{"token", regexp.MustCompile(`[A-Za-z0-9+/_-]{28,}={0,2}`), "<token>", func(m string) (string, bool) { return "<token>", tokenLike(m) }},
}

func redactSecret(m string) (string, bool) {
	sub := secretRx.FindStringSubmatch(m)
	if sub == nil || strings.HasPrefix(strings.Trim(sub[2], `"' `), "<") {
		return m, false
	}
	return sub[1] + `"<secret>"`, true
}

func keywordFix(m string) (string, bool) {
	sub := keywordRx.FindStringSubmatch(m)
	if sub == nil || !strings.ContainsAny(sub[2], "0123456789") {
		return m, false
	}
	return sub[1] + "<secret>", true
}

// tokenLike: a long run with a digit and a letter, or mixed case with a symbol (the watch token is base64url, so a
// digit-free one has _ or - in it with near certainty); plain lowercase words and paths do not count.
func tokenLike(s string) bool {
	var lo, up, dg, sy int
	for _, c := range s {
		switch {
		case c >= 'a' && c <= 'z':
			lo++
		case c >= 'A' && c <= 'Z':
			up++
		case c >= '0' && c <= '9':
			dg++
		default:
			sy++
		}
	}
	if len(s) >= 28 && dg > 0 && lo+up > 0 && (up > 0 && lo > 0) {
		return true
	}
	return len(s) >= 32 && lo > 0 && up > 0 && (dg > 0 || sy > 0)
}

// Literals are the strings of this machine and the known secrets (tokens, home directory, host names), compiled once.
type Literals []*regexp.Regexp

// NewLiterals compiles the extra strings: a plain word only as a whole word, case-insensitive; anything else as is,
// plus its JSON form (doubled backslashes) and forward-slash form. Strings shorter than 3 characters are ignored.
func NewLiterals(extra []string) Literals {
	var out Literals
	seen := map[string]bool{}
	add := func(s string) {
		if len(s) < 3 || seen[s] {
			return
		}
		seen[s] = true
		q := regexp.QuoteMeta(s)
		if wordLit.MatchString(s) {
			q = `(?i)\b` + q + `\b`
		}
		out = append(out, regexp.MustCompile(q))
	}
	for _, e := range extra {
		e = strings.TrimSpace(e)
		add(e)
		if strings.Contains(e, `\`) {
			add(strings.ReplaceAll(e, `\`, `\\`))
			add(strings.ReplaceAll(e, `\`, `/`))
		}
	}
	return out
}

var wordLit = regexp.MustCompile(`^[\w-]+$`)

// Redact replaces secrets, personal data and machine identity with placeholders.
func Redact(s string, extra ...string) string {
	out, _ := RedactN(s, NewLiterals(extra))
	return out
}

// RedactN is Redact with precompiled literals and the number of replacements made.
func RedactN(s string, lits Literals) (string, int) {
	n := 0
	for _, rx := range lits {
		s = rx.ReplaceAllStringFunc(s, func(string) string { n++; return "<local>" })
	}
	for _, r := range rules {
		s = r.rx.ReplaceAllStringFunc(s, func(m string) string {
			if r.fix != nil {
				out, ok := r.fix(m)
				if ok {
					n++
				}
				return map[bool]string{true: out, false: m}[ok]
			}
			n++
			return r.rx.ReplaceAllString(m, r.repl)
		})
	}
	return s, n
}
