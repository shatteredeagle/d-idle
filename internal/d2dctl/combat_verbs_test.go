package d2dctl_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dctl/fake"
)

func TestCombatVerbs(t *testing.T) {
	path, cleanup, err := fake.TempSocket()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	srv, err := fake.New(path, fake.Sample(), fake.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	c := d2dctl.New(path, d2dctl.Options{Timeout: 2 * time.Second})
	defer c.Close()
	ctx := context.Background()
	ml, err := c.Monsters(ctx)
	if err != nil || len(ml.Items) != 2 {
		t.Fatal(err, ml.Items)
	}
	m, ok := d2dctl.NearestAlive(ml.Items, 13, 29.4, 5)
	if !ok || m.Unit != 5 {
		t.Fatalf("nearest %+v %v", m, ok)
	}
	if _, ok := d2dctl.NearestAlive(ml.Items, 100, 100, 5); ok {
		t.Fatal("nothing should be in range")
	}
	for _, err := range []error{c.CastUnit(ctx, 36, m), c.CastCell(ctx, 44, 1, 2.5), c.Melee(ctx, m), c.Drink(ctx, 2), c.SetSkill(ctx, 36, true)} {
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := c.Drink(ctx, 4); err == nil {
		t.Fatal("slot 4 must be refused")
	}
	want := []string{"monsters", "cmd skill 36 13 29.5 5", "cmd skill 44 1 2.5", "cmd skill 0 13 29.5 5 left", "cmd belt 2", "debug skill left 36"}
	if got := srv.Lines(); !reflect.DeepEqual(got, want) {
		t.Fatalf("lines\n got %q\nwant %q", got, want)
	}
}
