package status

type Skill struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Level  int    `json:"level"`
	Hotkey string `json:"hotkey,omitempty"`
}

type Resists struct {
	Fire   int `json:"fire"`
	Cold   int `json:"cold"`
	Light  int `json:"light"`
	Poison int `json:"poison"`
}

// CharacterData is the contract of the "character" panel.
type CharacterData struct {
	Name       string  `json:"name"`
	Class      string  `json:"class"`
	Difficulty string  `json:"difficulty,omitempty"`
	Level      int     `json:"level"`
	Exp        uint32  `json:"exp"`
	ExpPrev    uint32  `json:"expPrev"` // exp needed for current level
	ExpNext    uint32  `json:"expNext"` // exp needed for next level, 0 unknown
	Life       int     `json:"life"`
	MaxLife    int     `json:"maxLife"`
	Mana       int     `json:"mana"`
	MaxMana    int     `json:"maxMana"`
	Stamina    int     `json:"stamina"`
	MaxStamina int     `json:"maxStamina"`
	Str        int     `json:"str"`
	Dex        int     `json:"dex"`
	Vit        int     `json:"vit"`
	Ene        int     `json:"ene"`
	StatPoints int     `json:"statPoints"`
	SkillPts   int     `json:"skillPoints"`
	Defense    int     `json:"defense,omitempty"`
	Resists    Resists `json:"resists"`
	Skills     []Skill `json:"skills"`
	LeftSkill  int     `json:"leftSkill"`
	RightSkill int     `json:"rightSkill"`
}

// CharacterSource is implemented by the adapter over internal/d2 (or a simulation).
type CharacterSource interface{ Character() CharacterData }

func NewCharacterPanel(src CharacterSource) Panel {
	return Panel{ID: "character", Title: "Character", Order: 20, Snapshot: func() any { return src.Character() }}
}
