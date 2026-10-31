package goals

import "fmt"

// Settings are user preferences. The manager acts on AutoResume and MaxDeaths; the rest is passed
// to the runner through Configurer.
type Settings struct {
	AutoResume     bool   `json:"autoResume"`     // start playing when the app starts
	MaxDeaths      int    `json:"maxDeaths"`      // pause a goal after this many deaths in it; 0 = never
	ChickenLifePct int    `json:"chickenLifePct"` // leave the game below this life percentage
	PotionLifePct  int    `json:"potionLifePct"`  // drink a healing potion below this
	Difficulty     string `json:"difficulty"`     // "auto" or normal, nightmare, hell
}

func DefaultSettings() Settings {
	return Settings{MaxDeaths: 5, ChickenLifePct: 30, PotionLifePct: 55, Difficulty: "auto"}
}

func (s Settings) Validate() error {
	if s.MaxDeaths < 0 || s.MaxDeaths > 1000 {
		return fmt.Errorf("maxDeaths must be 0..1000")
	}
	if s.ChickenLifePct < 0 || s.ChickenLifePct > 90 {
		return fmt.Errorf("chickenLifePct must be 0..90")
	}
	if s.PotionLifePct < 0 || s.PotionLifePct > 95 {
		return fmt.Errorf("potionLifePct must be 0..95")
	}
	if s.PotionLifePct != 0 && s.ChickenLifePct >= s.PotionLifePct {
		return fmt.Errorf("chickenLifePct must be below potionLifePct")
	}
	if s.Difficulty != "auto" && !validDifficulty(s.Difficulty) {
		return fmt.Errorf("difficulty must be auto, normal, nightmare or hell")
	}
	return nil
}
