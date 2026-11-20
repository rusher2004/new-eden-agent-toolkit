// Package industry holds neat's industry calculations: material quantities,
// job times and fees, blueprint tree expansion, stock netting,
// build-versus-buy, reprocessing yield, the compressed-ore linear program,
// and job scheduling. It uses nav/ for security bands, cargo/ for haul loads
// and market/ for order books; none of those depend on it.
//
// It is pure: no network, disk or clock access. Callers (mcp/) gather inputs
// from sde/, esi/ and config/ and pass plain values in. Every number that the
// SDE doesn't carry is a named constant in constants.go with its source.
//
// See docs/superpowers/specs/2026-11-18-industry-planning-design.md.
package industry
