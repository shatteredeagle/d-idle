package d2dcombat

import (
	"context"
	"testing"
	"time"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dctl/fake"
	"d2lod-bot/internal/d2dinput"
)

func TestMeterSamples(t *testing.T) {
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
	m := &Meter{P: &d2dinput.Pacer{C: c}, C: c, Ctx: context.Background()}
	m.Sleep(80 * time.Millisecond)
	if tot := m.Totals(); tot.Samples != 1 || tot.MinLife != 45 {
		t.Fatalf("%+v", tot)
	}
}
