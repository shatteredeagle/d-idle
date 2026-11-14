package main

import (
	"fmt"

	"d2lod-bot/tools/levelsim"
)

func main() {
	d, _ := levelsim.Load()
	p := levelsim.DefaultParams()
	p.Decay, p.Floor, p.Yield = 1.5, 0.3, 0.85
	fmt.Print(levelsim.CalibReport(d, p))
}
