# Public status page (tools/publish)

Generates a safe, public status page for friends. Dry run by design: the generator only reads local files and
writes into a local directory. Nothing is published by it; `push_gist.sh` is the separate, gated future step.

## What goes in
- `status.md`: intro, milestone ladder (ASCII bars, rollup rule from `docs/milestones/milestones.json`), current work,
  what shipped, bot stats, last-updated UTC.
- `stats.json`: the same aggregate numbers. `{"live": false}` when paused.
- `scene.html` + `scene.json`: the stream scene below.
- `latest.jpg` (only with `--frame`): JPEG with all metadata segments stripped; downscaled to 640 px wide when
  ImageMagick (`magick`/`convert`) exists (V: not installed on the head box, so there it is strip-only).

Sources (allow-list, V in `tests/test_publish.py`):
- Board `~/d2lod-status/agents/*.json`: ONLY `task` (ASCII, 100 chars) and `state`; `done` cards are hidden. Notes, branches,
  instances, ids, items and agent names are never read into the output.
- Git: first-parent merge commits on `master` of the last `--days` (default 14) whose subject is `Merge <branch>: <text>`;
  only date and `<text>`. A subject the sanitizer flags is dropped (count printed), never rewritten.
- Stats URL (`--stats-url`): JSON object, only numeric keys `clvl_max xp_per_hour kills deaths uptime_seconds` are kept.
  Unreachable, non-JSON, no known key, or `ts` (unix seconds, if present) older than 10 min: page says paused. http(s) only.

## Stream scene (scene.html + scene.json)
A 16:9 dark TV page for a headless browser that feeds the live stream (`tools/publish/scene.py`). Written by default next to
index.html; `--no-scene` skips it. `scene.json` carries only: brand, `ts`/`updated`, up to 4 public work cards (same
work-titles rewrite as the status page), the gate ladder (Alpha, Beta, Good bot with %), up to 12 shipped one-liners and the
live stats (`stats.live` false = the page shows a "warming up" line). `scene.html` is a static shell: its one inline script
(marker `<script id="d2-scene">`) fetches `scene.json?t=<ms>` from its own origin every 60 s, keeps the last good data when a
fetch fails (dot turns red, text "offline"), shows the clock and "updated N min ago". No external resources, no fonts from the
network, calm motion only (ring fill, ticker, pulsing dot, row fade-in).
- Layout: design size 1280x720 (rem scales with the viewport width). `?tile=1` is the compact 640x360 layout: 2 work items,
  rings in a row, no stats strip, shorter ticker.
- Sanitizer: the scene marker is accepted for `scene.html` only. Relaxations over the status page script: the exact statement
  `fetch(U + "?t=" + Date.now(), ...)`, `location.search`, `createElement`. Everything else in the deny-list (innerHTML, other
  fetches, WebSocket, cookies, storage, URLs, `//`) still blocks. Tests: `SceneTests`.
- Screenshots (V, Playwright chromium on the worker host, 127.0.0.1 http.server): `docs/publish/scene-1280.png`, `scene-tile.png` (sample
  data, stats row filled in the 1280 one).
- `push_pages.sh` copies both files too, still behind `D2_PUBLISH_APPROVED=yes`. Capture URL: `<pages site>/scene.html`
  (`?tile=1` for the tile). U: not tested through the real capture/stream stack, and `@property` ring animation needs a
  Chromium-class browser (older engines show the final ring without animating).

## Watch live (Kick)
`tools/publish/live.json` is `{"kick": "https://kick.com/CHANNEL"}`. The shipped value `CHANNEL` (also `your_channel`, `changeme`, ...)
is a placeholder: the feature is off and the page shows nothing about streaming. Set a real channel (`https://kick.com/<name>`,
3 to 25 of `A-Za-z0-9_`, nothing else) to turn it on.

    publish.py ... --stream-url http://worker.example:8433/api/stream/status --stream-token-file PATH [--live-json FILE]

The token file is read at runtime and sent only as `Authorization: Bearer`; it is never printed, stored or rendered. The
response is read for aggregate fields only: `state` (`live` counts, or `live: true`), `layout` (`single`|`grid`), `ts` (older
than 5 min = not live) and character names from `characters` (strings or `{name}` objects). U: the exact response shape of the
stream service is not verified yet (no service code in the repo when this was written); the reader accepts those keys and
otherwise says not live, so fix `tools/publish/live.py` `_names`/`LIVE_WORDS` once the real shape is known.
Names lose instance ids (`i3:Navi`, `Navi (i3)`, `instance 3 Navi` -> `Navi`) and must look like a D2 character name
(letters, `-`, `_`, 2 to 15 chars), else are dropped.
- Live: a LIVE pill, a `LIVE now on Kick` button and a line `Watching a grid of N characters: A, B.` / `Watching A.` in index.html;
  one bold link line in status.md.
- Configured but not live or service unreachable/stale/bad: `Streams when the bot is running.`
- Not configured: nothing.
Sanitizer: `kick.com` is NOT in `ALLOWED_URL_HOSTS`; only the exact shape `https://kick.com/<channel>` passes (no path, query,
port, `www.`, subdomain, trailing slash). `<domain>`, IPs and tokens stay blocked in every field.

## Run
    python3 tools/publish/publish.py --out DIR [--stats-url URL] [--frame FILE] [--days N]
    python3 -m unittest discover -s tools/publish        # tests
    UPDATE_GOLDEN=1 python3 -m unittest discover -s tools/publish   # after an intended layout change, review the diff

Default `--out` is `~/d2lod-status/public/`. Exit 0 = written (and `.sanitizer-ok` stamp); exit 2 = blocked, nothing written,
offending lines listed on stderr with the match redacted.

## Sanitizer (tools/publish/sanitize.py), fail closed
Every output is scanned before any file is written. A hit on any rule blocks the whole run:
long hex/base64-like tokens (>= 24 chars), IPv4/IPv6, `*.<domain>`/`<domain>`, any hostname except URLs on
`github.com`, `youtube.com`, `twitch.tv` (and the three output filenames), URLs with other schemes/ports/credentials or
secret-looking query strings (`?token=`), `/home/ /tmp/ /mnt/ /etc/ ~/` and Windows paths, `ssh://`, `token|password|key|secret`
followed by `:`/`=` and a value, CD-key shaped 5x(4-6) groups, email addresses, save-file names (`.d2s .d2x .d2i .ctl .key .map .bak`, `saves/`).
JPEGs must contain no APPn (except JFIF) or COM segments. Allow-listing is the constants `ALLOWED_URL_HOSTS` and
`ALLOWED_FILENAMES`; the test `test_allow_list_is_exact` pins them. Known cost: legitimate text that looks like a hostname
(`go.mod`) blocks a board title or is dropped from the feed; rephrase it.

## Publish step (future, gated)
`tools/publish/push_gist.sh DIR [GIST_ID]` (NOT run by its author). It exits 3 unless `D2_PUBLISH_APPROVED=yes`, re-runs
`publish.py --verify DIR` in the same invocation and exits 2 if that fails, then `gh gist create --public=false` (or
`gh gist edit ID -f ...`) for `status.md` and `stats.json`. The image cannot live in a gist: with `D2_IMAGE_REPO` set it copies
`latest.jpg` into that clone, commits and pushes (reminder: that repo is public once pushed; review the frame first).

## Timer (text only, not installed)
`~/.config/systemd/user/d2-publish.service`:

    [Service]
    Type=oneshot
    ExecStart=/usr/bin/python3 %h/code/d2lod-bot/tools/publish/publish.py --out %h/d2lod-status/public --stats-url http://HOST:PORT/STATS_PATH --stream-url http://worker.example:8433/api/stream/status --stream-token-file %h/.config/d2-publish/stream.token

`~/.config/systemd/user/d2-publish.timer`:

    [Timer]
    OnCalendar=*:0/5   # every 5 minutes while the live feature is on (use *:0/15 without it)
    Persistent=true
    [Install]
    WantedBy=timers.target

The timer only generates locally. Pushing stays a manual, approved step unless the user later adds a second unit that calls
`D2_PUBLISH_APPROVED=yes push_gist.sh`. `push_pages.sh` stays gated by `D2_PUBLISH_APPROVED=yes` and is never run by the generator or the timer above.

## Update procedure
1. Run the generator, read `~/d2lod-status/public/status.md` (and look at `latest.jpg`).
2. If blocked: fix the source text (board task, milestone title), never loosen a rule to get past a hit; a new allow-list entry needs a test.
3. `D2_PUBLISH_APPROVED=yes tools/publish/push_gist.sh ~/d2lod-status/public GIST_ID` (first time without GIST_ID, then keep the id).
