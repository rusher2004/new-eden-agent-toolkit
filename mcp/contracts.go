package mcp

import (
	"context"
	"fmt"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
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

var listCharacterContracts = func(ctx context.Context, clientID string, characterID int32, userAgent string, page int32) ([]Contract, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
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

func listCharacterContractsHandler(version, clientID string, store ConfigStore) mcpsdk.ToolHandlerFor[ListCharacterContractsInput, ListCharacterContractsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterContractsInput) (*mcpsdk.CallToolResult, ListCharacterContractsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterContractsOutput{}, err
		}
		if !found {
			return nil, ListCharacterContractsOutput{}, fmt.Errorf("list character contracts for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		contracts, totalPages, err := listCharacterContracts(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.Page)
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

var listCharacterContractBids = func(ctx context.Context, clientID string, characterID int32, userAgent string, contractID int64) ([]ContractBid, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
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

func listCharacterContractBidsHandler(version, clientID string, store ConfigStore) mcpsdk.ToolHandlerFor[ListCharacterContractBidsInput, ListCharacterContractBidsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterContractBidsInput) (*mcpsdk.CallToolResult, ListCharacterContractBidsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterContractBidsOutput{}, err
		}
		if !found {
			return nil, ListCharacterContractBidsOutput{}, fmt.Errorf("list character contract bids for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		bids, err := listCharacterContractBids(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.ContractID)
		if err != nil {
			return nil, ListCharacterContractBidsOutput{}, fmt.Errorf("list character contract bids: %w", err)
		}
		return nil, ListCharacterContractBidsOutput{CharacterID: in.CharacterID, ContractID: in.ContractID, Bids: bids}, nil
	}
}

// --- list_character_contract_items ---

var listCharacterContractItems = func(ctx context.Context, clientID string, characterID int32, userAgent string, contractID int64) ([]ContractItem, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
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

func listCharacterContractItemsHandler(version, clientID string, store ConfigStore, sdeStore SDEStore) mcpsdk.ToolHandlerFor[ListCharacterContractItemsInput, ListCharacterContractItemsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterContractItemsInput) (*mcpsdk.CallToolResult, ListCharacterContractItemsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterContractItemsOutput{}, err
		}
		if !found {
			return nil, ListCharacterContractItemsOutput{}, fmt.Errorf("list character contract items for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		items, err := listCharacterContractItems(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.ContractID)
		if err != nil {
			return nil, ListCharacterContractItemsOutput{}, fmt.Errorf("list character contract items: %w", err)
		}
		ids := make([]int64, len(items))
		for i, it := range items {
			ids[i] = it.TypeID
		}
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), ids)
		for i := range items {
			items[i].TypeName = names[items[i].TypeID]
		}
		return nil, ListCharacterContractItemsOutput{CharacterID: in.CharacterID, ContractID: in.ContractID, Items: items}, nil
	}
}

// --- list_corporation_contracts ---

var listCorporationContracts = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]Contract, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
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

func listCorporationContractsHandler(version, clientID string, store ConfigStore) mcpsdk.ToolHandlerFor[ListCorporationContractsInput, ListCorporationContractsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationContractsInput) (*mcpsdk.CallToolResult, ListCorporationContractsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationContractsOutput{}, err
		}
		if !found {
			return nil, ListCorporationContractsOutput{}, fmt.Errorf("list corporation contracts for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		contracts, totalPages, err := listCorporationContracts(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.Page)
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

var listCorporationContractBids = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, contractID int64, page int32) ([]ContractBid, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
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

func listCorporationContractBidsHandler(version, clientID string, store ConfigStore) mcpsdk.ToolHandlerFor[ListCorporationContractBidsInput, ListCorporationContractBidsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationContractBidsInput) (*mcpsdk.CallToolResult, ListCorporationContractBidsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationContractBidsOutput{}, err
		}
		if !found {
			return nil, ListCorporationContractBidsOutput{}, fmt.Errorf("list corporation contract bids for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		bids, totalPages, err := listCorporationContractBids(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.ContractID, in.Page)
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

var listCorporationContractItems = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, contractID int64) ([]ContractItem, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
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

func listCorporationContractItemsHandler(version, clientID string, store ConfigStore, sdeStore SDEStore) mcpsdk.ToolHandlerFor[ListCorporationContractItemsInput, ListCorporationContractItemsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationContractItemsInput) (*mcpsdk.CallToolResult, ListCorporationContractItemsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationContractItemsOutput{}, err
		}
		if !found {
			return nil, ListCorporationContractItemsOutput{}, fmt.Errorf("list corporation contract items for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		items, err := listCorporationContractItems(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.ContractID)
		if err != nil {
			return nil, ListCorporationContractItemsOutput{}, fmt.Errorf("list corporation contract items: %w", err)
		}
		ids := make([]int64, len(items))
		for i, it := range items {
			ids[i] = it.TypeID
		}
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), ids)
		for i := range items {
			items[i].TypeName = names[items[i].TypeID]
		}
		return nil, ListCorporationContractItemsOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, ContractID: in.ContractID, Items: items}, nil
	}
}

// --- list_public_contracts ---

var listPublicContracts = func(ctx context.Context, userAgent string, regionID int64, page int32) ([]Contract, int32, error) {
	client := esi.NewPublicClient(userAgent)
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

func listPublicContractsHandler(version string) mcpsdk.ToolHandlerFor[ListPublicContractsInput, ListPublicContractsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListPublicContractsInput) (*mcpsdk.CallToolResult, ListPublicContractsOutput, error) {
		contracts, totalPages, err := listPublicContracts(ctx, sso.UserAgent(version), in.RegionID, in.Page)
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

var listPublicContractBids = func(ctx context.Context, userAgent string, contractID int64, page int32) ([]ContractBid, int32, error) {
	client := esi.NewPublicClient(userAgent)
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

func listPublicContractBidsHandler(version string) mcpsdk.ToolHandlerFor[ListPublicContractBidsInput, ListPublicContractBidsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListPublicContractBidsInput) (*mcpsdk.CallToolResult, ListPublicContractBidsOutput, error) {
		bids, totalPages, err := listPublicContractBids(ctx, sso.UserAgent(version), in.ContractID, in.Page)
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

var listPublicContractItems = func(ctx context.Context, userAgent string, contractID int64, page int32) ([]ContractItem, int32, error) {
	client := esi.NewPublicClient(userAgent)
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

func listPublicContractItemsHandler(version string, sdeStore SDEStore) mcpsdk.ToolHandlerFor[ListPublicContractItemsInput, ListPublicContractItemsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListPublicContractItemsInput) (*mcpsdk.CallToolResult, ListPublicContractItemsOutput, error) {
		items, totalPages, err := listPublicContractItems(ctx, sso.UserAgent(version), in.ContractID, in.Page)
		if err != nil {
			return nil, ListPublicContractItemsOutput{}, fmt.Errorf("list public contract items: %w", err)
		}
		ids := make([]int64, len(items))
		for i, it := range items {
			ids[i] = it.TypeID
		}
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), ids)
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
