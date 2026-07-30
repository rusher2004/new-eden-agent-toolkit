package mcp

import (
	"context"
	"fmt"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

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

var listCharacterKillmailsRecent = func(ctx context.Context, clientID string, characterID int32, userAgent string, page int32) ([]esiapi.CharactersCharacterIdKillmailsRecentGetInner, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
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

func listCharacterKillmailsRecentHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCharacterKillmailsRecentInput, ListCharacterKillmailsRecentOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterKillmailsRecentInput) (*mcpsdk.CallToolResult, ListCharacterKillmailsRecentOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterKillmailsRecentOutput{}, err
		}
		if !found {
			return nil, ListCharacterKillmailsRecentOutput{}, fmt.Errorf("list character killmails recent for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		refs, totalPages, err := listCharacterKillmailsRecent(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.Page)
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

var listCorporationKillmailsRecent = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CharactersCharacterIdKillmailsRecentGetInner, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
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

func listCorporationKillmailsRecentHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationKillmailsRecentInput, ListCorporationKillmailsRecentOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationKillmailsRecentInput) (*mcpsdk.CallToolResult, ListCorporationKillmailsRecentOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationKillmailsRecentOutput{}, err
		}
		if !found {
			return nil, ListCorporationKillmailsRecentOutput{}, fmt.Errorf("list corporation killmails recent for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		refs, totalPages, err := listCorporationKillmailsRecent(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.Page)
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

var getKillmail = func(ctx context.Context, userAgent string, killmailID int64, killmailHash string) (*esiapi.KillmailsKillmailIdKillmailHashGet, error) {
	client := esi.NewPublicClient(userAgent)
	return client.KillmailDetail(ctx, killmailID, killmailHash)
}

type GetKillmailInput struct {
	KillmailID   int64  `json:"killmailId" jsonschema:"the killmail ID"`
	KillmailHash string `json:"killmailHash" jsonschema:"the killmail hash, from a killmail listing"`
}
type GetKillmailOutput struct {
	KillmailID    int64              `json:"killmailId"`
	KillmailTime  time.Time          `json:"killmailTime"`
	SolarSystemID int64              `json:"solarSystemId"`
	MoonID        *int64             `json:"moonId,omitempty"`
	WarID         *int64             `json:"warId,omitempty"`
	Attackers     []KillmailAttacker `json:"attackers"`
	Victim        KillmailVictim     `json:"victim"`
}

func getKillmailHandler(version string, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[GetKillmailInput, GetKillmailOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetKillmailInput) (*mcpsdk.CallToolResult, GetKillmailOutput, error) {
		detail, err := getKillmail(ctx, sso.UserAgent(version), in.KillmailID, in.KillmailHash)
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
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), ids)
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

		return nil, GetKillmailOutput{
			KillmailID: detail.KillmailId, KillmailTime: detail.KillmailTime, SolarSystemID: detail.SolarSystemId,
			MoonID: detail.MoonId, WarID: detail.WarId, Attackers: attackers, Victim: victim,
		}, nil
	}
}
