package main

import (
	"fmt"
	"os"

	"d2lod-bot/internal/d2dsoakstat"
)

// printKillTimes prints the killtime summary of each log that has `killtime` lines (additive to extract).
func printKillTimes(paths []string, top int) {
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		ks, err := d2dsoakstat.ParseKillTimes(f)
		f.Close()
		if err == nil && len(ks) > 0 {
			fmt.Print(d2dsoakstat.SummarizeKills(ks, top).Text(p))
		}
	}
}
