package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"d2lod-bot/internal/fleet"
)

// The tables are data (AGENTS.md build flexibility): file set, classes and goal presets are not in Go code.
// d2d_index is the d2d UI order (V docs/d2d-fidelity.md: 0 Barbarian, 1 Necromancer, 2 Paladin, 3 Amazon,
// 4 Sorceress, 5 Druid, 6 Assassin). min_size values are loose lower bounds against partial copies (U: not
// pinned to a real install). size_1_14d is the size in the owner's 1.14d install (V, 2026-11-06; shown, not enforced). The 1.14d
// version proof is the Patch_D2 hash list (V: sha256 of the owner's Patch_D2.mpq, whose Game.exe file version reads 1.14.3.71).

//go:embed tables.json
var tablesJSON []byte

type FileSpec struct {
	Name      string `json:"name"`
	Required  bool   `json:"required,omitempty"`
	MinSize   int64  `json:"min_size,omitempty"`
	Size1_14d int64  `json:"size_1_14d,omitempty"`
	Why       string `json:"why"`
}

type Class struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	D2DIndex  int    `json:"d2d_index"`
	Supported bool   `json:"supported"`
}

type Preset struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Tables struct {
	Source        string     `json:"source,omitempty"`
	Files         []FileSpec `json:"files"`
	PatchSHA256   []string   `json:"patch_sha256"`
	Classes       []Class    `json:"classes"`
	DefaultClass  string     `json:"default_class"`
	Presets       []Preset   `json:"presets"`
	DefaultPreset string     `json:"default_preset"`
}

// LoadTables decodes and validates a table set. Unknown fields, an unknown default or an unsupported default
// class are errors that name the offender.
func LoadTables(raw []byte) (*Tables, error) {
	var t Tables
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&t); err != nil {
		return nil, fmt.Errorf("tables: %w", err)
	}
	c, err := t.Class(t.DefaultClass)
	if err != nil {
		return nil, fmt.Errorf("tables: default class: %w", err)
	}
	if !c.Supported {
		return nil, fmt.Errorf("tables: default class %q is not supported", c.ID)
	}
	if _, err := t.Preset(t.DefaultPreset); err != nil {
		return nil, fmt.Errorf("tables: default preset: %w", err)
	}
	for _, p := range t.Presets {
		if _, err := (fleet.Spec{Goals: p.ID}).GoalsArg(); err != nil {
			return nil, fmt.Errorf("tables: preset %q: %w", p.ID, err)
		}
	}
	for _, f := range t.Files {
		if f.Name == "" || strings.ContainsAny(f.Name, "/\\") {
			return nil, fmt.Errorf("tables: bad file name %q", f.Name)
		}
	}
	return &t, nil
}

func DefaultTables() *Tables {
	t, err := LoadTables(tablesJSON)
	if err != nil {
		panic(err)
	}
	return t
}

func (t *Tables) Class(id string) (Class, error) {
	for _, c := range t.Classes {
		if c.ID == id {
			return c, nil
		}
	}
	return Class{}, fmt.Errorf("unknown class %q", id)
}

func (t *Tables) Preset(id string) (Preset, error) {
	for _, p := range t.Presets {
		if p.ID == id {
			return p, nil
		}
	}
	return Preset{}, fmt.Errorf("unknown goals preset %q", id)
}
