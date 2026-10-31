# d2lod-bot runtime image (spike). Nothing Blizzard owns is in here. Binaries are cross-compiled on the
# build host into ./bin (see tools/container-spike.sh), so the image needs no Go toolchain.
# Plain Wine from Debian (not Proton): Proton needs the Steam runtime, see README.md.
FROM debian:trixie-slim

ENV DEBIAN_FRONTEND=noninteractive
RUN dpkg --add-architecture i386 \
 && apt-get update \
 && apt-get install -y --no-install-recommends \
      wine wine32:i386 libwine:i386 xvfb xauth x11-apps x11-utils tini ca-certificates rsync \
 && apt-get clean && rm -rf /var/lib/apt/lists/* /usr/share/doc /usr/share/man /usr/share/locale/*/LC_MESSAGES/*.mo

# The Wine user is the Linux user; "steamuser" matches the Proton layout the Go code expects (saves path, desktop dir).
RUN useradd -m -u 1000 steamuser
COPY bin/d2container /usr/local/bin/d2container
COPY bin/winprobe.exe /opt/probe/winprobe.exe
# The product: d2bot (linux/amd64), the input helper (windows/386) and a Proton stand-in so plain Wine is used.
COPY bin/d2bot /usr/local/bin/d2bot
# d2fleet serves the setup wizard and runs d2bot per slot (finds d2bot next to itself); its web assets are embedded.
COPY bin/d2fleet /usr/local/bin/d2fleet
COPY bin/d2doctor /usr/local/bin/d2doctor
COPY bin/d2soak /usr/local/bin/d2soak
COPY bin/d2input.exe /opt/d2/d2input.exe
COPY runner /opt/d2/runner
COPY d2init.sh /usr/local/bin/d2init.sh
COPY d2start.sh /usr/local/bin/d2start.sh

# One data dir (D2BOT_DATA) holds the prefix, saves and state; mount a volume there.
ENV D2BOT_DATA=/data \
    WINEPREFIX=/data/proton/pfx \
    WINEDEBUG=-all \
    WINEDLLOVERRIDES=mscoree,mshtml= \
    DISPLAY=:99 \
    XDG_RUNTIME_DIR=/tmp/xdg \
    PROTON=/opt/d2/runner/proton \
    D2INPUT_EXE=/opt/d2/d2input.exe
# D2BOT_GAMEDIR is deliberately NOT set: each slot's prefix has its own game dir (the legacy d2init.sh flow passes it with -e).
RUN mkdir /data /game /tmp/.X11-unix /tmp/xdg && chown steamuser /data && chmod 1777 /tmp/.X11-unix && chmod 700 /tmp/xdg && chown steamuser /tmp/xdg
USER steamuser
VOLUME /data
# The only port: the wizard / fleet admin UI (docs/container.md). Publish it on loopback only.
EXPOSE 8431

# Go entrypoint is subreaper and handles SIGTERM, so it runs fine as PID 1; `docker run --init` also works.
# A fresh container serves the setup wizard (d2start.sh makes /data/token, then d2container run -- d2fleet serve -setup).
# Override the command for the proven direct run, e.g. `run -- d2bot --managed ...` or `hold`: d2container is still the entrypoint.
ENTRYPOINT ["/usr/local/bin/d2container"]
CMD ["run", "--", "/usr/local/bin/d2start.sh"]
