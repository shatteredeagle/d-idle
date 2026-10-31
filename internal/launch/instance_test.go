package launch

import "testing"

func TestInstanceConfig(t *testing.T) {
	base := Config{Root: "/h/Games/d2lod", HelperExe: "/h/Games/d2lod/helper/d2input.exe"}
	i := Instance{ID: "i3", Display: ":103"}
	c := i.Config(base)
	if c.Root != "/h/Games/d2lod-inst/i3" || c.HelperAddr != "127.0.0.1:47015" || c.Instance != "i3" {
		t.Fatalf("%+v", c)
	}
	if c.HelperExe != base.HelperExe || len(c.ExtraEnv) != 1 || c.ExtraEnv[0] != "DISPLAY=:103" {
		t.Fatalf("%+v", c)
	}
	e := c.Env()
	found := false
	for _, v := range e {
		if v == "D2BOT_INSTANCE=i3" {
			found = true
		}
	}
	if !found {
		t.Fatal("marker missing from env")
	}
	z := Instance{ID: "i0"}.Config(base)
	if z.Root != base.Root || z.HelperAddr != "" {
		t.Fatalf("%+v", z)
	}
	if got := newPIDs(map[int]bool{1: true}, map[int]bool{1: true, 5: true}); len(got) != 1 || got[0] != 5 {
		t.Fatal(got)
	}
}
