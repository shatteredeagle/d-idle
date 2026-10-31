package launch

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Optional graphics wrapper hook (D2DX, GPL-3.0, github.com/bolrog/d2dx; supports 1.14d per its README).
// Not required and not shipped: the user supplies glide3x.dll. D2DX installs by copying glide3x.dll into
// the game folder and launching Game.exe with -3dfx. Whether it works under Proton is UNVERIFIED (it is
// a Direct3D 11 renderer, which DXVK should translate). The bot does not need it: the stock DirectDraw
// path renders fine windowed 800x600 (VERIFIED).

// InstallGlideWrapper copies dll into GameDir as glide3x.dll and returns the extra game argument needed.
func InstallGlideWrapper(cfg Config, dll string) (extraArg string, err error) {
	src, err := os.Open(dll)
	if err != nil {
		return "", err
	}
	defer src.Close()
	dst, err := os.Create(filepath.Join(cfg.GameDir, "glide3x.dll"))
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		return "", err
	}
	if err := dst.Close(); err != nil {
		return "", fmt.Errorf("write glide3x.dll: %w", err)
	}
	return "-3dfx", nil
}

// RemoveGlideWrapper deletes glide3x.dll from GameDir (the game then falls back to DirectDraw when
// launched without -3dfx).
func RemoveGlideWrapper(cfg Config) error {
	err := os.Remove(filepath.Join(cfg.GameDir, "glide3x.dll"))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
