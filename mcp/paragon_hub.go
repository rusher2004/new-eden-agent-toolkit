package mcp

import (
	"context"
	"fmt"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
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

// paragonHubSkinrListingsESI is the ESI call list_paragon_hub_skinr_listings makes.
type paragonHubSkinrListingsESI interface {
	ParagonHubSkinrListings(ctx context.Context, after, before string, limit int64) ([]esiapi.ParagonHubSkinrInternalItem, *esiapi.Cursor, error)
}

type ListParagonHubSkinrListingsInput struct {
	After  string `json:"after,omitempty" jsonschema:"cursor to walk forward from; omit to start from the beginning. Mutually exclusive with before"`
	Before string `json:"before,omitempty" jsonschema:"cursor to walk backward from; omit to start from the end. Mutually exclusive with after"`
	Limit  int64  `json:"limit,omitempty" jsonschema:"records to return per page, 10-100; omit for ESI's default of 10"`
}
type ListParagonHubSkinrListingsOutput = SkinrListingsPage

// AddListParagonHubSkinrListings registers list_paragon_hub_skinr_listings. open builds the ESI client; any client with the
// paragonHubSkinrListingsESI methods will do.
func AddListParagonHubSkinrListings[C paragonHubSkinrListingsESI](s *mcpsdk.Server, open func() C) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_paragon_hub_skinr_listings",
		Description: "Browse the SKINR listings publicly available on the Paragon Hub, with ISK or PLEX prices, cursor-paginated (public data, no character required).",
	}, listParagonHubSkinrListingsHandler(open))
}

func listParagonHubSkinrListingsHandler[C paragonHubSkinrListingsESI](open func() C) mcpsdk.ToolHandlerFor[ListParagonHubSkinrListingsInput, ListParagonHubSkinrListingsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListParagonHubSkinrListingsInput) (*mcpsdk.CallToolResult, ListParagonHubSkinrListingsOutput, error) {
		client := open()
		items, cursor, err := client.ParagonHubSkinrListings(ctx, in.After, in.Before, in.Limit)
		if err != nil {
			return nil, ListParagonHubSkinrListingsOutput{}, fmt.Errorf("list paragon hub skinr listings: %w", err)
		}
		out := ListParagonHubSkinrListingsOutput{Listings: toSkinrListings(items)}
		out.NextAfter, out.NextBefore = nextCursor(cursor)
		return nil, out, nil
	}
}

// --- list_character_paragon_hub_skinr_listings (private, cursor-paginated) ---

// characterParagonHubSkinrListingsESI is the ESI call list_character_paragon_hub_skinr_listings makes.
type characterParagonHubSkinrListingsESI interface {
	CharacterParagonHubSkinrListings(ctx context.Context, after, before string, limit int64) ([]esiapi.CharactersParagonHubSkinrItem, *esiapi.Cursor, error)
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

// AddListCharacterParagonHubSkinrListings registers list_character_paragon_hub_skinr_listings. open builds the ESI client; any client with the
// characterParagonHubSkinrListingsESI methods will do.
func AddListCharacterParagonHubSkinrListings[C characterParagonHubSkinrListingsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_character_paragon_hub_skinr_listings",
		Description: "List the SKINR listings a logged-in character has posted on the Paragon Hub, including who each targets, cursor-paginated.",
	}, listCharacterParagonHubSkinrListingsHandler(open, store))
}

func listCharacterParagonHubSkinrListingsHandler[C characterParagonHubSkinrListingsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[ListCharacterParagonHubSkinrListingsInput, ListCharacterParagonHubSkinrListingsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterParagonHubSkinrListingsInput) (*mcpsdk.CallToolResult, ListCharacterParagonHubSkinrListingsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterParagonHubSkinrListingsOutput{}, err
		}
		if !found {
			return nil, ListCharacterParagonHubSkinrListingsOutput{}, fmt.Errorf("list character paragon hub skinr listings for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCharacterParagonHubSkinrListingsOutput{}, fmt.Errorf("build ESI client: %w", err)
		}
		items, cursor, err := client.CharacterParagonHubSkinrListings(ctx, in.After, in.Before, in.Limit)
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

// paragonHubSkinrListingsForAllianceESI is the ESI call list_paragon_hub_skinr_listings_for_alliance makes.
type paragonHubSkinrListingsForAllianceESI interface {
	ParagonHubSkinrListingsForAlliance(ctx context.Context, allianceID int64, after, before string, limit int64) ([]esiapi.ParagonHubSkinrInternalItem, *esiapi.Cursor, error)
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

// AddListParagonHubSkinrListingsForAlliance registers list_paragon_hub_skinr_listings_for_alliance. open builds the ESI client; any client with the
// paragonHubSkinrListingsForAllianceESI methods will do.
func AddListParagonHubSkinrListingsForAlliance[C paragonHubSkinrListingsForAllianceESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_paragon_hub_skinr_listings_for_alliance",
		Description: "Browse the Paragon Hub SKINR listings visible to an alliance, using a logged-in character to authorize, cursor-paginated.",
	}, listParagonHubSkinrListingsForAllianceHandler(open, store))
}

func listParagonHubSkinrListingsForAllianceHandler[C paragonHubSkinrListingsForAllianceESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[ListParagonHubSkinrListingsForAllianceInput, ListParagonHubSkinrListingsForAllianceOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListParagonHubSkinrListingsForAllianceInput) (*mcpsdk.CallToolResult, ListParagonHubSkinrListingsForAllianceOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListParagonHubSkinrListingsForAllianceOutput{}, err
		}
		if !found {
			return nil, ListParagonHubSkinrListingsForAllianceOutput{}, fmt.Errorf("list paragon hub skinr listings for alliance for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListParagonHubSkinrListingsForAllianceOutput{}, fmt.Errorf("build ESI client: %w", err)
		}
		items, cursor, err := client.ParagonHubSkinrListingsForAlliance(ctx, in.AllianceID, in.After, in.Before, in.Limit)
		if err != nil {
			return nil, ListParagonHubSkinrListingsForAllianceOutput{}, fmt.Errorf("list paragon hub skinr listings for alliance: %w", err)
		}
		out := ListParagonHubSkinrListingsForAllianceOutput{CharacterID: in.CharacterID, AllianceID: in.AllianceID, Listings: toSkinrListings(items)}
		out.NextAfter, out.NextBefore = nextCursor(cursor)
		return nil, out, nil
	}
}

// --- list_paragon_hub_skinr_listings_for_corporation (private, cursor-paginated) ---

// paragonHubSkinrListingsForCorporationESI is the ESI call list_paragon_hub_skinr_listings_for_corporation makes.
type paragonHubSkinrListingsForCorporationESI interface {
	ParagonHubSkinrListingsForCorporation(ctx context.Context, corporationID int64, after, before string, limit int64) ([]esiapi.ParagonHubSkinrInternalItem, *esiapi.Cursor, error)
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

// AddListParagonHubSkinrListingsForCorporation registers list_paragon_hub_skinr_listings_for_corporation. open builds the ESI client; any client with the
// paragonHubSkinrListingsForCorporationESI methods will do.
func AddListParagonHubSkinrListingsForCorporation[C paragonHubSkinrListingsForCorporationESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_paragon_hub_skinr_listings_for_corporation",
		Description: "Browse the Paragon Hub SKINR listings visible to a corporation, using a logged-in character to authorize, cursor-paginated.",
	}, listParagonHubSkinrListingsForCorporationHandler(open, store))
}

func listParagonHubSkinrListingsForCorporationHandler[C paragonHubSkinrListingsForCorporationESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[ListParagonHubSkinrListingsForCorporationInput, ListParagonHubSkinrListingsForCorporationOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListParagonHubSkinrListingsForCorporationInput) (*mcpsdk.CallToolResult, ListParagonHubSkinrListingsForCorporationOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListParagonHubSkinrListingsForCorporationOutput{}, err
		}
		if !found {
			return nil, ListParagonHubSkinrListingsForCorporationOutput{}, fmt.Errorf("list paragon hub skinr listings for corporation for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListParagonHubSkinrListingsForCorporationOutput{}, fmt.Errorf("build ESI client: %w", err)
		}
		items, cursor, err := client.ParagonHubSkinrListingsForCorporation(ctx, in.CorporationID, in.After, in.Before, in.Limit)
		if err != nil {
			return nil, ListParagonHubSkinrListingsForCorporationOutput{}, fmt.Errorf("list paragon hub skinr listings for corporation: %w", err)
		}
		out := ListParagonHubSkinrListingsForCorporationOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Listings: toSkinrListings(items)}
		out.NextAfter, out.NextBefore = nextCursor(cursor)
		return nil, out, nil
	}
}

// --- list_paragon_hub_skinr_listings_for_character (private, cursor-paginated) ---

// paragonHubSkinrListingsForCharacterESI is the ESI call list_paragon_hub_skinr_listings_for_character makes.
type paragonHubSkinrListingsForCharacterESI interface {
	ParagonHubSkinrListingsForCharacter(ctx context.Context, after, before string, limit int64) ([]esiapi.ParagonHubSkinrInternalItem, *esiapi.Cursor, error)
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

// AddListParagonHubSkinrListingsForCharacter registers list_paragon_hub_skinr_listings_for_character. open builds the ESI client; any client with the
// paragonHubSkinrListingsForCharacterESI methods will do.
func AddListParagonHubSkinrListingsForCharacter[C paragonHubSkinrListingsForCharacterESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_paragon_hub_skinr_listings_for_character",
		Description: "Browse the Paragon Hub SKINR listings visible to a logged-in character (listings other sellers targeted at it), cursor-paginated.",
	}, listParagonHubSkinrListingsForCharacterHandler(open, store))
}

func listParagonHubSkinrListingsForCharacterHandler[C paragonHubSkinrListingsForCharacterESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[ListParagonHubSkinrListingsForCharacterInput, ListParagonHubSkinrListingsForCharacterOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListParagonHubSkinrListingsForCharacterInput) (*mcpsdk.CallToolResult, ListParagonHubSkinrListingsForCharacterOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListParagonHubSkinrListingsForCharacterOutput{}, err
		}
		if !found {
			return nil, ListParagonHubSkinrListingsForCharacterOutput{}, fmt.Errorf("list paragon hub skinr listings for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListParagonHubSkinrListingsForCharacterOutput{}, fmt.Errorf("build ESI client: %w", err)
		}
		items, cursor, err := client.ParagonHubSkinrListingsForCharacter(ctx, in.After, in.Before, in.Limit)
		if err != nil {
			return nil, ListParagonHubSkinrListingsForCharacterOutput{}, fmt.Errorf("list paragon hub skinr listings for character: %w", err)
		}
		out := ListParagonHubSkinrListingsForCharacterOutput{CharacterID: in.CharacterID, Listings: toSkinrListings(items)}
		out.NextAfter, out.NextBefore = nextCursor(cursor)
		return nil, out, nil
	}
}
