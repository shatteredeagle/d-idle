package loot_test

import (
	"math/rand"
	"strings"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/loot"
)

// judge parses a pickit and runs random items through it; neither step may panic.
func judge(t testing.TB, src string, l *loot.Looter, rng *rand.Rand) {
	t.Helper()
	defer func() {
		if x := recover(); x != nil {
			t.Fatalf("panic on pickit %.80q: %v", src, x)
		}
	}()
	pk, err := loot.ParsePickit(strings.NewReader(src))
	if err != nil {
		return
	}
	l.SetPickit(pk)
	for i := 0; i < 20; i++ {
		it := &d2.Item{Quality: rng.Intn(20) - 5, Flags: rng.Uint32(), ILvl: int(int32(rng.Uint32())), GX: rng.Intn(100), GY: -rng.Intn(10)}
		st := d2.Stats{}
		for k := rng.Intn(5); k > 0; k-- {
			st[uint32(rng.Intn(600))] = int32(rng.Uint32())
		}
		u := d2.Unit{Class: rng.Intn(600) - 20, Item: it}
		if i%7 == 0 {
			u.Item = nil
		}
		l.EvaluateWith(u, loot.Ctx{CLvl: rng.Intn(200) - 50, Gold: int(int32(rng.Uint32())), Stats: st})
		if u.Item != nil {
			l.Evaluate(*u.Item)
		}
	}
}

var pickitSeeds = []string{
	``, `{}`, `null`, `[]`, `{"rules":null}`, `{"rules":[{}]}`, `{"default":{"action":"keep","tier":"S"}}`,
	`{"rules":[{"name":"x","action":"keep","tier":"S","match":{"codes":["["],"stats":[{"stat":1,"min":-5,"max":0,"shift":-3}]}}]}`,
	`{"rules":[{"name":"x","action":"keep","tier":"S","match":{"anyStats":[{"stat":-1,"shift":99999999999}],"clvlMin":-1,"ilvlMax":2147483647}}]}`,
	`{"rules":[{"name":"x","action":"sell","tier":"a","match":{"qualities":["Unique","RARE"],"needed":"gem","types":["weap",""],"identified":true}}]}`,
	`{"version":99999999999999999999}`,
}

func TestPickitGarbage(t *testing.T) {
	l, _ := newLooter(t)
	rng := rand.New(rand.NewSource(9))
	for _, s := range pickitSeeds {
		judge(t, s, l, rng)
	}
	good := loot.Starter()
	_ = good
	base := pickitSeeds[7]
	for i := 0; i < 3000; i++ {
		b := []byte(base)
		for k := rng.Intn(4); k >= 0; k-- {
			b[rng.Intn(len(b))] = byte(rng.Intn(256))
		}
		judge(t, string(b), l, rng)
	}
	for n := 0; n <= len(base); n++ {
		judge(t, base[:n], l, rng)
	}
}

func FuzzPickit(f *testing.F) {
	for _, s := range pickitSeeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		l, _ := newLooter(t)
		judge(t, s, l, rand.New(rand.NewSource(int64(len(s)))))
	})
}
