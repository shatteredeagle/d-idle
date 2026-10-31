// Package setup is the backend of the setup wizard and of `d2doctor`: data-driven host and game checks
// (check.go, checks.go) and a resumable setup state machine (wizard.go, steps.go) with an HTTP contract
// (http.go, docs/setup.md).
//
// Rules: nothing Blizzard owns is downloaded or bundled (every step takes a path the user supplies);
// CD keys are typed by the user in the installer window and never enter this package (the HTTP input
// decoder rejects unknown fields and log lines are scrubbed, see Redact).
package setup
