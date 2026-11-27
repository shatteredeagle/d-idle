package agency

// ModeDrive is mode 2 of the knob agency.planner: the planner's pick replaces the combat profile's (agencyp1b,
// docs/design/agency-p1b.md). The wiring into the fighter is not built; this is the decision the fighter would ask for.
const ModeDrive = 2

// Drive is what mode 2 hands the fighter for one plan.
type Drive struct {
	Attack string // the attack to use instead of the profile's; "" = keep the profile's
	Aura   string // the aura to use instead; "" = keep the profile's
	Why    string // one line for the log; always set
}

// DriveOf decides the override. The planner only ever overrides with a Supported attack it has points in and that
// differs from the profile's pick (Plan.Attack is already the Prior unless the estimate beats it by the margin), and
// never when the plan carries a warning that makes the pick unreliable (unresolved archetype). Everything else keeps
// the profile, so mode 2 can only differ from mode 0 where the shadow line already shows a different pick.
func DriveOf(pl Plan) Drive {
	if pl.Archetype == "" {
		return Drive{Why: "keep profile: archetype not resolved"}
	}
	var d Drive
	if a := pl.Attack; a != nil && a.Tier == Supported && a.Skill != pl.Prior && pl.Prior != "" {
		d.Attack = a.Skill
	}
	if au := pl.Aura; au != nil && au.Tier == Supported && au.Skill != pl.PriorAura && pl.PriorAura != "" {
		d.Aura = au.Skill
	}
	switch {
	case d.Attack != "" && d.Aura != "":
		d.Why = "drive: attack " + d.Attack + " (profile " + pl.Prior + "), aura " + d.Aura + " (profile " + pl.PriorAura + ")"
	case d.Attack != "":
		d.Why = "drive: attack " + d.Attack + " (profile " + pl.Prior + ")"
	case d.Aura != "":
		d.Why = "drive: aura " + d.Aura + " (profile " + pl.PriorAura + ")"
	default:
		d.Why = "keep profile: the plan agrees"
	}
	return d
}
