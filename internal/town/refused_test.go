package town

import (
	"testing"

	"d2lod-bot/internal/d2"
)

// A refused item is skipped at the same character level and offered again after a level-up (EQUIPLIVE).
func TestRefusedGearUntilLevelUp(t *testing.T) {
	const id = 900001
	if isRefused(id, charStamp{7, 10, 10}) {
		t.Fatal("unknown item refused")
	}
	markRefused(d2.Unit{ID: id}, charStamp{7, 10, 10})
	if !isRefused(id, charStamp{7, 10, 10}) {
		t.Error("refused item offered again at the same level")
	}
	if isRefused(id, charStamp{8, 10, 10}) {
		t.Error("still refused after a level-up")
	}
}

// A stat point also lifts the blacklist.
func TestRefusedLiftedByStat(t *testing.T) {
	markRefused(d2.Unit{ID: 900002}, charStamp{7, 10, 10})
	if isRefused(900002, charStamp{7, 11, 10}) {
		t.Error("still refused after a strength point")
	}
}
