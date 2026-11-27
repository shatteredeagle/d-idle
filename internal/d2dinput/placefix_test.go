package d2dinput

import (
	"strings"
	"testing"

	"d2lod-bot/internal/d2dctl/fake"
)

func placeVerbs(ls []string) []string {
	var out []string
	for _, l := range ls {
		if strings.HasPrefix(l, "cmd ") || l == "debug objectlist" {
			out = append(out, l)
		}
	}
	return out
}

// The orifice operates only when the hero arrives, ticks after "cmd interact"; a "cmd place" sent before that is dropped
// by the d2d without a reply (V lab drill 2026-11-27: the Staff stayed in the pack, 3 of 3). PlaceItem waits for the
// object to leave NU before it takes the Staff and places it.
func TestPlaceItemWaitsForTheOrificeToOpen(t *testing.T) {
	fix := fake.Sample()
	row := func(mode string) []string {
		return []string{"103\t100\t10.0\t10.0\t0\tNU\t0\t0\tchest", "104\t152\t20.0\t20.0\t25\t" + mode + "\t0\t0\torifice", "room\t1\t2\t3"}
	}
	fix.ObjectList = [][]string{row("NU"), row("NU"), row("OP")}
	c, srv := rig(t, fix)
	if err := c.PlaceItem(104, 25); err != nil {
		t.Fatal(err)
	}
	got := strings.Join(placeVerbs(srv.Lines()), "|")
	want := "cmd interact 104|debug objectlist|debug objectlist|debug objectlist|cmd hand 25|cmd place 104 25"
	if got != want {
		t.Fatalf("verbs\n got %s\nwant %s", got, want)
	}
}

// An orifice that never opens (no Staff carried, out of reach) is an error, not a silent place.
func TestPlaceItemOrificeNeverOpens(t *testing.T) {
	fix := fake.Sample()
	fix.ObjectList = [][]string{{"104\t152\t20.0\t20.0\t25\tNU\t0\t0\torifice"}}
	c, srv := rig(t, fix)
	err := c.PlaceItem(104, 25)
	if err == nil || !strings.Contains(err.Error(), "object 104 still NU") {
		t.Fatalf("err = %v", err)
	}
	for _, l := range srv.Lines() {
		if strings.HasPrefix(l, "cmd place") || strings.HasPrefix(l, "cmd hand") {
			t.Fatalf("sent %q to a closed orifice", l)
		}
	}
}
