package task

import (
	"os"
	"regexp"
	"testing"

	"d2lod-bot/internal/questboard"
	"d2lod-bot/internal/questspec"
)

// TestGuardCodesAreReasons is questspec rule 14: every reason code the guards of gate.go can return (the code argument
// of refusal, the code of gatedCode, a Fault literal's Code) is a row of the reason table with a remedy.
func TestGuardCodesAreReasons(t *testing.T) {
	src, err := os.ReadFile("gate.go")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	var codes []string
	for _, re := range []*regexp.Regexp{
		regexp.MustCompile(`refusal\(w, *"([a-z_]+)"`),
		regexp.MustCompile(`Code: *"([a-z_]+)"`),
		regexp.MustCompile(`return "([a-z_]+)"`),
	} {
		for _, m := range re.FindAllStringSubmatch(string(src), -1) {
			if !seen[m[1]] {
				seen[m[1]] = true
				codes = append(codes, m[1])
			}
		}
	}
	if len(codes) < 4 {
		t.Fatalf("found only %v: the scan patterns no longer match gate.go", codes)
	}
	known := func(c string) (string, bool) {
		r, ok := questboard.ReasonOf(c)
		return string(r.Remedy), ok
	}
	for _, f := range questspec.CheckGuardCodes("task/gate.go", codes, known) {
		t.Error(f)
	}
}
