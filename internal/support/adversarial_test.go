package support

import (
	"strings"
	"testing"
	"time"
)

// Card sec2: every known secret shape, in the places it turns up (log text, env dumps, links, JSON values). The
// bundle rule is fail closed: after Redact the secret is gone, or Scan has a finding and the bundle is not sent.
// Values are fake and built from parts.
func TestAdversarialCorpus(t *testing.T) {
	tok := "0123456789abcdef" + "0123456789abcdef"
	b64 := "aB3dE6gH9jK2mN5pQ8sT" + "1vW4yZ7_-xY0"
	home := "/ho" + "me/alice"
	ip := "192.168." + "1.20"
	cases := []struct{ name, in, secret string }{
		{"key lower dashed", `key abcd-1234-efgh-5678`, "1234-efgh"},
		{"key spaced", `key ABCD 1234 EFGH 5678`, "1234 EFGH"},
		{"passcode many words", `passcode: four random words here`, "random words"},
		{"passcode no separator", `passcode four random words here`, "random"},
		{"passcode json", `{"passcode":"four random words here"}`, "random"},
		{"passcode escaped quote", `{"passcode":"ab\"cd efgh"}`, "efgh"},
		{"pin", `pin=4821`, "4821"},
		{"env token", `D2DVIEW_TOKEN=` + b64, b64},
		{"env export", `export WATCH_TOKEN='` + b64 + `'`, b64},
		{"env password", `DB_PASSWORD=hunter2 with space`, "hunter2"},
		{"env home", `HOME=` + home, "alice"},
		{"env user", `USERNAME=alice`, "alice"},
		{"link token", `http://localhost:8891/watch?t=` + b64, b64},
		{"link token second", `GET /api/x?a=1&token=abc123def`, "abc123def"},
		{"link passcode", `GET /login?passcode=hunter2x`, "hunter2x"},
		{"link user info", `fetch https://alice:s3cr3t@box.example.org/x`, "s3cr3t"},
		{"link host", `open http://alicebox:8891/watch`, "alicebox"},
		{"json nested token", `{"a":{"b":{"watch_token":"` + b64 + `"}}}`, b64},
		{"json cookie", `{"cookie":"d2dview=abc123def456"}`, "abc123def456"},
		{"json unlabelled", `{"note":"` + tok + `"}`, tok},
		{"header cookie", `Cookie: d2dview=abc123def456ghi789`, "abc123def456"},
		{"header basic", `Authorization: Basic YWxpY2U6czNjcjN0`, "YWxpY2U6"},
		{"pem", "-----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaA\n-----END OPENSSH PRIVATE KEY-----", "b3BlbnNzaA"},
		{"private ip", `peer ` + ip + `:8420`, ip},
		{"mesh ip", `peer 100.101.102.103:51234`, "101.102"},
		{"windows home", `C:\Users\Bob Smith\Games`, "Smith"},
		{"save file", `wrote Alice.d2s`, "Alice"},
	}
	for _, c := range cases {
		out := Redact(c.in)
		if strings.Contains(strings.ToLower(out), strings.ToLower(c.secret)) && len(Scan("log.txt", out)) == 0 {
			t.Errorf("%s: %q keeps %q and the scan lets it through", c.name, out, c.secret)
		}
	}
}

// Redact and Scan are linear in the input (RE2); lines built to blow up a backtracking engine cost what their
// length costs. A ratio against a baseline of the same test, not a wall-clock ceiling (docs/process/timing-tests.md).
func TestRedactAndScanLongInputs(t *testing.T) {
	build := func(n int) string {
		rep := func(s string) string { return strings.Repeat(s, n/len(s)) }
		return strings.Join([]string{rep("a."), rep("a:"), rep("a-"), rep("ab12"), rep("1."), rep("AB12 "), rep(`\"`), rep("a@"),
			"passcode: " + rep("x "), `{"token":"` + rep(`\\`), "key " + rep("abcd 1234 "), rep("f:"), "http://" + rep("a.") + "?t=" + rep("b&t=")}, "\n")
	}
	run := func(n int) time.Duration {
		in := build(n)
		best := time.Duration(1 << 62)
		for i := 0; i < 3; i++ {
			start := time.Now()
			Scan("log.txt", Redact(in))
			if d := time.Since(start); d < best {
				best = d
			}
		}
		return best
	}
	small, big := run(2<<10), run(16<<10)
	t.Logf("2 KiB lines: %v, 16 KiB lines: %v, ratio %.1f (linear is 8)", small, big, float64(big)/float64(small))
	if big > 40*small {
		t.Errorf("8 times the input took %.0f times as long: a pattern is not linear", float64(big)/float64(small))
	}
}
