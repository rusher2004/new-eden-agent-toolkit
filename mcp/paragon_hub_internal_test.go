package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zalando/go-keyring"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func iskPrice(v float64) esiapi.CharactersParagonHubSkinrItemPrice {
	return esiapi.CharactersParagonHubSkinrItemPrice{Isk: &esiapi.Isk{Isk: &v}}
}

func plexPrice(v int64) esiapi.CharactersParagonHubSkinrItemPrice {
	return esiapi.CharactersParagonHubSkinrItemPrice{Plex: &esiapi.Plex{Plex: &v}}
}

func TestListParagonHubSkinrListingsHandler_FlattensPriceAndCursor(t *testing.T) {
	orig := listParagonHubSkinrListings
	t.Cleanup(func() { listParagonHubSkinrListings = orig })
	created := time.Unix(1000, 0)
	prevCursor := "prev"
	var gotLimit int64
	listParagonHubSkinrListings = func(ctx context.Context, userAgent, after, before string, limit int64) ([]esiapi.ParagonHubSkinrInternalItem, *esiapi.Cursor, error) {
		gotLimit = limit
		return []esiapi.ParagonHubSkinrInternalItem{
			{Id: "l-1", SkinrId: "s-1", SellerId: 11, State: "active", Quantity: 1, Created: created, Price: plexPrice(3899)},
			{Id: "l-2", SkinrId: "s-2", SellerId: 12, State: "sold_out", Quantity: 0, Price: iskPrice(1.5e8)},
		}, &esiapi.Cursor{Before: &prevCursor}, nil
	}

	_, out, err := listParagonHubSkinrListingsHandler("v1")(context.Background(), nil, ListParagonHubSkinrListingsInput{Limit: 50})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if gotLimit != 50 {
		t.Errorf("forwarded limit = %d, want 50", gotLimit)
	}
	if len(out.Listings) != 2 {
		t.Fatalf("out = %+v, want 2 listings", out)
	}
	if l := out.Listings[0]; l.ListingID != "l-1" || l.SkinrID != "s-1" || l.SellerID != 11 || !l.Created.Equal(created) ||
		l.PricePlex == nil || *l.PricePlex != 3899 || l.PriceIsk != nil || l.Target != nil {
		t.Errorf("listings[0] = %+v, want l-1 priced 3899 PLEX, no ISK, no target", l)
	}
	if l := out.Listings[1]; l.PriceIsk == nil || *l.PriceIsk != 1.5e8 || l.PricePlex != nil {
		t.Errorf("listings[1] = %+v, want priced 1.5e8 ISK, no PLEX", l)
	}
	if out.NextBefore == nil || *out.NextBefore != "prev" || out.NextAfter != nil {
		t.Errorf("NextAfter/NextBefore = %v/%v, want nil/prev", out.NextAfter, out.NextBefore)
	}
}

func TestListCharacterParagonHubSkinrListingsHandler_FlattensTarget(t *testing.T) {
	store := storeWithCharacter(t)
	orig := listCharacterParagonHubSkinrListings
	t.Cleanup(func() { listCharacterParagonHubSkinrListings = orig })
	listCharacterParagonHubSkinrListings = func(ctx context.Context, clientID string, characterID int32, userAgent, after, before string, limit int64) ([]esiapi.CharactersParagonHubSkinrItem, *esiapi.Cursor, error) {
		return []esiapi.CharactersParagonHubSkinrItem{
			{Id: "l-1", Price: iskPrice(1000), Target: esiapi.CharactersParagonHubSkinrItemTarget{Public: &esiapi.Public{Public: ptr(true)}}},
			{Id: "l-2", Price: plexPrice(10), Target: esiapi.CharactersParagonHubSkinrItemTarget{AllianceId: &esiapi.AllianceId{AllianceId: ptr[int64](99000001)}}},
			{Id: "l-3", Price: plexPrice(10), Target: esiapi.CharactersParagonHubSkinrItemTarget{CorporationId: &esiapi.CorporationId{CorporationId: ptr[int64](98000001)}}},
			{Id: "l-4", Price: plexPrice(10), Target: esiapi.CharactersParagonHubSkinrItemTarget{CharacterId: &esiapi.CharacterId{CharacterId: ptr[int64](2112000001)}}},
		}, nil, nil
	}

	_, out, err := listCharacterParagonHubSkinrListingsHandler("v1", "client-id", store)(context.Background(), nil, ListCharacterParagonHubSkinrListingsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || len(out.Listings) != 4 {
		t.Fatalf("out = %+v, want character 7 with 4 listings", out)
	}
	for i, want := range []struct {
		kind string
		id   int64
	}{{"public", 0}, {"alliance", 99000001}, {"corporation", 98000001}, {"character", 2112000001}} {
		tgt := out.Listings[i].Target
		if tgt == nil || tgt.Kind != want.kind {
			t.Errorf("listings[%d].Target = %+v, want kind %q", i, tgt, want.kind)
			continue
		}
		if want.id == 0 && tgt.ID != nil {
			t.Errorf("listings[%d].Target.ID = %v, want nil for a public listing", i, *tgt.ID)
		}
		if want.id != 0 && (tgt.ID == nil || *tgt.ID != want.id) {
			t.Errorf("listings[%d].Target.ID = %v, want %d", i, tgt.ID, want.id)
		}
	}
	if out.Listings[0].PriceIsk == nil || *out.Listings[0].PriceIsk != 1000 {
		t.Errorf("listings[0].PriceIsk = %v, want 1000", out.Listings[0].PriceIsk)
	}
}

func TestParagonHubTargetedListingHandlers_ForwardIDsAndEchoThem(t *testing.T) {
	store := storeWithCharacter(t)
	origA, origC, origCh := listParagonHubSkinrListingsForAlliance, listParagonHubSkinrListingsForCorporation, listParagonHubSkinrListingsForCharacter
	t.Cleanup(func() {
		listParagonHubSkinrListingsForAlliance, listParagonHubSkinrListingsForCorporation, listParagonHubSkinrListingsForCharacter = origA, origC, origCh
	})
	one := []esiapi.ParagonHubSkinrInternalItem{{Id: "l-1", Price: plexPrice(10)}}
	var gotAlliance, gotCorporation int64
	var gotCharacter int32
	listParagonHubSkinrListingsForAlliance = func(ctx context.Context, clientID string, characterID int32, userAgent string, allianceID int64, after, before string, limit int64) ([]esiapi.ParagonHubSkinrInternalItem, *esiapi.Cursor, error) {
		gotAlliance = allianceID
		return one, nil, nil
	}
	listParagonHubSkinrListingsForCorporation = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, after, before string, limit int64) ([]esiapi.ParagonHubSkinrInternalItem, *esiapi.Cursor, error) {
		gotCorporation = corporationID
		return one, nil, nil
	}
	listParagonHubSkinrListingsForCharacter = func(ctx context.Context, clientID string, characterID int32, userAgent, after, before string, limit int64) ([]esiapi.ParagonHubSkinrInternalItem, *esiapi.Cursor, error) {
		gotCharacter = characterID
		return one, nil, nil
	}

	_, a, err := listParagonHubSkinrListingsForAllianceHandler("v1", "client-id", store)(context.Background(), nil, ListParagonHubSkinrListingsForAllianceInput{CharacterID: 7, AllianceID: 99000001})
	if err != nil || gotAlliance != 99000001 || a.AllianceID != 99000001 || a.CharacterID != 7 || len(a.Listings) != 1 {
		t.Errorf("alliance: err=%v forwarded=%d out=%+v, want 99000001 forwarded and echoed with one listing", err, gotAlliance, a)
	}
	_, c, err := listParagonHubSkinrListingsForCorporationHandler("v1", "client-id", store)(context.Background(), nil, ListParagonHubSkinrListingsForCorporationInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil || gotCorporation != 98000001 || c.CorporationID != 98000001 || len(c.Listings) != 1 {
		t.Errorf("corporation: err=%v forwarded=%d out=%+v, want 98000001 forwarded and echoed with one listing", err, gotCorporation, c)
	}
	_, ch, err := listParagonHubSkinrListingsForCharacterHandler("v1", "client-id", store)(context.Background(), nil, ListParagonHubSkinrListingsForCharacterInput{CharacterID: 7})
	if err != nil || gotCharacter != 7 || ch.CharacterID != 7 || len(ch.Listings) != 1 {
		t.Errorf("character: err=%v forwarded=%d out=%+v, want character 7 with one listing", err, gotCharacter, ch)
	}
}

func TestParagonHubPrivateHandlers_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	ctx := context.Background()

	for name, call := range map[string]func() error{
		"own listings": func() error {
			_, _, err := listCharacterParagonHubSkinrListingsHandler("v1", "client-id", store)(ctx, nil, ListCharacterParagonHubSkinrListingsInput{CharacterID: 999})
			return err
		},
		"for alliance": func() error {
			_, _, err := listParagonHubSkinrListingsForAllianceHandler("v1", "client-id", store)(ctx, nil, ListParagonHubSkinrListingsForAllianceInput{CharacterID: 999, AllianceID: 1})
			return err
		},
		"for corporation": func() error {
			_, _, err := listParagonHubSkinrListingsForCorporationHandler("v1", "client-id", store)(ctx, nil, ListParagonHubSkinrListingsForCorporationInput{CharacterID: 999, CorporationID: 1})
			return err
		},
		"for character": func() error {
			_, _, err := listParagonHubSkinrListingsForCharacterHandler("v1", "client-id", store)(ctx, nil, ListParagonHubSkinrListingsForCharacterInput{CharacterID: 999})
			return err
		},
	} {
		if err := call(); !errors.Is(err, ErrCharacterNotFound) {
			t.Errorf("%s: err = %v, want wrapping ErrCharacterNotFound", name, err)
		}
	}
}

func TestListParagonHubSkinrListingsHandler_PropagatesError(t *testing.T) {
	orig := listParagonHubSkinrListings
	t.Cleanup(func() { listParagonHubSkinrListings = orig })
	listParagonHubSkinrListings = func(ctx context.Context, userAgent, after, before string, limit int64) ([]esiapi.ParagonHubSkinrInternalItem, *esiapi.Cursor, error) {
		return nil, nil, errors.New("boom")
	}
	if _, _, err := listParagonHubSkinrListingsHandler("v1")(context.Background(), nil, ListParagonHubSkinrListingsInput{}); err == nil {
		t.Fatal("handler returned nil error, want propagated error")
	}
}
