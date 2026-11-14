package levelplan

import (
	"bytes"
	_ "embed"
	"encoding/csv"
	"fmt"
	"strconv"
)

//go:embed areaxp.csv
var areaXPCSV []byte

// xpCols are the clvl columns of areaxp.csv (xp_kill_c1 .. xp_kill_c16).
var xpCols = [...]int{1, 3, 5, 8, 12, 16}

// AreaXP is one area row of the AREAMON table: monsters per full clear and XP per kill at the clvl columns (V run),
// minutes per clear (I, the AREAMON time model).
type AreaXP struct {
	Area     int
	Name     string
	Units    float64
	ClearMin float64
	perKill  [len(xpCols)]float64
}

// PerKill is the mean XP of one kill at clvl, linear between the table's clvl columns and flat outside them (I: the
// level-gap award is a step function; the table samples it).
func (a AreaXP) PerKill(clvl int) float64 {
	if clvl <= xpCols[0] {
		return a.perKill[0]
	}
	for i := 1; i < len(xpCols); i++ {
		if clvl <= xpCols[i] {
			lo, hi := xpCols[i-1], xpCols[i]
			f := float64(clvl-lo) / float64(hi-lo)
			return a.perKill[i-1] + f*(a.perKill[i]-a.perKill[i-1])
		}
	}
	return a.perKill[len(xpCols)-1]
}

// PerMin is XP per minute of a fresh clear at clvl.
func (a AreaXP) PerMin(clvl int) float64 { return a.PerKill(clvl) * a.KillsPerMin() }

// KillsPerMin is the kill rate the time model assumes for a fresh clear.
func (a AreaXP) KillsPerMin() float64 { return a.Units / a.ClearMin }

// Table is the AREAMON XP table by area id.
type Table map[int]AreaXP

// LoadTable parses the embedded table.
func LoadTable() (Table, error) { return ParseTable(areaXPCSV) }

// ParseTable parses the areaxp.csv format.
func ParseTable(b []byte) (Table, error) {
	r := csv.NewReader(bytes.NewReader(b))
	r.Comment = '#'
	rows, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("levelplan: areaxp: %w", err)
	}
	if len(rows) < 2 || len(rows[0]) != 4+len(xpCols) {
		return nil, fmt.Errorf("levelplan: areaxp: want a header and %d columns", 4+len(xpCols))
	}
	t := Table{}
	for _, row := range rows[1:] {
		var a AreaXP
		a.Name = row[1]
		nums := make([]float64, 0, len(row))
		for i, f := range row {
			if i == 1 {
				continue
			}
			v, err := strconv.ParseFloat(f, 64)
			if err != nil {
				return nil, fmt.Errorf("levelplan: areaxp %s column %d: %w", row[1], i, err)
			}
			nums = append(nums, v)
		}
		a.Area, a.Units, a.ClearMin = int(nums[0]), nums[1], nums[2]
		if a.ClearMin <= 0 {
			return nil, fmt.Errorf("levelplan: areaxp %s: clear minutes %v", a.Name, a.ClearMin)
		}
		copy(a.perKill[:], nums[3:])
		t[a.Area] = a
	}
	return t, nil
}
