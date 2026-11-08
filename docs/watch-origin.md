# Same-origin rule for state-changing requests (watch login, setup wizard)

Code: `internal/access/sameorigin.go` (`access.SameOrigin`), used by the login POST (`internal/watch/gate.go`) and all non-GET requests of `cmd/d2dsetup`.

Bug (2026-11-08): the pages sent `Referrer-Policy: no-referrer`. U (not reproduced on a Mac): WebKit then sends `Origin: null` on the same-origin form POST, which never equals the request host, so login answered 403 "forbidden origin". Fix: pages now send `Referrer-Policy: same-origin` (nothing leaks cross-origin), and the check accepts the signals below.

Rule, in order:
1. `Sec-Fetch-Site` present: `same-origin` and `none` pass; `cross-site` and `same-site` fail (decisive, even if Origin looks right).
2. `Origin` a real URL: scheme http or https, host compared case-insensitively with the request Host; a missing port equals the scheme's default (80/443). Anything else fails.
3. `Origin: null`: judged by `Referer` (same host passes); no Referer fails.
4. No Origin: a `Referer` must match; no headers at all (curl, native clients) passes, since a browser cross-site POST always carries Origin or Sec-Fetch-Site.

A rejection logs Origin, whether a Referer was set, Sec-Fetch-Site and Host (no cookies, no form data). `internal/api` (dashboard API) still has its own older Origin check; not changed here.
