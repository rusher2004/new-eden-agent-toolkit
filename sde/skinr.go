package sde

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// SkinrComponent is a minimal decode of a fuzzwork skinrComponents row -- one SKINR building
// block (a nanocoating or pattern), referenced by ID from ESI's Cosmetics tag (a SKINR's per-slot
// layout, a character's component licenses). Unlike the Universe tag's types, these are not an
// ESI resource at all -- there is no ESI endpoint to fall back to on a miss, so there is no
// PutSkinrComponent/overlay write-back either.
type SkinrComponent struct {
	ID   int64
	Name string
}

// SkinrComponent looks up a SKINR component by ID. ok is false if the ID isn't in the cache.
func (s *Store) SkinrComponent(ctx context.Context, userAgent string, id int64) (SkinrComponent, bool, error) {
	if s == nil {
		return SkinrComponent{}, false, nil
	}
	s.syncBestEffort(ctx, userAgent)
	if s.db == nil {
		return SkinrComponent{}, false, nil
	}
	var name sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT name FROM skinrComponents WHERE componentID = ?`, id).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return SkinrComponent{}, false, nil
	}
	if err != nil {
		return SkinrComponent{}, false, fmt.Errorf("query skinr component %d: %w", id, err)
	}
	return SkinrComponent{ID: id, Name: name.String}, true, nil
}

// SkinrSlot is a minimal decode of a fuzzwork skinrSlots row -- one of a SKINR layout's 8 fixed
// cosmetic slots (e.g. 1 = "Primary Slot", 5 = "Primary Pattern Slot"), keyed by the same slot ID
// ESI's Cosmetics tag returns. Read-only, no overlay, same as SkinrComponent.
type SkinrSlot struct {
	ID   int64
	Name string
}

// SkinrSlot looks up a SKINR cosmetic slot by ID. ok is false if the ID isn't in the cache.
func (s *Store) SkinrSlot(ctx context.Context, userAgent string, id int64) (SkinrSlot, bool, error) {
	if s == nil {
		return SkinrSlot{}, false, nil
	}
	s.syncBestEffort(ctx, userAgent)
	if s.db == nil {
		return SkinrSlot{}, false, nil
	}
	var name sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT name FROM skinrSlots WHERE slotID = ?`, id).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return SkinrSlot{}, false, nil
	}
	if err != nil {
		return SkinrSlot{}, false, fmt.Errorf("query skinr slot %d: %w", id, err)
	}
	return SkinrSlot{ID: id, Name: name.String}, true, nil
}
