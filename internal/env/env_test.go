package env

import "testing"

func TestDefaultTuningAndKnobs(t *testing.T) {
	_ = DefaultTuning()
	seen := map[string]bool{}
	for _, k := range Knobs() {
		if k.Key == "" || seen[k.Key] {
			t.Fatalf("bad or duplicate knob key %q", k.Key)
		}
		seen[k.Key] = true
		if k.Min > k.Max || k.Default < k.Min || k.Default > k.Max {
			t.Fatalf("knob %s: default %v outside [%v,%v]", k.Key, k.Default, k.Min, k.Max)
		}
	}
}
