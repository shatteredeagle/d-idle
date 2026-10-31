# memfd: passing the game-memory descriptor between processes (WP-MEMFD)

Verdict: **works**. VERIFIED 2026-10-28 on the head box (Proxmox LXC 102), kernel 7.0.14-8-pve, Yama `ptrace_scope=1`, euid 1000, CapEff=0, no setcap, no Wine.

## Evidence (`go test -race -count=5 ./internal/memfd`, all pass)
`TestFDPassingSiblingHelper`: the test process becomes subreaper (`launch.BecomeSubreaper`), starts a forker that starts a sleeper and exits, so the sleeper is reparented to the test (ppid == test pid, checked). The sleeper holds a marker at a fixed 32-bit address (0x30000000, mmap). The subreaper opens `/proc/<sleeper>/mem` once, a `Server` sends the fd with SCM_RIGHTS to a sibling helper process (a child of the test, not an ancestor of the sleeper). The helper reads the marker through the received fd: OK. The helper's own direct `open("/proc/<sleeper>/mem")` fails with `permission denied` (EACCES). An unrelated shell (`head -c4 /proc/<pid>/mem`) is denied too.

Meaning: Yama is checked when the file is opened, not on each read; the open file carries the permission, so a sentinel/watcher process that is not an ancestor can read game memory without CAP_SYS_PTRACE if the subreaper hands it the fd.

## Caveats
- Test skips the EACCES part with a message when scope is 0 (a direct open would succeed), or when the helper is root. The fd-passing half runs regardless.
- Not tested with the real Game.exe or in a container (WP-CONTAINER). A passed fd stays valid for the old game process only; after a game restart reads fail and the receiver must ask again (`Remote` does).
- The passed fd keeps access alive after the sender dies, as long as the target process lives (standard fd semantics, UNVERIFIED here).

## API
- `Source` (`FromFile`, `OpenPID`, `File`, `ReadAt(p, addr uint32)`, `Close`) implements `mem.Source`.
- `Server{Path, Prov}.Serve(ctx)`: unixpacket socket mode 0600; `Prov(name)` returns the current `*os.File` plus an info string such as `pid=N`. Protocol: `GET <name>` -> `OK <info>` + SCM_RIGHTS, or `ERR <msg>`.
- `Client{Path}.Get(name)` redials once on a broken connection. `Remote{C, Name}` is a `mem.Source` that refetches the fd after a failed read (game restart, server restart) and retries once.
- Consequence for D1: the in-process sentinel plan stays valid, and an out-of-process reader/watcher is possible via this package. The Server must live in the subreaper (the app).
