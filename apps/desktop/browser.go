package main

import (
	"os"
	"os/exec"
	"path/filepath"
)

// browserNames are tried on PATH in this order (Linux); Chromium-family browsers all understand --app.
var browserNames = []string{"google-chrome-stable", "google-chrome", "chromium", "chromium-browser", "microsoft-edge-stable", "microsoft-edge", "brave-browser"}

// windowsBrowsers are paths relative to the ProgramFiles(x86), ProgramFiles and LocalAppData roots. Edge ships with
// Windows 10/11, so the Windows path normally ends at the first entry.
var windowsBrowsers = []string{
	`Microsoft\Edge\Application\msedge.exe`,
	`Google\Chrome\Application\chrome.exe`,
	`BraveSoftware\Brave-Browser\Application\brave.exe`,
}

// FindBrowser returns the Chromium-family executable to use, or "" when there is none (the caller then opens the
// default browser in a normal tab). override is $D2IDLE_BROWSER. lookPath and exists are injected for tests.
func FindBrowser(goos, override string, roots []string, lookPath func(string) (string, error), exists func(string) bool) string {
	if override != "" {
		if p, err := lookPath(override); err == nil {
			return p
		}
		return ""
	}
	if goos == "windows" {
		for _, rel := range windowsBrowsers {
			for _, r := range roots {
				if r == "" {
					continue
				}
				if p := filepath.Join(r, rel); exists(p) {
					return p
				}
			}
		}
		return ""
	}
	for _, n := range browserNames {
		if p, err := lookPath(n); err == nil {
			return p
		}
	}
	return ""
}

// AppArgs is the command line of an app-mode window: no tabs or address bar, its own profile (so the watch session
// cookie persists and the user's normal browser profile is untouched), its own window class for the task bar.
func AppArgs(url, profile string) []string {
	return []string{
		"--app=" + url,
		"--user-data-dir=" + profile,
		"--class=d2idle-watch",
		"--no-first-run",
		"--no-default-browser-check",
		"--window-size=1280,900",
	}
}

func systemBrowser() string {
	roots := []string{os.Getenv("ProgramFiles(x86)"), os.Getenv("ProgramFiles"), os.Getenv("LocalAppData")}
	return FindBrowser(goos, os.Getenv("D2IDLE_BROWSER"), roots, exec.LookPath, func(p string) bool { _, err := os.Stat(p); return err == nil })
}

// openWindow starts the app window (or, with no Chromium-family browser, the default browser) and does not wait.
func openWindow(url, profile string) error {
	var cmd *exec.Cmd
	if b := systemBrowser(); b != "" {
		cmd = exec.Command(b, AppArgs(url, profile)...)
	} else if goos == "windows" {
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	} else {
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
