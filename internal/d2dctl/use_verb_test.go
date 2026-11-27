package d2dctl_test

import (
	"testing"

	"d2lod-bot/internal/d2dctl/fake"
)

func TestCmdUseLine(t *testing.T) {
	srv, c := start(t, fake.Options{})
	if err := c.CmdUse(ctx(), 77); err != nil {
		t.Fatal(err)
	}
	if got := srv.Lines(); len(got) != 1 || got[0] != "cmd use 77" {
		t.Fatalf("got %q", got)
	}
}
