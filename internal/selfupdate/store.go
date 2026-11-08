package selfupdate

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Caps on what an unpack may write, so a hostile (but correctly signed or corrupted) tarball cannot fill the disk.
const (
	maxEntryBytes = 600 << 20
	maxTotalBytes = 900 << 20
	maxEntries    = 2000
)

// Unpack extracts a tar.gz into dir. Only plain files and directories are accepted; absolute names, "..",
// links, devices and anything that resolves outside dir are refused (design 3.12).
func Unpack(tgz io.Reader, dir string) error {
	gz, err := gzip.NewReader(tgz)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	var total int64
	for n := 0; ; n++ {
		if n > maxEntries {
			return errors.New("unpack: too many entries")
		}
		h, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		name := path.Clean(h.Name)
		if path.IsAbs(h.Name) || name == ".." || strings.HasPrefix(name, "../") || strings.ContainsRune(h.Name, 0) || strings.Contains(h.Name, `\`) {
			return fmt.Errorf("unpack: unsafe path %q", h.Name)
		}
		target := filepath.Join(dir, filepath.FromSlash(name))
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if h.Size < 0 || h.Size > maxEntryBytes {
				return fmt.Errorf("unpack: %q too large", h.Name)
			}
			if total += h.Size; total > maxTotalBytes {
				return errors.New("unpack: archive too large")
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			mode := os.FileMode(0o644)
			if h.Mode&0o111 != 0 {
				mode = 0o755
			}
			f, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
			if err != nil {
				return err
			}
			_, cerr := io.Copy(f, io.LimitReader(tr, h.Size))
			if err := f.Close(); cerr == nil {
				cerr = err
			}
			if cerr != nil {
				return cerr
			}
		default:
			return fmt.Errorf("unpack: %q: entry type %q is not allowed", h.Name, string(h.Typeflag))
		}
	}
}

// InstallRelease unpacks every file of m found in stagingDir into releases/<ver>.tmp/<dest> and renames the
// directory to releases/<ver>. The signed manifest and its signature are kept inside (MANIFEST.json, MANIFEST.json.sig)
// so the boot shim can re-check the compatibility gate and the signature. A leftover .tmp from an earlier crash is removed first. selftest (may be nil) runs
// on the finished directory before the rename.
func (l Layout) InstallRelease(m Manifest, raw, sig []byte, stagingDir string, selftest func(dir string) error) error {
	final := l.Release(m.Version)
	if _, err := os.Stat(final); err == nil {
		return fmt.Errorf("selfupdate: release %s already exists", m.Version)
	}
	tmp := final + ".tmp"
	os.RemoveAll(tmp)
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		return err
	}
	ok := false
	defer func() {
		if !ok {
			os.RemoveAll(tmp)
		}
	}()
	if err := os.WriteFile(filepath.Join(tmp, ManifestName), raw, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(tmp, ManifestName+".sig"), sig, 0o644); err != nil {
		return err
	}
	for _, f := range m.Files {
		fh, err := os.Open(filepath.Join(stagingDir, f.Name))
		if err != nil {
			return err
		}
		err = Unpack(fh, filepath.Join(tmp, filepath.FromSlash(f.Dest)))
		fh.Close()
		if err != nil {
			return fmt.Errorf("%s: %w", f.Name, err)
		}
	}
	if selftest != nil {
		if err := selftest(tmp); err != nil {
			return fmt.Errorf("selftest: %w", err)
		}
	}
	if err := os.Rename(tmp, final); err != nil {
		return err
	}
	ok = true
	return nil
}

// SwapCurrent points `current` at releases/<ver> with an atomic rename of a temporary symlink.
func (l Layout) SwapCurrent(ver string) error {
	if _, err := os.Stat(l.Release(ver)); err != nil {
		return err
	}
	tmp := l.Current() + ".new"
	os.Remove(tmp)
	if err := os.Symlink(filepath.Join("releases", ver), tmp); err != nil {
		return err
	}
	return os.Rename(tmp, l.Current())
}

// CurrentVersion is the release `current` points at ("" when there is none).
func (l Layout) CurrentVersion() string {
	t, err := os.Readlink(l.Current())
	if err != nil {
		return ""
	}
	return filepath.Base(t)
}

// Prune deletes every release directory except those named in keep (the current and the previous release),
// and any leftover .tmp directories.
func (l Layout) Prune(keep ...string) error {
	ents, err := os.ReadDir(l.Releases())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	keepSet := map[string]bool{}
	for _, k := range keep {
		keepSet[k] = true
	}
	for _, e := range ents {
		if !keepSet[e.Name()] {
			if err := os.RemoveAll(filepath.Join(l.Releases(), e.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}
