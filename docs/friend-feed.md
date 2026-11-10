# Friend feed: D2BOT_UPDATE_FEED

> status: BUILT offline (TASK-272, 2026-11-10); never run against a real feed. Design: docs/research/friend-build.md section 5.4.

A normal (non-DEV) image reads releases from GitHub. Setting `D2BOT_UPDATE_FEED=https://host[:port][/path]` makes it read a static directory of your own instead.

- Shape: the same JSON the GitHub API returns at `<feed>/repos/<repo>/releases` (V internal/selfupdate/fetch.go, the DEV server does the same); assets (`MANIFEST.json`, `MANIFEST.json.sig`, tarballs) are listed with `browser_download_url` on the SAME host. A redirect or asset URL on another host is refused (exact host match, no wildcard).
- Value rules (V feed_test.go): https only, a host, no credentials, query or fragment. A bad value is logged and ignored, so the image falls back to GitHub, never to an unchecked source.
- No trust is added (V TestNewUpdaterFeedAddsNoTrust): the keys are the baked `pubkeys/*.pub` only; the DEV key and `D2BOT_UPDATE_PUBKEY` still need `DIDLE_DEV=1`. A feed signed by any other key offers nothing (TestFeedWrongKeyRefused); tampered manifest (TestFeedTamperedManifestRefused) and missing signature (TestFeedUnsignedRefused) are refused. Downgrade, revoke and hash checks are the same engine as before.
- DEV images with `D2BOT_UPDATE_URL` keep that source; the feed is ignored there.
- `D2BOT_UPDATE_CHECK=off|manual` and the banner and Update button are unchanged: one engine.

Hosting (U, the user's choice): any static HTTPS host works (object storage, a web server directory). The publisher copies signed release assets there; the feed script is a later PR (friend-build.md section 8). The beta signing key stays outside the repo (section 5.2).
