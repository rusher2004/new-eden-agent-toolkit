// Package industry holds neat's industry, hauling and routing calculations:
// material quantities, job times and fees, blueprint tree expansion, stock
// netting, order-book pricing, build-versus-buy, reprocessing yield, the
// compressed-ore linear program, jump fuel and fatigue, cargo packing,
// jump/gate routing with dotlan links, and job scheduling.
//
// It is pure: no network, disk or clock access. Callers (mcp/) gather inputs
// from sde/, esi/ and config/ and pass plain values in. Every number that the
// SDE doesn't carry is a named constant in constants.go with its source.
//
// See docs/superpowers/specs/2026-11-18-industry-planning-design.md.
package industry
