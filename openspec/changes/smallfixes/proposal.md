# smallfixes

## Why

After the public split the d2dfarm offline test failed (its capacity script moved to the ops repo), two livelock fixes for the fresh-game path existed in parallel, and Go processes started on the worker never saw the role variables from local.env. These small repairs close those gaps and tidy two doc leftovers.

## What Changes

- d2dfarm probe uses the real capacity script (via ops.sh); its test uses a fake one.
- freshGameToLevel resets sweepKills, needBuff and stepArea next to the cleared set, with a test.
- tools/env.sh gains d2_role_env; d2dview-demo and soak start forward D2_COLD_DIR and the host words.
- docs/nas-container.md heading, BACKLOG lines, executable bit of tools/hooks/test-pre-push.sh.
