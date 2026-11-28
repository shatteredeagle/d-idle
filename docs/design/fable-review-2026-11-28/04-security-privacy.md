# Fable review 4 of 8: security and privacy of what changed on 2026-11-27

> status: REVIEW, docs only. No code changed, no sandbox probed, no exploit code. Read at origin/main `415b11f25`.
> Tags: V = read in the tree or run offline (said how), S = read in a secondary doc, U = not verified.
> Threat model: a home LAN that is "hostile-ish" (friends, tester machines, phones), a tester who shares a link or a
> report bundle, and a second process running as the same user on the worker host. Not in scope: a root attacker.

Severity scale: blocker (fix before the next tester release), high, medium, low. Each finding has attacker or
trigger, impact, steps in words, and a fix sketch.

## Summary table

| # | Sev | Finding | Where |
|---|---|---|---|
| 1 | blocker | The report scrubber does not hide passcodes, 16-character keys, several address and path shapes | `deploy/d2d-container/tester-report.sh` |
| 2 | high | `docker volume prune -a -f` is the wrong tool next to the hero volumes; two foreign volumes were lost | ops practice, `docs/probes/sandbox-deploy-plan-2026-11-27.md` |
| 3 | high | The watch token rides in the URL forever and is copied into every fetch | `internal/watch/serve.go:182`, `internal/watch/web/shell.js:928` |
| 4 | medium | `POST /api/difficulty/go`: no audit, no rate limit, no measured-run refusal, no cancel, a persisted go that is not bound to a hero | `internal/watch/difficultydesk.go`, `internal/bot/diffready.go`, `internal/diffready/consent.go` |
| 5 | medium | The flip tool checks `D2BOT_RUN` in the operator's shell, not in the container it restarts | `tools/flip/flip.py:370` |
| 6 | medium | The dev update server is an open directory listing on every interface for 8 hours | `tools/dev-update-host.sh:44` |
| 7 | medium | `ops log` stops at the first very long line, which is exactly what NUL damage produces; control characters reach the terminal | `internal/logx/sink.go` (Tail) |
| 8 | medium | Board: stored `javascript:` link on the hub page, token over plain HTTP, no limiter on the main write path | ops `tools/agentstatus/hub.html:23`, `agentstatus.py` |
| 9 | low | Login limiter keyed by peer address locks everyone out behind the container NAT | `internal/watch/gate.go:24` |
| 10 | low | Option injection through unvalidated names (`--commit`, `--to`, flip instance) | `tools/dev-update.sh`, `tools/flip/flip.py` |
| 11 | low | Error text with file paths is returned to the page verbatim | several `policyReply{Error: err.Error()}` |
| 12 | low | The mergeq env file is group-writable and can name the merge script | `~/.config/d2-mergeq/env` on the head box |

What I checked and found sound is listed at the end; it matters as much as the findings.

## 1. Blocker: the tester report scrubber misses the product's own secrets

Evidence (V, run offline: `REPORT_SCRUB_ONLY=1 sh deploy/d2d-container/tester-report.sh` on synthetic lines; no
container, no real data). These lines came out UNCHANGED:

- `login passcode=correct-horse-battery`, `{"passcode":"four random words here"}`, `passcode: hunter2hunter2`.
  The `words` list has `passw(or)?d|passphrase|pwd` but not `passcode`, which is the word this product uses
  (`cmd/d2dsetup/wizard.go:262` carries `json:"passcode"`).
- `cdkey ABCD1234EFGH5678 plain`: a 16-character key without dashes. The rules catch dashed groups and 26
  characters only. The header promises "CD-key-like text".
- `peer 100.101.102.103:51234` (the 100.64/10 range that mesh VPNs use, likely on a friend's machine), IPv6
  addresses (link-local ones embed the MAC), and a public IPv4.
- `host=DESKTOP-K3J2L1 user=alice` (bare host names and user names with no dotted suffix).
- `/mnt/user/games/alice/...`, `/media/alice/...`, `C:\Games\...`: only `/home`, `/Users` and `X:\Users` count as home.
- `open /Users/Bob Smith/Games/x.mpq` became `open <home> Smith/Games/x.mpq`: a home folder with a space leaks the
  rest of the name.
- Cosmetic: `http://192.168.1.50:8891` became `http://<ip>8891` (the rule eats the colon).

Attacker or trigger: none needed. The tester runs the script as told and mails the tarball; the README inside says
"Tokens, key-shaped text, home folders, private host names, IP addresses ... were replaced".
Impact: a friend's passcode (people reuse them), a game key, their network layout or their name leaves their house
in a file they were told is clean. This is a trust failure more than an exploit.
I did not find a place where the bot logs the passcode today (V: grep of `cmd/d2dsetup`, `internal/access`), so the
passcode line is latent, not live. The key and address lines depend on what d2d and Wine-free tooling print (U).

Other points in the same script (V, read):
- The ladder step copies EVERY `*.json` and `*.jsonl` whose path contains `ladder`, by pattern. A new file dropped
  into such a directory later is bundled without anyone deciding so. An allowlist of file names is safer.
- The scrubber is a deny list of shapes. A deny list cannot be complete; the second line of defence should be a
  scan that REFUSES to write the tarball when a known-secret shape survives (the in-app support bundle already has
  "never a bundle with a finding", `internal/watch/supportapi.go:17`).
- Two bundle paths now exist (this script, and `/api/support/bundle` in Go) with two different redactors. They
  will drift. The Go one should be the single implementation and the script should call it (`d2bot support ...`).
- The tarball lands in `/data/reports` with the default umask and is never cleaned up; old bundles accumulate in
  the data volume and travel with any volume backup.
- Sound: path traversal is not possible (`rel` is squeezed to `[A-Za-z0-9._-]`, V); `watch.token` is removed by
  value and the token alphabet (URL-safe base64, `cmd/d2bot/ladder.go:526`) needs no extra escaping (V);
  `access.json`, saves and keys are left out (V).

Fix sketch: add `passcode|pass[_-]?code|pin` to `words`; add 16-character key shape with a digit-and-letter mix
guard; add 100.64/10, IPv6 and "any dotted quad that is not 127.x" rules; treat `(/home|/Users|/mnt/user|/media)/`
up to the next path separator INCLUDING spaces; add a post-scan that fails the run; add each line above to
`tester_report_test.go`. Make the README claim only what the test proves.

## 2. High: the volume prune

Facts: the plan written earlier the same day said the volumes `docker-model-runner-models` (28 GB) and
`palimpsest-test` (2.2 GB) "belong to other projects, not ours: ask the owner" (V,
`docs/probes/sandbox-deploy-plan-2026-11-27.md` section 0.2). Later the user said "Reclaim docker volumes" and
`docker volume prune -a -f` was run; both volumes and some empty ones are gone, the two hero volumes stayed (S,
coordinator's decision log, not in this repo).

What was lost: a model cache (S: can be downloaded again, costs time and bandwidth) and a test volume of another
project (U: I cannot say whether it held anything that cannot be rebuilt; the owner of that project must confirm).

Why this is a security and safety finding and not only housekeeping:
- `prune -a -f` deletes every volume with no container, named or not, with no prompt. The hero volumes survived
  ONLY because their containers existed at that second. `tools/dev-sandbox-host.sh` and a compose `down` both have
  moments where a sandbox container does not exist. A prune in such a window deletes a hero with weeks of play.
- The go was for "docker volumes"; the tool widened it to other projects' data that the plan had marked as "ask".
  A go that names a class of things should be answered with the exact list before the destructive call.
- There is no backup of the hero volumes that I could find in the repo (U; `dev-sandbox-host.sh` archives state
  only on a reset).

Fix sketch: (a) a rule in the ops rules: never `docker volume prune`; remove volumes by name, after printing the
list and getting a go for that list; (b) label the hero volumes (`io.didle.keep=1`) and make the reclaim helper
refuse any volume with the label; (c) a nightly `tar` of each hero volume to cold storage, at most a few MB; (d)
tell the owner of the other project what was removed and when, today.

## 3. High: the watch token lives in the address bar

Evidence (V, read):
- `tokenGate` (`internal/watch/serve.go:182`) accepts `?t=`, sets the cookie and serves the page at the same URL.
  It does not redirect to the clean URL. `cmd/d2dsetup/server.go:158` does strip it; the watch gate does not.
- `shell.js:928` copies `t` from the page URL into the query string of its own fetches, so the token is in many
  request lines, not one.
- The watch pages send a CSP but no `Referrer-Policy` (only the login page and d2dsetup do, `gate.go:121`).
- The cookie is set from `?t=` BEFORE the token is compared, so a wrong value is stored as the cookie too.
- The token never expires and there is one token for viewing and for the answer listener.

Attacker or trigger: a friend screenshots or shares the URL ("look at my hero"); browser history sync; a shoulder
surfer; any proxy or log on the path (plain HTTP). In token mode the link is the whole credential.
Impact: full control of the answer routes: discard items, sell, hard reset, the difficulty go, the CPU profiler.
Steps: obtain the URL once; open it from any machine that reaches the published port. `tools/play-sandbox/up.sh`
(ops) publishes on every interface, and the tester compose allows `D2_PUBLISH=0.0.0.0`.

Fix sketch: on a valid `?t=`, set the cookie and answer 303 to the same path without `t` (as d2dsetup does); never
set the cookie for a wrong token; drop the `?t=` copy in `shell.js` (the cookie already rides on same-origin
fetches); add `Referrer-Policy: no-referrer` and `Cache-Control: no-store` to the gated pages; give the token a
rotation command and print "this link is a key" next to it. Longer term: a read-only token for sharing and a
separate owner credential for the answer listener.

Related, sound: state-changing routes are POST only, need `application/json` (not a simple cross-site request),
reject any foreign `Origin`, and the passcode cookie is `SameSite=Strict` (V, `policysheet.go:733`,
`access.go:128`). I found no CSRF path from another site. A DNS-rebinding page can read the viewer only in the
loopback no-token mode, which the tester image does not use (V, `serve.go:73`; the viewer has no Host check).

## 4. Medium: the difficulty go

Evidence (V, read `difficultydesk.go`, `diffready.go`, `consent.go`, `cmd/d2bot/funwire.go:302`, `ops.go:517`):
- The operator verb `go-difficulty` needs a reason, is rate limited and writes an audit event. The HTTP route
  calls `RequestDifficultyGo` directly: no reason, no audit event, no log line, no limiter.
- Neither path refuses in a measured run. The operator socket is not started in one, but the watch listener is.
  A go given during a gate or soak changes the difficulty mid-run and the record does not show who did it.
- One tap, no confirm, no cancel route. The page says "recorded"; the owner cannot take it back.
- `state/diffgo.json` is loaded without validation and is not in `internal/charsel/perchar.json`, so it is shared
  by every hero of the instance. A go given for hero A while A was ready stays on disk; if the owner switches to
  hero B in the same difficulty, B hops as soon as B is ready, with no go for B. Readiness is still re-checked at
  the hop (V, `DifficultyGate`), so this is a consent bug, not a safety bypass.
- The go has no age limit: given today, consumed whenever readiness next holds, maybe after many deaths.

Attacker or trigger: anyone with the watch link (finding 3); or simply the owner changing hero.
Impact: an unwanted, hard to undo move to a harder difficulty; an unexplained change inside a measured run.

Fix sketch: route both doors through one function that audits (remote address, "page" or "operator"), applies the
same budget, and refuses when `D2BOT_RUN` is set; store `{target, hero, at}` and drop the go when the hero name
differs or it is older than N hours (a profile field, not a literal); add `POST /api/difficulty/cancel` and a
second tap that states the risk; add `state/diffgo.json` to `perchar.json`. Test against two classes.

## 5. Medium: flip tool measured-run check looks at the wrong machine

Evidence (V, `tools/flip/flip.py:370`): `if self.env.get("D2BOT_RUN")` reads the operator's own environment and
`local.env`. The docstring says "never in a measured run". The run marker of a sandbox lives in the CONTAINER's
environment. A sandbox in a soak is flipped and restarted without complaint; the watcher can later restart it
again on a revert.
Impact: a measured run is silently invalidated (restart plus a knob change mid-run) and may still be counted.
Fix sketch: before the write, `docker exec <c> sh -c 'tr "\0" "\n" </proc/1/environ | grep -q "^D2BOT_RUN=."'`
(or ask `ops status`) and refuse; record the answer in the window file. Same check in the watcher before a revert,
with the breach reported instead of a restart.

Other flip points (V, read): knob values travel as base64 inside one `sh -c` argument and labrun quotes with
`shlex.join`, so `--why` and knob text cannot inject (sound). The instance name is not validated: it is a docker
argument and a state file name (`Store.path` joins it, so `../x` escapes the state directory; a leading `-` is a
docker option). Validate it against `^[a-z0-9][a-z0-9_.-]*$`. `--why` is free text that lands in
`data/darklaunch.json` and a commit in the public repo: leakcheck at merge is the only guard, so run it in the tool
before the commit.

## 6. Medium: the dev update server

Evidence (V, `tools/dev-update-host.sh:44`): `python3 -m http.server 18899 --bind 0.0.0.0 --directory www`, TTL
8 hours, no authentication, directory listing on.

What is sound (V, read `internal/selfupdate`, `cmd/didleboot`): every release needs an ed25519 signature by a key
the image trusts; the tarball hash and size are in the signed manifest; an older release is refused (the `stale`
drill); unpack refuses absolute names, `..`, links and devices; the signing key is in a 0700 directory outside the
served tree; only an image built with `DEV=1` honours the dev URL and key. So a LAN attacker who spoofs the server
or the name CANNOT install code: the answer to "who can publish a release the sandboxes will install" is "whoever
can read the dev key file", and that is every process of that user on the worker host, every agent included.

What is left:
- Anyone on the LAN can download every dev build (five kept) for 8 hours: unreleased binaries with full symbols
  of the bot, the setup tool and the entrypoint. For a public repo this is low; it becomes medium the day a build
  contains something not yet public (the brief says friends' machines will be on this network).
- A `--fault crash` drill publishes a release that is validly signed with the real dev key and dies on start. It
  is deleted at the next publish, but anyone who fetched it in between holds a signed "brick" whose sequence number
  is newer than everything before it. With a spoofed server it wins until the next real publish. A drill should
  sign with a throwaway key that only the drill container trusts.
- A spoofed server can also freeze a sandbox on its current version (serve nothing newer). Detectable, low.
- The dev key has no passphrase and no separation from agents. Any agent run can sign. Acceptable for a dev loop
  ONLY while the release key is elsewhere; write that sentence into `docs/dev-update.md` and add a test that a
  non-DEV image ignores `D2BOT_UPDATE_URL` and `D2BOT_UPDATE_PUBKEY` (U: I did not find one by name).

Fix sketch: bind to the docker bridge address or the one interface the sandboxes use; `--serve start` prints the
bind address; shorten the default TTL to the length of a deploy window; separate drill key.

Shell injection in the scripts (V, read): arguments are passed as arrays and labrun quotes them, so I found no
injection through branch names. Low leftovers are in finding 10.

## 7. Medium: `d2bot ops log`

Evidence (V, read `internal/logx/sink.go` Tail and `cmd/d2bot/opscmd.go:277`):
- The scanner's line limit is 4 MiB. A NUL-damaged log is typically a long run of NUL bytes with no newline (a
  sparse hole after a crash). Such a run longer than 4 MiB makes `Scan` stop with "token too long"; Tail returns
  the lines before it and an error, and never reaches the newer lines after the hole. The reader built for damaged
  logs fails on the usual damage. `sink_test.go` has no case for it (V, grep).
- Only NUL is removed. Escape sequences and other control bytes in a log line are written to the operator's
  terminal as they are. Item names, hero names and HTTP error texts end up in logs; a crafted name can rewrite
  what the operator sees. The login path already quotes with `%q` (sound), the tail printer does not.
- `filepath.Glob(path + ".*")` treats every sibling `name.*` as a rotation (`bot.log.bak`, `bot.log.tmp`), sorts
  them by a number parsed from the suffix (0 when there is none) and treats glob characters in `--file` as a
  pattern.
- `--file PATH` reads any file the caller can read. It is a local CLI with the caller's own rights, so this is not
  a privilege problem today. It becomes one the day a web route or the operator socket forwards a path to it.
  Keep it out of both.

Fix sketch: read with a `bufio.Reader`, skip NUL runs in a streaming way, cap a line by truncating it instead of
failing; replace bytes below 0x20 (except tab) and 0x7f with a visible escape when writing to a terminal; accept
only `^<name>\.[0-9]+$` as rotations; add the three tests.

## 8. Medium: the status board (ops repo, private)

Evidence (V, read ops `tools/agentstatus/`; S for the host layout from the ops board doc):
- `hub.html:23` sets `a.href = i.text` for a hub item of kind link with no scheme check and `target=_blank`.
  `board.js:89` does check for `http(s)`. A hub link whose text starts with `javascript:` runs script in the
  board's origin when clicked; the cards token sits in `localStorage` there (`cards.html:21`).
  Attacker: anything that can make an agent write a hub link (every agent holds the board token; a hostile web
  page read by a research agent is enough). Impact: card decisions (approve, reject) made in the user's name.
- The token travels in a header over plain HTTP on the LAN on every status update. Anyone who can sniff the
  segment once owns the board writes.
- `write()` (agents, hub, milestones) has no limiter for wrong tokens; only the cards path has one. With a
  32-character token this is academic.
- The 403 fix: the server enables writes only when `D2_BOARD_TOKEN` is in its environment (V). Where that value
  is stored on the board host (a container template or an environment file) and its mode I could not read (U).
  Check: the file is 0600 and owned by the service user, the template is not world-readable, and the token is not
  in the unit text itself (unit files are world-readable). On the head box the client file is 0600 in a 0700
  directory (V, `ls -la`).
- Sound: agent names are squeezed to `[A-Za-z0-9_-]` before they become a file name (V), tokens are compared in
  constant time (V), text is put on the page with DOM text nodes, not HTML (V, grep found no `innerHTML`).

Fix sketch: the same `^https?://` test in `hub.html`; reject other schemes on the server in `PUT /api/hub`; add
`Content-Security-Policy: default-src 'self'` to the pages; move the token to a per-agent value or at least
rotate it after any session that printed its environment.

## 9 to 12. Low

9. `ClientKey` uses the peer address (`gate.go:24`). Behind the container's port publishing every LAN client can
   appear as one gateway address (U for the exact network mode), so five wrong passcodes from one device lock
   every device for five minutes. Not a bypass; a nuisance a child can trigger. Fix: also key on a random
   pre-login cookie, and never lock an address that holds a valid session.
   Same area: the passcode session value is an HMAC of the stored hash (`access.go:104`), so every device gets the
   same cookie, logout clears only the browser's copy, and a sniffed cookie works for 30 days or until the
   passcode changes. Documented as accepted for plain HTTP; say in the guide that "log out" does not revoke.
10. `tools/dev-update.sh --commit REF` passes REF to `git archive` and `git rev-parse` with no `--` or ref check;
    a value starting with `-` is read as an option. `--to C` and the flip instance go to `docker exec` the same
    way. `set -- $out` in `dev-update-host.sh` word-splits and glob-expands text printed by the container. All
    need a hostile operator argument or a hostile container, so low. Fix: `git rev-parse --verify --end-of-options`
    and a name pattern check; `set -f` around the split.
11. Many answer routes return `err.Error()` to the page (for example `vaultapi.go` "internal"). An `os.PathError`
    carries the full path of the state file; outside the container that is the user's home path, and the page is
    what people screenshot. Fix: log the error, answer a fixed sentence and a code.
12. `~/.config/d2-mergeq/env` is 0664 in a 0775 directory (V, `ls -la`). It holds no secret (V: key names only,
    values not read into this report), but `MERGEQ_MERGE` and `MERGEQ_GO` are read from the environment and decide
    which script the merge worker runs before it pushes to main. On a single-user box this is the same trust as
    the user; make it 0600 in a 0700 directory so it stays that way when the worker moves host.

## Privacy of runtime outputs

- Explain view, quests, town plan (`/api/townplan`, read-only, 503 until bound), return screen: the structs carry
  game facts, knob values and reason lines; I found no host name, path or account field in them (V, read
  `internal/bot/explain.go:47`, `townplanapi.go`, `difficultydesk.go`). Reason lines are built from `err.Error()`
  in places (finding 11), which is the one way a path can reach them.
- The hero name is in every output and in the report bundle on purpose. A tester who names the hero after
  themselves has put their name in the bundle; say so in the README of the bundle (one line).
- The play launcher prints the token link once to standard output and says it is not written to the log (V,
  `cmd/d2bot/play.go:92`). Container standard output IS kept by the container runtime's own log, so "not in the
  log" is true for `d2bot.log` only. The report script takes `d2bot.log` and `d2d.out`, not the runtime log (V).
- leakcheck covers repository files. Nothing checks runtime text. The report script's scrub test is the only
  guard and finding 1 shows its holes; a small "runtime leak" test that feeds real handler output of a fixture
  hero through the same patterns would close the gap.

## Checked and sound (so nobody re-reviews it)

- Operator socket: unix socket, 0600, one JSON request per connection, unknown fields refused, 15 s deadline,
  reason required and control characters refused in `--why`, per-quest and total hourly budgets, audit with remote
  "operator" (V, `internal/ops/sock.go`, `ops.go:223`). `ops quest auto` rides the same path.
- Vault carry, stow, equip on merc: behind the shared POST gate, refused in a measured run in the keeper
  (`internal/vaultkeep/carry.go:108`, `equipdirect.go:33`), ledgered by the fork verbs (S for the fork side).
- Onboarding POST: shared gate, small body, known fields only (V).
- Self-update: signature, hash, anti-rollback, safe unpack, DEV-only dev source (V, finding 6).
- `wizard.env` is sourced by the entrypoint; the hero name is letters only and goals must be a known preset
  before they are written inside single quotes (V, `cmd/d2dsetup/wizard.go:21,296,303,361`). Keep that validation
  next to the `Sprintf`; today they are 60 lines apart and a new field would be easy to add unquoted.
- Passcode storage: PBKDF2-SHA256 with a per-record salt, constant-time compares, same-origin check on the login
  POST, limiter (V, `internal/access`).

## Recommended order

1. Scrubber fixes and tests (finding 1) before any friend is asked for a bundle.
2. Tell the other project's owner about the pruned volume; write the "never prune, remove by name" rule; back up
   the hero volumes (finding 2).
3. Strip `?t=` with a redirect, stop copying it in `shell.js`, add the two headers (finding 3). Small diff.
4. One card for the difficulty go (audit, measured-run refusal, per-hero, cancel) (finding 4).
5. Flip measured-run check in the container (finding 5), then the rest as hygiene cards.

## Questions for the coordinator

- Was the content of the pruned test volume of the other project rebuildable? Only its owner can answer.
- Is the passcode ever meant to appear in `wizard.json` during setup (before it is hashed)? If yes, finding 1
  moves from latent to live.
- Should a read-only share link exist (finding 3)? It changes the product, so it is a user decision.
