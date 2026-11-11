package watch

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The gear-policy prompt (invdesk View.prompt, docs/gear-policy.md) is a note above the equipment on the inventory
// page: role=note, styled by tokens (a d-note), shown and hidden by the state alone, warning kind for the no-weapon
// text, and it never moves focus or scroll.
func TestInventoryGearPromptNote(t *testing.T) {
	if !strings.Contains(invPage, `<p class="d-note" data-sev="info" id="gearprompt" role="note" hidden></p>`) {
		t.Fatal("the inventory page has no gear prompt note")
	}
	if strings.Index(invPage, `id="gearprompt"`) > strings.Index(invPage, `id="equip"`) {
		t.Error("the prompt must sit above the equipment")
	}
	b, err := webFS.ReadFile("web/inventory.js")
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?s)function promptSev\(.*?\n  function paint\(`).Find(b)
	if m == nil {
		t.Fatal("inventory.js lost gearPrompt")
	}
	src := strings.TrimSuffix(string(m), "function paint(")
	if regexp.MustCompile(`focus\(|scroll|\.style\b|innerHTML|createElement`).MatchString(src) {
		t.Error("the prompt note must not touch focus, scroll or styles")
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not available")
	}
	js := `var n = { textContent: '', dataset: {}, hidden: true, writes: 0 }, text = '';
Object.defineProperty(n, 'textContent', { get() { return text; }, set(v) { text = v; n.writes++; } });
function $(id) { return id === 'gearprompt' ? n : null; }
` + src + `
const assert = require('assert');
gearPrompt('Your hero wears its starting gear.'); assert.deepStrictEqual([n.hidden, n.dataset.sev, text], [false, 'info', 'Your hero wears its starting gear.']);
gearPrompt('Your hero wears its starting gear.'); assert.strictEqual(n.writes, 1, 'an unchanged prompt is not rewritten');
gearPrompt('No weapon equipped. Equip one on the Inventory page; the bot will not do it for you.'); assert.deepStrictEqual([n.hidden, n.dataset.sev], [false, 'warn']);
gearPrompt(undefined); assert.strictEqual(n.hidden, true, 'no prompt: the note is gone');
n = null; gearPrompt('x'); // a page without the note (the DOM golden stub) is fine
`
	f := filepath.Join(t.TempDir(), "p.js")
	os.WriteFile(f, []byte(js), 0o600)
	if out, err := exec.Command(node, f).CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
}
