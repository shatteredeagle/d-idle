package overlay

import (
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"d2lod-bot/internal/data/mpq"
)

// The shipped overlay data/overlay/didle-v1 (docs/overlay.md, docs/research/currency-rows.md): the Raven's Mark
// currency rows. The offline checks need the user's own vanilla tables; they are read from an install at
// D2_INSTALL_DIR (default /tmp/d2mpq like internal/data's real tests) and skipped when absent. Nothing from
// the base tables is stored in the repo.
const didleDir = "../../data/overlay/didle-v1"

func rawTables(t *testing.T, names ...string) map[string]string {
	t.Helper()
	dir := os.Getenv("D2_INSTALL_DIR")
	if dir == "" {
		dir = "/tmp/d2mpq"
	}
	var as []*mpq.Archive
	for _, n := range []string{"patch_d2.mpq", "d2exp.mpq", "d2data.mpq"} {
		ents, _ := os.ReadDir(dir)
		for _, e := range ents {
			if strings.EqualFold(e.Name(), n) {
				a, err := mpq.Open(filepath.Join(dir, e.Name()))
				if err != nil {
					t.Fatal(err)
				}
				defer a.Close()
				as = append(as, a)
			}
		}
	}
	if len(as) < 2 {
		t.Skip("no game install at " + dir)
	}
	out := map[string]string{}
	for _, n := range names {
		p := `data\global\excel\` + n + ".txt"
		for _, a := range as {
			if a.Has(p) {
				b, err := a.ReadFile(p)
				if err != nil {
					t.Fatal(err)
				}
				out[n] = string(b)
				break
			}
		}
		if out[n] == "" {
			t.Fatalf("table %s not in the install", n)
		}
	}
	return out
}

func rows(text string) [][]string {
	var out [][]string
	for _, l := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		if l != "" {
			out = append(out, strings.Split(l, "\t"))
		}
	}
	return out
}

func col(t *testing.T, header []string, name string) int {
	t.Helper()
	for i, h := range header {
		if strings.EqualFold(h, name) {
			return i
		}
	}
	t.Fatalf("no column %q", name)
	return -1
}

func find(t *testing.T, tab [][]string, key, val string) []string {
	t.Helper()
	c := col(t, tab[0], key)
	for _, r := range tab[1:] {
		if c < len(r) && r[c] == val {
			return r
		}
	}
	return nil
}

func cell(t *testing.T, tab [][]string, r []string, name string) string {
	c := col(t, tab[0], name)
	if c >= len(r) {
		return ""
	}
	return r[c]
}

func TestDidleV1FragmentsLoad(t *testing.T) {
	frags, err := Load(didleDir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, f := range frags {
		names = append(names, f.File)
	}
	if got := strings.Join(names, ","); got != "CubeMain.txt,Misc.txt,TreasureClassEx.txt" {
		t.Fatalf("fragments %s", got)
	}
}

func TestDidleV1AgainstVanilla(t *testing.T) {
	base := rawTables(t, "Misc", "Armor", "Weapons", "TreasureClassEx", "MonStats", "CubeMain")
	frags, err := Load(didleDir)
	if err != nil {
		t.Fatal(err)
	}
	merged := map[string][][]string{}
	for _, f := range frags {
		name := strings.TrimSuffix(f.File, ".txt")
		m, err := Merge(base[name], f)
		if err != nil {
			t.Fatal(err)
		}
		merged[name] = rows(m.Text)
		switch name {
		case "Misc", "CubeMain":
			if m.Appended != 1 || m.Replaced != 0 {
				t.Errorf("%s: appended %d replaced %d, want 1 and 0", name, m.Appended, m.Replaced)
			}
		case "TreasureClassEx":
			if m.Appended != 1 || m.Replaced != 1 {
				t.Errorf("%s: appended %d replaced %d, want 1 and 1", name, m.Appended, m.Replaced)
			}
		}
	}

	// The code is unused in every item table, then present exactly once.
	for _, n := range []string{"Misc", "Armor", "Weapons"} {
		tab := rows(base[n])
		if find(t, tab, "code", "mrk") != nil {
			t.Fatalf("code mrk is already used in %s", n)
		}
	}
	mt := merged["Misc"]
	mrk := find(t, mt, "code", "mrk")
	if mrk == nil {
		t.Fatal("mrk missing after the merge")
	}
	if cell(t, mt, mrk, "stackable") != "1" || cell(t, mt, mrk, "maxstack") != "12" || cell(t, mt, mrk, "spawnstack") != "1" {
		t.Errorf("stack columns: %v", mrk)
	}
	if typ := cell(t, mt, mrk, "type"); find(t, rows(base["Misc"]), "type", typ) == nil {
		t.Errorf("type %q is not a type used by the base", typ)
	}
	// Strings: the name key must not collide with a vanilla namestr.
	if r := find(t, rows(base["Misc"]), "namestr", cell(t, mt, mrk, "namestr")); r != nil {
		t.Errorf("namestr %q is already used", cell(t, mt, mrk, "namestr"))
	}

	// The source: every Blood Raven treasure class column of the Normal row points at "Blood Raven".
	ms := rows(base["MonStats"])
	br := find(t, ms, "Id", "bloodraven")
	if br == nil {
		t.Fatal("no bloodraven in MonStats")
	}
	for _, c := range []string{"TreasureClass1", "TreasureClass2", "TreasureClass3"} {
		if got := cell(t, ms, br, c); got != "Blood Raven" {
			t.Errorf("MonStats %s = %q", c, got)
		}
	}
	tcBase := rows(base["TreasureClassEx"])
	if find(t, tcBase, "Treasure Class", "DI Mark A") != nil {
		t.Fatal("DI Mark A exists in the base")
	}
	if r := find(t, tcBase, "Treasure Class", "Blood Raven"); r == nil || cell(t, tcBase, r, "Item5") != "" {
		t.Fatalf("Blood Raven row missing or Item5 not free: %v", r)
	}
	tc := merged["TreasureClassEx"]
	raven := find(t, tc, "Treasure Class", "Blood Raven")
	if cell(t, tc, raven, "Item5") != "DI Mark A" || cell(t, tc, raven, "Prob5") != "1" {
		t.Fatalf("Blood Raven row after merge: %v", raven)
	}
	mark := find(t, tc, "Treasure Class", "DI Mark A")
	if mark == nil || cell(t, tc, mark, "Item1") != "mrk" {
		t.Fatalf("DI Mark A row: %v", mark)
	}

	// Per-kill probability: the fork's roll (roll_class: per pick rand(NoDrop+total), then the entry by weight).
	atoi := func(s string) int { n, _ := strconv.Atoi(s); return n }
	picks, nodrop := atoi(cell(t, tc, raven, "Picks")), atoi(cell(t, tc, raven, "NoDrop"))
	total := 0
	for i := 1; i <= 10; i++ {
		if cell(t, tc, raven, "Item"+strconv.Itoa(i)) != "" {
			total += atoi(cell(t, tc, raven, "Prob"+strconv.Itoa(i)))
		}
	}
	innerNoDrop, innerTotal := atoi(cell(t, tc, mark, "NoDrop")), atoi(cell(t, tc, mark, "Prob1"))
	perPick := float64(1) / float64(nodrop+total) * float64(innerTotal) / float64(innerNoDrop+innerTotal)
	want := 1 - math.Pow(1-perPick, float64(picks))
	t.Logf("Blood Raven picks %d nodrop %d total %d; per pick %.5f; per kill %.5f (1 in %.1f)", picks, nodrop, total, perPick, want, 1/want)
	if want < 1.0/50 || want > 1.0/30 {
		t.Errorf("per-kill mark chance %.4f is outside 1/50..1/30", want)
	}
	rng := rand.New(rand.NewSource(1))
	const n = 400000
	hits := 0
	for k := 0; k < n; k++ {
		got := false
		for p := 0; p < picks; p++ {
			if r := rng.Intn(nodrop + total); r >= nodrop && r-nodrop >= total-atoi(cell(t, tc, raven, "Prob5")) {
				if rng.Intn(innerNoDrop+innerTotal) >= innerNoDrop {
					got = true
				}
			}
		}
		if got {
			hits++
		}
	}
	sd := math.Sqrt(float64(n) * want * (1 - want))
	if math.Abs(float64(hits)-float64(n)*want) > 5*sd {
		t.Errorf("Monte Carlo %d hits of %d, expected %.0f +- %.0f", hits, n, float64(n)*want, sd)
	}

	// Cube: the bank row is the only enabled one-input recipe that the stack (code mrk, type of the row) can match.
	cm := merged["CubeMain"]
	bank := cm[len(cm)-1]
	if cell(t, cm, bank, "input 1") != "mrk" || cell(t, cm, bank, "numinputs") != "1" {
		t.Fatalf("bank row: %v", bank)
	}
	typ := cell(t, mt, mrk, "type")
	for _, r := range cm[1 : len(cm)-1] {
		in := cell(t, cm, r, "input 1")
		if cell(t, cm, r, "enabled") == "1" && cell(t, cm, r, "numinputs") == "1" && (in == "mrk" || in == typ || in == "any" || strings.HasPrefix(in, "mrk,")) {
			t.Errorf("an earlier recipe %q takes a lone mrk", cell(t, cm, r, "description"))
		}
	}
	if out := cell(t, cm, bank, "output"); !strings.HasPrefix(out, "isc") || find(t, rows(base["Misc"]), "code", "isc") == nil {
		t.Errorf("bank output %q is not a base item", out)
	}
}
