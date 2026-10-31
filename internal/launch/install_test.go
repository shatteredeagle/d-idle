package launch

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseRegSection(t *testing.T) {
	reg := `[Software\\Blizzard Entertainment\\Diablo II] 1793204161
#time=1dd66f79fb2b3f8
"CmdLine"="-w -skiptobnet"
"Contrast"=dword:00000064
"InstallPath"="C:\\Program Files (x86)\\Diablo II\\"

[Software\\Blizzard Entertainment\\Diablo II\\VideoConfig] 1793203673
"Render"=dword:00000001
`
	m, err := parseRegSection(bufio.NewScanner(strings.NewReader(reg)), RegKey)
	if err != nil {
		t.Fatal(err)
	}
	if m["Contrast"] != "100" || m["CmdLine"] != "-w -skiptobnet" || m["InstallPath"] != `C:\Program Files (x86)\Diablo II\` || m["Render"] != "" {
		t.Fatalf("%v", m)
	}
	v, _ := parseRegSection(bufio.NewScanner(strings.NewReader(reg)), RegKey+`\VideoConfig`)
	if v["Render"] != "1" {
		t.Fatalf("%v", v)
	}
	if _, err := parseRegSection(bufio.NewScanner(strings.NewReader(reg)), `Software\Nope`); err == nil {
		t.Fatal("want error")
	}
}

func TestVerifyInstallAndVersion(t *testing.T) {
	root := t.TempDir()
	cfg := Config{Root: root, PrefixDir: filepath.Join(root, "proton"), GameDir: filepath.Join(root, "g")}
	in := &Installer{Cfg: cfg}
	os.MkdirAll(cfg.GameDir, 0o755)
	if in.verifyBase() == nil {
		t.Fatal("empty dir must not verify")
	}
	w := func(n, s string) { os.WriteFile(filepath.Join(cfg.GameDir, n), []byte(s), 0o644) }
	w("Diablo II Install Log.html", "Installation completed successfully.")
	for _, f := range []string{"d2data.mpq", "d2char.mpq", "d2sfx.mpq"} {
		w(f, "x")
	}
	w("Game.exe", string(buildPE(Version{1, 13, 0, 0})))
	if err := in.verifyBase(); err != nil {
		t.Fatal(err)
	}
	if in.verifyVersion() == nil {
		t.Fatal("1.13 must not pass as 1.14d")
	}
	w("Game.exe", string(buildPE(Want114d)))
	if err := in.verifyVersion(); err != nil {
		t.Fatal(err)
	}
}
