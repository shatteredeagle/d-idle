package data

// schemaVersion is stamped into the cache. Bump it whenever the column lists
// or the cache layout change, so old caches are re-extracted.
const schemaVersion = 2

type tableSpec struct {
	name     string // file name under data\global\excel\, without .txt
	cols     []string
	optional bool // extraction does not fail when no archive has it
}

var specs = []tableSpec{
	{"monstats", monsterCols, false},
	{"levels", levelCols, false},
	{"skills", skillCols, false},
	{"skilldesc", skillDescCols, true},
	{"charstats", charCols, false},
	{"experience", expCols, false},
	{"itemtypes", itemTypeCols, false},
	{"weapons", itemCols, false},
	{"armor", itemCols, false},
	{"misc", itemCols, false},
	{"objects", objectCols, true},
	{"superuniques", superUniqueCols, true},
	{"inventory", inventoryCols, true},
	{"belts", beltCols, true},
	{"treasureclassex", tcCols, true},
	{"uniqueitems", uniqueCols, true},
	{"setitems", setItemCols, true},
	{"runes", runewordCols, true},
	{"cubemain", cubeCols, true},
	{"monlvl", monLevelCols, true},
	{"difficultylevels", difficultyCols, true},
	{"hireling", hirelingCols, true},
	{"itemstatcost", statCostCols, true},
	{"properties", propertyCols, true},
	{"magicprefix", affixCols, true},
	{"magicsuffix", affixCols, true},
	{"states", stateCols, true},
	{"missiles", missileCols, true},
	{"lvlwarp", warpCols, true},
}

// String tables in priority order, highest first (the game's own lookup order).
var stringTables = []string{"patchstring.tbl", "expansionstring.tbl", "string.tbl"}

const (
	excelDir = `data\global\excel\`
	lngDir   = `data\local\lng\eng\`
)

// Archives in priority order, highest first: the 1.14 patch overrides the
// expansion, which overrides the base game.
var archiveNames = []string{"patch_d2.mpq", "d2exp.mpq", "d2data.mpq"}
