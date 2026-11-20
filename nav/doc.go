// Package nav holds neat's calculations for moving ships and characters
// through New Eden, for any activity (PvP, PvE or industry): security bands
// and displayed security, where a cyno can be lit, light years, jump range,
// jump fuel and fatigue, and jump/gate route search with pilot preferences
// and dotlan links.
//
// It is pure: no network, disk or clock access, and it imports nothing else
// from neat. Callers (mcp/) gather inputs from sde/, esi/ and config/ and pass
// plain values in. Every number that the SDE doesn't carry is a named
// constant in constants.go with its source.
//
// See docs/superpowers/specs/2026-11-20-logistics-planning-design.md.
package nav
