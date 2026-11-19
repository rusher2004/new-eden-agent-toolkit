package mcp

import (
	"context"
	"fmt"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// --- shared types ---

// SkinrListingTarget flattens CharactersParagonHubSkinrItemTarget, a 4-variant oneOf. Kind names
// the active variant ("public", "alliance", "corporation", "character"); ID is set for every Kind
// but "public".
type SkinrListingTarget struct {
	Kind string `json:"kind"`
	ID   *int64 `json:"id,omitempty"`
}

func toSkinrListingTarget(t esiapi.CharactersParagonHubSkinrItemTarget) *SkinrListingTarget {
	switch {
	case t.Public != nil && t.Public.Public != nil:
		return &SkinrListingTarget{Kind: "public"}
	case t.AllianceId != nil && t.AllianceId.AllianceId != nil:
		return &SkinrListingTarget{Kind: "alliance", ID: t.AllianceId.AllianceId}
	case t.CorporationId != nil && t.CorporationId.CorporationId != nil:
		return &SkinrListingTarget{Kind: "corporation", ID: t.CorporationId.CorporationId}
	case t.CharacterId != nil && t.CharacterId.CharacterId != nil:
		return &SkinrListingTarget{Kind: "character", ID: t.CharacterId.CharacterId}
	default:
		return nil
	}
}

// SkinrListing is one Paragon Hub SKINR listing. Price is a 2-variant oneOf in ESI ({"isk": n}
// or {"plex": n}); exactly one of PriceIsk/PricePlex is set. Target is populated only by
// list_character_paragon_hub_skinr_listings -- ESI omits it from every other listing endpoint.
type SkinrListing struct {
	ListingID    string              `json:"listingId"`
	SkinrID      string              `json:"skinrId"`
	SellerID     int64               `json:"sellerId"`
	State        string              `json:"state"`
	Quantity     int64               `json:"quantity"`
	PriceIsk     *float64            `json:"priceIsk,omitempty"`
	PricePlex    *int64              `json:"pricePlex,omitempty"`
	Created      time.Time           `json:"created"`
	Expires      time.Time           `json:"expires"`
	LastModified time.Time           `json:"lastModified"`
	Target       *SkinrListingTarget `json:"target,omitempty"`
}

func setSkinrListingPrice(l *SkinrListing, p esiapi.CharactersParagonHubSkinrItemPrice) {
	if p.Isk != nil {
		l.PriceIsk = p.Isk.Isk
	}
	if p.Plex != nil {
		l.PricePlex = p.Plex.Plex
	}
}

func toSkinrListings(items []esiapi.ParagonHubSkinrInternalItem) []SkinrListing {
	out := make([]SkinrListing, len(items))
	for i, it := range items {
		out[i] = SkinrListing{
			ListingID: it.Id, SkinrID: it.SkinrId, SellerID: it.SellerId, State: it.State, Quantity: it.Quantity,
			Created: it.Created, Expires: it.Expires, LastModified: it.LastModified,
		}
		setSkinrListingPrice(&out[i], it.Price)
	}
	return out
}

// SkinrListingsPage is the shared output shape of every Paragon Hub tool that has no extra
// identifying input to echo back beyond the character.
type SkinrListingsPage struct {
	Listings   []SkinrListing `json:"listings"`
	NextAfter  *string        `json:"nextAfter,omitempty"`
	NextBefore *string        `json:"nextBefore,omitempty"`
}

func nextCursor(c *esiapi.Cursor) (after, before *string) {
	if c == nil {
		return nil, nil
	}
	return c.After, c.Before
}

// --- list_paragon_hub_skinr_listings (public, cursor-paginated) ---

var listParagonHubSkinrListings = func(ctx context.Context, userAgent, after, before string, limit int64) ([]esiapi.ParagonHubSkinrInternalItem, *esiapi.Cursor, error) {
	return esi.NewPublicClient(userAgent).ParagonHubSkinrListings(ctx, after, before, limit)
}

type ListParagonHubSkinrListingsInput struct {
	After  string `json:"after,omitempty" jsonschema:"cursor to walk forward from; omit to start from the beginning. Mutually exclusive with before"`
	Before string `json:"before,omitempty" jsonschema:"cursor to walk backward from; omit to start from the end. Mutually exclusive with after"`
	Limit  int64  `json:"limit,omitempty" jsonschema:"records to return per page, 10-100; omit for ESI's default of 10"`
}
type ListParagonHubSkinrListingsOutput = SkinrListingsPage

func listParagonHubSkinrListingsHandler(version string) mcpsdk.ToolHandlerFor[ListParagonHubSkinrListingsInput, ListParagonHubSkinrListingsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListParagonHubSkinrListingsInput) (*mcpsdk.CallToolResult, ListParagonHubSkinrListingsOutput, error) {
		items, cursor, err := listParagonHubSkinrListings(ctx, sso.UserAgent(version), in.After, in.Before, in.Limit)
		if err != nil {
			return nil, ListParagonHubSkinrListingsOutput{}, fmt.Errorf("list paragon hub skinr listings: %w", err)
		}
		out := ListParagonHubSkinrListingsOutput{Listings: toSkinrListings(items)}
		out.NextAfter, out.NextBefore = nextCursor(cursor)
		return nil, out, nil
	}
}

// --- list_character_paragon_hub_skinr_listings (private, cursor-paginated) ---

var listCharacterParagonHubSkinrListings = func(ctx context.Context, clientID string, characterID int32, userAgent, after, before string, limit int64) ([]esiapi.CharactersParagonHubSkinrItem, *esiapi.Cursor, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, nil, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CharacterParagonHubSkinrListings(ctx, after, before, limit)
}

type ListCharacterParagonHubSkinrListingsInput struct {
	CharacterID int32  `json:"characterId" jsonschema:"the character ID to check"`
	After       string `json:"after,omitempty" jsonschema:"cursor to walk forward from; omit to start from the beginning. Mutually exclusive with before"`
	Before      string `json:"before,omitempty" jsonschema:"cursor to walk backward from; omit to start from the end. Mutually exclusive with after"`
	Limit       int64  `json:"limit,omitempty" jsonschema:"records to return per page, 10-100; omit for ESI's default of 10"`
}
type ListCharacterParagonHubSkinrListingsOutput struct {
	CharacterID int32          `json:"characterId"`
	Listings    []SkinrListing `json:"listings"`
	NextAfter   *string        `json:"nextAfter,omitempty"`
	NextBefore  *string        `json:"nextBefore,omitempty"`
}

func listCharacterParagonHubSkinrListingsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCharacterParagonHubSkinrListingsInput, ListCharacterParagonHubSkinrListingsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterParagonHubSkinrListingsInput) (*mcpsdk.CallToolResult, ListCharacterParagonHubSkinrListingsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterParagonHubSkinrListingsOutput{}, err
		}
		if !found {
			return nil, ListCharacterParagonHubSkinrListingsOutput{}, fmt.Errorf("list character paragon hub skinr listings for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		items, cursor, err := listCharacterParagonHubSkinrListings(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.After, in.Before, in.Limit)
		if err != nil {
			return nil, ListCharacterParagonHubSkinrListingsOutput{}, fmt.Errorf("list character paragon hub skinr listings: %w", err)
		}
		out := ListCharacterParagonHubSkinrListingsOutput{CharacterID: in.CharacterID, Listings: make([]SkinrListing, len(items))}
		for i, it := range items {
			out.Listings[i] = SkinrListing{
				ListingID: it.Id, SkinrID: it.SkinrId, SellerID: it.SellerId, State: it.State, Quantity: it.Quantity,
				Created: it.Created, Expires: it.Expires, LastModified: it.LastModified, Target: toSkinrListingTarget(it.Target),
			}
			setSkinrListingPrice(&out.Listings[i], it.Price)
		}
		out.NextAfter, out.NextBefore = nextCursor(cursor)
		return nil, out, nil
	}
}

// --- list_paragon_hub_skinr_listings_for_alliance (private, cursor-paginated) ---

var listParagonHubSkinrListingsForAlliance = func(ctx context.Context, clientID string, characterID int32, userAgent string, allianceID int64, after, before string, limit int64) ([]esiapi.ParagonHubSkinrInternalItem, *esiapi.Cursor, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, nil, fmt.Errorf("build ESI client: %w", err)
	}
	return client.ParagonHubSkinrListingsForAlliance(ctx, allianceID, after, before, limit)
}

type ListParagonHubSkinrListingsForAllianceInput struct {
	CharacterID int32  `json:"characterId" jsonschema:"the logged-in character whose token authorizes the request"`
	AllianceID  int64  `json:"allianceId" jsonschema:"the alliance whose visible listings to browse"`
	After       string `json:"after,omitempty" jsonschema:"cursor to walk forward from; omit to start from the beginning. Mutually exclusive with before"`
	Before      string `json:"before,omitempty" jsonschema:"cursor to walk backward from; omit to start from the end. Mutually exclusive with after"`
	Limit       int64  `json:"limit,omitempty" jsonschema:"records to return per page, 10-100; omit for ESI's default of 10"`
}
type ListParagonHubSkinrListingsForAllianceOutput struct {
	CharacterID int32          `json:"characterId"`
	AllianceID  int64          `json:"allianceId"`
	Listings    []SkinrListing `json:"listings"`
	NextAfter   *string        `json:"nextAfter,omitempty"`
	NextBefore  *string        `json:"nextBefore,omitempty"`
}

func listParagonHubSkinrListingsForAllianceHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListParagonHubSkinrListingsForAllianceInput, ListParagonHubSkinrListingsForAllianceOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListParagonHubSkinrListingsForAllianceInput) (*mcpsdk.CallToolResult, ListParagonHubSkinrListingsForAllianceOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListParagonHubSkinrListingsForAllianceOutput{}, err
		}
		if !found {
			return nil, ListParagonHubSkinrListingsForAllianceOutput{}, fmt.Errorf("list paragon hub skinr listings for alliance for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		items, cursor, err := listParagonHubSkinrListingsForAlliance(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.AllianceID, in.After, in.Before, in.Limit)
		if err != nil {
			return nil, ListParagonHubSkinrListingsForAllianceOutput{}, fmt.Errorf("list paragon hub skinr listings for alliance: %w", err)
		}
		out := ListParagonHubSkinrListingsForAllianceOutput{CharacterID: in.CharacterID, AllianceID: in.AllianceID, Listings: toSkinrListings(items)}
		out.NextAfter, out.NextBefore = nextCursor(cursor)
		return nil, out, nil
	}
}

// --- list_paragon_hub_skinr_listings_for_corporation (private, cursor-paginated) ---

var listParagonHubSkinrListingsForCorporation = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, after, before string, limit int64) ([]esiapi.ParagonHubSkinrInternalItem, *esiapi.Cursor, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, nil, fmt.Errorf("build ESI client: %w", err)
	}
	return client.ParagonHubSkinrListingsForCorporation(ctx, corporationID, after, before, limit)
}

type ListParagonHubSkinrListingsForCorporationInput struct {
	CharacterID   int32  `json:"characterId" jsonschema:"the logged-in character whose token authorizes the request"`
	CorporationID int64  `json:"corporationId" jsonschema:"the corporation whose visible listings to browse"`
	After         string `json:"after,omitempty" jsonschema:"cursor to walk forward from; omit to start from the beginning. Mutually exclusive with before"`
	Before        string `json:"before,omitempty" jsonschema:"cursor to walk backward from; omit to start from the end. Mutually exclusive with after"`
	Limit         int64  `json:"limit,omitempty" jsonschema:"records to return per page, 10-100; omit for ESI's default of 10"`
}
type ListParagonHubSkinrListingsForCorporationOutput struct {
	CharacterID   int32          `json:"characterId"`
	CorporationID int64          `json:"corporationId"`
	Listings      []SkinrListing `json:"listings"`
	NextAfter     *string        `json:"nextAfter,omitempty"`
	NextBefore    *string        `json:"nextBefore,omitempty"`
}

func listParagonHubSkinrListingsForCorporationHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListParagonHubSkinrListingsForCorporationInput, ListParagonHubSkinrListingsForCorporationOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListParagonHubSkinrListingsForCorporationInput) (*mcpsdk.CallToolResult, ListParagonHubSkinrListingsForCorporationOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListParagonHubSkinrListingsForCorporationOutput{}, err
		}
		if !found {
			return nil, ListParagonHubSkinrListingsForCorporationOutput{}, fmt.Errorf("list paragon hub skinr listings for corporation for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		items, cursor, err := listParagonHubSkinrListingsForCorporation(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.After, in.Before, in.Limit)
		if err != nil {
			return nil, ListParagonHubSkinrListingsForCorporationOutput{}, fmt.Errorf("list paragon hub skinr listings for corporation: %w", err)
		}
		out := ListParagonHubSkinrListingsForCorporationOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Listings: toSkinrListings(items)}
		out.NextAfter, out.NextBefore = nextCursor(cursor)
		return nil, out, nil
	}
}

// --- list_paragon_hub_skinr_listings_for_character (private, cursor-paginated) ---

var listParagonHubSkinrListingsForCharacter = func(ctx context.Context, clientID string, characterID int32, userAgent, after, before string, limit int64) ([]esiapi.ParagonHubSkinrInternalItem, *esiapi.Cursor, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, nil, fmt.Errorf("build ESI client: %w", err)
	}
	return client.ParagonHubSkinrListingsForCharacter(ctx, after, before, limit)
}

type ListParagonHubSkinrListingsForCharacterInput struct {
	CharacterID int32  `json:"characterId" jsonschema:"the character whose visible listings to browse"`
	After       string `json:"after,omitempty" jsonschema:"cursor to walk forward from; omit to start from the beginning. Mutually exclusive with before"`
	Before      string `json:"before,omitempty" jsonschema:"cursor to walk backward from; omit to start from the end. Mutually exclusive with after"`
	Limit       int64  `json:"limit,omitempty" jsonschema:"records to return per page, 10-100; omit for ESI's default of 10"`
}
type ListParagonHubSkinrListingsForCharacterOutput struct {
	CharacterID int32          `json:"characterId"`
	Listings    []SkinrListing `json:"listings"`
	NextAfter   *string        `json:"nextAfter,omitempty"`
	NextBefore  *string        `json:"nextBefore,omitempty"`
}

func listParagonHubSkinrListingsForCharacterHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListParagonHubSkinrListingsForCharacterInput, ListParagonHubSkinrListingsForCharacterOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListParagonHubSkinrListingsForCharacterInput) (*mcpsdk.CallToolResult, ListParagonHubSkinrListingsForCharacterOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListParagonHubSkinrListingsForCharacterOutput{}, err
		}
		if !found {
			return nil, ListParagonHubSkinrListingsForCharacterOutput{}, fmt.Errorf("list paragon hub skinr listings for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		items, cursor, err := listParagonHubSkinrListingsForCharacter(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.After, in.Before, in.Limit)
		if err != nil {
			return nil, ListParagonHubSkinrListingsForCharacterOutput{}, fmt.Errorf("list paragon hub skinr listings for character: %w", err)
		}
		out := ListParagonHubSkinrListingsForCharacterOutput{CharacterID: in.CharacterID, Listings: toSkinrListings(items)}
		out.NextAfter, out.NextBefore = nextCursor(cursor)
		return nil, out, nil
	}
}
