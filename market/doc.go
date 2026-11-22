// Package market holds neat's buying calculations, for any purchase rather
// than only build materials: walking a sell order book for a quantity at a
// station, and formatting in-game Multibuy lists.
//
// It is pure: no network, disk or clock access, and it imports nothing else
// from neat. Callers (mcp/) gather inputs from sde/, esi/ and config/ and pass
// plain values in.
//
// See docs/superpowers/specs/2026-11-20-logistics-planning-design.md.
package market
