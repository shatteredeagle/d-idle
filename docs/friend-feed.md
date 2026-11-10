# Friend feed: D2BOT_UPDATE_FEED

> status: BUILT offline (TASK-272, 2026-11-10); never run against a real feed. Design: docs/research/friend-build.md section 5.4.

A normal (non-DEV) image reads releases from GitHub. Setting `D2BOT_UPDATE_FEED=https://host[:port][/path]` makes it read a static directory of your own instead.

- Shape: the same JSON the GitHub API returns at `<feed>/repos/<repo>/releases` (V internal/selfupdate/fetch.go, the DEV server does the same); assets (`MANIFEST.json`, `MANIFEST.json.sig`, tarballs) are listed with `browser_download_url` on the SAME host. A redirect or asset URL on another host is refused (exact host match, no wildcard).
- Value rules (V feed_test.go): https only, a host, no credentials, query or fragment. A bad value is logged and ignored, so the image falls back to GitHub, never to an unchecked source.
- No trust is added (V TestNewUpdaterFeedAddsNoTrust): the keys are the baked `pubkeys/*.pub` only; the DEV key and `D2BOT_UPDATE_PUBKEY` still need `DIDLE_DEV=1`. A feed signed by any other key offers nothing (TestFeedWrongKeyRefused); tampered manifest (TestFeedTamperedManifestRefused) and missing signature (TestFeedUnsignedRefused) are refused. Downgrade, revoke and hash checks are the same engine as before.
- DEV images with `D2BOT_UPDATE_URL` keep that source; the feed is ignored there.
- `D2BOT_UPDATE_CHECK=off|manual` and the banner and Update button are unchanged: one engine.

Hosting (U, the user's choice): any static HTTPS host works (object storage, a web server directory). The publisher copies signed release assets there; the feed script is a later PR (friend-build.md section 8). The beta signing key stays outside the repo (section 5.2). Two keys are pinned in the image: `pubkeys/beta.pub` signs this feed, `pubkeys/current.pub` signs GitHub releases (its private half is held offline by the owner). The updater accepts a manifest signed by either and refuses any other key (V TestFeedTwoPinnedKeys, TestSignVerify); the key count is not asserted anywhere else, `TestPinnedKeysBothPresent` checks both files parse and differ.

## Release runbook (TASK-275)

`tools/release/feed-publish.sh` and `cmd/d2feed` turn one signed release into a static tree and check it with the real client before anything is uploaded. Host-agnostic: the example host below (a static https directory on the NAS behind the existing tunnel) is a likely choice, not a decision (U, with the user); any static https host works. Nothing here is stored or run by CI: the key path is passed by the person releasing and only ever reaches `d2sign`.

Tree written (V cmd/d2feed tests): `repos/<owner>/<repo>/releases` (the GitHub API list, newest first, drafts hidden from clients) and `dl/app-v<ver>/<asset>` for every file of the manifest, `MANIFEST.json`, `MANIFEST.json.sig`, optional `REVOKED` and `REVOKED.sig`, `SOURCE.txt`. Asset URLs are `<base-url>/dl/app-v<ver>/<name>`; the base URL is exactly what friends set as `D2BOT_UPDATE_FEED` (https only).

### Cut a release

1. Notes first: the CHANGELOG section for the version must exist (rule: every release has notes). The manifest `notes` field carries the text the friend sees.
2. Build the assets and the unsigned manifest: `tools/release/app-assets.sh --version 0.4.0 --seq 17 --issued <UTC time> --out REL --d2d-bin PATH --channel beta ...` (docs/self-update-release.md). `--channel` is where "bump channel" happens: a beta feed carries `beta` manifests only; the client does not filter by it (U, friend-build.md section 6). `seq` must rise with every release.
3. Read `d2sign summary -in REL/MANIFEST.json` (file names, sizes, sources) before signing.
4. Publish and verify: `tools/release/feed-publish.sh --release REL --base-url https://FEED/d2 --out FEEDTREE --sign-key /path/outside/the/repo/beta.key` (the real beta key; without `--sign-key` the script refuses a release with no `MANIFEST.json.sig`). It verifies signature, file hashes and REVOKED, adds the release, then runs the real client against the tree with the pinned key (`--pub`, default `deploy/d2d-container/pubkeys/beta.pub`). Nothing is uploaded yet.
5. Upload: rerun with `--upload`. Set one of `D2_FEED_UPLOAD_CMD` (a hook, run as `sh -c "$D2_FEED_UPLOAD_CMD" feed-upload TREE`) or `D2_FEED_RSYNC_DEST` (an rsync destination; assets go first, the list last, never `--delete`) in the environment or the gitignored `local.env`. No host or credential is in the repo.
6. Canary: your own container takes the release first (`./didle update`), ten minutes of play, then tell the friends.

Publishing the same release again changes nothing. A different release with an existing version is refused, and so is a version below the newest listed one. Old releases and their files stay in the tree.

### Rollback

- Bad release, nobody should take it: `feed-publish.sh --withdraw 0.5.0 --base-url ... --out FEEDTREE [--upload]`. Clients then see the next highest listed release (files stay).
- Friends who already installed it: automatic rollback covers a failed start; a release that plays badly is undone by `./didle` rollback in the app or, for everyone, by a higher fixed version, or a signed `REVOKED` file (`d2sign revoked ...`) next to the newest manifest (docs/self-update-release.md). A revoked or lower-seq release is refused by the client.
- Suspected key theft: ship a new pack with a new baked key (friend-build.md section 5.2).

### Static server snippets

The client asks `<base>/repos/<owner>/<repo>/releases?per_page=10`; a static server ignores the query and must serve the file `releases` as JSON.

Caddy (the directory is `FEEDTREE`, the tunnel terminates TLS):

```
feed.example {
	root * /srv/feed
	@api path /repos/*
	header @api Content-Type application/json
	file_server
}
```

nginx:

```
server {
	listen 443 ssl;
	server_name feed.example;
	root /srv/feed;
	location /repos/ { default_type application/json; }
	location /dl/    { default_type application/octet-stream; }
}
```

If the feed lives under a path (`https://host/d2`), mount the tree at that path and use the same URL as `--base-url`.
