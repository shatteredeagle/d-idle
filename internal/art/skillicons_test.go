package art

import "testing"

func TestSkillKeys(t *testing.T) {
	for _, k := range []string{"skill-pal-0", "skill-sor-29", "skill-ass-12"} {
		if !ValidKey(k) {
			t.Errorf("%s should be valid", k)
		}
	}
	for _, k := range []string{"skill-xxx-1", "skill-pal-", "skill-pal-1000", "skill-pal-1/../x", "skill-PAL-1", "skill--1"} {
		if ValidKey(k) {
			t.Errorf("%s should be refused", k)
		}
	}
	if SkillKey("pal", 3) != "skill-pal-3" || SkillKey("zzz", 3) != "" || SkillKey("pal", -1) != "" {
		t.Error("SkillKey")
	}
}

func TestRealSkillIcon(t *testing.T) {
	s, err := Open(realInstall(t), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for _, k := range []string{"skill-pal-0", "skill-sor-0", "skill-pal-58", "skill-ass-12"} {
		if b, err := s.PNG(k); err != nil || len(b) < 100 {
			t.Errorf("%s: %v (%d bytes)", k, err, len(b))
		}
	}
	if _, err := s.PNG("skill-pal-60"); err == nil {
		t.Error("a cel past the file must be not found")
	}
}
