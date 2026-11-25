package mcp

import (
	"context"
	"fmt"
	"time"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// killmailsESI is the ESI calls the recent-killmail listing tools make.
type killmailsESI interface {
	CharacterKillmailsRecent(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdKillmailsRecentGetInner, int32, error)
	CorporationKillmailsRecent(ctx context.Context, corporationID int64, page int32) ([]esiapi.CharactersCharacterIdKillmailsRecentGetInner, int32, error)
}

// killmailDetailESI is the ESI call get_killmail makes; it is public.
type killmailDetailESI interface {
	KillmailDetail(ctx context.Context, killmailID int64, killmailHash string) (*esiapi.KillmailsKillmailIdKillmailHashGet, error)
}

// killmailNameCache is the SDE lookups get_killmail makes: type, system, moon and faction names.
type killmailNameCache interface {
	typeNameCache
	systemNameCache
	moonNameCache
	factionNameCache
}

// KillmailRef is shared by list_character_killmails_recent and list_corporation_killmails_recent
// since both endpoints return the exact same generated model.
type KillmailRef struct {
	KillmailID   int64  `json:"killmailId"`
	KillmailHash string `json:"killmailHash"`
}

func toKillmailRefs(refs []esiapi.CharactersCharacterIdKillmailsRecentGetInner) []KillmailRef {
	out := make([]KillmailRef, len(refs))
	for i, r := range refs {
		out[i] = KillmailRef{KillmailID: r.KillmailId, KillmailHash: r.KillmailHash}
	}
	return out
}

// --- list_character_killmails_recent (private, paginated) ---

func listCharacterKillmailsRecent(ctx context.Context, client killmailsESI, page int32) ([]esiapi.CharactersCharacterIdKillmailsRecentGetInner, int32, error) {
	return client.CharacterKillmailsRecent(ctx, page)
}

type ListCharacterKillmailsRecentInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
	Page        int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCharacterKillmailsRecentOutput struct {
	CharacterID int32         `json:"characterId"`
	Page        int32         `json:"page"`
	TotalPages  int32         `json:"totalPages"`
	Killmails   []KillmailRef `json:"killmails"`
}

// AddListCharacterKillmailsRecent registers list_character_killmails_recent. open builds the ESI client; any client with the
// killmailsESI methods will do.
func AddListCharacterKillmailsRecent[C killmailsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_character_killmails_recent",
		Description: "List a logged-in EVE Online character's recent killmail references.",
	}, listCharacterKillmailsRecentHandler(open, store))
}

func listCharacterKillmailsRecentHandler[C killmailsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[ListCharacterKillmailsRecentInput, ListCharacterKillmailsRecentOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterKillmailsRecentInput) (*mcpsdk.CallToolResult, ListCharacterKillmailsRecentOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterKillmailsRecentOutput{}, err
		}
		if !found {
			return nil, ListCharacterKillmailsRecentOutput{}, fmt.Errorf("list character killmails recent for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCharacterKillmailsRecentOutput{}, fmt.Errorf("list character killmails recent: %w", err)
		}
		refs, totalPages, err := listCharacterKillmailsRecent(ctx, client, in.Page)
		if err != nil {
			return nil, ListCharacterKillmailsRecentOutput{}, fmt.Errorf("list character killmails recent: %w", err)
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListCharacterKillmailsRecentOutput{CharacterID: in.CharacterID, Page: page, TotalPages: totalPages, Killmails: toKillmailRefs(refs)}, nil
	}
}

// --- list_corporation_killmails_recent (private, corp-scoped, paginated) ---

func listCorporationKillmailsRecent(ctx context.Context, client killmailsESI, corporationID int64, page int32) ([]esiapi.CharactersCharacterIdKillmailsRecentGetInner, int32, error) {
	return client.CorporationKillmailsRecent(ctx, corporationID, page)
}

type ListCorporationKillmailsRecentInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	Page          int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCorporationKillmailsRecentOutput struct {
	CharacterID   int32         `json:"characterId"`
	CorporationID int64         `json:"corporationId"`
	Page          int32         `json:"page"`
	TotalPages    int32         `json:"totalPages"`
	Killmails     []KillmailRef `json:"killmails"`
}

// AddListCorporationKillmailsRecent registers list_corporation_killmails_recent. open builds the ESI client; any client with the
// killmailsESI methods will do.
func AddListCorporationKillmailsRecent[C killmailsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_corporation_killmails_recent",
		Description: "List a corporation's recent killmail references, using a logged-in character's roles to authorize.",
	}, listCorporationKillmailsRecentHandler(open, store))
}

func listCorporationKillmailsRecentHandler[C killmailsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[ListCorporationKillmailsRecentInput, ListCorporationKillmailsRecentOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationKillmailsRecentInput) (*mcpsdk.CallToolResult, ListCorporationKillmailsRecentOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationKillmailsRecentOutput{}, err
		}
		if !found {
			return nil, ListCorporationKillmailsRecentOutput{}, fmt.Errorf("list corporation killmails recent for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCorporationKillmailsRecentOutput{}, fmt.Errorf("list corporation killmails recent: %w", err)
		}
		refs, totalPages, err := listCorporationKillmailsRecent(ctx, client, in.CorporationID, in.Page)
		if err != nil {
			return nil, ListCorporationKillmailsRecentOutput{}, fmt.Errorf("list corporation killmails recent: %w", err)
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListCorporationKillmailsRecentOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Page: page, TotalPages: totalPages, Killmails: toKillmailRefs(refs)}, nil
	}
}

// --- get_killmail (public) ---

type KillmailAttacker struct {
	AllianceID     *int64  `json:"allianceId,omitempty"`
	CharacterID    *int64  `json:"characterId,omitempty"`
	CorporationID  *int64  `json:"corporationId,omitempty"`
	DamageDone     int64   `json:"damageDone"`
	FactionID      *int64  `json:"factionId,omitempty"`
	FactionName    string  `json:"factionName,omitempty"`
	FinalBlow      bool    `json:"finalBlow"`
	SecurityStatus float64 `json:"securityStatus"`
	ShipTypeID     *int64  `json:"shipTypeId,omitempty"`
	ShipTypeName   string  `json:"shipTypeName,omitempty"`
	WeaponTypeID   *int64  `json:"weaponTypeId,omitempty"`
	WeaponTypeName string  `json:"weaponTypeName,omitempty"`
}

// KillmailInnerItem mirrors the innermost generated type (items destroyed inside a destroyed
// container), which has no further nested items field.
//
// NOTE: KillmailItem/KillmailInnerItem were originally designed as a single self-referential
// type, matching ESI's one-level-deep nesting with a recursive Go type. That shape compiles and
// vets cleanly but panics at server startup ("cycle detected") in the real go-sdk's output-schema
// generator, which cannot represent a self-referential struct as JSON Schema — go build/vet/gofmt
// don't catch this since it's a runtime failure in mcpsdk.AddTool's reflection-based schema
// walk, not a compile error. Two distinct, non-recursive types (this one plus KillmailItem below)
// avoid the cycle while still mirroring ESI's actual one-level-deep nesting exactly.
type KillmailInnerItem struct {
	Flag              int64  `json:"flag"`
	ItemTypeID        int64  `json:"itemTypeId"`
	ItemTypeName      string `json:"itemTypeName,omitempty"`
	Singleton         int64  `json:"singleton"`
	QuantityDestroyed *int64 `json:"quantityDestroyed,omitempty"`
	QuantityDropped   *int64 `json:"quantityDropped,omitempty"`
}

// KillmailItem is the outer container-item level; its Items field holds the innermost items
// (containers cannot themselves contain further nested containers per ESI's own schema).
type KillmailItem struct {
	Flag              int64               `json:"flag"`
	ItemTypeID        int64               `json:"itemTypeId"`
	ItemTypeName      string              `json:"itemTypeName,omitempty"`
	Singleton         int64               `json:"singleton"`
	QuantityDestroyed *int64              `json:"quantityDestroyed,omitempty"`
	QuantityDropped   *int64              `json:"quantityDropped,omitempty"`
	Items             []KillmailInnerItem `json:"items,omitempty"`
}

type KillmailPosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type KillmailVictim struct {
	AllianceID    *int64            `json:"allianceId,omitempty"`
	CharacterID   *int64            `json:"characterId,omitempty"`
	CorporationID *int64            `json:"corporationId,omitempty"`
	DamageTaken   int64             `json:"damageTaken"`
	FactionID     *int64            `json:"factionId,omitempty"`
	FactionName   string            `json:"factionName,omitempty"`
	ShipTypeID    int64             `json:"shipTypeId"`
	ShipTypeName  string            `json:"shipTypeName,omitempty"`
	Items         []KillmailItem    `json:"items,omitempty"`
	Position      *KillmailPosition `json:"position,omitempty"`
}

func toKillmailInnerItems(items []esiapi.KillmailsKillmailIdKillmailHashGetVictimItemsInnerItemsInner) []KillmailInnerItem {
	out := make([]KillmailInnerItem, len(items))
	for i, it := range items {
		out[i] = KillmailInnerItem{
			Flag: it.Flag, ItemTypeID: it.ItemTypeId, Singleton: it.Singleton,
			QuantityDestroyed: it.QuantityDestroyed, QuantityDropped: it.QuantityDropped,
		}
	}
	return out
}

func toKillmailItems(items []esiapi.KillmailsKillmailIdKillmailHashGetVictimItemsInner) []KillmailItem {
	out := make([]KillmailItem, len(items))
	for i, it := range items {
		out[i] = KillmailItem{
			Flag: it.Flag, ItemTypeID: it.ItemTypeId, Singleton: it.Singleton,
			QuantityDestroyed: it.QuantityDestroyed, QuantityDropped: it.QuantityDropped,
			Items: toKillmailInnerItems(it.Items),
		}
	}
	return out
}

func getKillmail(ctx context.Context, client killmailDetailESI, killmailID int64, killmailHash string) (*esiapi.KillmailsKillmailIdKillmailHashGet, error) {
	return client.KillmailDetail(ctx, killmailID, killmailHash)
}

type GetKillmailInput struct {
	KillmailID   int64  `json:"killmailId" jsonschema:"the killmail ID"`
	KillmailHash string `json:"killmailHash" jsonschema:"the killmail hash, from a killmail listing"`
}
type GetKillmailOutput struct {
	KillmailID      int64              `json:"killmailId"`
	KillmailTime    time.Time          `json:"killmailTime"`
	SolarSystemID   int64              `json:"solarSystemId"`
	SolarSystemName string             `json:"solarSystemName,omitempty"`
	MoonID          *int64             `json:"moonId,omitempty"`
	MoonName        string             `json:"moonName,omitempty"`
	WarID           *int64             `json:"warId,omitempty"`
	Attackers       []KillmailAttacker `json:"attackers"`
	Victim          KillmailVictim     `json:"victim"`
}

// AddGetKillmail registers get_killmail. open builds the ESI client; any client with the
// killmailDetailESI methods will do.
func AddGetKillmail[C killmailDetailESI](s *mcpsdk.Server, userAgent string, open func() C, sdeStore killmailNameCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_killmail",
		Description: "Get the full details of a killmail, given its ID and hash (public data, no character required).",
	}, getKillmailHandler(userAgent, open, sdeStore))
}

func getKillmailHandler[C killmailDetailESI](userAgent string, open func() C, sdeStore killmailNameCache) mcpsdk.ToolHandlerFor[GetKillmailInput, GetKillmailOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetKillmailInput) (*mcpsdk.CallToolResult, GetKillmailOutput, error) {
		detail, err := getKillmail(ctx, open(), in.KillmailID, in.KillmailHash)
		if err != nil {
			return nil, GetKillmailOutput{}, fmt.Errorf("get killmail: %w", err)
		}
		attackers := make([]KillmailAttacker, len(detail.Attackers))
		for i, a := range detail.Attackers {
			attackers[i] = KillmailAttacker{
				AllianceID: a.AllianceId, CharacterID: a.CharacterId, CorporationID: a.CorporationId,
				DamageDone: a.DamageDone, FactionID: a.FactionId, FinalBlow: a.FinalBlow,
				SecurityStatus: a.SecurityStatus, ShipTypeID: a.ShipTypeId, WeaponTypeID: a.WeaponTypeId,
			}
		}
		var position *KillmailPosition
		if detail.Victim.Position != nil {
			position = &KillmailPosition{X: detail.Victim.Position.X, Y: detail.Victim.Position.Y, Z: detail.Victim.Position.Z}
		}
		victim := KillmailVictim{
			AllianceID: detail.Victim.AllianceId, CharacterID: detail.Victim.CharacterId,
			CorporationID: detail.Victim.CorporationId, DamageTaken: detail.Victim.DamageTaken,
			FactionID: detail.Victim.FactionId, ShipTypeID: detail.Victim.ShipTypeId,
			Items: toKillmailItems(detail.Victim.Items), Position: position,
		}

		ids := make([]int64, 0)
		for _, a := range attackers {
			if a.ShipTypeID != nil {
				ids = append(ids, *a.ShipTypeID)
			}
			if a.WeaponTypeID != nil {
				ids = append(ids, *a.WeaponTypeID)
			}
		}
		ids = append(ids, victim.ShipTypeID)
		for _, it := range victim.Items {
			ids = append(ids, it.ItemTypeID)
			for _, inner := range it.Items {
				ids = append(ids, inner.ItemTypeID)
			}
		}
		names := resolveTypeNames(ctx, sdeStore, userAgent, ids)
		for i := range attackers {
			if attackers[i].ShipTypeID != nil {
				attackers[i].ShipTypeName = names[*attackers[i].ShipTypeID]
			}
			if attackers[i].WeaponTypeID != nil {
				attackers[i].WeaponTypeName = names[*attackers[i].WeaponTypeID]
			}
		}
		victim.ShipTypeName = names[victim.ShipTypeID]
		for i := range victim.Items {
			victim.Items[i].ItemTypeName = names[victim.Items[i].ItemTypeID]
			for j := range victim.Items[i].Items {
				victim.Items[i].Items[j].ItemTypeName = names[victim.Items[i].Items[j].ItemTypeID]
			}
		}

		systemNames := resolveSystemNames(ctx, sdeStore, userAgent, []int64{detail.SolarSystemId})
		var moonNames map[int64]string
		if detail.MoonId != nil {
			moonNames = resolveMoonNames(ctx, sdeStore, userAgent, []int64{*detail.MoonId})
		}
		var moonName string
		if detail.MoonId != nil {
			moonName = moonNames[*detail.MoonId]
		}

		var factionIDs []int64
		for _, a := range attackers {
			if a.FactionID != nil {
				factionIDs = append(factionIDs, *a.FactionID)
			}
		}
		if victim.FactionID != nil {
			factionIDs = append(factionIDs, *victim.FactionID)
		}
		factionNames := resolveFactionNames(ctx, sdeStore, userAgent, factionIDs)
		for i := range attackers {
			if attackers[i].FactionID != nil {
				attackers[i].FactionName = factionNames[*attackers[i].FactionID]
			}
		}
		if victim.FactionID != nil {
			victim.FactionName = factionNames[*victim.FactionID]
		}

		return nil, GetKillmailOutput{
			KillmailID: detail.KillmailId, KillmailTime: detail.KillmailTime, SolarSystemID: detail.SolarSystemId,
			SolarSystemName: systemNames[detail.SolarSystemId],
			MoonID:          detail.MoonId, MoonName: moonName, WarID: detail.WarId, Attackers: attackers, Victim: victim,
		}, nil
	}
}
