package main

import (
	"errors"
	"fmt"
	"testing"

	"d2lod-bot/internal/bot"
)

// A campaign stall exits with its own status, neither the no-restart fatal nor the panic-storm one.
func TestHaltErrorCampaignStalled(t *testing.T) {
	err := haltError(fmt.Errorf("%w: no progress for 60 virtual min", bot.ErrCampaignStalled))
	var ee *exitError
	if !errors.As(err, &ee) || ee.code != exitStalled {
		t.Fatalf("got %v", err)
	}
	if exitStalled == exitFatal || exitStalled == exitStorm || exitStalled == 0 {
		t.Fatal("status must be distinct and non-zero")
	}
}
