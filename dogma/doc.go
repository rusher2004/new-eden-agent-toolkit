// Package dogma evaluates EVE's dogma for a ship, its fitted modules and
// its pilot's skills: each attribute's base value from the SDE, changed by
// every effect modifier that targets it, applied in the game's operation
// order, with stacking penalties where the attribute isn't stackable. It
// answers questions such as a fitted cargo hold or a cyno generator's
// Liquid Ozone per cycle the way the game computes them, rather than with a
// formula per hull.
//
// It is pure: no network, disk or clock access, and it imports nothing else
// from neat. Callers (mcp/) read types, effects and attributes from sde/.
//
// See docs/superpowers/specs/2026-11-20-logistics-planning-design.md.
package dogma
