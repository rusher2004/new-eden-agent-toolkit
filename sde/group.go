package sde

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Group is a minimal decode of a fuzzwork invGroups row -- EVE's item
// hierarchy level between Category and Type (e.g. "Frigate", within the
// "Ship" category). TypeIDs is not a stored column -- it's computed via a
// reverse query over invTypes at read time.
type Group struct {
	ID         int64
	CategoryID int64
	Name       string
	Published  bool
	TypeIDs    []int64
}

// Group looks up a group by ID. ok is false if the ID isn't in the cache.
func (s *Store) Group(ctx context.Context, userAgent string, id int64) (Group, bool, error) {
	if s == nil {
		return Group{}, false, nil
	}
	if v, ok := getOverlay[Group](s, "group", id); ok {
		return v, true, nil
	}
	s.syncBestEffort(ctx, userAgent)
	if s.db == nil {
		return Group{}, false, nil
	}

	var (
		categoryID int64
		name       sql.NullString
		published  sql.NullBool
	)
	row := s.db.QueryRowContext(ctx, `SELECT categoryID, groupName, published FROM invGroups WHERE groupID = ?`, id)
	if err := row.Scan(&categoryID, &name, &published); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Group{}, false, nil
		}
		return Group{}, false, fmt.Errorf("query group %d: %w", id, err)
	}

	typeIDs, err := s.queryInt64Column(ctx, `SELECT typeID FROM invTypes WHERE groupID = ? ORDER BY typeID`, id)
	if err != nil {
		return Group{}, false, fmt.Errorf("query group %d type ids: %w", id, err)
	}

	return Group{ID: id, CategoryID: categoryID, Name: name.String, Published: published.Bool, TypeIDs: typeIDs}, true, nil
}

// PutGroup stores g in the overlay, keyed by g.ID -- used by mcp/'s
// read-through fallback.
func (s *Store) PutGroup(g Group) error {
	if s == nil {
		return nil
	}
	putOverlay(s, "group", g.ID, g)
	return nil
}

// GroupIDs returns every group ID currently cached, ascending.
func (s *Store) GroupIDs(ctx context.Context, userAgent string) ([]int64, error) {
	if s == nil {
		return nil, nil
	}
	s.syncBestEffort(ctx, userAgent)
	if s.db == nil {
		return nil, nil
	}
	ids, err := s.queryInt64Column(ctx, `SELECT groupID FROM invGroups ORDER BY groupID`)
	if err != nil {
		return nil, fmt.Errorf("query group ids: %w", err)
	}
	return ids, nil
}
