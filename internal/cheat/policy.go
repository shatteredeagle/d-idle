package cheat

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"d2lod-bot/internal/mem"
)

// Allowed lists the built-in throwaway characters cheats may touch (AGENTS.md). Anything else is refused unless the
// config allow-list below names it. Palli and Palli2 are the throwaway Paladins (i1 smoke test, i2 leveler run).
var Allowed = []string{"Testy", "Botty", "Navi", "Combi", "Towni", "Booti", "Multi", "Palli", "Palli2", "Amazi", "Necki", "Druki", "Assi", "Barby"}

// Config-driven additions, read on every check so a new throwaway character needs no code change or rebuild:
//
//	D2CHEAT_ALLOW       comma-separated names
//	D2CHEAT_ALLOWFILE   path of a file with one name per line ('#' comments); default ~/.config/d2lod/cheat-allow.txt
//
// The same character names are what internal/cheat refuses on every other path, so only name throwaway characters.
const (
	AllowEnv     = "D2CHEAT_ALLOW"
	AllowFileEnv = "D2CHEAT_ALLOWFILE"
)

func allowFilePath() string {
	if p := os.Getenv(AllowFileEnv); p != "" {
		return p
	}
	if h, err := os.UserHomeDir(); err == nil {
		return filepath.Join(h, ".config", "d2lod", "cheat-allow.txt")
	}
	return ""
}

// AllowList returns every name currently allowed: built-ins, the environment list and the allow file.
func AllowList() []string {
	out := append([]string(nil), Allowed...)
	for _, n := range strings.Split(os.Getenv(AllowEnv), ",") {
		if n = strings.TrimSpace(n); n != "" {
			out = append(out, n)
		}
	}
	if p := allowFilePath(); p != "" {
		if b, err := os.ReadFile(p); err == nil {
			for _, ln := range strings.Split(string(b), "\n") {
				if i := strings.IndexByte(ln, '#'); i >= 0 {
					ln = ln[:i]
				}
				if ln = strings.TrimSpace(ln); ln != "" {
					out = append(out, ln)
				}
			}
		}
	}
	return out
}

var (
	ErrNoReason    = errors.New("cheat: a reason is required")
	ErrNotAllowed  = errors.New("cheat: character not on the allowlist")
	ErrNotOffline  = errors.New("cheat: not an offline single-player game")
	ErrGameRunning = errors.New("cheat: save edits need the instance's game stopped")
)

// CheckCharacter applies the allowlist (case-insensitive: the save file name and the in-game name differ in case rules).
func CheckCharacter(name string) error {
	for _, a := range AllowList() {
		if strings.EqualFold(a, name) {
			return nil
		}
	}
	return fmt.Errorf("%w: %q", ErrNotAllowed, name)
}

// GameInfo layout (docs/d2-1.14d-memory.md, offsets shifted +4 in 1.14d): the pointer is at RVA 0x3A0438,
// szGameServerIp +0x37, szAccountName +0x8D, szCharName +0xBD, szRealmName +0xD5.
const (
	rvaGameInfo = 0x3A0438
	giServerIP  = 0x37
	giAccount   = 0x8D
	giCharName  = 0xBD
	giRealm     = 0xD5
)

func cstr(src mem.Source, addr uint32, n int) (string, error) {
	b := make([]byte, n)
	if err := src.ReadAt(b, addr); err != nil {
		return "", err
	}
	for i, c := range b {
		if c == 0 {
			return string(b[:i]), nil
		}
	}
	return string(b), nil
}

// OfflineInfo reads the online markers of the running game: for single player they must all be empty.
// Whether they are empty offline was verified live (docs/cheats.md); a populated one refuses the cheat.
func OfflineInfo(src mem.Source) (server, account, realm string, err error) {
	gi, err := mem.U32(src, mem.Base+rvaGameInfo)
	if err != nil {
		return "", "", "", err
	}
	if gi < 0x10000 {
		return "", "", "", errors.New("cheat: no game info (not in a game)")
	}
	if server, err = cstr(src, gi+giServerIP, 0x56); err != nil {
		return
	}
	if account, err = cstr(src, gi+giAccount, 0x30); err != nil {
		return
	}
	realm, err = cstr(src, gi+giRealm, 0x18)
	return
}

// CheckOffline refuses anything that looks like a Battle.net or TCP/IP game.
func CheckOffline(src mem.Source) error {
	server, account, realm, err := OfflineInfo(src)
	if err != nil {
		return err
	}
	if server != "" || account != "" || realm != "" {
		return fmt.Errorf("%w (server=%q account=%q realm=%q)", ErrNotOffline, server, account, realm)
	}
	return nil
}
