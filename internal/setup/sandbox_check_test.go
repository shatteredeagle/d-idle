package setup

import (
	"strings"
	"testing"
)

func TestSandboxCheck(t *testing.T) {
	e := fakeEnv(nil)
	if r := checkSandbox(t.Context(), e, &Options{}); r.Status != Warn || !strings.Contains(r.Detail, "not found") || r.Fix == "" {
		t.Errorf("no bwrap: %+v", r)
	}
	e = fakeEnv(map[string]string{"/proc/sys/user/max_user_namespaces": "256711\n"})
	e.LookPath = func(s string) (string, error) { return "/usr/bin/" + s, nil }
	if r := checkSandbox(t.Context(), e, &Options{}); r.Status != Pass {
		t.Errorf("present: %+v", r)
	}
	e = fakeEnv(map[string]string{"/proc/sys/user/max_user_namespaces": "0\n"})
	e.LookPath = func(s string) (string, error) { return "/usr/bin/" + s, nil }
	if r := checkSandbox(t.Context(), e, &Options{}); r.Status != Warn || !strings.Contains(r.Detail, "user namespaces") {
		t.Errorf("userns off: %+v", r)
	}
	e.Getenv = func(k string) string {
		if k == "D2_SANDBOX" {
			return "0"
		}
		return ""
	}
	if r := checkSandbox(t.Context(), e, &Options{}); r.Status != Warn || !strings.Contains(r.Detail, "D2_SANDBOX=0") {
		t.Errorf("switched off: %+v", r)
	}
}
