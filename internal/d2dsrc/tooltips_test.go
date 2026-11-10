package d2dsrc

import (
	"context"
	"reflect"
	"testing"
)

// The stock items verb's text is attached by item id when Options.Tooltips is on, cached between reads, and absent
// when the option is off or the verb is unknown.
func TestTooltipsByID(t *testing.T) {
	r := lateReplies(t)
	r["items"] = "[wnd loc=1 slot=4 q=7 panel=0 id=9 size=1x2 at=0,0]\n  Wand\n  One-Hand Damage: 2 to 4\n[cap loc=3 q=6 id=12 size=2x2 at=3,1]\n  Cap\n  Ethereal\nok\n"
	s, c := startScript(t, r)
	src := New(c, Options{Tables: fakeTables{}, Logf: quiet, Tooltips: true})
	for i := 0; i < 2; i++ {
		res, err := src.Read(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if got, want := res.Extras.Items[9].Tooltip, []string{"Wand", "One-Hand Damage: 2 to 4"}; !reflect.DeepEqual(got, want) {
			t.Fatalf("read %d: tooltip of 9 = %q", i, got)
		}
		if len(res.Extras.Items[12].Tooltip) != 2 || res.Extras.Items[11].Tooltip != nil {
			t.Fatalf("read %d: tooltips %+v", i, res.Extras.Items)
		}
	}
	n := 0
	for _, l := range s.seen {
		if l == "items" {
			n++
		}
	}
	if n != 1 {
		t.Errorf("the items verb ran %d times for an unchanged item set, want 1 (cache)", n)
	}

	_, c2 := startScript(t, r)
	res, err := New(c2, Options{Tables: fakeTables{}, Logf: quiet}).Read(context.Background())
	if err != nil || res.Extras.Items[9].Tooltip != nil {
		t.Errorf("tooltips off: %v %+v", err, res.Extras.Items[9])
	}

	delete(r, "items")
	_, c3 := startScript(t, r)
	res, err = New(c3, Options{Tables: fakeTables{}, Logf: quiet, Tooltips: true}).Read(context.Background())
	if err != nil || res.Extras.Items[9].Tooltip != nil {
		t.Errorf("no items verb must not fail the snapshot: %v", err)
	}
}
