package launch

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// RegKey is the Diablo II settings key inside the prefix. VERIFIED in user.reg of the live install.
const RegKey = `Software\Blizzard Entertainment\Diablo II`

// ReadUserReg reads values of one HKCU key from the prefix's user.reg (Wine's text registry). It is
// read-only and safe while Wine runs (Wine rewrites the file only at its own save points, so values may
// lag by a few minutes; UNVERIFIED how long). dword values come back as decimal strings.
func (c Config) ReadUserReg(key string) (map[string]string, error) {
	f, err := os.Open(filepath.Join(c.PrefixDir, "pfx", "user.reg"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return parseRegSection(bufio.NewScanner(f), key)
}

func parseRegSection(sc *bufio.Scanner, key string) (map[string]string, error) {
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	want := "[" + strings.ReplaceAll(key, `\`, `\\`) + "]"
	out := map[string]string{}
	in := false
	for sc.Scan() {
		l := sc.Text()
		if strings.HasPrefix(l, "[") {
			if in {
				break
			}
			in = strings.HasPrefix(l, want+" ") || l == want
			continue
		}
		if !in || !strings.HasPrefix(l, `"`) {
			continue
		}
		k, v, ok := strings.Cut(l, `"=`)
		if !ok {
			continue
		}
		k = strings.ReplaceAll(strings.TrimPrefix(k, `"`), `\\`, `\`)
		switch {
		case strings.HasPrefix(v, "dword:"):
			n, _ := strconv.ParseUint(v[6:], 16, 32)
			v = strconv.FormatUint(n, 10)
		case strings.HasPrefix(v, `"`):
			v = strings.ReplaceAll(strings.Trim(v, `"`), `\\`, `\`)
		}
		out[k] = v
	}
	if !in && len(out) == 0 {
		return nil, fmt.Errorf("key %s not found", key)
	}
	return out, sc.Err()
}

// SaveDir returns the host path of the character save directory ("Saved Games\Diablo II" in the
// prefix; VERIFIED NewSavePath = C:\users\steamuser\Saved Games\Diablo II\). The bot uses it to back
// up .d2s files before risky operations.
func (c Config) SaveDir() string {
	return filepath.Join(c.PrefixDir, "pfx", "drive_c", "users", "steamuser", "Saved Games", "Diablo II")
}

// RegAddArgs returns the arguments for `proton run` that set one HKCU DWORD of the Diablo II key, for
// writing settings through Wine itself instead of editing user.reg behind its back. UNVERIFIED: reg.exe
// under Proton behaves as documented by Wine, but this exact call was not run on a real install.
func RegAddArgs(name string, dword uint32) []string {
	return []string{"reg.exe", "add", `HKCU\` + RegKey, "/v", name, "/t", "REG_DWORD", "/d", strconv.FormatUint(uint64(dword), 10), "/f"}
}
