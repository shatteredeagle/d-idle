package buildfile

import (
	"fmt"
	"strings"
)

// Share and import (agencyp3, docs/design/agency-p3.md). A build file is shared as its canonical JSON (SaveFile /
// Marshal) or the one-line chat form (Encode); both come back through ReadShared or Decode and are validated before
// anything uses them. A kolbot list is imported with ImportKolbotBuild.

// ReadShared parses and validates shared build file bytes. Parse errors are errors; validation problems are returned
// as problems with the build so the caller can show every one at once.
func ReadShared(raw []byte, rules Rules) (*Build, []Problem, error) {
	b, err := Parse(raw)
	if err != nil {
		return nil, nil, err
	}
	return b, Validate(b, rules), nil
}

// ImportKolbotBuild turns a kolbot AutoSkill/AutoStat list (skill names, not ids; see ImportKolbot) into a build
// file for class (full name or the three-letter code) and validates it under rules. The build is returned with its
// problems, so a list that does not fit the class (a Sorceress skill in a Paladin list) is reported, never guessed.
func ImportKolbotBuild(name, class string, skillsJSON, statsJSON []byte, rules Rules) (*Build, []Problem, error) {
	class = strings.ToLower(strings.TrimSpace(class))
	if full, ok := classCode[class]; ok {
		class = full
	}
	known := false
	for _, c := range ClassNames() {
		known = known || c == class
	}
	if !known {
		return nil, nil, fmt.Errorf("buildfile: kolbot import: unknown class %q", class)
	}
	goals, stats, err := ImportKolbot(skillsJSON, statsJSON)
	if err != nil {
		return nil, nil, err
	}
	b := &Build{Schema: Schema, Name: name, Class: class, Source: "kolbot-list", Skills: goals, Stats: stats}
	return b, Validate(b, rules), nil
}
