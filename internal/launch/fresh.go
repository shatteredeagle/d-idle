package launch

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Errors of the fresh-character flow.
var (
	ErrBadName      = errors.New("invalid character name")
	ErrVerifyFailed = errors.New("created character does not match the request")
)

// Classes returns the seven class keys the create screen supports, sorted.
func Classes() []string {
	out := make([]string, 0, len(ClassClick))
	for c := range ClassClick {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

// ValidCharName applies the classic D2 naming rules the create screen enforces: 2 to 15 characters, letters
// only, at most one '-' or '_' that is neither first nor last. UNVERIFIED live for the digit and separator
// rules (kolbot and the 1.14d client reject digits); the checks are deliberately stricter than the game so
// a name that passes here never reaches an in-game error popup.
func ValidCharName(name string) error {
	if len(name) < 2 || len(name) > 15 {
		return fmt.Errorf("%w: %q must be 2 to 15 characters", ErrBadName, name)
	}
	sep := 0
	for i, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
		case (r == '-' || r == '_') && i > 0 && i < len(name)-1:
			sep++
		default:
			return fmt.Errorf("%w: %q has %q (letters only, one inner - or _)", ErrBadName, name, r)
		}
	}
	if sep > 1 {
		return fmt.Errorf("%w: %q has more than one separator", ErrBadName, name)
	}
	return nil
}

// Who is what the running game says about the character it has just entered.
type Who struct {
	Name  string
	Class string // "Sorceress", ...
	Level int
}

// FreshOptions drives CreateVerified.
type FreshOptions struct {
	Attempts int                 // create attempts for a failure that left nothing behind; default 2
	Who      func() (Who, error) // reads the in-game player; required
	Leave    time.Duration       // Save and Exit budget after verifying; default 25 s, negative = stay in game
	Log      func(format string, a ...any)
}

// CreateVerified creates a softcore expansion Normal character and proves it: the in-game player must carry
// the requested name and class at level 1, then (unless Leave < 0) the game is left with Save and Exit so
// the .d2s exists on disk. Retries are bounded and only for failures that cannot have created the character
// (class not confirmed, stuck before OK); a name clash or a popup is final, because a retry would then
// either duplicate or select an older character. Never call this for a name that is not a throwaway.
func (f *Flow) CreateVerified(ctx context.Context, name, class string, o FreshOptions) error {
	class = strings.ToLower(class)
	if _, ok := ClassClick[class]; !ok {
		return fmt.Errorf("unknown class %q (have %s)", class, strings.Join(Classes(), ", "))
	}
	if err := ValidCharName(name); err != nil {
		return err
	}
	if o.Who == nil {
		return errors.New("CreateVerified: Who reader required")
	}
	logf := o.Log
	if logf == nil {
		logf = func(string, ...any) {}
	}
	n := o.Attempts
	if n <= 0 {
		n = 2
	}
	var err error
	for i := 1; i <= n; i++ {
		if err = f.CreateCharacter(ctx, name, class); err == nil {
			break
		}
		logf("create attempt %d/%d: %v", i, n, err)
		if errors.Is(err, ErrNameTaken) || errors.Is(err, ctx.Err()) || !(errors.Is(err, ErrWrongClass) || errors.Is(err, ErrStuck) || errors.Is(err, ErrTimeout)) {
			return err
		}
		// Back to a clean create screen state before trying again: the flow itself recovers from char-create.
	}
	if err != nil {
		return err
	}
	// The player unit can lag a moment behind the phase flip: retry the read errors only.
	var w Who
	for i := 0; i < 15; i++ {
		if w, err = o.Who(); err == nil {
			break
		}
		f.sleep(time.Second)
	}
	if err != nil {
		return fmt.Errorf("%w: reading the player: %v", ErrVerifyFailed, err)
	}
	if !strings.EqualFold(w.Name, name) || !strings.EqualFold(w.Class, class) || w.Level != 1 {
		return fmt.Errorf("%w: wanted %s/%s level 1, game says %s/%s level %d", ErrVerifyFailed, name, class, w.Name, w.Class, w.Level)
	}
	logf("verified in game: %s %s level %d", w.Name, w.Class, w.Level)
	if o.Leave < 0 {
		return nil
	}
	d := o.Leave
	if d == 0 {
		d = 25 * time.Second
	}
	return f.LeaveGame(ctx, d)
}
