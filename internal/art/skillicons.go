package art

import (
	"fmt"
	"image"
	"regexp"
	"strconv"
)

// Skill icons: key skill-<charclass>-<cel>, e.g. skill-pal-4, the icon IconCel 4 of the class's icon file. The
// charclass is the skills.txt code, the cel is skilldesc.txt IconCel. Each DC6 holds two frames per icon (V: the
// Paladin, Sorceress and Assassin files hold 60 frames for 30 skills, read from the real install); the first of the
// pair is used (U: that it is the unpressed one, to be looked at in a browser).
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
	if 2*cel >= len(fr) {
		return nil, fmt.Errorf("%w: %s cel %d", ErrNotFound, skillFiles[m[1]], cel)
	}
	return fr[2*cel].Image(s.pal), nil
}
