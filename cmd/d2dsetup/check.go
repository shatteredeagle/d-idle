package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Check statuses. "info" is for optional files: absent is fine.
const (
	StatusPass = "pass"
	StatusFail = "fail"
	StatusWarn = "warn"
	StatusInfo = "info"
)

type FileCheck struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
	Status   string `json:"status"`
	Size     int64  `json:"size,omitempty"`
	Message  string `json:"message"`
}

type Report struct {
	Dir   string      `json:"dir"`
	OK    bool        `json:"ok"` // no required file failed
	Files []FileCheck `json:"files"`
	// Version is the 1.14d verdict from the Patch_D2 hash; "unverified" while the hash list is empty (U).
	Version     string `json:"version"`
	PatchSHA256 string `json:"patch_sha256,omitempty"`
	Summary     string `json:"summary"`
}

// CheckGame inspects the mounted game folder. It never writes and never fails as a whole: every problem is a
// row with a plain-language message.
func CheckGame(dir string, t *Tables) Report {
	r := Report{Dir: dir, OK: true, Version: "unverified"}
	entries, err := os.ReadDir(dir)
	if err != nil {
		r.OK = false
		r.Summary = fmt.Sprintf("Cannot read the game folder %s (%s). Mount your Diablo II folder read-only at /game.", dir, reason(err))
		return r
	}
	byFold := map[string]string{}
	for _, e := range entries {
		byFold[strings.ToLower(e.Name())] = e.Name()
	}
	var missing []string
	for _, f := range t.Files {
		fc := checkFile(dir, byFold, f)
		if fc.Status == StatusFail {
			r.OK = false
			missing = append(missing, f.Name)
		}
		r.Files = append(r.Files, fc)
	}
	if r.OK {
		r.Version, r.PatchSHA256 = patchVersion(dir, byFold, t)
		if r.Version == "mismatch" {
			r.OK = false
		}
	}
	switch {
	case len(missing) > 0 && !hasAnyMPQ(byFold):
		r.Summary = "No Diablo II archive (.mpq) files in the folder. This build needs the classic Diablo II Lord of Destruction files, version 1.14d. Diablo II Resurrected files do not work."
	case len(missing) > 0:
		r.Summary = "Problems with: " + strings.Join(missing, ", ") + "."
	case r.Version == "mismatch":
		r.Summary = "Patch_D2.mpq is not a known 1.14d file. Patch the game to 1.14d and copy the folder again."
	default:
		r.Summary = "All required game files are present."
	}
	return r
}

func hasAnyMPQ(by map[string]string) bool {
	for k := range by {
		if strings.HasSuffix(k, ".mpq") {
			return true
		}
	}
	return false
}

func checkFile(dir string, by map[string]string, f FileSpec) FileCheck {
	fc := FileCheck{Name: f.Name, Required: f.Required}
	actual, ok := by[strings.ToLower(f.Name)]
	if !ok {
		if f.Required {
			fc.Status, fc.Message = StatusFail, f.Name+" is missing ("+f.Why+"). Copy it from your Diablo II folder."
		} else {
			fc.Status, fc.Message = StatusInfo, "optional, not found ("+f.Why+")"
		}
		return fc
	}
	p := filepath.Join(dir, actual)
	fi, err := os.Stat(p)
	if err != nil {
		return failOrInfo(fc, f, actual+" cannot be read ("+reason(err)+")")
	}
	fc.Size = fi.Size()
	if !fi.Mode().IsRegular() {
		return failOrInfo(fc, f, actual+" is not a regular file")
	}
	fh, err := os.Open(p)
	if err != nil {
		return failOrInfo(fc, f, actual+" cannot be read ("+reason(err)+"). Check the file permissions: the container user must be able to read it.")
	}
	defer fh.Close()
	var head [4]byte
	if _, err := io.ReadFull(fh, head[:]); err != nil || string(head[:3]) != "MPQ" || (head[3] != 0x1a && head[3] != 0x1b) {
		return failOrInfo(fc, f, actual+" is not a Diablo II archive (wrong file header). Copy the original file.")
	}
	if fi.Size() < f.MinSize {
		return failOrInfo(fc, f, fmt.Sprintf("%s is only %s, too small: the copy looks incomplete.", actual, human(fi.Size())))
	}
	fc.Status, fc.Message = StatusPass, human(fi.Size())+", readable"
	return fc
}

func failOrInfo(fc FileCheck, f FileSpec, msg string) FileCheck {
	fc.Message = msg
	if f.Required {
		fc.Status = StatusFail
	} else {
		fc.Status, fc.Message = StatusWarn, msg+" Optional file: it will be ignored."
	}
	return fc
}

// patchVersion hashes Patch_D2.mpq (about 6 MB). With an empty pinned list the verdict stays "unverified" and the
// hash is shown so a maintainer can pin it from a real 1.14d install (U).
func patchVersion(dir string, by map[string]string, t *Tables) (verdict, sum string) {
	name, ok := by["patch_d2.mpq"]
	if !ok {
		return "unverified", ""
	}
	f, err := os.Open(filepath.Join(dir, name))
	if err != nil {
		return "unverified", ""
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "unverified", ""
	}
	sum = hex.EncodeToString(h.Sum(nil))
	if len(t.PatchSHA256) == 0 {
		return "unverified", sum
	}
	for _, k := range t.PatchSHA256 {
		if strings.EqualFold(k, sum) {
			return "1.14d", sum
		}
	}
	return "mismatch", sum
}

func human(n int64) string {
	if n >= 1<<20 {
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	}
	return fmt.Sprintf("%d bytes", n)
}

// reason strips the path out of an os error so messages stay short and path-free.
func reason(err error) string {
	switch {
	case os.IsNotExist(err):
		return "not found"
	case os.IsPermission(err):
		return "permission denied"
	}
	if pe, ok := err.(*os.PathError); ok {
		return pe.Err.Error()
	}
	return err.Error()
}
