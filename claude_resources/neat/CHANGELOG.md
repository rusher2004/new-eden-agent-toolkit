# neat plugin — changelog

## 0.4.0
- Adds SDE-resolved `*Name` fields (`RaceName`, `BloodlineName`, `FactionName`, `GroupName`)
  alongside the existing bare `*ID` fields across 8 non-Universe MCP files (alliance, character,
  corporation, corporation projects, faction warfare, incursions, killmails, sovereignty). Closes
  out the last of the ID kinds flagged in batch 1's own follow-up list. Names resolve from the
  local SDE cache only, same as batches 1-2's enrichment — no ESI calls are made on this path.
  Purely additive.

## 0.3.0
- Adds SDE-resolved location `*Name` fields (`SystemName`, `RegionName`, `ConstellationName`,
  `StationName`, `PlanetName`, `MoonName`, plus the index-aligned `*Names` list variants) alongside
  the existing bare `*ID` fields across 14 non-Universe MCP files (corporation, industry,
  killmails, market, planetary interaction, structures, sovereignty, fleets, and more). Names
  resolve from the local SDE cache only, same as batch 1's `type_id` enrichment — no ESI calls are
  made on this path. Purely additive.

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
