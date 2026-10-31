# d2d container: the bot inside (card d2d-container-bot, first cut)

> status: entry point done and smoke-tested only as far as `docs/d2d-container/README.md` records; the fleet wizard, dashboard viewer and soak sidecar (tasks 1.2 to 1.4) are NOT done. Tags: V verified (how stated), S secondary, U unverified.

## Decision: same image, bot is the default command
`deploy/d2d-container/entrypoint.sh bot` (the image's default `CMD`) runs `d2bot --backend d2d --managed --d2d-dial <tap ctl socket> --d2d-sock <d2d socket>` behind the `d2dview` tap. d2bot starts d2d itself (`--managed`: it also saves and quits it, V docs/d2d-bot.md), so there is one process owner for the d2d lifecycle and no separate sidecar for the bot. A sidecar image is not needed (U until the soak driver exists: it would reuse this image with `entrypoint.sh d2dsoak ...`).

## Slot semantics
`tools/d2dslot.sh` gives a host a pool of numbered slots (lock file, private dir, socket, ports 8475+n and 8891+n). In the container the container is the slot: one d2d per container, fixed paths (`/run/d2d/d2d.sock`, bot API `127.0.0.1:8420`, viewer `127.0.0.1:8891`), no lock. Parallel slots = parallel containers (each with its own `/data` volume and character), so ports never clash inside and only published ports matter on the host.

## Socket topology and why
d2d takes one control client (V docs/d2d-wire.md, `err busy`). The tap (`cmd/d2dview`, V docs/d2d-view.md) holds the only upstream connection and serves `d2d.sock.ctl` (everything, for d2bot and `docker exec` probes) and `d2d.sock.view` (read verbs only, for the healthcheck). The tap redials upstream lazily, which lets it start before d2d exists (U: confirmed for the managed launch only by the smoke test in `d2d` mode, not yet by a bot run).

## Modes and env
`D2D_MODE=lockstep|realtime|timescale:N` selects d2d flags. U: in lockstep d2bot must drive `step` itself (Pacer, V d2dinput); `entrypoint.sh` passes `--d2d-args '--lockstep'` and relies on `internal/d2dinput.Pacer`. Whether a realtime bot run is better for the live product is a policy question (`design.md` section 6). A character with a save of the same `D2D_NAME` is loaded (`--d2d-class -1`), otherwise created (`--d2d-class $D2D_CLASS`).

## Not done
- `d2fleet` wizard for d2d (1.2), viewer source in the dashboard (1.3, the tap's own page on 8891 exists), soak driver sidecar (1.4).
- Gate "one slot runs in the container on the worker host for 1 h": not run (no live bot run was made by this card; the smoke test drives a bare d2d only).
