package gear

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"d2lod-bot/internal/data/mpq"
)

// The gear tables (item types, weapon/shield/ammo bases, skills.txt prerequisites, charstats, armor.txt) are NOT in the
// repo: d2tables extracts them from the user's own MPQ files into a cache dir (THIRD_PARTY.md). Extraction is a port of
// the former gen.py / gen_armor.py (V: output proven equal to the old committed JSON, docs/gear.md).

const tablesSchema = 1

var archiveOrder = []string{"patch_d2.mpq", "d2exp.mpq", "d2data.mpq"} // highest priority first

// TablesStamp records which game files the cache was extracted from.
type TablesStamp struct {
	Schema int               `json:"schema"`
	Sizes  map[string]int64  `json:"sizes"`
	SHA256 map[string]string `json:"sha256"`
}

// TablesDir is the cache directory: $D2_TABLES_DIR, else <user cache dir>/d2lod/tables.
func TablesDir() string {
	if d := os.Getenv("D2_TABLES_DIR"); d != "" {
		return d
	}
	c, err := os.UserCacheDir()
	if err != nil {
		c = os.TempDir()
	}
	return filepath.Join(c, "d2lod", "tables")
}

// ErrNoTables is returned (and panicked by the lookups) when the cache is missing and no data dir is configured.
var ErrNoTables = errors.New("gear: tables not extracted: run `go run ./cmd/d2tables --data <MPQ dir>` first (or set D2_DATA_DIR)")

func findFold(dir, name string) (string, bool) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return "", false
	}
	for _, e := range ents {
		if strings.EqualFold(e.Name(), name) {
			return filepath.Join(dir, e.Name()), true
		}
	}
	return "", false
}

func sourceStamp(dataDir string, withHash bool) (TablesStamp, error) {
	st := TablesStamp{Schema: tablesSchema, Sizes: map[string]int64{}, SHA256: map[string]string{}}
	for _, n := range archiveOrder {
		p, ok := findFold(dataDir, n)
		if !ok {
			if n == "patch_d2.mpq" {
				continue
			}
			return st, fmt.Errorf("gear: %s not found in %s", n, dataDir)
		}
		fi, err := os.Stat(p)
		if err != nil {
			return st, err
		}
		st.Sizes[n] = fi.Size()
		if withHash || n == "patch_d2.mpq" {
			if st.SHA256[n], err = fileSHA256(p); err != nil {
				return st, err
			}
		}
	}
	return st, nil
}

func fileSHA256(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// TablesCurrent reports whether cacheDir was extracted from the current state of dataDir (sizes of all archives and the
// patch hash; the full hashes are recorded for provenance only, hashing d2data on every start would be slow).
func TablesCurrent(dataDir, cacheDir string) bool {
	b, err := os.ReadFile(filepath.Join(cacheDir, "stamp.json"))
	if err != nil {
		return false
	}
	var have TablesStamp
	if json.Unmarshal(b, &have) != nil || have.Schema != tablesSchema {
		return false
	}
	want, err := sourceStamp(dataDir, false)
	if err != nil || len(want.Sizes) != len(have.Sizes) {
		return false
	}
	for k, v := range want.Sizes {
		if have.Sizes[k] != v {
			return false
		}
	}
	return want.SHA256["patch_d2.mpq"] == have.SHA256["patch_d2.mpq"]
}

// ExtractTables reads the tables from the MPQs in dataDir (Patch_D2 over d2exp over d2data) and writes gear.json,
// armor.json and stamp.json (last, so a partial extraction is never taken for a valid one) into cacheDir.
func ExtractTables(dataDir, cacheDir string) error {
	stamp, err := sourceStamp(dataDir, true)
	if err != nil {
		return err
	}
	var archives []*mpq.Archive
	defer func() {
		for _, a := range archives {
			a.Close()
		}
	}()
	for _, n := range archiveOrder {
		p, ok := findFold(dataDir, n)
		if !ok {
			continue
		}
		a, err := mpq.Open(p)
		if err != nil {
			return fmt.Errorf("gear: %s: %w", p, err)
		}
		archives = append(archives, a)
	}
	read := func(name string) ([]byte, error) {
		f := `data\global\excel\` + name + ".txt"
		for _, a := range archives {
			if a.Has(f) {
				return a.ReadFile(f)
			}
		}
		return nil, fmt.Errorf("gear: %s not found in the MPQs", f)
	}
	return writeTables(read, stamp, cacheDir)
}

func writeTables(read func(string) ([]byte, error), stamp TablesStamp, cacheDir string) error {
	t, a, err := generate(read)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}
	_ = os.Remove(filepath.Join(cacheDir, "stamp.json"))
	for name, v := range map[string]any{"gear.json": t, "armor.json": map[string]any{"armor": a}} {
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(cacheDir, name), b, 0o644); err != nil {
			return err
		}
	}
	b, _ := json.Marshal(stamp)
	return os.WriteFile(filepath.Join(cacheDir, "stamp.json"), b, 0o644)
}

// rows parses a raw table into header-keyed rows. firstWins picks the first column of a duplicated header (armor.txt),
// otherwise the last one wins (the other tables, as gen.py did). Bytes are latin1.
func rows(raw []byte, firstWins bool) []map[string]string {
	lines := strings.Split(string(raw), "\n")
	var head []string
	var out []map[string]string
	for i, l := range lines {
		l = strings.TrimRight(l, "\r")
		if i == 0 {
			head = strings.Split(latin1(l), "\t")
			continue
		}
		if l == "" {
			continue
		}
		f := strings.Split(latin1(l), "\t")
		r := make(map[string]string, len(head))
		for j, c := range head {
			v := ""
			if j < len(f) {
				v = f[j]
			}
			if _, dup := r[c]; dup && firstWins {
				continue
			}
			r[c] = v
		}
		out = append(out, r)
	}
	return out
}

func latin1(s string) string {
	ascii := true
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			ascii = false
			break
		}
	}
	if ascii {
		return s
	}
	var sb strings.Builder
	for i := 0; i < len(s); i++ {
		sb.WriteRune(rune(s[i]))
	}
	return sb.String()
}

func num(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

func nonEmpty(vs ...string) []string {
	out := []string{}
	for _, v := range vs {
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func generate(read func(string) ([]byte, error)) (*table, map[string]Armor, error) {
	get := func(name string, firstWins bool) ([]map[string]string, error) {
		b, err := read(name)
		if err != nil {
			return nil, err
		}
		return rows(b, firstWins), nil
	}
	t := &table{Types: map[string]typeRow{}, Bases: map[string]Base{}, Skills: map[string]skillRow{}, Classes: map[string]ClassInfo{}}

	its, err := get("itemtypes", false)
	if err != nil {
		return nil, nil, err
	}
	for _, r := range its {
		c := r["Code"]
		if c == "" {
			continue
		}
		t.Types[c] = typeRow{Equiv: nonEmpty(r["Equiv1"], r["Equiv2"]), Shoots: r["Shoots"], Quiver: r["Quiver"],
			Throwable: r["Throwable"] == "1", Class: r["Class"], Body: nonEmpty(r["BodyLoc1"], r["BodyLoc2"])}
	}
	isa := func(typ, anc string) bool {
		seen, todo := map[string]bool{}, []string{typ}
		for len(todo) > 0 {
			x := todo[len(todo)-1]
			todo = todo[:len(todo)-1]
			if x == anc {
				return true
			}
			if seen[x] {
				continue
			}
			if _, ok := t.Types[x]; !ok {
				continue
			}
			seen[x] = true
			todo = append(todo, t.Types[x].Equiv...)
		}
		return false
	}
	addBases := func(table string, keep func(typ string) bool) error {
		rs, err := get(table, false)
		if err != nil {
			return err
		}
		for _, r := range rs {
			c := r["code"]
			if c == "" || !keep(r["type"]) {
				continue
			}
			t.Bases[c] = Base{Name: r["name"], Type: r["type"], TwoHanded: r["2handed"] == "1",
				ReqStr: num(r["reqstr"]), ReqDex: num(r["reqdex"]), LevelReq: num(r["levelreq"]),
				Stackable: r["stackable"] == "1", MinStack: num(r["minstack"]), MaxStack: num(r["maxstack"]),
				Cost: num(r["cost"]), PermStore: r["PermStoreItem"] == "1"}
		}
		return nil
	}
	if err := addBases("weapons", func(string) bool { return true }); err != nil {
		return nil, nil, err
	}
	if err := addBases("misc", func(ty string) bool {
		return ty == "bowq" || ty == "xboq" || ty == "mboq" || ty == "mxbq" || isa(ty, "bowq") || isa(ty, "xboq")
	}); err != nil {
		return nil, nil, err
	}
	if err := addBases("armor", func(ty string) bool { return isa(ty, "shld") }); err != nil {
		return nil, nil, err
	}

	sks, err := get("skills", false)
	if err != nil {
		return nil, nil, err
	}
	for _, r := range sks {
		k := r["skill"]
		if k == "" || r["charclass"] == "" {
			continue
		}
		t.Skills[k] = skillRow{Class: r["charclass"], ReqLevel: num(r["reqlevel"]),
			Req:    nonEmpty(r["reqskill1"], r["reqskill2"], r["reqskill3"]),
			ItypeA: nonEmpty(r["itypea1"], r["itypea2"], r["itypea3"]), EtypeA: nonEmpty(r["etypea1"], r["etypea2"]),
			Left: r["leftskill"] == "1", Passive: r["passive"] == "1", ID: num(r["Id"])}
	}

	css, err := get("charstats", false)
	if err != nil {
		return nil, nil, err
	}
	for _, r := range css {
		n := r["class"]
		if n == "" || n == "Expansion" {
			continue
		}
		ci := ClassInfo{Name: n, Str: num(r["str"]), Dex: num(r["dex"]), Vit: num(r["vit"]), Ene: num(r["int"]), Start: r["StartSkill"]}
		for i := 1; i <= 10; i++ {
			c := r[fmt.Sprintf("item%d", i)]
			if c == "" || c == "0" {
				continue
			}
			ci.Items = append(ci.Items, struct {
				Code  string `json:"code"`
				Loc   string `json:"loc"`
				Count int    `json:"count"`
			}{c, r[fmt.Sprintf("item%dloc", i)], max(1, num(r[fmt.Sprintf("item%dcount", i)]))})
		}
		code := n
		if len(code) > 3 {
			code = code[:3]
		}
		t.Classes[strings.ToLower(code)] = ci
	}

	ars, err := get("armor", true)
	if err != nil {
		return nil, nil, err
	}
	armor := map[string]Armor{}
	tier := map[string]int{}
	for _, r := range ars {
		if r["code"] == "" {
			continue
		}
		for col, ti := range map[string]int{"normcode": 0, "ubercode": 1, "ultracode": 2} {
			if r[col] != "" {
				tier[r[col]] = ti
			}
		}
	}
	for _, r := range ars {
		c := r["code"]
		if c == "" {
			continue
		}
		armor[c] = Armor{Name: r["name"], Type: r["type"], Tier: tier[c], MinAC: num(r["minac"]), MaxAC: num(r["maxac"]),
			ReqStr: num(r["reqstr"]), LevelReq: num(r["levelreq"]), Level: num(r["level"]), Speed: num(r["speed"]),
			Block: num(r["block"]), Durability: num(r["durability"]), NoDurability: r["nodurability"] == "1",
			MaxSockets: num(r["gemsockets"]), StrBonus: num(r["StrBonus"]), DexBonus: num(r["DexBonus"]),
			InvWidth: num(r["invwidth"]), InvHeight: num(r["invheight"])}
	}
	return t, armor, nil
}

var (
	once     sync.Once
	tab      table
	armorTab map[string]Armor
	loadErr  error
)

// Available reports whether the tables can be loaded now (extracting them first when D2_DATA_DIR is set and the cache
// is missing or stale). Tests that need the real tables skip on its error.
func Available() error {
	loadAll()
	return loadErr
}

func loadAll() {
	once.Do(func() { loadErr = loadTables(os.Getenv("D2_DATA_DIR"), TablesDir()) })
}

func loadTables(dataDir, cacheDir string) error {
	if dataDir != "" && !TablesCurrent(dataDir, cacheDir) {
		if err := ExtractTables(dataDir, cacheDir); err != nil {
			return err
		}
	}
	gb, err := os.ReadFile(filepath.Join(cacheDir, "gear.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return ErrNoTables
		}
		return err
	}
	ab, err := os.ReadFile(filepath.Join(cacheDir, "armor.json"))
	if err != nil {
		return err
	}
	var f struct {
		Armor map[string]Armor `json:"armor"`
	}
	if err := json.Unmarshal(gb, &tab); err != nil {
		return fmt.Errorf("gear: gear.json: %w", err)
	}
	if err := json.Unmarshal(ab, &f); err != nil {
		return fmt.Errorf("gear: armor.json: %w", err)
	}
	armorTab = f.Armor
	return nil
}

func load() *table {
	loadAll()
	if loadErr != nil {
		panic(loadErr.Error())
	}
	return &tab
}
