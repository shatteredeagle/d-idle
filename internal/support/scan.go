package support

import (
	"bytes"
	"net"
	"regexp"
	"strings"
)

// Finding is one thing the scan still sees after redaction. Context is the line with the match blanked, never the
// match itself.
type Finding struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Class   string `json:"class"`
	Context string `json:"context"`
}

// d2sMagic starts every character save file (V: the 1.14d .d2s header, docs/d2-1.14d-memory.md).
var d2sMagic = []byte{0x55, 0xAA, 0x55, 0xAA}

// The scan's own pattern set. It is written separately from the redaction rules on purpose (broader, simpler, no
// shared table), so a shape the redaction misses can still be caught here. A hit is a placeholder-free match: the
// redaction's placeholders start with "<" and no pattern below matches text that begins with it.
type scanRule struct {
	class string
	rx    *regexp.Regexp
	// ok, when set, decides from the match whether it is a hit
	hit func(sub []string) bool
}

var scanRules = []scanRule{
	{"homepath", regexp.MustCompile(`(?i)(?:/home/|/users/|[a-z]:[\\/]+(?:users|home|documents and settings)\b|%2f(?:home|users)%2f)`), nil},
	{"email", regexp.MustCompile(`[^\s@<>"']+@[^\s@<>"']+\.[A-Za-z]{2,}`), nil},
	{"ip4", regexp.MustCompile(`(?:\d{1,3}\.){3}\d{1,3}`), func(s []string) bool {
		for _, o := range strings.Split(s[0], ".") {
			if len(o) > 3 || (len(o) == 3 && o > "255") {
				return false
			}
		}
		return true
	}},
	{"ip6", regexp.MustCompile(`[0-9A-Fa-f:]{2,}:[0-9A-Fa-f:.]*`), func(s []string) bool {
		c := strings.Trim(s[0], ".")
		return net.ParseIP(c) != nil && strings.Contains(c, ":") && (strings.Contains(c, "::") || strings.Count(c, ":") >= 5)
	}},
	{"secret-assignment", regexp.MustCompile(`(?i)(?:pass|passcode|pass[_-]code|passwd|password|passphrase|pwd|pw|secret|token|apikey|api[_-]key|cookie|authoriz` + `ation|credentials?|private[_-]key|cdkey|cd[_-]key)["']?\s*[:=]\s*["']?([^<"'\s,}])`), nil},
	{"pack-path", regexp.MustCompile(`/p/[A-Za-z0-9_-]{12,}`), nil},
	{"bearer", regexp.MustCompile(`(?i)\b(?:bearer|basic)\s+[^<\s]{6,}`), nil},
	{"url-userinfo", regexp.MustCompile(`://[^/\s@<]+@`), nil},
	{"url-host", regexp.MustCompile(`(?i)https?://[^<\s/"']+`), func(s []string) bool {
		h := strings.ToLower(s[0][strings.Index(s[0], "//")+2:])
		if i := strings.LastIndexByte(h, ':'); i > 0 {
			h = h[:i]
		}
		return !publicHosts[h]
	}},
	{"hostname", regexp.MustCompile(`(?i)\b(?:[a-z0-9_-]+\.)+(?:lan|local|localdomain|internal|home|arpa|box|net|com|org|io|dev|app|cloud|test)\b`), func(s []string) bool { return !publicHosts[strings.ToLower(s[0])] }},
	{"hex-run", regexp.MustCompile(`[0-9A-Fa-f]{32,}`), nil},
	{"uuid", regexp.MustCompile(`(?i)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`), nil},
	{"key-block", regexp.MustCompile(`-----BEGIN [A-Z ]*(?:PRIVATE KEY|CERTIFICATE)`), nil},
	{"key-file", regexp.MustCompile(`(?i)[\w.-]+\.key\b|[\w.-]+\.d2s\b`), func(s []string) bool { return !strings.HasPrefix(strings.ToLower(s[0]), "<save>") }},
	{"cd-key", regexp.MustCompile(`\b[A-Z0-9]{5,6}(?:-[A-Z0-9]{4,6}){2,4}\b|\b[A-Z0-9]{26}\b`), func(s []string) bool { return strings.ContainsAny(s[0], "0123456789") }},
	{"key-word", regexp.MustCompile(`(?i)\bkey\s+[A-Za-z0-9-]{8,}`), func(s []string) bool { return strings.ContainsAny(s[0], "0123456789") }},
	{"cd-key-16", regexp.MustCompile(`\b[A-Z0-9]{16}\b`), func(s []string) bool {
		return strings.ContainsAny(s[0], "0123456789") && strings.ContainsAny(s[0], "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	}},
	{"token-run", regexp.MustCompile(`[A-Za-z0-9+/_-]{28,}`), func(s []string) bool { return tokenLike(strings.Trim(s[0], "/")) && !strings.ContainsAny(s[0], "<>") }},
}

// Scan looks for what Redact should have removed and blocks the upload on any hit. lits are the literal strings of
// this machine and the known secrets (a hit on one of them is class "local").
func Scan(file string, text string, extra ...string) []Finding {
	return ScanL(file, text, NewLiterals(extra))
}

// ScanL is Scan with precompiled literals.
func ScanL(file, text string, lits Literals) []Finding {
	var out []Finding
	for i, line := range strings.Split(text, "\n") {
		for _, rx := range lits {
			if rx.MatchString(line) {
				out = append(out, Finding{file, i + 1, "local", ctx(rx.ReplaceAllString(line, "[***]"), lits)})
			}
		}
		for _, r := range scanRules {
			for _, loc := range r.rx.FindAllStringSubmatchIndex(line, -1) {
				sub := make([]string, 0, len(loc)/2)
				for k := 0; k+1 < len(loc); k += 2 {
					if loc[k] < 0 {
						sub = append(sub, "")
					} else {
						sub = append(sub, line[loc[k]:loc[k+1]])
					}
				}
				if r.hit != nil && !r.hit(sub) {
					continue
				}
				rest := line[loc[1]:]
				if r.class == "secret-assignment" { // the value runs on (a passphrase has spaces): blank the line's rest
					rest = ""
				}
				out = append(out, Finding{file, i + 1, r.class, ctx(line[:loc[0]]+"[***]"+rest, lits)})
			}
		}
	}
	return out
}

// ctx is the line as the sheet shows it: the match blanked, everything else the redaction can still find replaced,
// so a finding never echoes a second secret of the same line.
func ctx(line string, lits Literals) string {
	line, _ = RedactN(line, lits)
	if len(line) > 160 {
		line = line[:160]
	}
	return line
}

// hasSaveMagic reports whether data looks like a character save.
func hasSaveMagic(data []byte) bool { return bytes.HasPrefix(data, d2sMagic) }
