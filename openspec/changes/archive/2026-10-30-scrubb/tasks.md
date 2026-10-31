# Tasks

```yaml
change: scrubb
status: archived
agent: scrubb
branch: agent/scrubb
worktree: ~/code/d2lod-wt/scrubb
owns:
  - openspec/changes/scrubb/**
  - docs/process/leakcheck-allow.txt
  - docs/process/public-manifest.json
  - cmd/leakcheck/**
  - .forgejo/workflows/ci.yml
  - cmd/d2dsoak/lab_other.go
  - cmd/d2dview/pub.go
  - cmd/d2dview/view_test.go
  - cmd/d2game/main.go
  - cmd/d2input/main.go
  - cmd/d2mapdump/main.go
  - cmd/d2sentinel/main.go
  - cmd/d2watch/main.go
  - cmd/d2watch/main_test.go
  - cmd/npcprobe/main.go
  - cmd/origcasts/main.go
  - deploy/container/compose.example.yml
  - deploy/container/d2start.sh
  - deploy/nas/my-d2idle.xml
  - docs/d2d-probes/c
  - docs/d2d-probes/d2p.py
  - docs/d2d-probes/death.py
  - docs/d2d-probes/fid.py
  - docs/d2d-probes/fid_vend3.py
  - docs/d2d-probes/fid_vendor.py
  - docs/process/d2dfarm.md
  - internal/api/api_test.go
  - internal/boss/act1_real_test.go
  - internal/cheat/paths.go
  - internal/combat/deathfacts.go
  - internal/d2/offsets.go
  - internal/d2dreplay/lock_linux.go
  - internal/d2dworld/integration_test.go
  - internal/fleet/admission.go
  - internal/input/client.go
  - internal/input/cmd/inputcal/peek.go
  - internal/launch/clean_test.go
  - internal/launch/lifecycle.go
  - internal/launch/mounts.go
  - internal/launch/oog_table.go
  - internal/launch/reap_test.go
  - internal/launch/sandbox.go
  - internal/launch/sandbox_test.go
  - internal/launch/subreaper.go
  - internal/leveler/class_test.go
  - internal/leveler/score_test.go
  - internal/mem/proc.go
  - internal/nav/cmd/navlive/main.go
  - internal/paths/legacy.go
  - internal/paths/paths.go
  - internal/paths/paths_test.go
  - internal/quest/act2_real_test.go
  - internal/quest/act3_real_test.go
  - internal/quest/act45_real_test.go
  - internal/questmem/questmem.go
  - internal/sentinel/server_test.go
  - internal/setup/check.go
  - internal/setup/checks.go
  - internal/telemetry/exp.go
  - internal/town/merc.go
  - internal/town/merc_test.go
  - openspec/specs/d2d-farm/spec.md
  - tools/container-build.sh
  - tools/d2ddet/build.sh
  - tools/d2ddet/pair.sh
  - tools/d2dfarm-test.sh
  - tools/d2dfarm.hosts.example
  - tools/d2dfarm.sh
  - tools/d2dloop/econ.sh
  - tools/d2dloop/kfbuild.sh
  - tools/d2dloop/pm.sh
  - tools/d2dloop/seeds.sh
  - tools/d2dslot-test.sh
  - tools/d2dslot.sh
  - tools/d2dsoak/run.sh
  - tools/d2dview-demo.sh
  - tools/env.sh
  - tools/hooks/pre-push
  - tools/lib-paths.sh
  - tools/ops-cutover.sh
  - tools/prmeasure/rerun.sh
  - tools/prmeasure/runall.sh
  - tools/prmeasure/runctl.sh
  - tools/prmeasure/runone.sh
  - tools/rigs/sandboxlive/main.go
  - tools/runab/ab.sh
  - tools/runab/one.sh
  - tools/soak/contsoak.sh
  - tools/soak/crashdrill.sh
  - tools/soak/fresh.sh
  - tools/soak/layout.sh
  - tools/soak/mounts.sh
  - tools/soak/soak.sh
  - tools/soak/soak24.sh
  - tools/soak/soak24_test.sh
  - tools/soak/stopclean.sh
  - tools/soakfull/run.sh
  - tools/test/agentrun_test.sh
  - tools/winereap.sh
exclusive: []
gates:
  - "go vet + go test -short on the worker; go test -p 1 ./cmd/leakcheck; changecheck 0 warnings; openspec validate --all --strict"
merge: d394dcf
result: "public scrub package B: host-config refactor: ~100 files read hosts/paths from role env vars via tools/env.sh/local.env (D2_WORKER_HOST/USER, D2_NAS_HOST, D2_BOARD_URL, D2_FORGE_URL, D2_COLD_DIR); leakcheck hits by class before->after: domain 53->41, email 13->6, homepath 67->9, ip 11->8, machine 1178->932, person 83->62, secret 53->5; env-var files 100->0; remaining hits are docs (package C) plus deploy/d2d-container; allow-list entries documented"
```

- [x] 1.1 Role variables (env.sh, local.env.example) and shell scripts read them
- [x] 1.2 Go code, tests and comments free of hostnames, IPs and home paths
- [x] 1.3 leakcheck game phrases, allow-list, manifest regenerated
- [x] 1.4 Verification (worker tests, changecheck, openspec)
