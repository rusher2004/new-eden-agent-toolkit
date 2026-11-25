package mcp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	goesi "github.com/fnt-eve/goesi-openapi"
	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
)

// fakeESIJSONServer serves body (with an optional X-Pages header, "" to
// omit) to any request, for tests exercising the cache-miss/ESI-fallback
// path -- the underlying esi/ package's own tests already verify each
// endpoint's exact URL, so these only need a plausible response body.
func fakeESIJSONServer(t *testing.T, body, totalPages string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if totalPages != "" {
			w.Header().Set("X-Pages", totalPages)
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// useFakeESIClient points newPublicESIClient at srv for the duration of
// the test, restoring the real constructor afterward.
func useFakeESIClient(t *testing.T, srv *httptest.Server) {
	t.Helper()
	orig := newPublicESIClient
	t.Cleanup(func() { newPublicESIClient = orig })
	newPublicESIClient = func(userAgent string) *esi.Client {
		return esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: userAgent})
	}
}

// --- list_universe_ancestries / list_universe_bloodlines / list_universe_races /
// list_universe_factions ---
//
// These 4 are whole-collection read-through, no Put* (list-only, no
// per-ID tool exists for any of them).

type fakeAncestryCache struct {
	ancestries []sde.Ancestry
	err        error
}

func (f *fakeAncestryCache) Ancestries(ctx context.Context, userAgent string) ([]sde.Ancestry, error) {
	return f.ancestries, f.err
}

type fakeBloodlineCache struct {
	bloodlines []sde.Bloodline
	err        error
}

func (f *fakeBloodlineCache) Bloodlines(ctx context.Context, userAgent string) ([]sde.Bloodline, error) {
	return f.bloodlines, f.err
}

type fakeRaceCache struct {
	races []sde.Race
	err   error
}

func (f *fakeRaceCache) Races(ctx context.Context, userAgent string) ([]sde.Race, error) {
	return f.races, f.err
}

type fakeFactionCache struct {
	factions []sde.Faction
	err      error
}

func (f *fakeFactionCache) Factions(ctx context.Context, userAgent string) ([]sde.Faction, error) {
	return f.factions, f.err
}

func TestListUniverseAncestries_CacheHit_MapsFields(t *testing.T) {
	iconID := int64(1641)
	shortDesc := "Progressive members of the upper class."
	cache := &fakeAncestryCache{ancestries: []sde.Ancestry{
		{ID: 1, BloodlineID: 5, Description: "Holders are...", Name: "Liberal Holders", IconID: &iconID, ShortDescription: &shortDesc},
	}}

	got, err := listUniverseAncestries(context.Background(), cache, "test-agent")
	if err != nil {
		t.Fatalf("listUniverseAncestries returned error: %v", err)
	}
	if len(got) != 1 || got[0].ID != 1 || got[0].Name != "Liberal Holders" || got[0].Description != "Holders are..." {
		t.Errorf("got = %+v, want one entry ID=1 Name=\"Liberal Holders\" Description=\"Holders are...\"", got)
	}
	if got[0].ShortDescription == nil || *got[0].ShortDescription != shortDesc {
		t.Errorf("ShortDescription = %v, want %q", got[0].ShortDescription, shortDesc)
	}
}

func TestListUniverseAncestries_CacheError_ReturnsError(t *testing.T) {
	cache := &fakeAncestryCache{err: errors.New("boom")}

	_, err := listUniverseAncestries(context.Background(), cache, "test-agent")
	if err == nil {
		t.Fatal("listUniverseAncestries returned nil error, want the wrapped cache error")
	}
}

func TestListUniverseAncestriesHandler_MapsFields(t *testing.T) {
	orig := listUniverseAncestries
	t.Cleanup(func() { listUniverseAncestries = orig })
	iconID := int64(100)
	shortDesc := "short"
	listUniverseAncestries = func(ctx context.Context, sdeStore ancestryCache, userAgent string) ([]UniverseAncestry, error) {
		return []UniverseAncestry{{ID: 11, BloodlineID: 1, Description: "desc", Name: "Tribal", IconID: &iconID, ShortDescription: &shortDesc}}, nil
	}

	handler := listUniverseAncestriesHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, ListUniverseAncestriesInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Ancestries) != 1 || out.Ancestries[0].ID != 11 || out.Ancestries[0].Name != "Tribal" {
		t.Errorf("out = %+v, want one entry ID=11 Name=Tribal", out)
	}
}

func TestListUniverseAncestriesHandler_PropagatesError(t *testing.T) {
	orig := listUniverseAncestries
	t.Cleanup(func() { listUniverseAncestries = orig })
	wantErr := errors.New("upstream failure")
	listUniverseAncestries = func(ctx context.Context, sdeStore ancestryCache, userAgent string) ([]UniverseAncestry, error) {
		return nil, wantErr
	}

	handler := listUniverseAncestriesHandler("v1", nil)
	_, _, err := handler(context.Background(), nil, ListUniverseAncestriesInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListUniverseBloodlines_CacheHit_MapsFieldsAndDefaultsMissingShipTypeID(t *testing.T) {
	cache := &fakeBloodlineCache{bloodlines: []sde.Bloodline{
		{ID: 1, Charisma: 6, CorporationID: 1000006, Description: "The Deteis are...", Intelligence: 7, Memory: 7, Name: "Deteis", Perception: 5, RaceID: 1, Willpower: 5},
	}}

	got, err := listUniverseBloodlines(context.Background(), cache, "test-agent")
	if err != nil {
		t.Fatalf("listUniverseBloodlines returned error: %v", err)
	}
	if len(got) != 1 || got[0].ID != 1 || got[0].Name != "Deteis" || got[0].ShipTypeID != 0 {
		t.Errorf("got = %+v, want one entry ID=1 Name=Deteis ShipTypeID=0 (no SDE source for this record)", got)
	}
}

func TestListUniverseBloodlinesHandler_MapsFields(t *testing.T) {
	orig := listUniverseBloodlines
	t.Cleanup(func() { listUniverseBloodlines = orig })
	listUniverseBloodlines = func(ctx context.Context, sdeStore bloodlineCache, userAgent string) ([]UniverseBloodline, error) {
		return []UniverseBloodline{{ID: 1, Name: "Deteis", RaceID: 1, CorporationID: 1000166}}, nil
	}

	handler := listUniverseBloodlinesHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, ListUniverseBloodlinesInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Bloodlines) != 1 || out.Bloodlines[0].ID != 1 || out.Bloodlines[0].Name != "Deteis" {
		t.Errorf("out = %+v, want one entry ID=1 Name=Deteis", out)
	}
}

func TestListUniverseBloodlinesHandler_PropagatesError(t *testing.T) {
	orig := listUniverseBloodlines
	t.Cleanup(func() { listUniverseBloodlines = orig })
	wantErr := errors.New("upstream failure")
	listUniverseBloodlines = func(ctx context.Context, sdeStore bloodlineCache, userAgent string) ([]UniverseBloodline, error) {
		return nil, wantErr
	}

	handler := listUniverseBloodlinesHandler("v1", nil)
	_, _, err := handler(context.Background(), nil, ListUniverseBloodlinesInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListUniverseRaces_CacheHit_ReturnsAllianceID(t *testing.T) {
	allianceID := int64(500001)
	cache := &fakeRaceCache{races: []sde.Race{
		{ID: 1, AllianceID: &allianceID, Description: "Founded on the tenets...", Name: "Caldari"},
	}}

	got, err := listUniverseRaces(context.Background(), cache, "test-agent")
	if err != nil {
		t.Fatalf("listUniverseRaces returned error: %v", err)
	}
	if len(got) != 1 || got[0].RaceID != 1 || got[0].Name != "Caldari" || got[0].AllianceID == nil || *got[0].AllianceID != 500001 {
		t.Errorf("got = %+v, want one entry RaceID=1 Name=Caldari AllianceID=500001 (from a cache hit, not only a cache miss)", got)
	}
}

func TestListUniverseRaces_CacheHit_NoAllianceID_ReturnsNil(t *testing.T) {
	cache := &fakeRaceCache{races: []sde.Race{
		{ID: 1, Description: "Founded on the tenets...", Name: "Caldari"},
	}}

	got, err := listUniverseRaces(context.Background(), cache, "test-agent")
	if err != nil {
		t.Fatalf("listUniverseRaces returned error: %v", err)
	}
	if len(got) != 1 || got[0].AllianceID != nil {
		t.Errorf("got = %+v, want AllianceID=nil when sde.Race.AllianceID is nil (distinguishable from a confirmed-no-alliance race, which ESI's schema never actually returns)", got)
	}
}

func TestListUniverseRacesHandler_MapsFields(t *testing.T) {
	orig := listUniverseRaces
	t.Cleanup(func() { listUniverseRaces = orig })
	allianceID := int64(500001)
	listUniverseRaces = func(ctx context.Context, sdeStore raceCache, userAgent string) ([]UniverseRace, error) {
		return []UniverseRace{{RaceID: 1, AllianceID: &allianceID, Name: "Caldari"}}, nil
	}

	handler := listUniverseRacesHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, ListUniverseRacesInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Races) != 1 || out.Races[0].RaceID != 1 || out.Races[0].Name != "Caldari" {
		t.Errorf("out = %+v, want one entry RaceID=1 Name=Caldari", out)
	}
}

func TestListUniverseRacesHandler_PropagatesError(t *testing.T) {
	orig := listUniverseRaces
	t.Cleanup(func() { listUniverseRaces = orig })
	wantErr := errors.New("upstream failure")
	listUniverseRaces = func(ctx context.Context, sdeStore raceCache, userAgent string) ([]UniverseRace, error) {
		return nil, wantErr
	}

	handler := listUniverseRacesHandler("v1", nil)
	_, _, err := handler(context.Background(), nil, ListUniverseRacesInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListUniverseFactions_CacheHit_MapsFieldsIncludingStationCounts(t *testing.T) {
	corpID := int64(1000035)
	cache := &fakeFactionCache{factions: []sde.Faction{
		// IsUnique omitted -- a sqlite-served Faction never has it (see
		// sde.Faction's doc comment), so the fixture matches real behavior.
		{ID: 500001, Name: "Caldari State", Description: "The Caldari State...", SizeFactor: 5.0, CorporationID: &corpID, StationCount: 120, StationSystemCount: 45},
	}}

	got, err := listUniverseFactions(context.Background(), cache, "test-agent")
	if err != nil {
		t.Fatalf("listUniverseFactions returned error: %v", err)
	}
	if len(got) != 1 || got[0].FactionID != 500001 || got[0].StationCount != 120 || got[0].StationSystemCount != 45 {
		t.Errorf("got = %+v, want one entry FactionID=500001 StationCount=120 StationSystemCount=45", got)
	}
	if got[0].IsUnique != nil {
		t.Errorf("IsUnique = %v, want nil on a cache hit (chrFactions has no source for it)", *got[0].IsUnique)
	}
}

func TestListUniverseFactionsHandler_MapsFields(t *testing.T) {
	orig := listUniverseFactions
	t.Cleanup(func() { listUniverseFactions = orig })
	corpID := int64(1000035)
	isUnique := true
	listUniverseFactions = func(ctx context.Context, sdeStore factionCache, userAgent string) ([]UniverseFaction, error) {
		return []UniverseFaction{{FactionID: 500001, Name: "Caldari State", IsUnique: &isUnique, CorporationID: &corpID}}, nil
	}

	handler := listUniverseFactionsHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, ListUniverseFactionsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Factions) != 1 || out.Factions[0].FactionID != 500001 || out.Factions[0].IsUnique == nil || !*out.Factions[0].IsUnique {
		t.Errorf("out = %+v, want one entry FactionID=500001 IsUnique=true", out)
	}
	if out.Factions[0].CorporationID == nil || *out.Factions[0].CorporationID != 1000035 {
		t.Errorf("CorporationID = %v, want 1000035", out.Factions[0].CorporationID)
	}
}

func TestListUniverseFactionsHandler_PropagatesError(t *testing.T) {
	orig := listUniverseFactions
	t.Cleanup(func() { listUniverseFactions = orig })
	wantErr := errors.New("upstream failure")
	listUniverseFactions = func(ctx context.Context, sdeStore factionCache, userAgent string) ([]UniverseFaction, error) {
		return nil, wantErr
	}

	handler := listUniverseFactionsHandler("v1", nil)
	_, _, err := handler(context.Background(), nil, ListUniverseFactionsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_universe_categories / get_universe_category / list_universe_groups /
// get_universe_group / list_universe_types / get_universe_type ---
//
// These 6 tools read through sde.Store (via the narrow categoryCache/
// groupCache/typeCache interfaces) with an ESI fallback on a cache miss. The
// fakeCategoryCache/fakeGroupCache/fakeTypeCache types below stand in for a
// real *sde.Store, which would otherwise attempt live network I/O on every
// test run.

type fakeCategoryCache struct {
	category    sde.Category
	categoryOK  bool
	categoryErr error
	ids         []int64
	idsErr      error
	putErr      error
	putCalls    []sde.Category
}

func (f *fakeCategoryCache) Category(ctx context.Context, userAgent string, id int64) (sde.Category, bool, error) {
	return f.category, f.categoryOK, f.categoryErr
}
func (f *fakeCategoryCache) PutCategory(c sde.Category) error {
	f.putCalls = append(f.putCalls, c)
	return f.putErr
}
func (f *fakeCategoryCache) CategoryIDs(ctx context.Context, userAgent string) ([]int64, error) {
	return f.ids, f.idsErr
}

type fakeGroupCache struct {
	group    sde.Group
	groupOK  bool
	groupErr error
	ids      []int64
	idsErr   error
	putErr   error
	putCalls []sde.Group
}

func (f *fakeGroupCache) Group(ctx context.Context, userAgent string, id int64) (sde.Group, bool, error) {
	return f.group, f.groupOK, f.groupErr
}
func (f *fakeGroupCache) PutGroup(g sde.Group) error {
	f.putCalls = append(f.putCalls, g)
	return f.putErr
}
func (f *fakeGroupCache) GroupIDs(ctx context.Context, userAgent string) ([]int64, error) {
	return f.ids, f.idsErr
}

type fakeTypeCache struct {
	typ      sde.Type
	typeOK   bool
	typeErr  error
	ids      []int64
	idsErr   error
	putErr   error
	putCalls []sde.Type
}

func (f *fakeTypeCache) Type(ctx context.Context, userAgent string, id int64) (sde.Type, bool, error) {
	return f.typ, f.typeOK, f.typeErr
}
func (f *fakeTypeCache) PutType(t sde.Type) error {
	f.putCalls = append(f.putCalls, t)
	return f.putErr
}
func (f *fakeTypeCache) TypeIDs(ctx context.Context, userAgent string) ([]int64, error) {
	return f.ids, f.idsErr
}

func TestListUniverseCategories_CacheHit_ReturnsCachedIDs(t *testing.T) {
	cache := &fakeCategoryCache{ids: []int64{4, 6}}

	ids, err := listUniverseCategories(context.Background(), cache, "test-agent")
	if err != nil {
		t.Fatalf("listUniverseCategories returned error: %v", err)
	}
	if len(ids) != 2 || ids[0] != 4 || ids[1] != 6 {
		t.Errorf("ids = %v, want [4 6]", ids)
	}
}

func TestListUniverseCategories_CacheError_ReturnsError(t *testing.T) {
	cache := &fakeCategoryCache{idsErr: errors.New("boom")}

	_, err := listUniverseCategories(context.Background(), cache, "test-agent")
	if err == nil {
		t.Fatal("listUniverseCategories returned nil error, want the wrapped cache error")
	}
}

func TestListUniverseCategoriesHandler_MapsFields(t *testing.T) {
	orig := listUniverseCategories
	t.Cleanup(func() { listUniverseCategories = orig })
	listUniverseCategories = func(ctx context.Context, sdeStore categoryCache, userAgent string) ([]int64, error) {
		return []int64{6, 7}, nil
	}

	handler := listUniverseCategoriesHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, ListUniverseCategoriesInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.IDs) != 2 || out.IDs[0] != 6 {
		t.Errorf("out = %+v, want IDs=[6 7]", out)
	}
}

func TestListUniverseCategoriesHandler_PropagatesError(t *testing.T) {
	orig := listUniverseCategories
	t.Cleanup(func() { listUniverseCategories = orig })
	wantErr := errors.New("upstream failure")
	listUniverseCategories = func(ctx context.Context, sdeStore categoryCache, userAgent string) ([]int64, error) {
		return nil, wantErr
	}

	handler := listUniverseCategoriesHandler("v1", nil)
	_, _, err := handler(context.Background(), nil, ListUniverseCategoriesInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetUniverseCategory_CacheHit_MapsFields(t *testing.T) {
	cache := &fakeCategoryCache{
		category:   sde.Category{ID: 6, Name: "Ship", Published: true, GroupIDs: []int64{25}},
		categoryOK: true,
	}

	got, err := getUniverseCategory(context.Background(), cache, "test-agent", 6)
	if err != nil {
		t.Fatalf("getUniverseCategory returned error: %v", err)
	}
	if got.CategoryID != 6 || got.Name != "Ship" || !got.Published || len(got.GroupIDs) != 1 || got.GroupIDs[0] != 25 {
		t.Errorf("got = %+v, want CategoryID=6 Name=Ship Published=true GroupIDs=[25]", got)
	}
	if len(cache.putCalls) != 0 {
		t.Errorf("PutCategory called %d times on a cache hit, want 0", len(cache.putCalls))
	}
}

func TestGetUniverseCategory_CacheError_ReturnsError(t *testing.T) {
	cache := &fakeCategoryCache{categoryErr: errors.New("boom")}

	_, err := getUniverseCategory(context.Background(), cache, "test-agent", 6)
	if err == nil {
		t.Fatal("getUniverseCategory returned nil error, want the wrapped cache error")
	}
}

func TestGetUniverseCategoryHandler_MapsFields(t *testing.T) {
	orig := getUniverseCategory
	t.Cleanup(func() { getUniverseCategory = orig })
	getUniverseCategory = func(ctx context.Context, sdeStore categoryCache, userAgent string, categoryID int64) (UniverseCategory, error) {
		return UniverseCategory{CategoryID: 6, Name: "Ship", Published: true, GroupIDs: []int64{25, 26}}, nil
	}

	handler := getUniverseCategoryHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, GetUniverseCategoryInput{CategoryID: 6})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CategoryID != 6 || out.Name != "Ship" || !out.Published || len(out.GroupIDs) != 2 {
		t.Errorf("out = %+v, want CategoryID=6 Name=Ship Published=true GroupIDs=[25 26]", out)
	}
}

func TestGetUniverseCategoryHandler_PropagatesError(t *testing.T) {
	orig := getUniverseCategory
	t.Cleanup(func() { getUniverseCategory = orig })
	wantErr := errors.New("upstream failure")
	getUniverseCategory = func(ctx context.Context, sdeStore categoryCache, userAgent string, categoryID int64) (UniverseCategory, error) {
		return UniverseCategory{}, wantErr
	}

	handler := getUniverseCategoryHandler("v1", nil)
	_, _, err := handler(context.Background(), nil, GetUniverseCategoryInput{CategoryID: 6})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListUniverseGroups_CacheHit_PaginatesAt1000(t *testing.T) {
	ids := make([]int64, 1500)
	for i := range ids {
		ids[i] = int64(i + 1)
	}
	cache := &fakeGroupCache{ids: ids}

	page1, totalPages, err := listUniverseGroups(context.Background(), cache, "test-agent", 1)
	if err != nil {
		t.Fatalf("listUniverseGroups page 1 returned error: %v", err)
	}
	if len(page1) != 1000 || page1[0] != 1 || page1[999] != 1000 {
		t.Errorf("page 1 = %d ids starting %d ending %d, want 1000 ids from 1 to 1000", len(page1), page1[0], page1[len(page1)-1])
	}
	if totalPages != 2 {
		t.Errorf("totalPages = %d, want 2", totalPages)
	}

	page2, _, err := listUniverseGroups(context.Background(), cache, "test-agent", 2)
	if err != nil {
		t.Fatalf("listUniverseGroups page 2 returned error: %v", err)
	}
	if len(page2) != 500 || page2[0] != 1001 {
		t.Errorf("page 2 = %d ids starting %d, want 500 ids starting 1001", len(page2), page2[0])
	}
}

func TestListUniverseGroups_CacheHit_PageZeroDefaultsToOne(t *testing.T) {
	cache := &fakeGroupCache{ids: []int64{25, 26}}

	ids, totalPages, err := listUniverseGroups(context.Background(), cache, "test-agent", 0)
	if err != nil {
		t.Fatalf("listUniverseGroups returned error: %v", err)
	}
	if len(ids) != 2 || totalPages != 1 {
		t.Errorf("ids = %v, totalPages = %d, want [25 26], 1", ids, totalPages)
	}
}

func TestListUniverseGroupsHandler_MapsFields(t *testing.T) {
	orig := listUniverseGroups
	t.Cleanup(func() { listUniverseGroups = orig })
	listUniverseGroups = func(ctx context.Context, sdeStore groupCache, userAgent string, page int32) ([]int64, int32, error) {
		return []int64{25, 26}, 3, nil
	}

	handler := listUniverseGroupsHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, ListUniverseGroupsInput{Page: 2})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.IDs) != 2 || out.Page != 2 || out.TotalPages != 3 {
		t.Errorf("out = %+v, want IDs=[25 26] Page=2 TotalPages=3", out)
	}
}

func TestListUniverseGroupsHandler_DefaultsPageToOne(t *testing.T) {
	orig := listUniverseGroups
	t.Cleanup(func() { listUniverseGroups = orig })
	listUniverseGroups = func(ctx context.Context, sdeStore groupCache, userAgent string, page int32) ([]int64, int32, error) {
		return []int64{25}, 3, nil
	}

	handler := listUniverseGroupsHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, ListUniverseGroupsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Page != 1 {
		t.Errorf("Page = %d, want 1", out.Page)
	}
}

func TestListUniverseGroupsHandler_PropagatesError(t *testing.T) {
	orig := listUniverseGroups
	t.Cleanup(func() { listUniverseGroups = orig })
	wantErr := errors.New("upstream failure")
	listUniverseGroups = func(ctx context.Context, sdeStore groupCache, userAgent string, page int32) ([]int64, int32, error) {
		return nil, 0, wantErr
	}

	handler := listUniverseGroupsHandler("v1", nil)
	_, _, err := handler(context.Background(), nil, ListUniverseGroupsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetUniverseGroup_CacheHit_MapsFields(t *testing.T) {
	cache := &fakeGroupCache{
		group:   sde.Group{ID: 25, CategoryID: 6, Name: "Frigate", Published: true, TypeIDs: []int64{587}},
		groupOK: true,
	}

	got, err := getUniverseGroup(context.Background(), cache, "test-agent", 25)
	if err != nil {
		t.Fatalf("getUniverseGroup returned error: %v", err)
	}
	want := UniverseGroup{GroupID: 25, CategoryID: 6, Name: "Frigate", Published: true, TypeIDs: []int64{587}}
	if got.GroupID != want.GroupID || got.CategoryID != want.CategoryID || got.Name != want.Name || got.Published != want.Published || len(got.TypeIDs) != 1 || got.TypeIDs[0] != 587 {
		t.Errorf("got = %+v, want %+v", got, want)
	}
}

func TestGetUniverseGroupHandler_MapsFields(t *testing.T) {
	orig := getUniverseGroup
	t.Cleanup(func() { getUniverseGroup = orig })
	getUniverseGroup = func(ctx context.Context, sdeStore groupCache, userAgent string, groupID int64) (UniverseGroup, error) {
		return UniverseGroup{GroupID: 25, CategoryID: 6, Name: "Frigate", Published: true, TypeIDs: []int64{587, 588}}, nil
	}

	handler := getUniverseGroupHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, GetUniverseGroupInput{GroupID: 25})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.GroupID != 25 || out.CategoryID != 6 || out.Name != "Frigate" || len(out.TypeIDs) != 2 {
		t.Errorf("out = %+v, want GroupID=25 CategoryID=6 Name=Frigate TypeIDs=[587 588]", out)
	}
}

func TestGetUniverseGroupHandler_PropagatesError(t *testing.T) {
	orig := getUniverseGroup
	t.Cleanup(func() { getUniverseGroup = orig })
	wantErr := errors.New("upstream failure")
	getUniverseGroup = func(ctx context.Context, sdeStore groupCache, userAgent string, groupID int64) (UniverseGroup, error) {
		return UniverseGroup{}, wantErr
	}

	handler := getUniverseGroupHandler("v1", nil)
	_, _, err := handler(context.Background(), nil, GetUniverseGroupInput{GroupID: 25})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListUniverseTypes_CacheHit_Paginates(t *testing.T) {
	cache := &fakeTypeCache{ids: []int64{587, 588}}

	ids, totalPages, err := listUniverseTypes(context.Background(), cache, "test-agent", 1)
	if err != nil {
		t.Fatalf("listUniverseTypes returned error: %v", err)
	}
	if len(ids) != 2 || totalPages != 1 {
		t.Errorf("ids = %v, totalPages = %d, want [587 588], 1", ids, totalPages)
	}
}

func TestListUniverseTypesHandler_MapsFields(t *testing.T) {
	orig := listUniverseTypes
	t.Cleanup(func() { listUniverseTypes = orig })
	listUniverseTypes = func(ctx context.Context, sdeStore typeCache, userAgent string, page int32) ([]int64, int32, error) {
		return []int64{587, 588}, 5, nil
	}

	handler := listUniverseTypesHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, ListUniverseTypesInput{Page: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.IDs) != 2 || out.Page != 1 || out.TotalPages != 5 {
		t.Errorf("out = %+v, want IDs=[587 588] Page=1 TotalPages=5", out)
	}
}

func TestListUniverseTypesHandler_PropagatesError(t *testing.T) {
	orig := listUniverseTypes
	t.Cleanup(func() { listUniverseTypes = orig })
	wantErr := errors.New("upstream failure")
	listUniverseTypes = func(ctx context.Context, sdeStore typeCache, userAgent string, page int32) ([]int64, int32, error) {
		return nil, 0, wantErr
	}

	handler := listUniverseTypesHandler("v1", nil)
	_, _, err := handler(context.Background(), nil, ListUniverseTypesInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetUniverseType_CacheHit_MapsFieldsIncludingPackagedVolume(t *testing.T) {
	mass := 1067000.0
	packagedVolume := 2500.0
	cache := &fakeTypeCache{
		typ: sde.Type{
			ID: 587, GroupID: 25, Name: "Rifter", Description: "Fast.",
			Published: true, Mass: &mass, PackagedVolume: &packagedVolume,
			DogmaAttributes: []sde.TypeDogmaAttribute{{AttributeID: 182, Value: 3386.0}},
			DogmaEffects:    []sde.TypeDogmaEffect{{EffectID: 4, IsDefault: true}},
		},
		typeOK: true,
	}

	got, err := getUniverseType(context.Background(), cache, "test-agent", 587)
	if err != nil {
		t.Fatalf("getUniverseType returned error: %v", err)
	}
	if got.TypeID != 587 || got.Name != "Rifter" || got.Description != "Fast." || got.GroupID != 25 || !got.Published {
		t.Errorf("got = %+v, want TypeID=587 Name=Rifter Description=Fast. GroupID=25 Published=true", got)
	}
	if got.Mass == nil || *got.Mass != 1067000.0 {
		t.Errorf("Mass = %v, want 1067000.0", got.Mass)
	}
	if got.PackagedVolume == nil || *got.PackagedVolume != 2500 {
		t.Errorf("PackagedVolume = %v, want 2500 (from a cache hit, not only a cache miss)", got.PackagedVolume)
	}
	if len(got.DogmaAttributes) != 1 || got.DogmaAttributes[0].AttributeID != 182 {
		t.Errorf("DogmaAttributes = %+v, want one entry with AttributeID=182", got.DogmaAttributes)
	}
	if len(got.DogmaEffects) != 1 || got.DogmaEffects[0].EffectID != 4 {
		t.Errorf("DogmaEffects = %+v, want one entry with EffectID=4", got.DogmaEffects)
	}
	if len(cache.putCalls) != 0 {
		t.Errorf("PutType called %d times on a cache hit, want 0", len(cache.putCalls))
	}
}

func TestGetUniverseTypeHandler_MapsFields(t *testing.T) {
	orig := getUniverseType
	t.Cleanup(func() { getUniverseType = orig })
	mass := 1200000.0
	getUniverseType = func(ctx context.Context, sdeStore typeCache, userAgent string, typeID int64) (UniverseType, error) {
		return UniverseType{
			TypeID: 587, Name: "Rifter", GroupID: 25, Published: true, Mass: &mass,
			DogmaAttributes: []DynamicItemAttribute{{AttributeID: 9, Value: 2500}},
			DogmaEffects:    []DynamicItemEffect{{EffectID: 1, IsDefault: true}},
		}, nil
	}

	handler := getUniverseTypeHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, GetUniverseTypeInput{TypeID: 587})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.TypeID != 587 || out.Name != "Rifter" || out.GroupID != 25 || !out.Published {
		t.Errorf("out = %+v, want TypeID=587 Name=Rifter GroupID=25 Published=true", out)
	}
	if out.Mass == nil || *out.Mass != 1200000 {
		t.Errorf("Mass = %v, want 1200000", out.Mass)
	}
	if len(out.DogmaAttributes) != 1 || out.DogmaAttributes[0].AttributeID != 9 || out.DogmaAttributes[0].Value != 2500 {
		t.Errorf("DogmaAttributes = %+v, want one entry AttributeID=9 Value=2500", out.DogmaAttributes)
	}
	if len(out.DogmaEffects) != 1 || out.DogmaEffects[0].EffectID != 1 || !out.DogmaEffects[0].IsDefault {
		t.Errorf("DogmaEffects = %+v, want one entry EffectID=1 IsDefault=true", out.DogmaEffects)
	}
}

func TestGetUniverseTypeHandler_PropagatesError(t *testing.T) {
	orig := getUniverseType
	t.Cleanup(func() { getUniverseType = orig })
	wantErr := errors.New("upstream failure")
	getUniverseType = func(ctx context.Context, sdeStore typeCache, userAgent string, typeID int64) (UniverseType, error) {
		return UniverseType{}, wantErr
	}

	handler := getUniverseTypeHandler("v1", nil)
	_, _, err := handler(context.Background(), nil, GetUniverseTypeInput{TypeID: 587})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_universe_regions / get_universe_region / list_universe_constellations /
// get_universe_constellation / list_universe_systems / get_universe_system ---
//
// Same shape as the categories/groups/types fakes above: fakeRegionCache/
// fakeConstellationCache/fakeSystemCache stand in for a real *sde.Store.

type fakeRegionCache struct {
	region    sde.Region
	regionOK  bool
	regionErr error
	ids       []int64
	idsErr    error
	putErr    error
	putCalls  []sde.Region
}

func (f *fakeRegionCache) Region(ctx context.Context, userAgent string, id int64) (sde.Region, bool, error) {
	return f.region, f.regionOK, f.regionErr
}
func (f *fakeRegionCache) PutRegion(r sde.Region) error {
	f.putCalls = append(f.putCalls, r)
	return f.putErr
}
func (f *fakeRegionCache) RegionIDs(ctx context.Context, userAgent string) ([]int64, error) {
	return f.ids, f.idsErr
}

type fakeConstellationCache struct {
	constellation   sde.Constellation
	constellationOK bool
	err             error
	ids             []int64
	idsErr          error
	putErr          error
	putCalls        []sde.Constellation
}

func (f *fakeConstellationCache) Constellation(ctx context.Context, userAgent string, id int64) (sde.Constellation, bool, error) {
	return f.constellation, f.constellationOK, f.err
}
func (f *fakeConstellationCache) PutConstellation(c sde.Constellation) error {
	f.putCalls = append(f.putCalls, c)
	return f.putErr
}
func (f *fakeConstellationCache) ConstellationIDs(ctx context.Context, userAgent string) ([]int64, error) {
	return f.ids, f.idsErr
}

type fakeSystemCache struct {
	system   sde.System
	systemOK bool
	err      error
	ids      []int64
	idsErr   error
	putErr   error
	putCalls []sde.System
}

func (f *fakeSystemCache) System(ctx context.Context, userAgent string, id int64) (sde.System, bool, error) {
	return f.system, f.systemOK, f.err
}
func (f *fakeSystemCache) PutSystem(s sde.System) error {
	f.putCalls = append(f.putCalls, s)
	return f.putErr
}
func (f *fakeSystemCache) SystemIDs(ctx context.Context, userAgent string) ([]int64, error) {
	return f.ids, f.idsErr
}

func TestListUniverseRegions_CacheHit_ReturnsCachedIDs(t *testing.T) {
	cache := &fakeRegionCache{ids: []int64{10000001, 10000002}}

	ids, err := listUniverseRegions(context.Background(), cache, "test-agent")
	if err != nil {
		t.Fatalf("listUniverseRegions returned error: %v", err)
	}
	if len(ids) != 2 || ids[0] != 10000001 {
		t.Errorf("ids = %v, want [10000001 10000002]", ids)
	}
}

func TestListUniverseRegionsHandler_MapsFields(t *testing.T) {
	orig := listUniverseRegions
	t.Cleanup(func() { listUniverseRegions = orig })
	listUniverseRegions = func(ctx context.Context, sdeStore regionCache, userAgent string) ([]int64, error) {
		return []int64{10000002, 10000043}, nil
	}

	handler := listUniverseRegionsHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, ListUniverseRegionsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.IDs) != 2 || out.IDs[0] != 10000002 {
		t.Errorf("out = %+v, want IDs=[10000002 10000043]", out)
	}
}

func TestListUniverseRegionsHandler_PropagatesError(t *testing.T) {
	orig := listUniverseRegions
	t.Cleanup(func() { listUniverseRegions = orig })
	wantErr := errors.New("upstream failure")
	listUniverseRegions = func(ctx context.Context, sdeStore regionCache, userAgent string) ([]int64, error) {
		return nil, wantErr
	}

	handler := listUniverseRegionsHandler("v1", nil)
	_, _, err := handler(context.Background(), nil, ListUniverseRegionsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetUniverseRegion_CacheHit_MapsFields(t *testing.T) {
	cache := &fakeRegionCache{
		region: sde.Region{
			ID: 10000001, Name: "Derelik",
			Description: new("A dangerous frontier region."), ConstellationIDs: []int64{20000001},
		},
		regionOK: true,
	}

	got, err := getUniverseRegion(context.Background(), cache, "test-agent", 10000001)
	if err != nil {
		t.Fatalf("getUniverseRegion returned error: %v", err)
	}
	if got.RegionID != 10000001 || got.Name != "Derelik" || got.Description == nil || *got.Description != "A dangerous frontier region." {
		t.Errorf("got = %+v, want RegionID=10000001 Name=Derelik Description=\"A dangerous frontier region.\"", got)
	}
	if len(got.ConstellationIDs) != 1 || got.ConstellationIDs[0] != 20000001 {
		t.Errorf("ConstellationIDs = %v, want [20000001]", got.ConstellationIDs)
	}
	if len(cache.putCalls) != 0 {
		t.Errorf("PutRegion called %d times on a cache hit, want 0", len(cache.putCalls))
	}
}

func TestGetUniverseRegion_CacheHit_NoDescription_ReturnsNilDescription(t *testing.T) {
	cache := &fakeRegionCache{
		region:   sde.Region{ID: 10000002, Name: "No Description"},
		regionOK: true,
	}

	got, err := getUniverseRegion(context.Background(), cache, "test-agent", 10000002)
	if err != nil {
		t.Fatalf("getUniverseRegion returned error: %v", err)
	}
	if got.Description != nil {
		t.Errorf("Description = %v, want nil", got.Description)
	}
}

func TestGetUniverseRegionHandler_MapsFields(t *testing.T) {
	orig := getUniverseRegion
	t.Cleanup(func() { getUniverseRegion = orig })
	desc := "desc"
	getUniverseRegion = func(ctx context.Context, sdeStore regionCache, userAgent string, regionID int64) (UniverseRegion, error) {
		return UniverseRegion{RegionID: 10000002, Name: "The Forge", Description: &desc, ConstellationIDs: []int64{20000020}}, nil
	}

	handler := getUniverseRegionHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, GetUniverseRegionInput{RegionID: 10000002})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.RegionID != 10000002 || out.Name != "The Forge" || len(out.ConstellationIDs) != 1 {
		t.Errorf("out = %+v, want RegionID=10000002 Name=The Forge ConstellationIDs=[20000020]", out)
	}
}

func TestGetUniverseRegionHandler_PropagatesError(t *testing.T) {
	orig := getUniverseRegion
	t.Cleanup(func() { getUniverseRegion = orig })
	wantErr := errors.New("upstream failure")
	getUniverseRegion = func(ctx context.Context, sdeStore regionCache, userAgent string, regionID int64) (UniverseRegion, error) {
		return UniverseRegion{}, wantErr
	}

	handler := getUniverseRegionHandler("v1", nil)
	_, _, err := handler(context.Background(), nil, GetUniverseRegionInput{RegionID: 10000002})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListUniverseConstellationsHandler_MapsFields(t *testing.T) {
	orig := listUniverseConstellations
	t.Cleanup(func() { listUniverseConstellations = orig })
	listUniverseConstellations = func(ctx context.Context, sdeStore constellationCache, userAgent string) ([]int64, error) {
		return []int64{20000020}, nil
	}

	handler := listUniverseConstellationsHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, ListUniverseConstellationsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.IDs) != 1 || out.IDs[0] != 20000020 {
		t.Errorf("out = %+v, want IDs=[20000020]", out)
	}
}

func TestListUniverseConstellationsHandler_PropagatesError(t *testing.T) {
	orig := listUniverseConstellations
	t.Cleanup(func() { listUniverseConstellations = orig })
	wantErr := errors.New("upstream failure")
	listUniverseConstellations = func(ctx context.Context, sdeStore constellationCache, userAgent string) ([]int64, error) {
		return nil, wantErr
	}

	handler := listUniverseConstellationsHandler("v1", nil)
	_, _, err := handler(context.Background(), nil, ListUniverseConstellationsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetUniverseConstellation_CacheHit_MapsFields(t *testing.T) {
	cache := &fakeConstellationCache{
		constellation: sde.Constellation{
			ID: 20000001, RegionID: 10000001, Name: "San Matar",
			Position: sde.Position{X: 1, Y: 2, Z: 3}, SystemIDs: []int64{30000001},
		},
		constellationOK: true,
	}

	got, err := getUniverseConstellation(context.Background(), cache, "test-agent", 20000001)
	if err != nil {
		t.Fatalf("getUniverseConstellation returned error: %v", err)
	}
	if got.ConstellationID != 20000001 || got.RegionID != 10000001 || got.Name != "San Matar" {
		t.Errorf("got = %+v, want ConstellationID=20000001 RegionID=10000001 Name=San Matar", got)
	}
	if got.Position != (Position{X: 1, Y: 2, Z: 3}) {
		t.Errorf("Position = %+v, want {1 2 3}", got.Position)
	}
	if len(got.SystemIDs) != 1 || got.SystemIDs[0] != 30000001 {
		t.Errorf("SystemIDs = %v, want [30000001]", got.SystemIDs)
	}
}

func TestGetUniverseConstellationHandler_MapsFields(t *testing.T) {
	orig := getUniverseConstellation
	t.Cleanup(func() { getUniverseConstellation = orig })
	getUniverseConstellation = func(ctx context.Context, sdeStore constellationCache, userAgent string, constellationID int64) (UniverseConstellation, error) {
		return UniverseConstellation{
			ConstellationID: 20000020, RegionID: 10000002, Name: "Kimotoro",
			Position: Position{X: 1, Y: 2, Z: 3}, SystemIDs: []int64{30000142, 30000144},
		}, nil
	}

	handler := getUniverseConstellationHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, GetUniverseConstellationInput{ConstellationID: 20000020})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.ConstellationID != 20000020 || out.RegionID != 10000002 || out.Name != "Kimotoro" || len(out.SystemIDs) != 2 {
		t.Errorf("out = %+v, want ConstellationID=20000020 RegionID=10000002 Name=Kimotoro SystemIDs=[30000142 30000144]", out)
	}
	if out.Position.X != 1 || out.Position.Y != 2 || out.Position.Z != 3 {
		t.Errorf("Position = %+v, want X=1 Y=2 Z=3", out.Position)
	}
}

func TestGetUniverseConstellationHandler_PropagatesError(t *testing.T) {
	orig := getUniverseConstellation
	t.Cleanup(func() { getUniverseConstellation = orig })
	wantErr := errors.New("upstream failure")
	getUniverseConstellation = func(ctx context.Context, sdeStore constellationCache, userAgent string, constellationID int64) (UniverseConstellation, error) {
		return UniverseConstellation{}, wantErr
	}

	handler := getUniverseConstellationHandler("v1", nil)
	_, _, err := handler(context.Background(), nil, GetUniverseConstellationInput{ConstellationID: 20000020})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListUniverseSystemsHandler_MapsFields(t *testing.T) {
	orig := listUniverseSystems
	t.Cleanup(func() { listUniverseSystems = orig })
	listUniverseSystems = func(ctx context.Context, sdeStore systemCache, userAgent string) ([]int64, error) {
		return []int64{30000142}, nil
	}

	handler := listUniverseSystemsHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, ListUniverseSystemsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.IDs) != 1 || out.IDs[0] != 30000142 {
		t.Errorf("out = %+v, want IDs=[30000142]", out)
	}
}

func TestListUniverseSystemsHandler_PropagatesError(t *testing.T) {
	orig := listUniverseSystems
	t.Cleanup(func() { listUniverseSystems = orig })
	wantErr := errors.New("upstream failure")
	listUniverseSystems = func(ctx context.Context, sdeStore systemCache, userAgent string) ([]int64, error) {
		return nil, wantErr
	}

	handler := listUniverseSystemsHandler("v1", nil)
	_, _, err := handler(context.Background(), nil, ListUniverseSystemsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetUniverseSystem_CacheHit_MapsFieldsIncludingPlanets(t *testing.T) {
	secClass := "B"
	starID := int64(40000001)
	cache := &fakeSystemCache{
		system: sde.System{
			ID: 30000001, ConstellationID: 20000001, Name: "Tanoo",
			Position: sde.Position{X: 1, Y: 2, Z: 3}, SecurityStatus: 0.85, SecurityClass: &secClass,
			StarID: &starID, StargateIDs: []int64{50000056}, StationIDs: []int64{60000001},
			Planets: []sde.SystemPlanet{{PlanetID: 40000002, MoonIDs: []int64{40000004}, AsteroidBeltIDs: []int64{40000003}}},
		},
		systemOK: true,
	}

	got, err := getUniverseSystem(context.Background(), cache, "test-agent", 30000001)
	if err != nil {
		t.Fatalf("getUniverseSystem returned error: %v", err)
	}
	if got.SystemID != 30000001 || got.ConstellationID != 20000001 || got.Name != "Tanoo" || got.SecurityStatus != 0.85 {
		t.Errorf("got = %+v, want SystemID=30000001 ConstellationID=20000001 Name=Tanoo SecurityStatus=0.85", got)
	}
	if len(got.StargateIDs) != 1 || got.StargateIDs[0] != 50000056 {
		t.Errorf("StargateIDs = %v, want [50000056]", got.StargateIDs)
	}
	if len(got.StationIDs) != 1 || got.StationIDs[0] != 60000001 {
		t.Errorf("StationIDs = %v, want [60000001]", got.StationIDs)
	}
	if len(got.Planets) != 1 || got.Planets[0].PlanetID != 40000002 || len(got.Planets[0].MoonIDs) != 1 || len(got.Planets[0].AsteroidBeltIDs) != 1 {
		t.Errorf("Planets = %+v, want one entry PlanetID=40000002 with 1 moon and 1 belt (from a cache hit, not only a cache miss)", got.Planets)
	}
	if got.SecurityDisplay != "0.9" || got.SecurityColor != "#399AEB" {
		t.Errorf("security = %q %q, want the displayed 0.9 and its in-game color", got.SecurityDisplay, got.SecurityColor)
	}
	if len(cache.putCalls) != 0 {
		t.Errorf("PutSystem called %d times on a cache hit, want 0", len(cache.putCalls))
	}
}

func TestGetUniverseSystemHandler_MapsFields(t *testing.T) {
	orig := getUniverseSystem
	t.Cleanup(func() { getUniverseSystem = orig })
	starID := int64(40000001)
	secClass := "B"
	getUniverseSystem = func(ctx context.Context, sdeStore systemCache, userAgent string, systemID int64) (UniverseSystem, error) {
		return UniverseSystem{
			SystemID: 30000142, ConstellationID: 20000020, Name: "Jita",
			Position: Position{X: 1, Y: 2, Z: 3}, SecurityStatus: 0.9459, SecurityClass: &secClass,
			StarID: &starID, StargateIDs: []int64{50000342}, StationIDs: []int64{60003760},
			Planets: []UniverseSystemPlanet{{PlanetID: 40000004, AsteroidBeltIDs: []int64{40000002}, MoonIDs: []int64{40000003}}},
		}, nil
	}

	handler := getUniverseSystemHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, GetUniverseSystemInput{SystemID: 30000142})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.SystemID != 30000142 || out.Name != "Jita" || out.ConstellationID != 20000020 {
		t.Errorf("out = %+v, want SystemID=30000142 Name=Jita ConstellationID=20000020", out)
	}
	if out.SecurityStatus != 0.9459 || out.SecurityClass == nil || *out.SecurityClass != "B" {
		t.Errorf("SecurityStatus/Class = %v/%v, want 0.9459/B", out.SecurityStatus, out.SecurityClass)
	}
	if len(out.Planets) != 1 || out.Planets[0].PlanetID != 40000004 || len(out.Planets[0].AsteroidBeltIDs) != 1 || len(out.Planets[0].MoonIDs) != 1 {
		t.Errorf("Planets = %+v, want one entry PlanetID=40000004 with one asteroid belt and one moon", out.Planets)
	}
}

func TestGetUniverseSystemHandler_PropagatesError(t *testing.T) {
	orig := getUniverseSystem
	t.Cleanup(func() { getUniverseSystem = orig })
	wantErr := errors.New("upstream failure")
	getUniverseSystem = func(ctx context.Context, sdeStore systemCache, userAgent string, systemID int64) (UniverseSystem, error) {
		return UniverseSystem{}, wantErr
	}

	handler := getUniverseSystemHandler("v1", nil)
	_, _, err := handler(context.Background(), nil, GetUniverseSystemInput{SystemID: 30000142})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- get_universe_asteroid_belt ---

type fakeAsteroidBeltCache struct {
	belt    sde.AsteroidBelt
	ok      bool
	err     error
	putErr  error
	putCall *sde.AsteroidBelt
}

func (f *fakeAsteroidBeltCache) AsteroidBelt(ctx context.Context, userAgent string, id int64) (sde.AsteroidBelt, bool, error) {
	return f.belt, f.ok, f.err
}
func (f *fakeAsteroidBeltCache) PutAsteroidBelt(b sde.AsteroidBelt) error {
	f.putCall = &b
	return f.putErr
}

func TestGetUniverseAsteroidBelt_CacheHit_ReturnsRecordWithoutESICall(t *testing.T) {
	cache := &fakeAsteroidBeltCache{belt: sde.AsteroidBelt{ID: 40000003, Name: "Tanoo I - Asteroid Belt 1", SystemID: 30000001}, ok: true}
	out, err := getUniverseAsteroidBelt(context.Background(), cache, "test-agent", 40000003)
	if err != nil {
		t.Fatalf("getUniverseAsteroidBelt returned error: %v", err)
	}
	if out.Name != "Tanoo I - Asteroid Belt 1" || out.SystemID != 30000001 {
		t.Errorf("got %+v, want Name=\"Tanoo I - Asteroid Belt 1\" SystemID=30000001", out)
	}
}

func TestGetUniverseAsteroidBelt_CacheReadError_ReturnsError(t *testing.T) {
	cache := &fakeAsteroidBeltCache{err: errors.New("boom")}
	_, err := getUniverseAsteroidBelt(context.Background(), cache, "test-agent", 40000003)
	if err == nil {
		t.Fatal("getUniverseAsteroidBelt returned nil error, want the cache read error surfaced")
	}
}

func TestGetUniverseAsteroidBeltHandler_MapsFields(t *testing.T) {
	orig := getUniverseAsteroidBelt
	t.Cleanup(func() { getUniverseAsteroidBelt = orig })
	getUniverseAsteroidBelt = func(ctx context.Context, sdeStore asteroidBeltCache, userAgent string, asteroidBeltID int64) (UniverseAsteroidBelt, error) {
		return UniverseAsteroidBelt{Name: "Jita IV - Asteroid Belt 1", Position: Position{X: 1, Y: 2, Z: 3}, SystemID: 30000142}, nil
	}

	handler := getUniverseAsteroidBeltHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, GetUniverseAsteroidBeltInput{AsteroidBeltID: 40000002})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Name != "Jita IV - Asteroid Belt 1" || out.SystemID != 30000142 {
		t.Errorf("out = %+v, want Name=Jita IV - Asteroid Belt 1 SystemID=30000142", out)
	}
}

func TestGetUniverseAsteroidBeltHandler_PropagatesError(t *testing.T) {
	orig := getUniverseAsteroidBelt
	t.Cleanup(func() { getUniverseAsteroidBelt = orig })
	wantErr := errors.New("upstream failure")
	getUniverseAsteroidBelt = func(ctx context.Context, sdeStore asteroidBeltCache, userAgent string, asteroidBeltID int64) (UniverseAsteroidBelt, error) {
		return UniverseAsteroidBelt{}, wantErr
	}

	handler := getUniverseAsteroidBeltHandler("v1", nil)
	_, _, err := handler(context.Background(), nil, GetUniverseAsteroidBeltInput{AsteroidBeltID: 40000002})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- get_universe_moon ---

type fakeMoonCache struct {
	moon    sde.Moon
	ok      bool
	err     error
	putErr  error
	putCall *sde.Moon
}

func (f *fakeMoonCache) Moon(ctx context.Context, userAgent string, id int64) (sde.Moon, bool, error) {
	return f.moon, f.ok, f.err
}
func (f *fakeMoonCache) PutMoon(m sde.Moon) error {
	f.putCall = &m
	return f.putErr
}

func TestGetUniverseMoon_CacheHit_ReturnsRecordWithoutESICall(t *testing.T) {
	cache := &fakeMoonCache{moon: sde.Moon{ID: 40000004, Name: "Tanoo I - Moon 1", SystemID: 30000001}, ok: true}
	out, err := getUniverseMoon(context.Background(), cache, "test-agent", 40000004)
	if err != nil {
		t.Fatalf("getUniverseMoon returned error: %v", err)
	}
	if out.MoonID != 40000004 || out.Name != "Tanoo I - Moon 1" {
		t.Errorf("got %+v, want MoonID=40000004 Name=\"Tanoo I - Moon 1\"", out)
	}
}

func TestGetUniverseMoonHandler_MapsFields(t *testing.T) {
	orig := getUniverseMoon
	t.Cleanup(func() { getUniverseMoon = orig })
	getUniverseMoon = func(ctx context.Context, sdeStore moonCache, userAgent string, moonID int64) (UniverseMoon, error) {
		return UniverseMoon{MoonID: 40000003, SystemID: 30000142, Name: "Jita IV - Moon 1", Position: Position{X: 1, Y: 2, Z: 3}}, nil
	}

	handler := getUniverseMoonHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, GetUniverseMoonInput{MoonID: 40000003})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.MoonID != 40000003 || out.SystemID != 30000142 || out.Name != "Jita IV - Moon 1" {
		t.Errorf("out = %+v, want MoonID=40000003 SystemID=30000142 Name=Jita IV - Moon 1", out)
	}
}

func TestGetUniverseMoonHandler_PropagatesError(t *testing.T) {
	orig := getUniverseMoon
	t.Cleanup(func() { getUniverseMoon = orig })
	wantErr := errors.New("upstream failure")
	getUniverseMoon = func(ctx context.Context, sdeStore moonCache, userAgent string, moonID int64) (UniverseMoon, error) {
		return UniverseMoon{}, wantErr
	}

	handler := getUniverseMoonHandler("v1", nil)
	_, _, err := handler(context.Background(), nil, GetUniverseMoonInput{MoonID: 40000003})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- get_universe_planet ---

type fakePlanetCache struct {
	planet  sde.Planet
	ok      bool
	err     error
	putErr  error
	putCall *sde.Planet
}

func (f *fakePlanetCache) Planet(ctx context.Context, userAgent string, id int64) (sde.Planet, bool, error) {
	return f.planet, f.ok, f.err
}
func (f *fakePlanetCache) PutPlanet(p sde.Planet) error {
	f.putCall = &p
	return f.putErr
}

func TestGetUniversePlanet_CacheHit_ReturnsRecordWithoutESICall(t *testing.T) {
	cache := &fakePlanetCache{planet: sde.Planet{ID: 40000002, Name: "Tanoo I", TypeID: 11, SystemID: 30000001}, ok: true}
	out, err := getUniversePlanet(context.Background(), cache, "test-agent", 40000002)
	if err != nil {
		t.Fatalf("getUniversePlanet returned error: %v", err)
	}
	if out.Name != "Tanoo I" || out.TypeID != 11 {
		t.Errorf("got %+v, want Name=\"Tanoo I\" TypeID=11", out)
	}
}

func TestGetUniversePlanetHandler_MapsFields(t *testing.T) {
	orig := getUniversePlanet
	t.Cleanup(func() { getUniversePlanet = orig })
	getUniversePlanet = func(ctx context.Context, sdeStore planetCache, userAgent string, planetID int64) (UniversePlanet, error) {
		return UniversePlanet{PlanetID: 40000004, SystemID: 30000142, TypeID: 2016, Name: "Jita IV", Position: Position{X: 1, Y: 2, Z: 3}}, nil
	}

	handler := getUniversePlanetHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, GetUniversePlanetInput{PlanetID: 40000004})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.PlanetID != 40000004 || out.SystemID != 30000142 || out.TypeID != 2016 || out.Name != "Jita IV" {
		t.Errorf("out = %+v, want PlanetID=40000004 SystemID=30000142 TypeID=2016 Name=Jita IV", out)
	}
}

func TestGetUniversePlanetHandler_PropagatesError(t *testing.T) {
	orig := getUniversePlanet
	t.Cleanup(func() { getUniversePlanet = orig })
	wantErr := errors.New("upstream failure")
	getUniversePlanet = func(ctx context.Context, sdeStore planetCache, userAgent string, planetID int64) (UniversePlanet, error) {
		return UniversePlanet{}, wantErr
	}

	handler := getUniversePlanetHandler("v1", nil)
	_, _, err := handler(context.Background(), nil, GetUniversePlanetInput{PlanetID: 40000004})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- get_universe_stargate ---

type fakeStargateCache struct {
	stargate sde.Stargate
	ok       bool
	err      error
	putErr   error
	putCall  *sde.Stargate
}

func (f *fakeStargateCache) Stargate(ctx context.Context, userAgent string, id int64) (sde.Stargate, bool, error) {
	return f.stargate, f.ok, f.err
}
func (f *fakeStargateCache) PutStargate(g sde.Stargate) error {
	f.putCall = &g
	return f.putErr
}

func TestGetUniverseStargate_CacheHit_ReturnsRecordWithoutESICall(t *testing.T) {
	cache := &fakeStargateCache{stargate: sde.Stargate{ID: 50000001, Name: "Stargate (G-EURJ)", DestinationStargateID: 50000482}, ok: true}
	out, err := getUniverseStargate(context.Background(), cache, "test-agent", 50000001)
	if err != nil {
		t.Fatalf("getUniverseStargate returned error: %v", err)
	}
	if out.Name != "Stargate (G-EURJ)" || out.DestinationStargateID != 50000482 {
		t.Errorf("got %+v, want Name=\"Stargate (G-EURJ)\" DestinationStargateID=50000482", out)
	}
}

func TestGetUniverseStargateHandler_MapsFields(t *testing.T) {
	orig := getUniverseStargate
	t.Cleanup(func() { getUniverseStargate = orig })
	getUniverseStargate = func(ctx context.Context, sdeStore stargateCache, userAgent string, stargateID int64) (UniverseStargate, error) {
		return UniverseStargate{
			StargateID: 50000342, SystemID: 30000142, TypeID: 16, Name: "Stargate (Perimeter)",
			Position: Position{X: 1, Y: 2, Z: 3}, DestinationStargateID: 50000343, DestinationSystemID: 30000144,
		}, nil
	}

	handler := getUniverseStargateHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, GetUniverseStargateInput{StargateID: 50000342})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.StargateID != 50000342 || out.SystemID != 30000142 || out.TypeID != 16 {
		t.Errorf("out = %+v, want StargateID=50000342 SystemID=30000142 TypeID=16", out)
	}
	if out.DestinationStargateID != 50000343 || out.DestinationSystemID != 30000144 {
		t.Errorf("Destination = %d/%d, want 50000343/30000144", out.DestinationStargateID, out.DestinationSystemID)
	}
}

func TestGetUniverseStargateHandler_PropagatesError(t *testing.T) {
	orig := getUniverseStargate
	t.Cleanup(func() { getUniverseStargate = orig })
	wantErr := errors.New("upstream failure")
	getUniverseStargate = func(ctx context.Context, sdeStore stargateCache, userAgent string, stargateID int64) (UniverseStargate, error) {
		return UniverseStargate{}, wantErr
	}

	handler := getUniverseStargateHandler("v1", nil)
	_, _, err := handler(context.Background(), nil, GetUniverseStargateInput{StargateID: 50000342})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- get_universe_star ---

type fakeStarCache struct {
	star    sde.Star
	ok      bool
	err     error
	putErr  error
	putCall *sde.Star
}

func (f *fakeStarCache) Star(ctx context.Context, userAgent string, id int64) (sde.Star, bool, error) {
	return f.star, f.ok, f.err
}
func (f *fakeStarCache) PutStar(s sde.Star) error {
	f.putCall = &s
	return f.putErr
}

func TestGetUniverseStar_CacheHit_ReturnsRecordWithoutESICall(t *testing.T) {
	cache := &fakeStarCache{star: sde.Star{ID: 40000001, Name: "Tanoo - Star", SpectralClass: "K2 V"}, ok: true}
	out, err := getUniverseStar(context.Background(), cache, "test-agent", 40000001)
	if err != nil {
		t.Fatalf("getUniverseStar returned error: %v", err)
	}
	if out.Name != "Tanoo - Star" || out.SpectralClass != "K2 V" {
		t.Errorf("got %+v, want Name=\"Tanoo - Star\" SpectralClass=\"K2 V\"", out)
	}
}

func TestGetUniverseStarHandler_MapsFields(t *testing.T) {
	orig := getUniverseStar
	t.Cleanup(func() { getUniverseStar = orig })
	getUniverseStar = func(ctx context.Context, sdeStore starCache, userAgent string, starID int64) (UniverseStar, error) {
		return UniverseStar{
			StarID: 40000001, Age: 15000000000, Luminosity: 0.02, Name: "Jita - Star", Radius: 100000,
			SolarSystemID: 30000142, SpectralClass: "K2 V", Temperature: 4465, TypeID: 3796,
		}, nil
	}

	handler := getUniverseStarHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, GetUniverseStarInput{StarID: 40000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.StarID != 40000001 || out.SolarSystemID != 30000142 || out.Name != "Jita - Star" || out.SpectralClass != "K2 V" {
		t.Errorf("out = %+v, want StarID=40000001 SolarSystemID=30000142 Name=Jita - Star SpectralClass=K2 V", out)
	}
}

func TestGetUniverseStarHandler_PropagatesError(t *testing.T) {
	orig := getUniverseStar
	t.Cleanup(func() { getUniverseStar = orig })
	wantErr := errors.New("upstream failure")
	getUniverseStar = func(ctx context.Context, sdeStore starCache, userAgent string, starID int64) (UniverseStar, error) {
		return UniverseStar{}, wantErr
	}

	handler := getUniverseStarHandler("v1", nil)
	_, _, err := handler(context.Background(), nil, GetUniverseStarInput{StarID: 40000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- get_universe_station ---

type fakeStationCache struct {
	station sde.Station
	ok      bool
	err     error
	putErr  error
	putCall *sde.Station
}

func (f *fakeStationCache) Station(ctx context.Context, userAgent string, id int64) (sde.Station, bool, error) {
	return f.station, f.ok, f.err
}
func (f *fakeStationCache) PutStation(s sde.Station) error {
	f.putCall = &s
	return f.putErr
}

func TestGetUniverseStation_CacheHit_ReturnsRecordWithoutESICall(t *testing.T) {
	cache := &fakeStationCache{station: sde.Station{ID: 60000001, Name: "Test Station", SystemID: 30000001}, ok: true}
	out, err := getUniverseStation(context.Background(), cache, "test-agent", 60000001)
	if err != nil {
		t.Fatalf("getUniverseStation returned error: %v", err)
	}
	if out.Name != "Test Station" || out.SystemID != 30000001 {
		t.Errorf("got %+v, want Name=\"Test Station\" SystemID=30000001", out)
	}
}

func TestGetUniverseStationHandler_MapsFields(t *testing.T) {
	orig := getUniverseStation
	t.Cleanup(func() { getUniverseStation = orig })
	owner := int64(1000035)
	services := []string{"market", "repair-facilities"}
	getUniverseStation = func(ctx context.Context, sdeStore stationCache, userAgent string, stationID int64) (UniverseStation, error) {
		return UniverseStation{
			StationID: 60003760, SystemID: 30000142, TypeID: 1529,
			Name: "Jita IV - Moon 4 - Caldari Navy Assembly Plant", Position: Position{X: 1, Y: 2, Z: 3},
			MaxDockableShipVolume: 50000000, OfficeRentalCost: 10000, Owner: &owner,
			ReprocessingEfficiency: 0.5, ReprocessingStationsTake: 0.05, Services: &services,
		}, nil
	}

	handler := getUniverseStationHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, GetUniverseStationInput{StationID: 60003760})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.StationID != 60003760 || out.SystemID != 30000142 || out.TypeID != 1529 {
		t.Errorf("out = %+v, want StationID=60003760 SystemID=30000142 TypeID=1529", out)
	}
	if out.Owner == nil || *out.Owner != 1000035 {
		t.Errorf("Owner = %v, want 1000035", out.Owner)
	}
	if out.Services == nil || len(*out.Services) != 2 {
		t.Errorf("Services = %v, want 2 entries", out.Services)
	}
}

func TestGetUniverseStationHandler_PropagatesError(t *testing.T) {
	orig := getUniverseStation
	t.Cleanup(func() { getUniverseStation = orig })
	wantErr := errors.New("upstream failure")
	getUniverseStation = func(ctx context.Context, sdeStore stationCache, userAgent string, stationID int64) (UniverseStation, error) {
		return UniverseStation{}, wantErr
	}

	handler := getUniverseStationHandler("v1", nil)
	_, _, err := handler(context.Background(), nil, GetUniverseStationInput{StationID: 60003760})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_universe_graphics / get_universe_graphic ---

type fakeGraphicCache struct {
	graphic   sde.Graphic
	graphicOK bool
	err       error
	ids       []int64
	idsErr    error
	putErr    error
	putCalls  []sde.Graphic
}

func (f *fakeGraphicCache) Graphic(ctx context.Context, userAgent string, id int64) (sde.Graphic, bool, error) {
	return f.graphic, f.graphicOK, f.err
}
func (f *fakeGraphicCache) PutGraphic(g sde.Graphic) error {
	f.putCalls = append(f.putCalls, g)
	return f.putErr
}
func (f *fakeGraphicCache) GraphicIDs(ctx context.Context, userAgent string) ([]int64, error) {
	return f.ids, f.idsErr
}

func TestListUniverseGraphics_CacheHit_ReturnsCachedIDs(t *testing.T) {
	cache := &fakeGraphicCache{ids: []int64{10, 38}}

	ids, err := listUniverseGraphics(context.Background(), cache, "test-agent")
	if err != nil {
		t.Fatalf("listUniverseGraphics returned error: %v", err)
	}
	if len(ids) != 2 || ids[0] != 10 {
		t.Errorf("ids = %v, want [10 38]", ids)
	}
}

func TestListUniverseGraphicsHandler_MapsFields(t *testing.T) {
	orig := listUniverseGraphics
	t.Cleanup(func() { listUniverseGraphics = orig })
	listUniverseGraphics = func(ctx context.Context, sdeStore graphicCache, userAgent string) ([]int64, error) {
		return []int64{1, 2, 3}, nil
	}

	handler := listUniverseGraphicsHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, ListUniverseGraphicsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.IDs) != 3 || out.IDs[0] != 1 {
		t.Errorf("out = %+v, want IDs=[1 2 3]", out)
	}
}

func TestListUniverseGraphicsHandler_PropagatesError(t *testing.T) {
	orig := listUniverseGraphics
	t.Cleanup(func() { listUniverseGraphics = orig })
	wantErr := errors.New("upstream failure")
	listUniverseGraphics = func(ctx context.Context, sdeStore graphicCache, userAgent string) ([]int64, error) {
		return nil, wantErr
	}

	handler := listUniverseGraphicsHandler("v1", nil)
	_, _, err := handler(context.Background(), nil, ListUniverseGraphicsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetUniverseGraphic_CacheHit_MapsFieldsIncludingSofDNA(t *testing.T) {
	iconFolder := "res:/dx9/model/ship/caldari/frigate/cf1/icons"
	factionName := "caldaribase"
	sofDNA := "cf1_t1:caldaribase:caldari"
	cache := &fakeGraphicCache{
		graphic:   sde.Graphic{ID: 38, IconFolder: &iconFolder, SofDNA: &sofDNA, SofFactionName: &factionName},
		graphicOK: true,
	}

	got, err := getUniverseGraphic(context.Background(), cache, "test-agent", 38)
	if err != nil {
		t.Fatalf("getUniverseGraphic returned error: %v", err)
	}
	if got.GraphicID != 38 || got.IconFolder == nil || *got.IconFolder != iconFolder {
		t.Errorf("got = %+v, want GraphicID=38 IconFolder=%q", got, iconFolder)
	}
	if got.SofDNA == nil || *got.SofDNA != sofDNA {
		t.Errorf("SofDNA = %v, want %q (from a cache hit, not only a cache miss)", got.SofDNA, sofDNA)
	}
	if got.CollisionFile != nil {
		t.Errorf("CollisionFile = %v, want nil on a cache hit (permanently unsourced, confirmed absent from esi-scrape too)", got.CollisionFile)
	}
	if len(cache.putCalls) != 0 {
		t.Errorf("PutGraphic called %d times on a cache hit, want 0", len(cache.putCalls))
	}
}

func TestGetUniverseGraphicHandler_MapsFields(t *testing.T) {
	orig := getUniverseGraphic
	t.Cleanup(func() { getUniverseGraphic = orig })
	graphicFile := "file.gr2"
	getUniverseGraphic = func(ctx context.Context, sdeStore graphicCache, userAgent string, graphicID int64) (UniverseGraphic, error) {
		return UniverseGraphic{GraphicID: 1234, GraphicFile: &graphicFile}, nil
	}

	handler := getUniverseGraphicHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, GetUniverseGraphicInput{GraphicID: 1234})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.GraphicID != 1234 || out.GraphicFile == nil || *out.GraphicFile != "file.gr2" {
		t.Errorf("out = %+v, want GraphicID=1234 GraphicFile=file.gr2", out)
	}
}

func TestGetUniverseGraphicHandler_PropagatesError(t *testing.T) {
	orig := getUniverseGraphic
	t.Cleanup(func() { getUniverseGraphic = orig })
	wantErr := errors.New("upstream failure")
	getUniverseGraphic = func(ctx context.Context, sdeStore graphicCache, userAgent string, graphicID int64) (UniverseGraphic, error) {
		return UniverseGraphic{}, wantErr
	}

	handler := getUniverseGraphicHandler("v1", nil)
	_, _, err := handler(context.Background(), nil, GetUniverseGraphicInput{GraphicID: 1234})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_universe_structures ---

func TestListUniverseStructuresHandler_MapsFields(t *testing.T) {
	orig := listUniverseStructures
	t.Cleanup(func() { listUniverseStructures = orig })
	var gotFilter string
	listUniverseStructures = func(ctx context.Context, userAgent string, filter string) ([]int64, error) {
		gotFilter = filter
		return []int64{1000000000001}, nil
	}

	handler := listUniverseStructuresHandler("v1")
	_, out, err := handler(context.Background(), nil, ListUniverseStructuresInput{Filter: "market"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if gotFilter != "market" {
		t.Errorf("captured filter = %q, want market", gotFilter)
	}
	if len(out.IDs) != 1 || out.IDs[0] != 1000000000001 {
		t.Errorf("out = %+v, want IDs=[1000000000001]", out)
	}
}

func TestListUniverseStructuresHandler_RejectsInvalidFilter(t *testing.T) {
	orig := listUniverseStructures
	t.Cleanup(func() { listUniverseStructures = orig })
	listUniverseStructures = func(ctx context.Context, userAgent string, filter string) ([]int64, error) {
		t.Fatal("listUniverseStructures should not be called with an invalid filter")
		return nil, nil
	}

	handler := listUniverseStructuresHandler("v1")
	_, _, err := handler(context.Background(), nil, ListUniverseStructuresInput{Filter: "bogus"})
	if err == nil {
		t.Fatal("expected error for invalid filter")
	}
}

func TestListUniverseStructuresHandler_PropagatesError(t *testing.T) {
	orig := listUniverseStructures
	t.Cleanup(func() { listUniverseStructures = orig })
	wantErr := errors.New("upstream failure")
	listUniverseStructures = func(ctx context.Context, userAgent string, filter string) ([]int64, error) {
		return nil, wantErr
	}

	handler := listUniverseStructuresHandler("v1")
	_, _, err := handler(context.Background(), nil, ListUniverseStructuresInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- get_universe_structure (private -- the only private tool in this batch) ---

func TestGetUniverseStructureHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := fakeUniverseStructureESI{universeStructureDetail: func(_ context.Context, structureID int64) (*esiapi.UniverseStructuresStructureIdGet, error) {
		return &esiapi.UniverseStructuresStructureIdGet{Name: "Test Citadel", OwnerId: 98000001, SolarSystemId: 30000142, Position: &esiapi.UniverseStructuresStructureIdGetPosition{X: 1, Y: 2, Z: 3}}, nil
	}}

	handler := getUniverseStructureHandler(openNarrow[universeStructureESI](fake), store)
	_, out, err := handler(context.Background(), nil, GetUniverseStructureInput{CharacterID: 7, StructureID: 1000000000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Name != "Test Citadel" || out.OwnerID != 98000001 || out.SolarSystemID != 30000142 {
		t.Errorf("out = %+v, want Name=Test Citadel OwnerID=98000001 SolarSystemID=30000142", out)
	}
	if out.Position == nil || out.Position.X != 1 {
		t.Errorf("Position = %v, want X=1", out.Position)
	}
}

func TestGetUniverseStructureHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	handler := getUniverseStructureHandler(noNarrow[universeStructureESI](t), store)
	_, _, err := handler(context.Background(), nil, GetUniverseStructureInput{CharacterID: 999, StructureID: 1000000000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetUniverseStructureHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	wantErr := errors.New("upstream failure")
	fake := fakeUniverseStructureESI{universeStructureDetail: func(_ context.Context, structureID int64) (*esiapi.UniverseStructuresStructureIdGet, error) {
		return &esiapi.UniverseStructuresStructureIdGet{}, wantErr
	}}

	handler := getUniverseStructureHandler(openNarrow[universeStructureESI](fake), store)
	_, _, err := handler(context.Background(), nil, GetUniverseStructureInput{CharacterID: 7, StructureID: 1000000000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_universe_system_jumps ---

func TestListUniverseSystemJumpsHandler_MapsFields(t *testing.T) {
	orig := listUniverseSystemJumps
	t.Cleanup(func() { listUniverseSystemJumps = orig })
	listUniverseSystemJumps = func(ctx context.Context, userAgent string) ([]UniverseSystemJumps, error) {
		return []UniverseSystemJumps{{SystemID: 30000142, ShipJumps: 10}}, nil
	}

	handler := listUniverseSystemJumpsHandler("v1")
	_, out, err := handler(context.Background(), nil, ListUniverseSystemJumpsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.SystemJumps) != 1 || out.SystemJumps[0].SystemID != 30000142 || out.SystemJumps[0].ShipJumps != 10 {
		t.Errorf("out = %+v, want one entry SystemID=30000142 ShipJumps=10", out)
	}
}

func TestListUniverseSystemJumpsHandler_PropagatesError(t *testing.T) {
	orig := listUniverseSystemJumps
	t.Cleanup(func() { listUniverseSystemJumps = orig })
	wantErr := errors.New("upstream failure")
	listUniverseSystemJumps = func(ctx context.Context, userAgent string) ([]UniverseSystemJumps, error) {
		return nil, wantErr
	}

	handler := listUniverseSystemJumpsHandler("v1")
	_, _, err := handler(context.Background(), nil, ListUniverseSystemJumpsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- list_universe_system_kills ---

func TestListUniverseSystemKillsHandler_MapsFields(t *testing.T) {
	fake := fakeUniverseSystemKillsESI{universeSystemKills: func(_ context.Context) ([]esiapi.UniverseSystemKillsGetInner, error) {
		return []esiapi.UniverseSystemKillsGetInner{{SystemId: 30000142, NpcKills: 5, PodKills: 1, ShipKills: 2}}, nil
	}}

	handler := listUniverseSystemKillsHandler(publicNarrow[universeSystemKillsESI](fake))
	_, out, err := handler(context.Background(), nil, ListUniverseSystemKillsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.SystemKills) != 1 || out.SystemKills[0].SystemID != 30000142 || out.SystemKills[0].NPCKills != 5 {
		t.Errorf("out = %+v, want one entry SystemID=30000142 NPCKills=5", out)
	}
}

func TestListUniverseSystemKillsHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakeUniverseSystemKillsESI{universeSystemKills: func(_ context.Context) ([]esiapi.UniverseSystemKillsGetInner, error) {
		return nil, wantErr
	}}

	handler := listUniverseSystemKillsHandler(publicNarrow[universeSystemKillsESI](fake))
	_, _, err := handler(context.Background(), nil, ListUniverseSystemKillsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- resolve_universe_ids ---

func TestResolveUniverseIDsHandler_MapsFields_FullFixture(t *testing.T) {
	orig := resolveUniverseIDs
	t.Cleanup(func() { resolveUniverseIDs = orig })
	charID := int64(12345)
	charName := "Test Pilot"
	sysID := int64(30000142)
	sysName := "Jita"
	resolveUniverseIDs = func(ctx context.Context, userAgent string, names []string) (ResolveUniverseIDsOutput, error) {
		return ResolveUniverseIDsOutput{
			Characters: []IDNamePair{{ID: &charID, Name: &charName}},
			Systems:    []IDNamePair{{ID: &sysID, Name: &sysName}},
		}, nil
	}

	handler := resolveUniverseIDsHandler("v1")
	_, out, err := handler(context.Background(), nil, ResolveUniverseIDsInput{Names: []string{"Test Pilot", "Jita"}})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Characters) != 1 || out.Characters[0].ID == nil || *out.Characters[0].ID != 12345 {
		t.Errorf("Characters = %+v, want one entry ID=12345", out.Characters)
	}
	if len(out.Systems) != 1 || out.Systems[0].Name == nil || *out.Systems[0].Name != "Jita" {
		t.Errorf("Systems = %+v, want one entry Name=Jita", out.Systems)
	}
	if len(out.Alliances) != 0 || len(out.Corporations) != 0 || len(out.Factions) != 0 || len(out.Regions) != 0 || len(out.Stations) != 0 || len(out.Agents) != 0 || len(out.Constellations) != 0 || len(out.InventoryTypes) != 0 {
		t.Errorf("out = %+v, want only Characters and Systems populated", out)
	}
}

func TestResolveUniverseIDsHandler_RejectsEmptyNames(t *testing.T) {
	orig := resolveUniverseIDs
	t.Cleanup(func() { resolveUniverseIDs = orig })
	resolveUniverseIDs = func(ctx context.Context, userAgent string, names []string) (ResolveUniverseIDsOutput, error) {
		t.Fatal("resolveUniverseIDs should not be called with empty names")
		return ResolveUniverseIDsOutput{}, nil
	}

	handler := resolveUniverseIDsHandler("v1")
	_, _, err := handler(context.Background(), nil, ResolveUniverseIDsInput{Names: []string{}})
	if err == nil {
		t.Fatal("expected error for empty names")
	}
}

func TestResolveUniverseIDsHandler_RejectsOverCapNames(t *testing.T) {
	orig := resolveUniverseIDs
	t.Cleanup(func() { resolveUniverseIDs = orig })
	resolveUniverseIDs = func(ctx context.Context, userAgent string, names []string) (ResolveUniverseIDsOutput, error) {
		t.Fatal("resolveUniverseIDs should not be called with over-cap names")
		return ResolveUniverseIDsOutput{}, nil
	}

	names := make([]string, 501)
	for i := range names {
		names[i] = "x"
	}
	handler := resolveUniverseIDsHandler("v1")
	_, _, err := handler(context.Background(), nil, ResolveUniverseIDsInput{Names: names})
	if err == nil {
		t.Fatal("expected error for over-cap names (501 > ESI's 500 max)")
	}
}

func TestResolveUniverseIDsHandler_PropagatesError(t *testing.T) {
	orig := resolveUniverseIDs
	t.Cleanup(func() { resolveUniverseIDs = orig })
	wantErr := errors.New("upstream failure")
	resolveUniverseIDs = func(ctx context.Context, userAgent string, names []string) (ResolveUniverseIDsOutput, error) {
		return ResolveUniverseIDsOutput{}, wantErr
	}

	handler := resolveUniverseIDsHandler("v1")
	_, _, err := handler(context.Background(), nil, ResolveUniverseIDsInput{Names: []string{"Jita"}})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- resolve_universe_names ---

func TestResolveUniverseNamesHandler_MapsFields(t *testing.T) {
	fake := fakeUniverseNamesESI{universeNamesByID: func(_ context.Context, ids []int64) ([]esiapi.UniverseNamesPostInner, error) {
		return []esiapi.UniverseNamesPostInner{{Category: "solar_system", Id: 30000142, Name: "Jita"}}, nil
	}}

	handler := resolveUniverseNamesHandler(publicNarrow[universeNamesESI](fake))
	_, out, err := handler(context.Background(), nil, ResolveUniverseNamesInput{IDs: []int64{30000142}})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Names) != 1 || out.Names[0].Category != "solar_system" || out.Names[0].ID != 30000142 || out.Names[0].Name != "Jita" {
		t.Errorf("out = %+v, want one entry Category=solar_system ID=30000142 Name=Jita", out)
	}
}

func TestResolveUniverseNamesHandler_RejectsEmptyIDs(t *testing.T) {
	fake := fakeUniverseNamesESI{universeNamesByID: func(_ context.Context, ids []int64) ([]esiapi.UniverseNamesPostInner, error) {
		t.Fatal("resolveUniverseNames should not be called with empty ids")
		return nil, nil
	}}

	handler := resolveUniverseNamesHandler(publicNarrow[universeNamesESI](fake))
	_, _, err := handler(context.Background(), nil, ResolveUniverseNamesInput{IDs: []int64{}})
	if err == nil {
		t.Fatal("expected error for empty ids")
	}
}

func TestResolveUniverseNamesHandler_RejectsOverCapIDs(t *testing.T) {
	fake := fakeUniverseNamesESI{universeNamesByID: func(_ context.Context, ids []int64) ([]esiapi.UniverseNamesPostInner, error) {
		t.Fatal("resolveUniverseNames should not be called with over-cap ids")
		return nil, nil
	}}

	ids := make([]int64, 1001)
	handler := resolveUniverseNamesHandler(publicNarrow[universeNamesESI](fake))
	_, _, err := handler(context.Background(), nil, ResolveUniverseNamesInput{IDs: ids})
	if err == nil {
		t.Fatal("expected error for over-cap ids (1001 > ESI's 1000 max)")
	}
}

func TestResolveUniverseNamesHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakeUniverseNamesESI{universeNamesByID: func(_ context.Context, ids []int64) ([]esiapi.UniverseNamesPostInner, error) {
		return nil, wantErr
	}}

	handler := resolveUniverseNamesHandler(publicNarrow[universeNamesESI](fake))
	_, _, err := handler(context.Background(), nil, ResolveUniverseNamesInput{IDs: []int64{30000142}})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- Cache-miss/ESI-fallback/write-back coverage for all 24 sde/-backed
// Universe tools ---
//
// Every test above that swaps a getUniverseX/listUniverseX var only ever
// exercises the cache-HIT branch (a pre-populated fake). These tests call
// the real, unswapped functions with a cache miss (ok=false / empty IDs)
// and a fake ESI server via newPublicESIClient, proving: (1) the ESI
// response maps onto the sde.* struct correctly, (2) Put* is called with
// that value, and (3) a Put* failure is logged-and-swallowed rather than
// failing the tool call.

func TestGetUniverseCategory_CacheMiss_FetchesFromESIAndWritesBack(t *testing.T) {
	srv := fakeESIJSONServer(t, `{"category_id":6,"groups":[25,26],"name":"Ship","published":true}`, "")
	useFakeESIClient(t, srv)
	cache := &fakeCategoryCache{categoryOK: false, putErr: errors.New("put failed")}

	got, err := getUniverseCategory(context.Background(), cache, "test-agent", 6)
	if err != nil {
		t.Fatalf("getUniverseCategory returned error: %v", err)
	}
	if got.CategoryID != 6 || got.Name != "Ship" || !got.Published || len(got.GroupIDs) != 2 {
		t.Errorf("got = %+v, want CategoryID=6 Name=Ship Published=true GroupIDs len=2", got)
	}
	if len(cache.putCalls) != 1 || cache.putCalls[0].ID != 6 {
		t.Errorf("PutCategory calls = %+v, want one call with ID=6", cache.putCalls)
	}
}

func TestListUniverseCategories_CacheMiss_FetchesFromESI(t *testing.T) {
	srv := fakeESIJSONServer(t, `[6,7,8]`, "")
	useFakeESIClient(t, srv)
	cache := &fakeCategoryCache{ids: nil}

	got, err := listUniverseCategories(context.Background(), cache, "test-agent")
	if err != nil {
		t.Fatalf("listUniverseCategories returned error: %v", err)
	}
	if len(got) != 3 || got[0] != 6 {
		t.Errorf("got = %v, want [6 7 8]", got)
	}
}

func TestGetUniverseGroup_CacheMiss_FetchesFromESIAndWritesBack(t *testing.T) {
	srv := fakeESIJSONServer(t, `{"category_id":6,"group_id":25,"name":"Frigate","published":true,"types":[587,588]}`, "")
	useFakeESIClient(t, srv)
	cache := &fakeGroupCache{groupOK: false, putErr: errors.New("put failed")}

	got, err := getUniverseGroup(context.Background(), cache, "test-agent", 25)
	if err != nil {
		t.Fatalf("getUniverseGroup returned error: %v", err)
	}
	if got.GroupID != 25 || got.CategoryID != 6 || got.Name != "Frigate" || len(got.TypeIDs) != 2 {
		t.Errorf("got = %+v, want GroupID=25 CategoryID=6 Name=Frigate TypeIDs len=2", got)
	}
	if len(cache.putCalls) != 1 || cache.putCalls[0].ID != 25 {
		t.Errorf("PutGroup calls = %+v, want one call with ID=25", cache.putCalls)
	}
}

func TestListUniverseGroups_CacheMiss_FetchesFromESIWithPagination(t *testing.T) {
	srv := fakeESIJSONServer(t, `[25,26,27]`, "3")
	useFakeESIClient(t, srv)
	cache := &fakeGroupCache{ids: nil}

	got, totalPages, err := listUniverseGroups(context.Background(), cache, "test-agent", 2)
	if err != nil {
		t.Fatalf("listUniverseGroups returned error: %v", err)
	}
	if len(got) != 3 || got[0] != 25 || totalPages != 3 {
		t.Errorf("got = %v, totalPages = %d, want [25 26 27] totalPages=3", got, totalPages)
	}
}

func TestGetUniverseType_CacheMiss_FetchesFromESIAndWritesBack(t *testing.T) {
	srv := fakeESIJSONServer(t, `{
		"capacity": 400, "description": "desc", "graphic_id": 1234, "group_id": 25,
		"icon_id": 100, "market_group_id": 5, "mass": 1200000, "name": "Rifter",
		"packaged_volume": 2500, "portion_size": 1, "published": true, "radius": 40,
		"type_id": 587, "volume": 27289,
		"dogma_attributes": [{"attribute_id": 9, "value": 2500}],
		"dogma_effects": [{"effect_id": 1, "is_default": true}]
	}`, "")
	useFakeESIClient(t, srv)
	cache := &fakeTypeCache{typeOK: false, putErr: errors.New("put failed")}

	got, err := getUniverseType(context.Background(), cache, "test-agent", 587)
	if err != nil {
		t.Fatalf("getUniverseType returned error: %v", err)
	}
	if got.TypeID != 587 || got.Name != "Rifter" || got.GroupID != 25 || !got.Published {
		t.Errorf("got = %+v, want TypeID=587 Name=Rifter GroupID=25 Published=true", got)
	}
	if got.PackagedVolume == nil || *got.PackagedVolume != 2500 {
		t.Errorf("PackagedVolume = %v, want 2500 (fresh off ESI on a cache miss)", got.PackagedVolume)
	}
	if len(got.DogmaAttributes) != 1 || got.DogmaAttributes[0].AttributeID != 9 {
		t.Errorf("DogmaAttributes = %+v, want one entry AttributeID=9", got.DogmaAttributes)
	}
	if len(cache.putCalls) != 1 || cache.putCalls[0].ID != 587 {
		t.Errorf("PutType calls = %+v, want one call with ID=587", cache.putCalls)
	}
}

func TestListUniverseTypes_CacheMiss_FetchesFromESIWithPagination(t *testing.T) {
	srv := fakeESIJSONServer(t, `[587,588]`, "5")
	useFakeESIClient(t, srv)
	cache := &fakeTypeCache{ids: nil}

	got, totalPages, err := listUniverseTypes(context.Background(), cache, "test-agent", 1)
	if err != nil {
		t.Fatalf("listUniverseTypes returned error: %v", err)
	}
	if len(got) != 2 || totalPages != 5 {
		t.Errorf("got = %v, totalPages = %d, want [587 588] totalPages=5", got, totalPages)
	}
}

func TestGetUniverseRegion_CacheMiss_FetchesFromESIAndWritesBack(t *testing.T) {
	srv := fakeESIJSONServer(t, `{"constellations":[20000020,20000021],"description":"desc","name":"The Forge","region_id":10000002}`, "")
	useFakeESIClient(t, srv)
	cache := &fakeRegionCache{regionOK: false, putErr: errors.New("put failed")}

	got, err := getUniverseRegion(context.Background(), cache, "test-agent", 10000002)
	if err != nil {
		t.Fatalf("getUniverseRegion returned error: %v", err)
	}
	if got.RegionID != 10000002 || got.Name != "The Forge" || len(got.ConstellationIDs) != 2 {
		t.Errorf("got = %+v, want RegionID=10000002 Name=The Forge ConstellationIDs len=2", got)
	}
	if got.Description == nil || *got.Description != "desc" {
		t.Errorf("Description = %v, want desc", got.Description)
	}
	if len(cache.putCalls) != 1 || cache.putCalls[0].ID != 10000002 {
		t.Errorf("PutRegion calls = %+v, want one call with ID=10000002", cache.putCalls)
	}
}

func TestListUniverseRegions_CacheMiss_FetchesFromESI(t *testing.T) {
	srv := fakeESIJSONServer(t, `[10000002,10000043]`, "")
	useFakeESIClient(t, srv)
	cache := &fakeRegionCache{ids: nil}

	got, err := listUniverseRegions(context.Background(), cache, "test-agent")
	if err != nil {
		t.Fatalf("listUniverseRegions returned error: %v", err)
	}
	if len(got) != 2 || got[0] != 10000002 {
		t.Errorf("got = %v, want [10000002 10000043]", got)
	}
}

func TestGetUniverseConstellation_CacheMiss_FetchesFromESIAndWritesBack(t *testing.T) {
	srv := fakeESIJSONServer(t, `{"constellation_id":20000020,"name":"Kimotoro","position":{"x":1,"y":2,"z":3},"region_id":10000002,"systems":[30000142,30000144]}`, "")
	useFakeESIClient(t, srv)
	cache := &fakeConstellationCache{constellationOK: false, putErr: errors.New("put failed")}

	got, err := getUniverseConstellation(context.Background(), cache, "test-agent", 20000020)
	if err != nil {
		t.Fatalf("getUniverseConstellation returned error: %v", err)
	}
	if got.ConstellationID != 20000020 || got.Name != "Kimotoro" || got.RegionID != 10000002 || len(got.SystemIDs) != 2 {
		t.Errorf("got = %+v, want ConstellationID=20000020 Name=Kimotoro RegionID=10000002 SystemIDs len=2", got)
	}
	if len(cache.putCalls) != 1 || cache.putCalls[0].ID != 20000020 {
		t.Errorf("PutConstellation calls = %+v, want one call with ID=20000020", cache.putCalls)
	}
}

func TestListUniverseConstellations_CacheMiss_FetchesFromESI(t *testing.T) {
	srv := fakeESIJSONServer(t, `[20000020,20000021]`, "")
	useFakeESIClient(t, srv)
	cache := &fakeConstellationCache{ids: nil}

	got, err := listUniverseConstellations(context.Background(), cache, "test-agent")
	if err != nil {
		t.Fatalf("listUniverseConstellations returned error: %v", err)
	}
	if len(got) != 2 || got[0] != 20000020 {
		t.Errorf("got = %v, want [20000020 20000021]", got)
	}
}

func TestGetUniverseSystem_CacheMiss_FetchesFromESIAndWritesBack(t *testing.T) {
	srv := fakeESIJSONServer(t, `{
		"constellation_id": 20000020, "name": "Jita", "position": {"x":1,"y":2,"z":3},
		"security_class": "B", "security_status": 0.9459, "star_id": 40000001,
		"stargates": [50000342], "stations": [60003760], "system_id": 30000142,
		"planets": [{"asteroid_belts": [40000002], "moons": [40000003], "planet_id": 40000004}]
	}`, "")
	useFakeESIClient(t, srv)
	cache := &fakeSystemCache{systemOK: false, putErr: errors.New("put failed")}

	got, err := getUniverseSystem(context.Background(), cache, "test-agent", 30000142)
	if err != nil {
		t.Fatalf("getUniverseSystem returned error: %v", err)
	}
	if got.SystemID != 30000142 || got.Name != "Jita" || got.ConstellationID != 20000020 {
		t.Errorf("got = %+v, want SystemID=30000142 Name=Jita ConstellationID=20000020", got)
	}
	if len(got.Planets) != 1 || got.Planets[0].PlanetID != 40000004 || len(got.Planets[0].AsteroidBeltIDs) != 1 || len(got.Planets[0].MoonIDs) != 1 {
		t.Errorf("Planets = %+v, want one entry PlanetID=40000004 with one belt and one moon (fresh off ESI on a cache miss)", got.Planets)
	}
	if len(cache.putCalls) != 1 || cache.putCalls[0].ID != 30000142 {
		t.Errorf("PutSystem calls = %+v, want one call with ID=30000142", cache.putCalls)
	}
}

func TestListUniverseSystems_CacheMiss_FetchesFromESI(t *testing.T) {
	srv := fakeESIJSONServer(t, `[30000142,30000144]`, "")
	useFakeESIClient(t, srv)
	cache := &fakeSystemCache{ids: nil}

	got, err := listUniverseSystems(context.Background(), cache, "test-agent")
	if err != nil {
		t.Fatalf("listUniverseSystems returned error: %v", err)
	}
	if len(got) != 2 || got[0] != 30000142 {
		t.Errorf("got = %v, want [30000142 30000144]", got)
	}
}

func TestGetUniverseAsteroidBelt_CacheMiss_FetchesFromESIAndWritesBack(t *testing.T) {
	srv := fakeESIJSONServer(t, `{"name":"Jita IV - Asteroid Belt 1","position":{"x":1,"y":2,"z":3},"system_id":30000142}`, "")
	useFakeESIClient(t, srv)
	cache := &fakeAsteroidBeltCache{ok: false, putErr: errors.New("put failed")}

	got, err := getUniverseAsteroidBelt(context.Background(), cache, "test-agent", 40000002)
	if err != nil {
		t.Fatalf("getUniverseAsteroidBelt returned error: %v", err)
	}
	if got.Name != "Jita IV - Asteroid Belt 1" || got.SystemID != 30000142 {
		t.Errorf("got = %+v, want Name=Jita IV - Asteroid Belt 1 SystemID=30000142", got)
	}
	if cache.putCall == nil || cache.putCall.ID != 40000002 {
		t.Errorf("PutAsteroidBelt call = %v, want one call with ID=40000002", cache.putCall)
	}
}

func TestGetUniverseMoon_CacheMiss_FetchesFromESIAndWritesBack(t *testing.T) {
	srv := fakeESIJSONServer(t, `{"moon_id":40000003,"name":"Jita IV - Moon 1","position":{"x":1,"y":2,"z":3},"system_id":30000142}`, "")
	useFakeESIClient(t, srv)
	cache := &fakeMoonCache{ok: false, putErr: errors.New("put failed")}

	got, err := getUniverseMoon(context.Background(), cache, "test-agent", 40000003)
	if err != nil {
		t.Fatalf("getUniverseMoon returned error: %v", err)
	}
	if got.MoonID != 40000003 || got.Name != "Jita IV - Moon 1" || got.SystemID != 30000142 {
		t.Errorf("got = %+v, want MoonID=40000003 Name=Jita IV - Moon 1 SystemID=30000142", got)
	}
	if cache.putCall == nil || cache.putCall.ID != 40000003 {
		t.Errorf("PutMoon call = %v, want one call with ID=40000003", cache.putCall)
	}
}

func TestGetUniversePlanet_CacheMiss_FetchesFromESIAndWritesBack(t *testing.T) {
	srv := fakeESIJSONServer(t, `{"name":"Jita IV","planet_id":40000004,"position":{"x":1,"y":2,"z":3},"system_id":30000142,"type_id":2016}`, "")
	useFakeESIClient(t, srv)
	cache := &fakePlanetCache{ok: false, putErr: errors.New("put failed")}

	got, err := getUniversePlanet(context.Background(), cache, "test-agent", 40000004)
	if err != nil {
		t.Fatalf("getUniversePlanet returned error: %v", err)
	}
	if got.PlanetID != 40000004 || got.Name != "Jita IV" || got.SystemID != 30000142 || got.TypeID != 2016 {
		t.Errorf("got = %+v, want PlanetID=40000004 Name=Jita IV SystemID=30000142 TypeID=2016", got)
	}
	if cache.putCall == nil || cache.putCall.ID != 40000004 {
		t.Errorf("PutPlanet call = %v, want one call with ID=40000004", cache.putCall)
	}
}

func TestGetUniverseStargate_CacheMiss_FetchesFromESIAndWritesBack(t *testing.T) {
	srv := fakeESIJSONServer(t, `{
		"destination": {"stargate_id": 50000343, "system_id": 30000144},
		"name": "Stargate (Perimeter)", "position": {"x":1,"y":2,"z":3},
		"stargate_id": 50000342, "system_id": 30000142, "type_id": 16
	}`, "")
	useFakeESIClient(t, srv)
	cache := &fakeStargateCache{ok: false, putErr: errors.New("put failed")}

	got, err := getUniverseStargate(context.Background(), cache, "test-agent", 50000342)
	if err != nil {
		t.Fatalf("getUniverseStargate returned error: %v", err)
	}
	if got.StargateID != 50000342 || got.SystemID != 30000142 || got.TypeID != 16 {
		t.Errorf("got = %+v, want StargateID=50000342 SystemID=30000142 TypeID=16", got)
	}
	if got.DestinationStargateID != 50000343 || got.DestinationSystemID != 30000144 {
		t.Errorf("Destination = %d/%d, want 50000343/30000144", got.DestinationStargateID, got.DestinationSystemID)
	}
	if cache.putCall == nil || cache.putCall.ID != 50000342 {
		t.Errorf("PutStargate call = %v, want one call with ID=50000342", cache.putCall)
	}
}

func TestGetUniverseStar_CacheMiss_FetchesFromESIAndWritesBack(t *testing.T) {
	srv := fakeESIJSONServer(t, `{
		"age": 15000000000, "luminosity": 0.02, "name": "Jita - Star",
		"radius": 100000, "solar_system_id": 30000142, "spectral_class": "K2 V",
		"temperature": 4465, "type_id": 3796
	}`, "")
	useFakeESIClient(t, srv)
	cache := &fakeStarCache{ok: false, putErr: errors.New("put failed")}

	got, err := getUniverseStar(context.Background(), cache, "test-agent", 40000001)
	if err != nil {
		t.Fatalf("getUniverseStar returned error: %v", err)
	}
	if got.SolarSystemID != 30000142 || got.Name != "Jita - Star" || got.SpectralClass != "K2 V" || got.TypeID != 3796 {
		t.Errorf("got = %+v, want SolarSystemID=30000142 Name=Jita - Star SpectralClass=K2 V TypeID=3796", got)
	}
	if cache.putCall == nil || cache.putCall.ID != 40000001 {
		t.Errorf("PutStar call = %v, want one call with ID=40000001", cache.putCall)
	}
}

func TestGetUniverseStation_CacheMiss_FetchesFromESIAndWritesBack(t *testing.T) {
	srv := fakeESIJSONServer(t, `{
		"max_dockable_ship_volume": 50000000, "name": "Jita IV - Moon 4 - Caldari Navy Assembly Plant",
		"office_rental_cost": 10000, "owner": 1000035, "position": {"x":1,"y":2,"z":3},
		"race_id": 1, "reprocessing_efficiency": 0.5, "reprocessing_stations_take": 0.05,
		"services": ["market","repair-facilities"], "station_id": 60003760, "system_id": 30000142, "type_id": 1529
	}`, "")
	useFakeESIClient(t, srv)
	cache := &fakeStationCache{ok: false, putErr: errors.New("put failed")}

	got, err := getUniverseStation(context.Background(), cache, "test-agent", 60003760)
	if err != nil {
		t.Fatalf("getUniverseStation returned error: %v", err)
	}
	if got.StationID != 60003760 || got.SystemID != 30000142 || got.TypeID != 1529 {
		t.Errorf("got = %+v, want StationID=60003760 SystemID=30000142 TypeID=1529", got)
	}
	if got.Owner == nil || *got.Owner != 1000035 {
		t.Errorf("Owner = %v, want 1000035", got.Owner)
	}
	if got.Services == nil || len(*got.Services) != 2 {
		t.Errorf("Services = %v, want 2 entries", got.Services)
	}
	if cache.putCall == nil || cache.putCall.ID != 60003760 {
		t.Errorf("PutStation call = %v, want one call with ID=60003760", cache.putCall)
	}
}

func TestGetUniverseGraphic_CacheMiss_FetchesFromESIAndWritesBack(t *testing.T) {
	srv := fakeESIJSONServer(t, `{"graphic_file":"file.gr2","graphic_id":1234,"sof_hull_name":"caldari_frigate"}`, "")
	useFakeESIClient(t, srv)
	cache := &fakeGraphicCache{graphicOK: false, putErr: errors.New("put failed")}

	got, err := getUniverseGraphic(context.Background(), cache, "test-agent", 1234)
	if err != nil {
		t.Fatalf("getUniverseGraphic returned error: %v", err)
	}
	if got.GraphicID != 1234 || got.GraphicFile == nil || *got.GraphicFile != "file.gr2" {
		t.Errorf("got = %+v, want GraphicID=1234 GraphicFile=file.gr2", got)
	}
	if len(cache.putCalls) != 1 || cache.putCalls[0].ID != 1234 {
		t.Errorf("PutGraphic calls = %+v, want one call with ID=1234", cache.putCalls)
	}
}

func TestListUniverseGraphics_CacheMiss_FetchesFromESI(t *testing.T) {
	srv := fakeESIJSONServer(t, `[1,2,3]`, "")
	useFakeESIClient(t, srv)
	cache := &fakeGraphicCache{ids: nil}

	got, err := listUniverseGraphics(context.Background(), cache, "test-agent")
	if err != nil {
		t.Fatalf("listUniverseGraphics returned error: %v", err)
	}
	if len(got) != 3 || got[0] != 1 {
		t.Errorf("got = %v, want [1 2 3]", got)
	}
}

func TestListUniverseAncestries_CacheMiss_FetchesFromESI(t *testing.T) {
	srv := fakeESIJSONServer(t, `[{"bloodline_id":1,"description":"desc","icon_id":100,"id":11,"name":"Tribal","short_description":"short"}]`, "")
	useFakeESIClient(t, srv)
	cache := &fakeAncestryCache{ancestries: nil}

	got, err := listUniverseAncestries(context.Background(), cache, "test-agent")
	if err != nil {
		t.Fatalf("listUniverseAncestries returned error: %v", err)
	}
	if len(got) != 1 || got[0].ID != 11 || got[0].BloodlineID != 1 || got[0].Name != "Tribal" {
		t.Errorf("got = %+v, want one entry ID=11 BloodlineID=1 Name=Tribal", got)
	}
}

func TestListUniverseBloodlines_CacheMiss_FetchesFromESI(t *testing.T) {
	srv := fakeESIJSONServer(t, `[{"bloodline_id":1,"charisma":6,"corporation_id":1000166,"description":"desc","intelligence":7,"memory":5,"name":"Deteis","perception":5,"race_id":1,"ship_type_id":601,"willpower":5}]`, "")
	useFakeESIClient(t, srv)
	cache := &fakeBloodlineCache{bloodlines: nil}

	got, err := listUniverseBloodlines(context.Background(), cache, "test-agent")
	if err != nil {
		t.Fatalf("listUniverseBloodlines returned error: %v", err)
	}
	if len(got) != 1 || got[0].ID != 1 || got[0].Name != "Deteis" || got[0].RaceID != 1 {
		t.Errorf("got = %+v, want one entry ID=1 Name=Deteis RaceID=1", got)
	}
}

func TestListUniverseRaces_CacheMiss_FetchesFromESI(t *testing.T) {
	srv := fakeESIJSONServer(t, `[{"alliance_id":500001,"description":"desc","name":"Caldari","race_id":1}]`, "")
	useFakeESIClient(t, srv)
	cache := &fakeRaceCache{races: nil}

	got, err := listUniverseRaces(context.Background(), cache, "test-agent")
	if err != nil {
		t.Fatalf("listUniverseRaces returned error: %v", err)
	}
	if len(got) != 1 || got[0].RaceID != 1 || got[0].Name != "Caldari" {
		t.Errorf("got = %+v, want one entry RaceID=1 Name=Caldari", got)
	}
	if got[0].AllianceID == nil || *got[0].AllianceID != 500001 {
		t.Errorf("AllianceID = %v, want 500001 (ESI's schema always returns a real value)", got[0].AllianceID)
	}
}

func TestListUniverseFactions_CacheMiss_FetchesFromESI(t *testing.T) {
	srv := fakeESIJSONServer(t, `[{"corporation_id":1000035,"description":"desc","faction_id":500001,"is_unique":true,"militia_corporation_id":1000180,"name":"Caldari State","size_factor":5,"solar_system_id":30000144,"station_count":100,"station_system_count":50}]`, "")
	useFakeESIClient(t, srv)
	cache := &fakeFactionCache{factions: nil}

	got, err := listUniverseFactions(context.Background(), cache, "test-agent")
	if err != nil {
		t.Fatalf("listUniverseFactions returned error: %v", err)
	}
	if len(got) != 1 || got[0].FactionID != 500001 || got[0].IsUnique == nil || !*got[0].IsUnique || got[0].Name != "Caldari State" {
		t.Errorf("got = %+v, want one entry FactionID=500001 IsUnique=true Name=Caldari State", got)
	}
	if got[0].CorporationID == nil || *got[0].CorporationID != 1000035 {
		t.Errorf("CorporationID = %v, want 1000035", got[0].CorporationID)
	}
}

// fakeUniverseStructureESI fakes universeStructureESI; a nil universeStructureDetail fails as an unfaked call.
type fakeUniverseStructureESI struct {
	universeStructureDetail func(ctx context.Context, structureID int64) (*esiapi.UniverseStructuresStructureIdGet, error)
}

func (f fakeUniverseStructureESI) UniverseStructureDetail(ctx context.Context, structureID int64) (*esiapi.UniverseStructuresStructureIdGet, error) {
	if f.universeStructureDetail == nil {
		return nil, errNotFaked
	}
	return f.universeStructureDetail(ctx, structureID)
}

// fakeUniverseSystemKillsESI fakes universeSystemKillsESI; a nil universeSystemKills fails as an unfaked call.
type fakeUniverseSystemKillsESI struct {
	universeSystemKills func(ctx context.Context) ([]esiapi.UniverseSystemKillsGetInner, error)
}

func (f fakeUniverseSystemKillsESI) UniverseSystemKills(ctx context.Context) ([]esiapi.UniverseSystemKillsGetInner, error) {
	if f.universeSystemKills == nil {
		return nil, errNotFaked
	}
	return f.universeSystemKills(ctx)
}

// fakeUniverseNamesESI fakes universeNamesESI; a nil universeNamesByID fails as an unfaked call.
type fakeUniverseNamesESI struct {
	universeNamesByID func(ctx context.Context, ids []int64) ([]esiapi.UniverseNamesPostInner, error)
}

func (f fakeUniverseNamesESI) UniverseNamesByID(ctx context.Context, ids []int64) ([]esiapi.UniverseNamesPostInner, error) {
	if f.universeNamesByID == nil {
		return nil, errNotFaked
	}
	return f.universeNamesByID(ctx, ids)
}

// Each handler reports a client that can't be opened, wrapping the cause.
func TestUniverseStructureHandlers_PropagateOpenError(t *testing.T) {
	store := characterStore(t, config.Character{ID: 7, Name: "Test Pilot"})
	wantErr := errors.New("no refresh token")
	for name, call := range map[string]func() error{
		"get_universe_structure": func() error {
			_, _, err := getUniverseStructureHandler(failNarrow[universeStructureESI](wantErr), store)(context.Background(), nil, GetUniverseStructureInput{CharacterID: 7, StructureID: 1000000000001})
			return err
		},
	} {
		if err := call(); !errors.Is(err, wantErr) {
			t.Errorf("%s: err = %v, want wrapping %v", name, err, wantErr)
		}
	}
}
