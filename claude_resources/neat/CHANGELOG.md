# neat plugin — changelog

## 0.5.0
- `sde/` moves from a bbolt-backed cache seeded from CCP's own SDE jsonl export to a directly
  SQL-queried SQLite dump published by fuzzwork.co.uk (`~/.neat/sde/eve.db`, replacing
  `~/.neat/sde/sde.db`, which is left in place and orphaned by this change). No tool-facing
  behavior change other than the three field regressions below — same 24 Universe tools remain
  SDE-backed via the same read-through cache.
- Three tool outputs now always come back absent (previously populated for at least some rows):
  `get_universe_region`'s `description`, `get_universe_station`'s `services`, and
  `list_universe_factions`' `isUnique`. None have a source column in the fuzzwork dump; a
  cache-miss ESI fallback still populates them fresh, only cache hits are affected.

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
