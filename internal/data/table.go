package data

import (
	"bytes"
	"math"
	"strconv"
	"strings"
)

// Table is a parsed D2 tab-separated .txt table: a header row, then data rows.
// Column lookup is case-insensitive and tolerant: a missing column reads as
// the zero value, so a loader written against 1.14d keeps working on a table
// that lacks (or renames) a column.
type Table struct {
	Header []string
	rows   [][]string
	col    map[string]int
}

// ParseTable parses TSV bytes (CRLF or LF). Blank lines are skipped. Rows
// shorter than the header read as empty in the missing cells.
func ParseTable(b []byte) *Table {
	t := &Table{col: map[string]int{}}
	b = bytes.TrimPrefix(b, []byte{0xEF, 0xBB, 0xBF})
	first := true
	for len(b) > 0 {
		var line []byte
		if i := bytes.IndexByte(b, '\n'); i >= 0 {
			line, b = b[:i], b[i+1:]
		} else {
			line, b = b, nil
		}
		line = bytes.TrimRight(line, "\r")
		if len(line) == 0 {
			continue
		}
		cells := strings.Split(string(line), "\t")
		if first {
			first = false
			t.Header = cells
			for i, h := range cells {
				k := strings.ToLower(strings.TrimSpace(h))
				if _, dup := t.col[k]; !dup && k != "" {
					t.col[k] = i
				}
			}
			continue
		}
		t.rows = append(t.rows, cells)
	}
	return t
}

// Len is the number of data rows.
func (t *Table) Len() int { return len(t.rows) }

// Row returns data row i (0-based, header excluded).
func (t *Table) Row(i int) Row { return Row{t, t.rows[i]} }

// Has reports whether any of the named columns exists.
func (t *Table) Has(names ...string) bool {
	_, ok := t.find(names)
	return ok
}

func (t *Table) find(names []string) (int, bool) {
	for _, n := range names {
		if i, ok := t.col[strings.ToLower(n)]; ok {
			return i, true
		}
	}
	return 0, false
}

// Row is one data row. Every accessor takes one or more column names (aliases
// for the same column across game versions) and uses the first that exists.
type Row struct {
	t     *Table
	cells []string
}

// Str returns the cell text, or "" if the column is missing or the row short.
func (r Row) Str(names ...string) string {
	i, ok := r.t.find(names)
	if !ok || i >= len(r.cells) {
		return ""
	}
	return strings.TrimSpace(r.cells[i])
}

// Int returns the cell as an integer. Empty and non-numeric cells read as 0;
// decimals truncate.
func (r Row) Int(names ...string) int {
	s := r.Str(names...)
	if s == "" {
		return 0
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return int(n)
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil && !math.IsNaN(f) && !math.IsInf(f, 0) {
		return int(f)
	}
	return 0
}

// Float returns the cell as a float64 (0 if empty or invalid).
func (r Row) Float(names ...string) float64 {
	f, _ := strconv.ParseFloat(r.Str(names...), 64)
	return f
}

// Bool is true for any non-zero integer cell.
func (r Row) Bool(names ...string) bool { return r.Int(names...) != 0 }

// Empty reports whether the row has no non-empty cell in column names[0]
// (rows with a blank key column are separators, e.g. the "Expansion" row).
func (r Row) Empty(name string) bool { return r.Str(name) == "" }

func itoa(n int) string { return strconv.Itoa(n) }
