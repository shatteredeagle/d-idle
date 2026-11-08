package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"d2lod-bot/internal/plandata"
)

func planFile(build, class string) *fstest.MapFile {
	c := ""
	if class != "" {
		c = `"class":"` + class + `",`
	}
	return &fstest.MapFile{Data: []byte(`{"build":"` + build + `",` + c + `"rows":[]}`)}
}

// The class list derives from the plan files: a new normal30 table adds its class, a class with no table is absent.
func TestClassListFollowsPlanData(t *testing.T) {
	ids := func(tb *Tables) string {
		var s []string
		for _, c := range tb.Classes {
			s = append(s, c.ID+":"+c.Table)
		}
		return strings.Join(s, ",")
	}
	plan := fstest.MapFS{"build-necro_normal30.json": planFile("necromancer_x_normal30", "necromancer")}
	tb, err := LoadTablesWith([]byte(strings.Replace(smallTables, `"default_class": "sorceress"`, `"default_class": "necromancer"`, 1)), plan)
	if err != nil {
		t.Fatal(err)
	}
	if got := ids(tb); got != "necromancer:normal30" {
		t.Fatalf("before: %s", got)
	}
	plan["build-sorc_fallback.json"] = planFile("lightning", "")                  // no class key: the Sorceress
	plan["build-fallback-lightning-v2.json"] = planFile("lightning_chain_v2", "") // fallback by file name
	plan["build-witch_normal30.json"] = planFile("witch_normal30", "witch")
	tb, err = LoadTablesWith([]byte(smallTables), plan)
	if err != nil {
		t.Fatal(err)
	}
	if got := ids(tb); got != "sorceress:fallback,necromancer:normal30,witch:normal30" {
		t.Fatalf("after: %s", got)
	}
	// a normal30 table wins over a fallback one for the same class
	plan["build-sorc_normal30.json"] = planFile("sorceress_normal30", "sorceress")
	tb, _ = LoadTablesWith([]byte(smallTables), plan)
	if c, _ := tb.Class("sorceress"); c.Table != "normal30" || c.Note != "" {
		t.Errorf("sorceress with a normal30 table: %+v", c)
	}
	if _, err := LoadTablesWith([]byte(smallTables), fstest.MapFS{}); err == nil {
		t.Error("an empty plan accepted (default class has no table)")
	}
}

// Every class the plan can level has a d2d UI index in tables.json, so no table is silently unreachable.
func TestEveryPlanClassIsInCatalog(t *testing.T) {
	ids, err := planClasses(plandata.FS)
	if err != nil {
		t.Fatal(err)
	}
	tb := DefaultTables()
	for _, id := range ids {
		if _, err := tb.Class(id); err != nil {
			t.Errorf("plan class %s: %v", id, err)
		}
	}
	if len(ids) != len(tb.Classes) {
		t.Errorf("plan classes %v, wizard classes %d", ids, len(tb.Classes))
	}
}

func TestPaladinPreselectedSorceressFallbackNote(t *testing.T) {
	f := newFixture(t, strings.Replace(smallTables, `"default_class": "sorceress"`, `"default_class": "necromancer"`, 1))
	if v := f.state(t); v.Inputs.Class != "necromancer" {
		t.Fatalf("preselect follows default_class: %+v", v.Inputs)
	}
	w := &Wizard{Tables: DefaultTables()}
	if in := w.inputsLocked(); in.Class != "paladin" {
		t.Errorf("default tables preselect %q, want paladin", in.Class)
	}
	var sorc Class
	for _, c := range DefaultTables().Classes {
		if c.ID == "sorceress" {
			sorc = c
		}
	}
	if sorc.Table != "normal30" || sorc.Note != "" {
		t.Errorf("sorceress %+v", sorc)
	}
}

func TestSavedUnknownClassIsError(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, stateFile), []byte(`{"schema":1,"phase":"inputs","class":"witch"}`), 0o600)
	_, err := NewWizard(dir, t.TempDir(), tables(t, smallTables), nil)
	if err == nil || !strings.Contains(err.Error(), `"witch"`) {
		t.Fatalf("saved unknown class: %v", err)
	}
	os.WriteFile(filepath.Join(dir, stateFile), []byte(`{"schema":1,"phase":"inputs","class":"paladin"}`), 0o600)
	if _, err := NewWizard(dir, t.TempDir(), DefaultTables(), nil); err != nil {
		t.Errorf("saved paladin: %v", err)
	}
	f := newFixture(t, smallTables)
	f.do(t, "PUT", "/api/ahead", `{"ok":true}`)
	for _, c := range []string{"witch", "wizard"} {
		code, body, _ := f.do(t, "PUT", "/api/inputs", `{"char_name":"Nobody","class":"`+c+`","goals":"campaign:normal_act1"}`)
		if code != 400 || !strings.Contains(body, `\"`+c+`\"`) {
			t.Errorf("%s: %d %s", c, code, body)
		}
	}
}

// Golden of what the class step receives (/api/state "classes"). Regenerate with UPDATE_GOLDEN=1.
func TestClassStepGolden(t *testing.T) {
	got, _ := json.MarshalIndent(DefaultTables().Classes, "", " ")
	got = append(got, '\n')
	p := filepath.Join("testdata", "classes.golden.json")
	if os.Getenv("UPDATE_GOLDEN") != "" {
		os.WriteFile(p, got, 0o644)
	}
	want, err := os.ReadFile(p)
	if err != nil || string(want) != string(got) {
		t.Errorf("class payload differs from %s (UPDATE_GOLDEN=1 to regenerate)\n%s", p, got)
	}
}

// The written wizard.env is sourced by the container entrypoint, which passes $D2D_CLASS as --d2d-class /
// --start-class. Two classes through the real writer and a real shell.
func TestWizardEnvDrivesD2DClass(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no sh")
	}
	for class, want := range map[string]string{"paladin": "2", "sorceress": "4"} {
		f := newFixture(t, strings.Replace(smallTables, `{"id": "necromancer", "name": "Necromancer", "d2d_index": 1}`, `{"id": "paladin", "name": "Paladin", "d2d_index": 2}`, 1))
		f.do(t, "PUT", "/api/ahead", `{"ok":true}`)
		if code, body, _ := f.do(t, "PUT", "/api/inputs", `{"char_name":"Heroa","class":"`+class+`","goals":"campaign:normal_act1"}`); code != 200 {
			t.Fatalf("%s: %d %s", class, code, body)
		}
		f.do(t, "PUT", "/api/access", passChoice)
		f.do(t, "POST", "/api/start", "")
		f.waitPhase(t, "done")
		out, err := exec.Command(sh, "-c", `. "$1"; printf %s "$D2D_CLASS"`, "sh", filepath.Join(f.data, envFile)).Output()
		if err != nil || string(out) != want {
			t.Errorf("%s: D2D_CLASS %q (%v), want %s", class, out, err, want)
		}
	}
}
