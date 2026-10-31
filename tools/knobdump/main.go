// Command knobdump prints every env tuning knob (default, range, profile values, reader count) as Markdown tables
// for docs/knobs.md. It only reads the code: run from the repo root, `go run ./tools/knobdump`.
// Reader count is a heuristic: references to `.Field` in the group's own package directory plus `.Group.Field`
// anywhere else, tests and tuning definition files excluded. 0 means "probably dead"; verify by hand.
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/resource"
)

var groupDir = map[string]string{"nav": "internal/nav", "combat": "internal/combat", "town": "internal/town",
	"loot": "internal/loot", "bot": "internal/bot", "resource": "internal/resource", "boss": "internal/boss", "merc": "internal/merc",
	"campaign": "internal/task", "route": "internal/task"}

// annotated readers: live knobs the field-reference heuristic cannot count (read through accessor methods or a local copy).
var annotated = map[string]string{"nav.pollMs": "accessor", "nav.replanMs": "accessor", "nav.stuckMs": "accessor",
	"nav.walkTimeoutSec": "accessor", "nav.exploreTimeoutSec": "accessor", "nav.loadWaitSec": "accessor",
	"nav.panelWaitMs": "accessor", "bot.cubeMaxJobs": "1", "bot.cubeRuneCap": "1", "bot.cubeGemCap": "1", // read as env.BotTuning fields in internal/town
	"resource.waitMaxSec": "local var", "campaign.denMinLevelCaster": "1", "campaign.denMinLevelMelee": "1"} // read in bot.denFloor through a local copy

func main() {
	missing := render(os.Stdout)
	fmt.Println("\nFields without a knob entry:")
	for _, m := range missing {
		fmt.Println("- " + m)
	}
}

// render writes the per-group knob tables to w and returns the Tuning fields that have no Knob entry.
func render(w io.Writer) (missing []string) {
	env.RegisterKnobs(resource.Knobs)
	files := map[string]string{}
	filepath.Walk(".", func(p string, i os.FileInfo, err error) error {
		if err != nil || i.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") ||
			strings.HasPrefix(p, "internal/env/") || strings.HasPrefix(p, "tools/knobdump") ||
			p == "internal/bot/profile.go" || p == "internal/resource/tuning.go" {
			return nil
		}
		b, _ := os.ReadFile(p)
		files[p] = string(b)
		return nil
	})
	count := func(group, field string) int {
		own := regexp.MustCompile(`\.` + field + `\b`)
		any := regexp.MustCompile(`\.` + strings.Title(group) + `\.` + field + `\b`)
		n := 0
		for p, s := range files {
			if strings.HasPrefix(p, groupDir[group]+"/") {
				n += len(own.FindAllString(s, -1))
			} else {
				n += len(any.FindAllString(s, -1))
			}
		}
		return n
	}
	var t env.Tuning
	rookie := bot.Profiles()[4]
	rk := env.DefaultTuning()
	rookie.Apply(&rk)
	seen := map[string]bool{}
	last := ""
	for _, k := range env.Knobs() {
		grp, name, _ := strings.Cut(k.Key, ".")
		if grp != last {
			fmt.Fprintf(w, "\n### %s.*\n\n| key | default | range | unit | worst (rookie) | readers | what it changes |\n|---|---|---|---|---|---|---|\n", grp)
			last = grp
		}
		seen[strings.ToLower(k.Key)] = true
		fv := reflect.ValueOf(&t).Elem().FieldByNameFunc(func(s string) bool { return strings.EqualFold(s, grp) })
		fld, _ := fv.Type().FieldByNameFunc(func(s string) bool { return strings.EqualFold(s, name) })
		v, _ := bot.GetKnob(&rk, k)
		readers := fmt.Sprint(count(grp, fld.Name))
		if a, ok := annotated[k.Key]; ok {
			readers = a
		}
		fmt.Fprintf(w, "| %s | %.4g | %.4g..%.4g | %s | %.4g | %s | %s |\n", k.Key, k.Default, k.Min, k.Max, k.Unit, v, readers, k.Desc)
	}
	rt := reflect.TypeOf(t)
	for i := 0; i < rt.NumField(); i++ {
		g := rt.Field(i)
		if g.Type.Kind() != reflect.Struct {
			continue
		}
		for j := 0; j < g.Type.NumField(); j++ {
			if !seen[strings.ToLower(g.Name+"."+g.Type.Field(j).Name)] {
				missing = append(missing, fmt.Sprintf("%s.%s (readers %d)", strings.ToLower(g.Name), g.Type.Field(j).Name, count(strings.ToLower(g.Name), g.Type.Field(j).Name)))
			}
		}
	}
	return missing
}
