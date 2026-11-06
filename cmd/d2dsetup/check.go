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
	Issue    string `json:"issue,omitempty"` // missing, not_mpq, too_small, unreadable (empty when fine)
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
	// Verdict is the one state the page shows: ok, missing, too_small, not_mpq, unreadable, problems (mixed),
	// resurrected, wrong_version, unconfirmed or no_folder. Fix says in one sentence what to do about it.
	Verdict string `json:"verdict"`
	Fix     string `json:"fix,omitempty"`
}

// CheckGame inspects the mounted game folder. It never writes and never fails as a whole: every problem is a
// row with a plain-language message.
func CheckGame(dir string, t *Tables) Report {
	r := Report{Dir: dir, OK: true, Version: "unverified"}
	entries, err := os.ReadDir(dir)
	if err != nil {
		r.OK = false
		r.Summary = fmt.Sprintf("Cannot read the game folder %s (%s). Mount your Diablo II folder read-only at /game.", dir, reason(err))
		r.Verdict, r.Fix = "no_folder", "Give the container your classic Diablo II folder (the one with the .mpq files), read-only, then press Check again."
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
		r.Verdict, r.Fix = "resurrected", "Point the container at the folder of your classic install (the one that has the .mpq files), then press Check again."
	case len(missing) > 0:
		r.Summary = "Problems with: " + strings.Join(missing, ", ") + "."
		r.Verdict, r.Fix = problemVerdict(r.Files)
	case r.Version == "mismatch":
		r.Summary = "Patch_D2.mpq is not a known 1.14d file. Patch the game to 1.14d and copy the folder again."
		r.Verdict, r.Fix = "wrong_version", "Install the 1.14d patch in your own game, copy the new Patch_D2.mpq here, then press Check again."
	case r.Version == "unverified":
		r.Summary = "All required game files are present, but the version cannot be confirmed yet."
		r.Verdict, r.Fix = "unconfirmed", "You may continue. Put the fingerprint in your report if the hero behaves strangely."
	default:
		r.Summary = "All required game files are present."
		r.Verdict = "ok"
	}
	return r
}

// problemVerdict names the one problem when every failing required file has the same issue, else "problems".
func problemVerdict(files []FileCheck) (string, string) {
	issue := ""
	for _, f := range files {
		if f.Status != StatusFail {
			continue
		}
		if issue != "" && issue != f.Issue {
			return "problems", "Fix the files listed below (copy them again from your classic Diablo II folder), then press Check again. No restart is needed."
		}
		issue = f.Issue
	}
	switch issue {
	case "missing":
		return "missing", "Copy the missing files from your classic Diablo II folder into the folder you gave the container, then press Check again. No restart is needed."
	case "too_small":
		return "too_small", "A copy may have stopped half way. Copy the file again, then press Check again."
	case "not_mpq":
		return "not_mpq", "Copy the original file again; this one is not a Diablo II archive."
	}
	return "unreadable", "Check the file permissions: the container user must be able to read the files."
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
			fc.Status, fc.Issue, fc.Message = StatusFail, "missing", f.Name+" is missing ("+f.Why+"). Copy it from your Diablo II folder."
		} else {
			fc.Status, fc.Message = StatusInfo, "optional, not found ("+f.Why+")"
		}
		return fc
	}
	p := filepath.Join(dir, actual)
	fi, err := os.Stat(p)
	if err != nil {
		return failOrInfo(fc, f, "unreadable", actual+" cannot be read ("+reason(err)+")")
	}
	fc.Size = fi.Size()
	if !fi.Mode().IsRegular() {
		return failOrInfo(fc, f, "unreadable", actual+" is not a regular file")
	}
	fh, err := os.Open(p)
	if err != nil {
		return failOrInfo(fc, f, "unreadable", actual+" cannot be read ("+reason(err)+"). Check the file permissions: the container user must be able to read it.")
	}
	defer fh.Close()
	var head [4]byte
	if _, err := io.ReadFull(fh, head[:]); err != nil || string(head[:3]) != "MPQ" || (head[3] != 0x1a && head[3] != 0x1b) {
		return failOrInfo(fc, f, "not_mpq", actual+" is not a Diablo II archive (wrong file header). Copy the original file.")
	}
	if fi.Size() < f.MinSize {
		return failOrInfo(fc, f, "too_small", fmt.Sprintf("%s is only %s, too small: the copy looks incomplete.", actual, human(fi.Size())))
	}
	fc.Status, fc.Message = StatusPass, human(fi.Size())+", readable"
	return fc
}

func failOrInfo(fc FileCheck, f FileSpec, issue, msg string) FileCheck {
	fc.Message, fc.Issue = msg, issue
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
