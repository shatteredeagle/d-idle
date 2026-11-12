d.idle friends pack
===================
You need: Linux x86-64, Docker with the compose plugin (v2), your own Diablo II 1.14d folder.

  ./install.sh --game-dir /path/to/your/Diablo-II     install and start (asks you to accept licenses/licence-note.txt)
  ./didle status                                      version, state and the exact page URL with your token
  ./didle stop | start                                stop (saves the hero, takes up to 45 s) or start again
  ./didle update                                      ask the update source for a signed release and apply it
  ./didle report                                      write a scrubbed report file to send back when something breaks

Full guide: guide/friend-guide.md. Source code and licences: src/ and licenses/. Check this download with
`sha256sum -c SHA256SUMS`.

Not affiliated with or endorsed by Blizzard Entertainment. Bring your own legally owned Diablo II 1.14d game files.
Offline only (no Battle.net). GPL-3.0-or-later (licenses/GPL-3.0.txt, source in src/). No warranty.
