# Self-update release format and signing (Tier 2, part 1)

> status: CURRENT for the format and tools; the workflow `.github/workflows/app-release.yml` is a DRAFT and inert (never run). Design and decisions: docs/research/self-update-design.md. Task: TASK-121 (updrelease).
> Tags: V = verified by the unit tests or a local run named here; S = secondary; U = unverified (needs GitHub).

## Asset layout (release tag `app-vX.Y.Z`)

| Asset | Content |
|---|---|
| `MANIFEST.json` | the signed document, canonical bytes written by `d2sign manifest` |
| `MANIFEST.json.sig` | base64 Ed25519 signature over the exact bytes of MANIFEST.json |
| `didle-bin-vX.Y.Z-linux-amd64.tar.gz` | `d2bot d2dsoak d2dview d2dsetup d2dping`, dest `bin` |
| `didle-d2d-vX.Y.Z-linux-amd64.tar.gz` | the d2d binary built from the named public fork tag, dest `d2d` |
| `didle-data-vX.Y.Z.tar.gz` | overlay fragments and data, dest `data` |
| `SOURCE.txt` | source refs and the written offer |
| `REVOKED`, `REVOKED.sig` | optional, published with the latest release; see Revocation |

Tarballs are reproducible (sorted names, mtime 0, root owner, `gzip -n`); V: two runs of `tools/release/app-assets.sh` gave byte-identical MANIFEST.json.

## MANIFEST.json (contract; Go types in `internal/selfupdate/manifest.go`)

```
{ "schema": 1, "version": "0.4.0", "seq": 17, "issued": "<RFC 3339>", "channel": "stable", "notes": "...",
  "min_image": 3,
  "compat": { "bot_api": 4, "d2d_ctl": 2, "data_schema": 5 },
  "source": { "bot": "<tag/commit>", "d2d": "<tag/commit>", "offer": "SOURCE.txt" },
  "files": [ { "name": "...", "sha256": "<64 lower-case hex>", "size": 123, "dest": "bin" } ],
  "revoke": { "versions": ["0.3.1"], "min_version": "0.3.0" } }
```

Rules (V, `manifest_test.go`): unknown fields and trailing data are errors; `version` is strict `x.y.z`; `seq` >= 1 and monotonic across releases; file names are plain names (no path, no `..`), `dest` is a clean relative path, `sha256` is lower-case hex; files are listed sorted by name so the bytes are reproducible. `min_image` is the base-image ABI level (label `io.didle.image.abi`); `compat` pairs are compared by the updater and the boot shim (design 3.4).

## Revocation

Two ways, both signed with the release key (design 3.3): the `revoke` block of a manifest (versions banned, `min_version` = monotonic floor), and the standalone `REVOKED` document (`schema`, `seq`, `versions`, `min_version`) published next to the latest manifest so an already-shipped bad version can be banned without a new app release. The consumer treats a version as revoked when it is listed or below the floor (`Revoke.Bans`).

## d2sign (stdlib Ed25519)

```
go run ./cmd/d2sign keygen   -dir DIR [-name N]       # N.key (0600, never overwritten, never printed) + N.pub
go run ./cmd/d2sign manifest -dir ASSETS -version .. -seq .. -issued .. -file name=dest ... -out MANIFEST.json
go run ./cmd/d2sign revoked  -seq N -revoke V -min-version V -out REVOKED
go run ./cmd/d2sign sign     (-key FILE | -key-env NAME) -in MANIFEST.json     # writes MANIFEST.json.sig
go run ./cmd/d2sign verify   -pub A.pub[,B.pub] -in MANIFEST.json -dir ASSETS   # signature and asset hashes
go run ./cmd/d2sign summary  -in MANIFEST.json                                  # the approver's review text
```

V (tests): refuses a key file readable by group/others; refuses to sign anything that is not a valid manifest or REVOKED document; signing twice gives identical bytes (Ed25519 is deterministic); a tampered manifest or asset fails verify; a second trusted key (rotation) verifies. The key file is a header line plus the base64 32-byte seed; the key never appears in output or error text.

## Building an unsigned set

`tools/release/app-assets.sh --version 0.4.0 --seq 17 --issued <time> --out DIR --d2d-bin PATH --src-bot REF --src-d2d REF [--overlay DIR] [--notes TEXT]` (Linux, GNU tar, Go). The issue time is an argument, not "now", so the bytes are reproducible.

## The workflow (DRAFT, inert)

Jobs: `build` (no secrets), `sign` (environment `app-release`, secret `D2SIGN_KEY`, prints the review summary into the job summary, verifies against the pinned public key), `publish` (skipped on dry run). It is inert: only `workflow_dispatch`, every job needs the repository variable `APP_RELEASE_ENABLED=true`, and the d2d build step deliberately fails until it is wired to the Dockerfile.d2d builder stage (TASK-120). U: nothing has run on GitHub.

### Turning it on (user steps, TASK-124; not done)

1. Generate the pair with `d2sign keygen` on your machine; commit only the `.pub` as `deploy/d2d-container/pubkeys/current.pub` (the image bakes it).
2. Settings > Environments > create `app-release`, add yourself as required reviewer, add the secret `D2SIGN_KEY` (the body of the `.key` file) to the environment only.
3. Rulesets protecting tags `app-v*`, `tester-v*`, `release-*`; 2FA on the account.
4. Only then set the repository variable `APP_RELEASE_ENABLED=true`. GitHub creates a missing environment without reviewers on first use, so the order matters.
