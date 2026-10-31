# S: showcase, stream and public status (layer, gates nothing)

Added by MILESTONES2 (2026-10-28). Not part of alpha, beta or the good-bot gate: it does not make the bot play better. It exists so the stream and
public layer is counted honestly instead of being mistaken for bot progress. 56 % today, but almost every criterion is `partial` because it is
Python/shell tooling verified by mock tests; a criterion goes to done only with a doc naming a live check.

| piece | where | state |
|---|---|---|
| Kick/RTMP streamer with overlay and layouts | tools/stream/stream.sh, overlay.py | partial: dry-run tests; ingest fix 'verified live' in a commit message only |
| stream control service (API, scheduler, max-live guard, keep-alive) | tools/streamctl, docs/streamctl.md | partial: offline tests, installed in dry mode |
| scene page | tools/publish/scene.py, tools/stream/scene.sh | partial: screenshots in docs/publish |
| public status page generator | tools/publish, docs/publish.md | done (offline by nature) |
| public page published on a timer | board host (the NAS), Pages | partial: no recorded check |
| Home Assistant package and dashboard | tools/ha-stream | partial |
| StreamDesk tray app | tools/streamdesk | partial: mock tests |
| Kick title/category updater | tools/kick | partial: mock server only |

Work (U): about 3 agent-days, all user-facing: one recorded live stream (S.stream, S.scene), one flip of streamctl to live mode with the max-live
guard observed, a documented check of the published URL. Keep it behind alpha work; do not let it take a worker host instance from a soak.
The public generator reads milestones.json fields `gates[].id/title/milestones/includes`, `milestones[].id/title/phase/criteria[].text/status`; do not rename them.
