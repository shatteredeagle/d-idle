# Read-only Backlog.md web board for the NAS. Build context is this directory.
# Backlog.md's `browser` command binds 127.0.0.1 only (V: ss -ltn on 1.55.3), so
# nginx in the same container publishes it on 6420 and refuses every method that
# is not GET/HEAD (the board becomes read-only at the network edge).
FROM debian:bookworm-slim

ARG BACKLOG_VERSION=1.55.3
# sha256 of backlog-bun-linux-x64 for v1.55.3 (V: GitHub release digest and a local sha256sum agree)
ARG BACKLOG_SHA256=c20d5c51aa95b66d5c89428927bb0a16dc7beb9110683d66387e0514925f274a

RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates curl git nginx-light tini \
 && rm -rf /var/lib/apt/lists/* \
 && curl -fsSL -o /usr/local/bin/backlog \
      "https://github.com/MrLesk/Backlog.md/releases/download/v${BACKLOG_VERSION}/backlog-bun-linux-x64" \
 && echo "${BACKLOG_SHA256}  /usr/local/bin/backlog" | sha256sum -c - \
 && chmod 0755 /usr/local/bin/backlog \
 && rm -f /etc/nginx/sites-enabled/default

COPY nginx.conf /etc/nginx/board.conf
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod 0755 /usr/local/bin/entrypoint.sh

# 99:100 is NAS's nobody:users, so the appdata folder needs no chown games.
USER 99:100
ENV HOME=/tmp \
    REPO_URL=https://github.com/shatteredeagle/d-idle \
    BRANCH=main \
    SYNC_INTERVAL=300 \
    MIRROR=/data/repo
EXPOSE 6420
HEALTHCHECK --interval=30s --timeout=5s --start-period=90s --retries=3 \
  CMD curl -fsS -o /dev/null http://127.0.0.1:6420/api/tasks || exit 1
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/entrypoint.sh"]
