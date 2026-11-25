package mcp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// errNotFaked is what a fake ESI call no test set up returns, the way a
// real call fails with no network or login.
var errNotFaked = errors.New("ESI call not faked")

// fakeESI is the ESI client the planner family tests open, built from each
// tool file's narrow fake: it has every call any planner-family handler
// makes. A narrow fake left zero fails its calls with errNotFaked.
type fakeESI struct {
	fakeAllianceContactsESI
	fakeAssetNamesESI
	fakeAssetsESI
	fakeCharacterAffiliationsESI
	fakeCharacterBlueprintsESI
	fakeCharacterClonesESI
	fakeCharacterContactsESI
	fakeCharacterFatigueESI
	fakeCharacterImplantsESI
	fakeCharacterIndustryJobsESI
	fakeCharacterLocationESI
	fakeCharacterOnlineESI
	fakeCharacterShipESI
	fakeCharacterSkillsESI
	fakeCorporationAssetNamesESI
	fakeCorporationAssetsESI
	fakeCorporationBlueprintsESI
	fakeCorporationContactsESI
	fakeCorporationIndustryJobsESI
	fakeIncursionsESI
	fakeIndustrySystemsESI
	fakeMarketOrdersESI
	fakeMarketPricesESI
	fakePublicContractItemsESI
	fakePublicContractsESI
	fakeSovereigntySystemsESI
	fakeStructureMarketOrdersESI
	fakeUniverseNamesESI
	fakeUniverseStructureESI
	fakeUniverseSystemKillsESI
}

// plannerOpen is the planner's opener for authenticated clients.
type plannerOpen = func(ctx context.Context, characterID int32) (plannerClient, error)

// openFakeESI opens f for every character.
func openFakeESI(f fakeESI) plannerOpen {
	return func(context.Context, int32) (plannerClient, error) { return f, nil }
}

// openFakeESIFor opens the fake for each character that f returns.
func openFakeESIFor(f func(characterID int32) fakeESI) plannerOpen {
	return func(_ context.Context, characterID int32) (plannerClient, error) { return f(characterID), nil }
}

// openFakePublicESI opens f for public calls.
func openFakePublicESI(f fakeESI) func() plannerPublicClient {
	return func() plannerPublicClient { return f }
}

// failOpenESI is an opener whose clients can't be built.
func failOpenESI(err error) plannerOpen {
	return func(context.Context, int32) (plannerClient, error) { return nil, err }
}

// fakeOpener opens the fakeESI values p's opener opens, for handlers whose
// client interface is wider than the planner's.
func fakeOpener(p *planner) func(context.Context, int32) (fakeESI, error) {
	return func(ctx context.Context, characterID int32) (fakeESI, error) {
		c, err := p.openESI(ctx, characterID)
		if err != nil {
			return fakeESI{}, err
		}
		return c.(fakeESI), nil
	}
}

// openFakeFor opens the fake f returns for each character, as the fake
// itself rather than a planner interface.
func openFakeFor(f func(characterID int32) fakeESI) func(context.Context, int32) (fakeESI, error) {
	return func(_ context.Context, characterID int32) (fakeESI, error) { return f(characterID), nil }
}

// openNarrow opens c for every character, the way narrowESI hands a tool
// file its own small interface.
func openNarrow[C any](c C) func(context.Context, int32) (C, error) {
	return func(context.Context, int32) (C, error) { return c, nil }
}

// failNarrow is a narrow opener whose clients can't be built.
func failNarrow[C any](err error) func(context.Context, int32) (C, error) {
	return func(context.Context, int32) (C, error) {
		var zero C
		return zero, err
	}
}

// noNarrow is a narrow opener a test expects never to be called, such as
// for a character that isn't logged in.
func noNarrow[C any](t interface{ Fatal(...any) }) func(context.Context, int32) (C, error) {
	return func(context.Context, int32) (C, error) {
		t.Fatal("no ESI client should be opened")
		var zero C
		return zero, nil
	}
}

// publicNarrow opens c for public calls.
func publicNarrow[C any](c C) func() C {
	return func() C { return c }
}

// addESILayer makes p's characters' clients go through layer too: each
// opened client is the one before with layer's edits, so tests can stub
// skills, locations and clones apart. p's opener must open fakeESI values.
func addESILayer(p *planner, layer func(characterID int32, f *fakeESI)) {
	prev := p.openESI
	p.openESI = func(ctx context.Context, characterID int32) (plannerClient, error) {
		c, err := prev(ctx, characterID)
		if err != nil {
			return nil, err
		}
		f := c.(fakeESI)
		layer(characterID, &f)
		return f, nil
	}
}

// addPublicESILayer does the same for p's public client.
func addPublicESILayer(p *planner, layer func(f *fakeESI)) {
	prev := p.openPublicESI
	p.openPublicESI = func() plannerPublicClient {
		f := prev().(fakeESI)
		layer(&f)
		return f
	}
}

// stubClones answers each character's clones with clones(characterID),
// written the way get_character_clones reports them.
func stubClones(p *planner, clones func(characterID int32) (GetCharacterClonesOutput, error)) {
	addESILayer(p, func(id int32, f *fakeESI) {
		f.characterClones = func(context.Context) (*esiapi.CharactersCharacterIdClonesGet, error) {
			c, err := clones(id)
			if err != nil {
				return nil, err
			}
			out := &esiapi.CharactersCharacterIdClonesGet{LastCloneJumpDate: c.LastCloneJumpDate, LastStationChangeDate: c.LastStationChangeDate, JumpClones: []esiapi.CharactersCharacterIdClonesGetJumpClonesInner{}}
			if c.HomeLocationID != nil || c.HomeLocationType != nil {
				out.HomeLocation = &esiapi.CharactersCharacterIdClonesGetHomeLocation{LocationId: c.HomeLocationID, LocationType: c.HomeLocationType}
			}
			for _, jc := range c.JumpClones {
				out.JumpClones = append(out.JumpClones, esiapi.CharactersCharacterIdClonesGetJumpClonesInner{JumpCloneId: jc.JumpCloneID, LocationId: jc.LocationID, LocationType: jc.LocationType, Name: jc.Name, Implants: jc.Implants})
			}
			return out, nil
		}
	})
}

// stubImplants answers each character's current implants with
// implants(characterID).
func stubImplants(p *planner, implants func(characterID int32) ([]int64, error)) {
	addESILayer(p, func(id int32, f *fakeESI) {
		f.characterImplants = func(context.Context) ([]int64, error) { return implants(id) }
	})
}

// stubAssets answers each character's assets, one page, with
// assets(characterID), written the way list_assets reports them.
func stubAssets(p *planner, assets func(characterID int32) ([]AssetItem, error)) {
	addESILayer(p, func(id int32, f *fakeESI) {
		f.assets = func(context.Context, int32) ([]esiapi.CharactersCharacterIdAssetsGetInner, int32, error) {
			items, err := assets(id)
			if err != nil {
				return nil, 0, err
			}
			return toESIAssets(items), 1, nil
		}
	})
}

// toESIAssets writes assets the way ESI sends a character's.
func toESIAssets(items []AssetItem) []esiapi.CharactersCharacterIdAssetsGetInner {
	if items == nil {
		return nil
	}
	out := make([]esiapi.CharactersCharacterIdAssetsGetInner, len(items))
	for i, a := range items {
		out[i] = esiapi.CharactersCharacterIdAssetsGetInner{ItemId: a.ItemID, TypeId: a.TypeID, LocationId: a.LocationID, LocationType: a.LocationType,
			LocationFlag: a.LocationFlag, Quantity: a.Quantity, IsSingleton: a.IsSingleton, IsBlueprintCopy: a.IsBlueprintCopy}
	}
	return out
}

// stubCorporationAssets answers a corporation's assets, one page, read
// through any character.
func stubCorporationAssets(p *planner, assets func(corporationID int64) ([]AssetItem, error)) {
	addESILayer(p, func(_ int32, f *fakeESI) {
		f.corporationAssets = func(_ context.Context, corporationID int64, _ int32) ([]esiapi.CorporationsCorporationIdAssetsGetInner, int32, error) {
			items, err := assets(corporationID)
			if err != nil {
				return nil, 0, err
			}
			out := make([]esiapi.CorporationsCorporationIdAssetsGetInner, len(items))
			for i, a := range items {
				out[i] = esiapi.CorporationsCorporationIdAssetsGetInner{ItemId: a.ItemID, TypeId: a.TypeID, LocationId: a.LocationID, LocationType: a.LocationType,
					LocationFlag: a.LocationFlag, Quantity: a.Quantity, IsSingleton: a.IsSingleton, IsBlueprintCopy: a.IsBlueprintCopy}
			}
			return out, 1, nil
		}
	})
}

// stubAssetNames answers each character's item names with
// names(characterID, itemIDs).
func stubAssetNames(p *planner, names func(characterID int32, itemIDs []int64) ([]AssetName, error)) {
	addESILayer(p, func(id int32, f *fakeESI) {
		f.assetNames = func(_ context.Context, itemIDs []int64) ([]esiapi.CharactersCharacterIdAssetsNamesPostInner, error) {
			got, err := names(id, itemIDs)
			if err != nil {
				return nil, err
			}
			out := make([]esiapi.CharactersCharacterIdAssetsNamesPostInner, len(got))
			for i, n := range got {
				out[i] = esiapi.CharactersCharacterIdAssetsNamesPostInner{ItemId: n.ItemID, Name: n.Name}
			}
			return out, nil
		}
	})
}

// noESI is the openers for a test that makes no ESI call that succeeds:
// every client opens, and every call fails with errNotFaked.
func noESI() (plannerOpen, func() plannerPublicClient) {
	return openFakeESI(fakeESI{}), openFakePublicESI(fakeESI{})
}

// Every planner read through a character's client reports a client that
// can't be opened, with what it was reading.
func TestPlannerReads_ReportUnopenableClients(t *testing.T) {
	_, p := industryTestEnv(t)
	wantErr := errors.New("no refresh token")
	p.openESI = failOpenESI(wantErr)
	ctx := context.Background()
	for name, tt := range map[string]struct {
		read func() error
		want string
	}{
		"skills":                 {func() error { _, err := p.skills(ctx, testPilot); return err }, fmt.Sprintf("skills of %d", testPilot)},
		"character blueprints":   {func() error { _, err := p.loadOwnership(ctx, testPilot, 0, false); return err }, "character blueprints"},
		"corporation blueprints": {func() error { _, err := p.loadOwnership(ctx, testPilot, testCorp, false); return err }, "corporation blueprints"},
		"character jobs":         {func() error { _, err := p.findJob(ctx, testPilot, 0, 1); return err }, "character jobs"},
		"corporation jobs":       {func() error { _, err := p.findJob(ctx, testPilot, testCorp, 1); return err }, "corporation jobs"},
		"implants":               {func() error { _, _, err := p.implantRefiningBonus(ctx, testPilot); return err }, fmt.Sprintf("implants of %d", testPilot)},
		"contacts":               {func() error { _, err := p.badStandingAlliances(ctx, testPilot); return err }, "character contacts"},
		"structure market":       {func() error { _, err := p.structureOrders(ctx, testPilot, 1049345291823); return err }, "read structure 1049345291823's market through character"},
		"structure":              {func() error { _, err := p.hubFacts(ctx, 1049345291823, testPilot); return err }, "read structure 1049345291823 through character"},
	} {
		err := tt.read()
		if !errors.Is(err, wantErr) || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: err = %v, want %q wrapping %v", name, err, tt.want, wantErr)
		}
	}
}

// list_jump_clones names a character whose client can't be opened in its
// warnings instead of failing.
func TestListJumpClones_UnopenableClientIsAWarning(t *testing.T) {
	store, p := readinessEnv(t)
	p.openESI = failOpenESI(errors.New("no refresh token"))
	_, out, err := listJumpClonesHandler("test", fakeOpener(p), p.openPublicESI, store, p.sde)(context.Background(), nil, ListJumpClonesInput{CharacterIDs: []int32{reconPilot}})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Characters) != 0 || len(out.Warnings) != 2 || !strings.Contains(out.Warnings[0], "couldn't read Recon Pilot's clones: no refresh token") {
		t.Errorf("out = %+v, want both characters' clones unread with the open error", out)
	}
}
