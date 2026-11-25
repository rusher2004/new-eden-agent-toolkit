package mcp

import (
	"context"
	"fmt"
	"time"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Contract is one contract entry, shared by character/corporation/public listings. Status is
// empty ("") when populated from a public-region listing, which has no status field at all
// (public listings are implicitly outstanding).
type Contract struct {
	ContractID          int64      `json:"contractId"`
	AcceptorID          int64      `json:"acceptorId,omitempty"`
	AssigneeID          int64      `json:"assigneeId,omitempty"`
	IssuerID            int64      `json:"issuerId"`
	IssuerCorporationID int64      `json:"issuerCorporationId"`
	Availability        string     `json:"availability,omitempty"`
	Status              string     `json:"status,omitempty"`
	Type                string     `json:"type"`
	ForCorporation      bool       `json:"forCorporation"`
	DateIssued          time.Time  `json:"dateIssued"`
	DateExpired         time.Time  `json:"dateExpired"`
	DateAccepted        *time.Time `json:"dateAccepted,omitempty"`
	DateCompleted       *time.Time `json:"dateCompleted,omitempty"`
	DaysToComplete      *int64     `json:"daysToComplete,omitempty"`
	EndLocationID       *int64     `json:"endLocationId,omitempty"`
	StartLocationID     *int64     `json:"startLocationId,omitempty"`
	Buyout              *float64   `json:"buyout,omitempty"`
	Collateral          *float64   `json:"collateral,omitempty"`
	Price               *float64   `json:"price,omitempty"`
	Reward              *float64   `json:"reward,omitempty"`
	Volume              *float64   `json:"volume,omitempty"`
	Title               *string    `json:"title,omitempty"`
}

// ContractBid is one bid, shared by character/corporation/public listings. BidderID is populated
// for character-scoped and corporation-scoped bids (both use the same generated model with a
// required bidder_id field), but is genuinely absent for public bids (whose model has no such
// field).
type ContractBid struct {
	BidID    int64     `json:"bidId"`
	BidderID *int64    `json:"bidderId,omitempty"`
	Amount   float64   `json:"amount"`
	DateBid  time.Time `json:"dateBid"`
}

// ContractItem is one item, shared by character/corporation/public listings. Field availability
// differs slightly by scope (see esi/contracts.go's model documentation).
type ContractItem struct {
	RecordID           int64  `json:"recordId"`
	TypeID             int64  `json:"typeId"`
	TypeName           string `json:"typeName,omitempty"`
	Quantity           int64  `json:"quantity"`
	IsIncluded         bool   `json:"isIncluded"`
	IsSingleton        bool   `json:"isSingleton,omitempty"`
	RawQuantity        *int64 `json:"rawQuantity,omitempty"`
	ItemID             *int64 `json:"itemId,omitempty"`
	MaterialEfficiency *int64 `json:"materialEfficiency,omitempty"`
	Runs               *int64 `json:"runs,omitempty"`
	TimeEfficiency     *int64 `json:"timeEfficiency,omitempty"`
	IsBlueprintCopy    *bool  `json:"isBlueprintCopy,omitempty"`
}

// --- list_character_contracts ---

// characterContractsESI is the ESI call list_character_contracts makes.
type characterContractsESI interface {
	CharacterContracts(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdContractsGetInner, int32, error)
}

func listCharacterContracts(ctx context.Context, client characterContractsESI, page int32) ([]Contract, int32, error) {
	contracts, totalPages, err := client.CharacterContracts(ctx, page)
	if err != nil {
		return nil, 0, err
	}
	out := make([]Contract, len(contracts))
	for i, c := range contracts {
		out[i] = Contract{
			ContractID: c.ContractId, AcceptorID: c.AcceptorId, AssigneeID: c.AssigneeId,
			IssuerID: c.IssuerId, IssuerCorporationID: c.IssuerCorporationId, Availability: c.Availability,
			Status: c.Status, Type: c.Type, ForCorporation: c.ForCorporation,
			DateIssued: c.DateIssued, DateExpired: c.DateExpired, DateAccepted: c.DateAccepted,
			DateCompleted: c.DateCompleted, DaysToComplete: c.DaysToComplete, EndLocationID: c.EndLocationId,
			StartLocationID: c.StartLocationId, Buyout: c.Buyout, Collateral: c.Collateral,
			Price: c.Price, Reward: c.Reward, Volume: c.Volume, Title: c.Title,
		}
	}
	return out, totalPages, nil
}

type ListCharacterContractsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
	Page        int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCharacterContractsOutput struct {
	CharacterID int32      `json:"characterId"`
	Page        int32      `json:"page"`
	TotalPages  int32      `json:"totalPages"`
	Contracts   []Contract `json:"contracts"`
}

// AddListCharacterContracts registers list_character_contracts. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListCharacterContracts[C characterContractsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_character_contracts",
		Description: "List a logged-in EVE Online character's contracts.",
	}, listCharacterContractsHandler(open, store))
}

func listCharacterContractsHandler[C characterContractsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[ListCharacterContractsInput, ListCharacterContractsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterContractsInput) (*mcpsdk.CallToolResult, ListCharacterContractsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterContractsOutput{}, err
		}
		if !found {
			return nil, ListCharacterContractsOutput{}, fmt.Errorf("list character contracts for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCharacterContractsOutput{}, fmt.Errorf("list character contracts: %w", err)
		}
		contracts, totalPages, err := listCharacterContracts(ctx, client, in.Page)
		if err != nil {
			return nil, ListCharacterContractsOutput{}, fmt.Errorf("list character contracts: %w", err)
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListCharacterContractsOutput{CharacterID: in.CharacterID, Page: page, TotalPages: totalPages, Contracts: contracts}, nil
	}
}

// --- list_character_contract_bids ---

// characterContractBidsESI is the ESI call list_character_contract_bids makes.
type characterContractBidsESI interface {
	CharacterContractBids(ctx context.Context, contractID int64) ([]esiapi.CharactersCharacterIdContractsContractIdBidsGetInner, error)
}

func listCharacterContractBids(ctx context.Context, client characterContractBidsESI, contractID int64) ([]ContractBid, error) {
	bids, err := client.CharacterContractBids(ctx, contractID)
	if err != nil {
		return nil, err
	}
	out := make([]ContractBid, len(bids))
	for i, b := range bids {
		bidderID := b.BidderId
		out[i] = ContractBid{BidID: b.BidId, BidderID: &bidderID, Amount: b.Amount, DateBid: b.DateBid}
	}
	return out, nil
}

type ListCharacterContractBidsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
	ContractID  int64 `json:"contractId" jsonschema:"the contract ID, from list_character_contracts"`
}
type ListCharacterContractBidsOutput struct {
	CharacterID int32         `json:"characterId"`
	ContractID  int64         `json:"contractId"`
	Bids        []ContractBid `json:"bids"`
}

// AddListCharacterContractBids registers list_character_contract_bids. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListCharacterContractBids[C characterContractBidsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_character_contract_bids",
		Description: "List the bids on one of a logged-in character's contracts.",
	}, listCharacterContractBidsHandler(open, store))
}

func listCharacterContractBidsHandler[C characterContractBidsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[ListCharacterContractBidsInput, ListCharacterContractBidsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterContractBidsInput) (*mcpsdk.CallToolResult, ListCharacterContractBidsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterContractBidsOutput{}, err
		}
		if !found {
			return nil, ListCharacterContractBidsOutput{}, fmt.Errorf("list character contract bids for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCharacterContractBidsOutput{}, fmt.Errorf("list character contract bids: %w", err)
		}
		bids, err := listCharacterContractBids(ctx, client, in.ContractID)
		if err != nil {
			return nil, ListCharacterContractBidsOutput{}, fmt.Errorf("list character contract bids: %w", err)
		}
		return nil, ListCharacterContractBidsOutput{CharacterID: in.CharacterID, ContractID: in.ContractID, Bids: bids}, nil
	}
}

// --- list_character_contract_items ---

// characterContractItemsESI is the ESI call list_character_contract_items makes.
type characterContractItemsESI interface {
	CharacterContractItems(ctx context.Context, contractID int64) ([]esiapi.CharactersCharacterIdContractsContractIdItemsGetInner, error)
}

func listCharacterContractItems(ctx context.Context, client characterContractItemsESI, contractID int64) ([]ContractItem, error) {
	items, err := client.CharacterContractItems(ctx, contractID)
	if err != nil {
		return nil, err
	}
	out := make([]ContractItem, len(items))
	for i, it := range items {
		out[i] = ContractItem{RecordID: it.RecordId, TypeID: it.TypeId, Quantity: it.Quantity, IsIncluded: it.IsIncluded, IsSingleton: it.IsSingleton, RawQuantity: it.RawQuantity}
	}
	return out, nil
}

type ListCharacterContractItemsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
	ContractID  int64 `json:"contractId" jsonschema:"the contract ID, from list_character_contracts"`
}
type ListCharacterContractItemsOutput struct {
	CharacterID int32          `json:"characterId"`
	ContractID  int64          `json:"contractId"`
	Items       []ContractItem `json:"items"`
}

// AddListCharacterContractItems registers list_character_contract_items. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListCharacterContractItems[C characterContractItemsESI](s *mcpsdk.Server, userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore typeNameCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_character_contract_items",
		Description: "List the items on one of a logged-in character's contracts.",
	}, listCharacterContractItemsHandler(userAgent, open, store, sdeStore))
}

func listCharacterContractItemsHandler[C characterContractItemsESI](userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore typeNameCache) mcpsdk.ToolHandlerFor[ListCharacterContractItemsInput, ListCharacterContractItemsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterContractItemsInput) (*mcpsdk.CallToolResult, ListCharacterContractItemsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterContractItemsOutput{}, err
		}
		if !found {
			return nil, ListCharacterContractItemsOutput{}, fmt.Errorf("list character contract items for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCharacterContractItemsOutput{}, fmt.Errorf("list character contract items: %w", err)
		}
		items, err := listCharacterContractItems(ctx, client, in.ContractID)
		if err != nil {
			return nil, ListCharacterContractItemsOutput{}, fmt.Errorf("list character contract items: %w", err)
		}
		ids := make([]int64, len(items))
		for i, it := range items {
			ids[i] = it.TypeID
		}
		names := resolveTypeNames(ctx, sdeStore, userAgent, ids)
		for i := range items {
			items[i].TypeName = names[items[i].TypeID]
		}
		return nil, ListCharacterContractItemsOutput{CharacterID: in.CharacterID, ContractID: in.ContractID, Items: items}, nil
	}
}

// --- list_corporation_contracts ---

// corporationContractsESI is the ESI call list_corporation_contracts makes.
type corporationContractsESI interface {
	CorporationContracts(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdContractsGetInner, int32, error)
}

func listCorporationContracts(ctx context.Context, client corporationContractsESI, corporationID int64, page int32) ([]Contract, int32, error) {
	contracts, totalPages, err := client.CorporationContracts(ctx, corporationID, page)
	if err != nil {
		return nil, 0, err
	}
	out := make([]Contract, len(contracts))
	for i, c := range contracts {
		out[i] = Contract{
			ContractID: c.ContractId, AcceptorID: c.AcceptorId, AssigneeID: c.AssigneeId,
			IssuerID: c.IssuerId, IssuerCorporationID: c.IssuerCorporationId, Availability: c.Availability,
			Status: c.Status, Type: c.Type, ForCorporation: c.ForCorporation,
			DateIssued: c.DateIssued, DateExpired: c.DateExpired, DateAccepted: c.DateAccepted,
			DateCompleted: c.DateCompleted, DaysToComplete: c.DaysToComplete, EndLocationID: c.EndLocationId,
			StartLocationID: c.StartLocationId, Buyout: c.Buyout, Collateral: c.Collateral,
			Price: c.Price, Reward: c.Reward, Volume: c.Volume, Title: c.Title,
		}
	}
	return out, totalPages, nil
}

type ListCorporationContractsInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	Page          int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCorporationContractsOutput struct {
	CharacterID   int32      `json:"characterId"`
	CorporationID int64      `json:"corporationId"`
	Page          int32      `json:"page"`
	TotalPages    int32      `json:"totalPages"`
	Contracts     []Contract `json:"contracts"`
}

// AddListCorporationContracts registers list_corporation_contracts. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListCorporationContracts[C corporationContractsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_corporation_contracts",
		Description: "List a corporation's contracts, using a logged-in character's roles to authorize.",
	}, listCorporationContractsHandler(open, store))
}

func listCorporationContractsHandler[C corporationContractsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[ListCorporationContractsInput, ListCorporationContractsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationContractsInput) (*mcpsdk.CallToolResult, ListCorporationContractsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationContractsOutput{}, err
		}
		if !found {
			return nil, ListCorporationContractsOutput{}, fmt.Errorf("list corporation contracts for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCorporationContractsOutput{}, fmt.Errorf("list corporation contracts: %w", err)
		}
		contracts, totalPages, err := listCorporationContracts(ctx, client, in.CorporationID, in.Page)
		if err != nil {
			return nil, ListCorporationContractsOutput{}, fmt.Errorf("list corporation contracts: %w", err)
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListCorporationContractsOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Page: page, TotalPages: totalPages, Contracts: contracts}, nil
	}
}

// --- list_corporation_contract_bids ---

// corporationContractBidsESI is the ESI call list_corporation_contract_bids makes.
type corporationContractBidsESI interface {
	CorporationContractBids(ctx context.Context, corporationID int64, contractID int64, page int32) ([]esiapi.CharactersCharacterIdContractsContractIdBidsGetInner, int32, error)
}

func listCorporationContractBids(ctx context.Context, client corporationContractBidsESI, corporationID int64, contractID int64, page int32) ([]ContractBid, int32, error) {
	bids, totalPages, err := client.CorporationContractBids(ctx, corporationID, contractID, page)
	if err != nil {
		return nil, 0, err
	}
	out := make([]ContractBid, len(bids))
	for i, b := range bids {
		bidderID := b.BidderId
		out[i] = ContractBid{BidID: b.BidId, BidderID: &bidderID, Amount: b.Amount, DateBid: b.DateBid}
	}
	return out, totalPages, nil
}

type ListCorporationContractBidsInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	ContractID    int64 `json:"contractId" jsonschema:"the contract ID, from list_corporation_contracts"`
	Page          int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCorporationContractBidsOutput struct {
	CharacterID   int32         `json:"characterId"`
	CorporationID int64         `json:"corporationId"`
	ContractID    int64         `json:"contractId"`
	Page          int32         `json:"page"`
	TotalPages    int32         `json:"totalPages"`
	Bids          []ContractBid `json:"bids"`
}

// AddListCorporationContractBids registers list_corporation_contract_bids. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListCorporationContractBids[C corporationContractBidsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_corporation_contract_bids",
		Description: "List the bids on one of a corporation's contracts, using a logged-in character's roles to authorize.",
	}, listCorporationContractBidsHandler(open, store))
}

func listCorporationContractBidsHandler[C corporationContractBidsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[ListCorporationContractBidsInput, ListCorporationContractBidsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationContractBidsInput) (*mcpsdk.CallToolResult, ListCorporationContractBidsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationContractBidsOutput{}, err
		}
		if !found {
			return nil, ListCorporationContractBidsOutput{}, fmt.Errorf("list corporation contract bids for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCorporationContractBidsOutput{}, fmt.Errorf("list corporation contract bids: %w", err)
		}
		bids, totalPages, err := listCorporationContractBids(ctx, client, in.CorporationID, in.ContractID, in.Page)
		if err != nil {
			return nil, ListCorporationContractBidsOutput{}, fmt.Errorf("list corporation contract bids: %w", err)
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListCorporationContractBidsOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, ContractID: in.ContractID, Page: page, TotalPages: totalPages, Bids: bids}, nil
	}
}

// --- list_corporation_contract_items ---

// corporationContractItemsESI is the ESI call list_corporation_contract_items makes.
type corporationContractItemsESI interface {
	CorporationContractItems(ctx context.Context, corporationID int64, contractID int64) ([]esiapi.CharactersCharacterIdContractsContractIdItemsGetInner, error)
}

func listCorporationContractItems(ctx context.Context, client corporationContractItemsESI, corporationID int64, contractID int64) ([]ContractItem, error) {
	items, err := client.CorporationContractItems(ctx, corporationID, contractID)
	if err != nil {
		return nil, err
	}
	out := make([]ContractItem, len(items))
	for i, it := range items {
		out[i] = ContractItem{RecordID: it.RecordId, TypeID: it.TypeId, Quantity: it.Quantity, IsIncluded: it.IsIncluded, IsSingleton: it.IsSingleton, RawQuantity: it.RawQuantity}
	}
	return out, nil
}

type ListCorporationContractItemsInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	ContractID    int64 `json:"contractId" jsonschema:"the contract ID, from list_corporation_contracts"`
}
type ListCorporationContractItemsOutput struct {
	CharacterID   int32          `json:"characterId"`
	CorporationID int64          `json:"corporationId"`
	ContractID    int64          `json:"contractId"`
	Items         []ContractItem `json:"items"`
}

// AddListCorporationContractItems registers list_corporation_contract_items. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListCorporationContractItems[C corporationContractItemsESI](s *mcpsdk.Server, userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore typeNameCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_corporation_contract_items",
		Description: "List the items on one of a corporation's contracts, using a logged-in character's roles to authorize.",
	}, listCorporationContractItemsHandler(userAgent, open, store, sdeStore))
}

func listCorporationContractItemsHandler[C corporationContractItemsESI](userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore typeNameCache) mcpsdk.ToolHandlerFor[ListCorporationContractItemsInput, ListCorporationContractItemsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationContractItemsInput) (*mcpsdk.CallToolResult, ListCorporationContractItemsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationContractItemsOutput{}, err
		}
		if !found {
			return nil, ListCorporationContractItemsOutput{}, fmt.Errorf("list corporation contract items for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCorporationContractItemsOutput{}, fmt.Errorf("list corporation contract items: %w", err)
		}
		items, err := listCorporationContractItems(ctx, client, in.CorporationID, in.ContractID)
		if err != nil {
			return nil, ListCorporationContractItemsOutput{}, fmt.Errorf("list corporation contract items: %w", err)
		}
		ids := make([]int64, len(items))
		for i, it := range items {
			ids[i] = it.TypeID
		}
		names := resolveTypeNames(ctx, sdeStore, userAgent, ids)
		for i := range items {
			items[i].TypeName = names[items[i].TypeID]
		}
		return nil, ListCorporationContractItemsOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, ContractID: in.ContractID, Items: items}, nil
	}
}

// --- list_public_contracts ---

// publicContractsESI is the public ESI call list_public_contracts makes;
// validate_trade_hub reads a hub's contracts through it too.
type publicContractsESI interface {
	PublicContracts(ctx context.Context, regionID int64, page int32) ([]esiapi.ContractsPublicRegionIdGetInner, int32, error)
}

func listPublicContracts(ctx context.Context, client publicContractsESI, regionID int64, page int32) ([]Contract, int32, error) {
	contracts, totalPages, err := client.PublicContracts(ctx, regionID, page)
	if err != nil {
		return nil, 0, err
	}
	out := make([]Contract, len(contracts))
	for i, c := range contracts {
		var forCorp bool
		if c.ForCorporation != nil {
			forCorp = *c.ForCorporation
		}
		out[i] = Contract{
			ContractID: c.ContractId, IssuerID: c.IssuerId, IssuerCorporationID: c.IssuerCorporationId,
			Type: c.Type, ForCorporation: forCorp, DateIssued: c.DateIssued, DateExpired: c.DateExpired,
			DaysToComplete: c.DaysToComplete, EndLocationID: c.EndLocationId, StartLocationID: c.StartLocationId,
			Buyout: c.Buyout, Collateral: c.Collateral, Price: c.Price, Reward: c.Reward, Volume: c.Volume, Title: c.Title,
		}
	}
	return out, totalPages, nil
}

type ListPublicContractsInput struct {
	RegionID int64 `json:"regionId" jsonschema:"the region ID to look up"`
	Page     int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListPublicContractsOutput struct {
	RegionID   int64      `json:"regionId"`
	Page       int32      `json:"page"`
	TotalPages int32      `json:"totalPages"`
	Contracts  []Contract `json:"contracts"`
}

// AddListPublicContracts registers list_public_contracts. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListPublicContracts[C publicContractsESI](s *mcpsdk.Server, open func() C) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_public_contracts",
		Description: "List public contracts in a region (public data, no character required).",
	}, listPublicContractsHandler(open))
}

func listPublicContractsHandler[C publicContractsESI](open func() C) mcpsdk.ToolHandlerFor[ListPublicContractsInput, ListPublicContractsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListPublicContractsInput) (*mcpsdk.CallToolResult, ListPublicContractsOutput, error) {
		contracts, totalPages, err := listPublicContracts(ctx, open(), in.RegionID, in.Page)
		if err != nil {
			return nil, ListPublicContractsOutput{}, fmt.Errorf("list public contracts: %w", err)
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListPublicContractsOutput{RegionID: in.RegionID, Page: page, TotalPages: totalPages, Contracts: contracts}, nil
	}
}

// --- list_public_contract_bids ---

// publicContractBidsESI is the public ESI call list_public_contract_bids makes.
type publicContractBidsESI interface {
	PublicContractBids(ctx context.Context, contractID int64, page int32) ([]esiapi.ContractsPublicBidsContractIdGetInner, int32, error)
}

func listPublicContractBids(ctx context.Context, client publicContractBidsESI, contractID int64, page int32) ([]ContractBid, int32, error) {
	bids, totalPages, err := client.PublicContractBids(ctx, contractID, page)
	if err != nil {
		return nil, 0, err
	}
	out := make([]ContractBid, len(bids))
	for i, b := range bids {
		out[i] = ContractBid{BidID: b.BidId, Amount: b.Amount, DateBid: b.DateBid}
	}
	return out, totalPages, nil
}

type ListPublicContractBidsInput struct {
	ContractID int64 `json:"contractId" jsonschema:"the contract ID, from list_public_contracts"`
	Page       int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListPublicContractBidsOutput struct {
	ContractID int64         `json:"contractId"`
	Page       int32         `json:"page"`
	TotalPages int32         `json:"totalPages"`
	Bids       []ContractBid `json:"bids"`
}

// AddListPublicContractBids registers list_public_contract_bids. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListPublicContractBids[C publicContractBidsESI](s *mcpsdk.Server, open func() C) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_public_contract_bids",
		Description: "List the bids on a public contract (public data, no character required).",
	}, listPublicContractBidsHandler(open))
}

func listPublicContractBidsHandler[C publicContractBidsESI](open func() C) mcpsdk.ToolHandlerFor[ListPublicContractBidsInput, ListPublicContractBidsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListPublicContractBidsInput) (*mcpsdk.CallToolResult, ListPublicContractBidsOutput, error) {
		bids, totalPages, err := listPublicContractBids(ctx, open(), in.ContractID, in.Page)
		if err != nil {
			return nil, ListPublicContractBidsOutput{}, fmt.Errorf("list public contract bids: %w", err)
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListPublicContractBidsOutput{ContractID: in.ContractID, Page: page, TotalPages: totalPages, Bids: bids}, nil
	}
}

// --- list_public_contract_items ---

// publicContractItemsESI is the public ESI call list_public_contract_items
// makes; validate_trade_hub reads contract items through it too.
type publicContractItemsESI interface {
	PublicContractItems(ctx context.Context, contractID int64, page int32) ([]esiapi.ContractsPublicItemsContractIdGetInner, int32, error)
}

func listPublicContractItems(ctx context.Context, client publicContractItemsESI, contractID int64, page int32) ([]ContractItem, int32, error) {
	items, totalPages, err := client.PublicContractItems(ctx, contractID, page)
	if err != nil {
		return nil, 0, err
	}
	out := make([]ContractItem, len(items))
	for i, it := range items {
		out[i] = ContractItem{
			RecordID: it.RecordId, TypeID: it.TypeId, Quantity: it.Quantity, IsIncluded: it.IsIncluded,
			ItemID: it.ItemId, MaterialEfficiency: it.MaterialEfficiency, Runs: it.Runs,
			TimeEfficiency: it.TimeEfficiency, IsBlueprintCopy: it.IsBlueprintCopy,
		}
	}
	return out, totalPages, nil
}

type ListPublicContractItemsInput struct {
	ContractID int64 `json:"contractId" jsonschema:"the contract ID, from list_public_contracts"`
	Page       int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListPublicContractItemsOutput struct {
	ContractID int64          `json:"contractId"`
	Page       int32          `json:"page"`
	TotalPages int32          `json:"totalPages"`
	Items      []ContractItem `json:"items"`
}

// AddListPublicContractItems registers list_public_contract_items. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListPublicContractItems[C publicContractItemsESI](s *mcpsdk.Server, userAgent string, open func() C, sdeStore typeNameCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_public_contract_items",
		Description: "List the items on a public contract (public data, no character required).",
	}, listPublicContractItemsHandler(userAgent, open, sdeStore))
}

func listPublicContractItemsHandler[C publicContractItemsESI](userAgent string, open func() C, sdeStore typeNameCache) mcpsdk.ToolHandlerFor[ListPublicContractItemsInput, ListPublicContractItemsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListPublicContractItemsInput) (*mcpsdk.CallToolResult, ListPublicContractItemsOutput, error) {
		items, totalPages, err := listPublicContractItems(ctx, open(), in.ContractID, in.Page)
		if err != nil {
			return nil, ListPublicContractItemsOutput{}, fmt.Errorf("list public contract items: %w", err)
		}
		ids := make([]int64, len(items))
		for i, it := range items {
			ids[i] = it.TypeID
		}
		names := resolveTypeNames(ctx, sdeStore, userAgent, ids)
		for i := range items {
			items[i].TypeName = names[items[i].TypeID]
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListPublicContractItemsOutput{ContractID: in.ContractID, Page: page, TotalPages: totalPages, Items: items}, nil
	}
}
