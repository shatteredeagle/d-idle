package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"d2lod-bot/internal/setup"
)

func TestJSONAndExit(t *testing.T) {
	var out, errw bytes.Buffer
	code := run([]string{"--json", "--quick", "--pre-install", "--data", t.TempDir()}, &out, &errw, setup.NewEnv())
	var rep setup.Report
	if err := json.Unmarshal(out.Bytes(), &rep); err != nil {
		t.Fatalf("%v: %s", err, out.String())
	}
	if rep.Exit != code || len(rep.Results) == 0 || rep.Results[0].ID == "" {
		t.Errorf("code %d rep %+v", code, rep.Exit)
	}
	if run([]string{"--bogus"}, &out, &errw, setup.NewEnv()) != 3 {
		t.Error("usage exit")
	}
}
