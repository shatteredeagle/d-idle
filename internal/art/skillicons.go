package art

import (
	"fmt"
	"image"
	"regexp"
	"strconv"
)

// Skill icons: key skill-<charclass>-<cel>, e.g. skill-pal-4, the icon IconCel 4 of the class's icon file. The
// charclass is the skills.txt code, the cel is skilldesc.txt IconCel, which is already the frame number: the values
// step by two (0, 2, 4 ... 58 for the Paladin, V read from the real tables) because each icon has two frames, and the
// files hold 60 frames for 30 skills (V). The even frame is used; that it is the unpressed look is U (a browser check
// of the page showed a plausible, undimmed icon).
const skillDir = `data\global\ui\SPELLS\`

var skillKey = regexp.MustCompile(`^skill-([a-z]{3})-([0-9]{1,3})$`)

// skillFiles maps the skills.txt charclass code to the icon file.
var skillFiles = map[string]string{
	"ama": "AmSkillicon.dc6", "sor": "SoSkillicon.dc6", "nec": "NeSkillicon.dc6", "pal": "PaSkillicon.dc6",
	"bar": "BaSkillicon.dc6", "dru": "DrSkillicon.dc6", "ass": "AsSkillicon.dc6",
}

// SkillKey is the art key of a class skill icon, or "" when the class has no icon file.
func SkillKey(classCode string, cel int) string {
	if _, ok := skillFiles[classCode]; !ok || cel < 0 || cel > 999 {
		return ""
	}
	return "skill-" + classCode + "-" + strconv.Itoa(cel)
}

func validSkillKey(key string) bool {
	m := skillKey.FindStringSubmatch(key)
	if m == nil {
		return false
	}
	_, ok := skillFiles[m[1]]
	return ok
}

func (s *Store) renderSkill(key string) (image.Image, error) {
	m := skillKey.FindStringSubmatch(key)
	cel, _ := strconv.Atoi(m[2])
	fr, err := s.frames(skillDir + skillFiles[m[1]])
	if err != nil {
		return nil, err
	}
	if cel >= len(fr) {
		return nil, fmt.Errorf("%w: %s cel %d", ErrNotFound, skillFiles[m[1]], cel)
	}
	return fr[cel].Image(s.pal), nil
}
