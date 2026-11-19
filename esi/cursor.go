package esi

// pagedRequest is the cursor-pagination surface (after/before/limit) shared by every generated
// request builder for ESI's newer cursor-paginated endpoints, so applyCursor can set all three
// in one place. Used by military_campaigns.go and paragon_hub.go; the older cursor-paginated
// files (freelance_jobs.go, corporation_projects.go) predate it and set the params inline.
type pagedRequest[R any] interface {
	After(string) R
	Before(string) R
	Limit(int64) R
}

// applyCursor sets after/before/limit on req, omitting each one left at its zero value
// (after/before are mutually exclusive; ESI rejects a request that sets both).
func applyCursor[R pagedRequest[R]](req R, after, before string, limit int64) R {
	if after != "" {
		req = req.After(after)
	}
	if before != "" {
		req = req.Before(before)
	}
	if limit > 0 {
		req = req.Limit(limit)
	}
	return req
}
