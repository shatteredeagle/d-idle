Per-feature additions to the invariant policy (docs/process/generated-files.md). One file per feature,
`<feature>.json`, same keys as ../invariants.json (hard, soft, report_only, soft_in; no _doc). The loader appends them
to the base lists, so a new row never edits a shared line.
