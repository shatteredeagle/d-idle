## Why

Several agents need verbs that only exist in the private d2d fork (dump vendor, debug layout, debug nearby). The lab binary, its branches and the way to rebuild it must be one documented, protected thing, or agents overwrite each other and patches live on an unpushed branch.

## What Changes

- one lab binary ~/d2d-lab/fork/bin/d2d.lab with provenance, never overwritten
- branch map main, d2dtown-vendor, p3-layout-verb, lab (the one to use)
- start.sh defaults to the lab binary; D2D_BIN overrides
- docs/d2d-lab.md (more owned paths U: the brief may add scripts under the lab, which live on the worker host, not in this repo)
