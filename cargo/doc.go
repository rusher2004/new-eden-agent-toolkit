// Package cargo holds neat's calculations for what fits where: packing a
// cargo list into loads that fit a hold.
//
// It is pure: no network, disk or clock access, and it imports nothing else
// from neat. Callers (mcp/) gather inputs from sde/, esi/ and config/ and pass
// plain values in.
//
// See docs/superpowers/specs/2026-11-20-logistics-planning-design.md.
package cargo
