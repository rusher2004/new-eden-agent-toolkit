# neat plugin — changelog

## 0.2.0
- Adds SDE-resolved `*Name` fields (`TypeName`, `ShipTypeName`, etc.) alongside the existing bare
  `*ID` fields across 42 non-Universe MCP tools (assets, contracts, killmails, market, industry,
  planetary interaction, structures, and more). Names resolve from the local SDE cache only — no
  ESI calls are made on this path at all. Purely additive — no existing field was renamed, removed, or
  changed, and an unresolvable ID simply omits its `*Name` field rather than failing the call.

## 0.1.1
- No plugin-facing change; VERSION bump required by `versiongate` for CI/tooling fixes to
  `tools/versiongate` and `.github/workflows/version-gate.yml` (fail-closed on git errors,
  GITHUB_OUTPUT sanitization, wired the `go generate` no-diff guard into CI).

## 0.1.0
- Initial release: registers the `neat` MCP server (217 tools covering ESI + a local SDE cache) with Claude Code.
