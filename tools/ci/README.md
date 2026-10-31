# CI

- `test.sh` -- blocking test run with one retry for quarantined tests; `scripts.sh` -- bash -n and script tests.
- `doclinks.py` -- broken relative links in docs/*.md (run before moving or renaming a doc).
- `check.sh` -- gofmt, vet, repo hygiene (no MPQ/.d2s/.key/extracted tables, nothing over 1 MB outside testdata). Run locally before pushing.
- `flaky.txt` -- quarantined tests with owner; CI skips them in the blocking run and runs them report-only.
- `.forgejo/workflows/ci.yml` -- push/PR: check, ``test.sh`, `scripts.sh`, release job on tags. See docs/ci.md.
- `../../packaging/release.sh VERSION` -- versioned tarball.

Runner needs: see docs/ci.md (host runner, Go 1.27.3, no docker).
Preferred host: The worker host or the Mac mini via the lab (memory `lab-machines`); the 4-core the head box box can run it but is slow with `-race`.
Registering a runner needs a forge registration token (admin action, not done here).
