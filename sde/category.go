package sde

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Category is a minimal decode of a fuzzwork invCategories row -- the top
// level of EVE's item hierarchy (e.g. "Ship", "Module"). GroupIDs is not a
// stored column -- it's computed via a reverse query over invGroups at read
// time.
type Category struct {
	ID        int64
	Name      string
	Published bool
	GroupIDs  []int64
}

// Category looks up a category by ID. ok is false if the ID isn't in the
// cache -- either because the store has never synced successfully, or
// because no such category exists.
func (s *Store) Category(ctx context.Context, userAgent string, id int64) (Category, bool, error) {
	if s == nil {
		return Category{}, false, nil
	}
	if v, ok := getOverlay[Category](s, "category", id); ok {
		return v, true, nil
	}
	s.syncBestEffort(ctx, userAgent)
	if s.db == nil {
		return Category{}, false, nil
	}

	var (
		name      sql.NullString
		published sql.NullBool
	)
	row := s.db.QueryRowContext(ctx, `SELECT categoryName, published FROM invCategories WHERE categoryID = ?`, id)
	if err := row.Scan(&name, &published); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Category{}, false, nil
		}
		return Category{}, false, fmt.Errorf("query category %d: %w", id, err)
	}

	groupIDs, err := s.queryInt64Column(ctx, `SELECT groupID FROM invGroups WHERE categoryID = ? ORDER BY groupID`, id)
	if err != nil {
		return Category{}, false, fmt.Errorf("query category %d group ids: %w", id, err)
	}

	return Category{ID: id, Name: name.String, Published: published.Bool, GroupIDs: groupIDs}, true, nil
}

// PutCategory stores c in the overlay, keyed by c.ID -- used by mcp/'s
// read-through fallback to cache a category fetched live from ESI.
func (s *Store) PutCategory(c Category) error {
	if s == nil {
		return nil
	}
	putOverlay(s, "category", c.ID, c)
	return nil
}

// CategoryIDs returns every category ID currently cached, ascending. Returns
// an empty slice, not an error, if the store has never synced successfully;
// callers read that as "cache empty" and fall back to a live ESI list call,
// not "no categories exist".
func (s *Store) CategoryIDs(ctx context.Context, userAgent string) ([]int64, error) {
	if s == nil {
		return nil, nil
	}
	s.syncBestEffort(ctx, userAgent)
	if s.db == nil {
		return nil, nil
	}
	return s.queryInt64Column(ctx, `SELECT categoryID FROM invCategories ORDER BY categoryID`)
}
