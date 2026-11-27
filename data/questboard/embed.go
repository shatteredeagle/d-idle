// Package questboarddata embeds the quest board definitions (docs/questboard.md; the code is internal/questboard).
package questboarddata

import _ "embed"

// Families is families.json (class to family).
//
//go:embed families.json
var Families []byte

// Act1 is act1.json (quests.def/1).
//
//go:embed act1.json
var Act1 []byte

// Act2 is act2.json (quests.def/1).
//
//go:embed act2.json
var Act2 []byte

// Readiness is readiness.json (quests.readiness/1: dial lines, kit minimums, wording).
//
//go:embed readiness.json
var Readiness []byte

// Act3 is act3.json (quests.def/1).
//
//go:embed act3.json
var Act3 []byte

// Act4 is act4.json (quests.def/1).
//
//go:embed act4.json
var Act4 []byte

// Act5 is act5.json (quests.def/1).
//
//go:embed act5.json
var Act5 []byte

// Resume is resume.json (quests.resume/1: which parks may lift themselves, and when).
//
//go:embed resume.json
var Resume []byte
