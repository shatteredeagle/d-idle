package main

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"d2lod-bot/internal/goldfarm"
	"d2lod-bot/internal/repair"
)

// /api/repair carries the Farm gold status as one additive field; the order switches through the farm adapter and
// survives a restart of the controller.
func TestRepairViewCarriesFarmAndTheOrderPersists(t *testing.T) {
	dir := t.TempDir()
	rc, err := repair.New(repair.Config{})
	if err != nil {
		t.Fatal(err)
	}
	gf, err := goldfarm.New(filepath.Join(dir, goldfarm.StateFile), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if err := (farmAdapter{gf}).Set(true); err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(repairAdapter{rc, gf}.View())
	var got struct {
		Schema int `json:"schema"`
		Farm   *struct {
			On bool `json:"on"`
		} `json:"farm"`
	}
	if err := json.Unmarshal(b, &got); err != nil || got.Schema != 1 || got.Farm == nil || !got.Farm.On {
		t.Fatalf("%s %v", b, err)
	}
	gf2, _ := goldfarm.New(filepath.Join(dir, goldfarm.StateFile), time.Now)
	if !gf2.On() {
		t.Error("the order did not survive a restart")
	}
	b, _ = json.Marshal(repairAdapter{rc, nil}.View())
	if json.Unmarshal(b, &got); got.Farm != nil && !got.Farm.On {
		t.Error("without a controller the field must be absent")
	}
}
