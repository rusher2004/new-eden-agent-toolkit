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

func TestListCharacterKillmailsRecentHandler_MapsFieldsAndDefaultsPageTo1(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterKillmailsRecent
	t.Cleanup(func() { listCharacterKillmailsRecent = orig })
	listCharacterKillmailsRecent = func(ctx context.Context, clientID string, characterID int32, userAgent string, page int32) ([]esiapi.CharactersCharacterIdKillmailsRecentGetInner, int32, error) {
		return []esiapi.CharactersCharacterIdKillmailsRecentGetInner{{
			KillmailId: 123, KillmailHash: "abc123hash",
		}}, 5, nil
	}

	handler := listCharacterKillmailsRecentHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCharacterKillmailsRecentInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.Page != 1 || out.TotalPages != 5 || len(out.Killmails) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 Page=1 TotalPages=5 with one killmail", out)
	}
	k := out.Killmails[0]
	if k.KillmailID != 123 || k.KillmailHash != "abc123hash" {
		t.Errorf("killmail = %+v, want KillmailID=123 KillmailHash=abc123hash", k)
	}
}

func TestListCharacterKillmailsRecentHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCharacterKillmailsRecent
	t.Cleanup(func() { listCharacterKillmailsRecent = orig })
	listCharacterKillmailsRecent = func(ctx context.Context, clientID string, characterID int32, userAgent string, page int32) ([]esiapi.CharactersCharacterIdKillmailsRecentGetInner, int32, error) {
		t.Fatal("listCharacterKillmailsRecent should not be called for an unknown character")
		return nil, 0, nil
	}

	handler := listCharacterKillmailsRecentHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterKillmailsRecentInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationKillmailsRecentHandler_MapsFieldsAndDefaultsPageTo1(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationKillmailsRecent
	t.Cleanup(func() { listCorporationKillmailsRecent = orig })
	listCorporationKillmailsRecent = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CharactersCharacterIdKillmailsRecentGetInner, int32, error) {
		return []esiapi.CharactersCharacterIdKillmailsRecentGetInner{{
			KillmailId: 456, KillmailHash: "def456hash",
		}}, 3, nil
	}

	handler := listCorporationKillmailsRecentHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCorporationKillmailsRecentInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.CorporationID != 98000001 || out.Page != 1 || out.TotalPages != 3 || len(out.Killmails) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 CorporationID=98000001 Page=1 TotalPages=3 with one killmail", out)
	}
	k := out.Killmails[0]
	if k.KillmailID != 456 || k.KillmailHash != "def456hash" {
		t.Errorf("killmail = %+v, want KillmailID=456 KillmailHash=def456hash", k)
	}
}

func TestListCorporationKillmailsRecentHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCorporationKillmailsRecent
	t.Cleanup(func() { listCorporationKillmailsRecent = orig })
	listCorporationKillmailsRecent = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CharactersCharacterIdKillmailsRecentGetInner, int32, error) {
		t.Fatal("listCorporationKillmailsRecent should not be called for an unknown character")
		return nil, 0, nil
	}

	handler := listCorporationKillmailsRecentHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationKillmailsRecentInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestGetKillmailHandler_MapsFields(t *testing.T) {
	orig := getKillmail
	t.Cleanup(func() { getKillmail = orig })

	killmailTime := time.Unix(1000, 0)
	moonID := int64(40000001)
	warID := int64(500)

	attackerAllianceID := int64(1)
	attackerCharacterID := int64(2)
	attackerCorporationID := int64(3)
	attackerFactionID := int64(4)
	attackerShipTypeID := int64(5)
	attackerWeaponTypeID := int64(6)

	victimAllianceID := int64(10)
	victimCharacterID := int64(20)
	victimCorporationID := int64(30)
	victimFactionID := int64(40)

	outerQuantityDestroyed := int64(100)
	outerQuantityDropped := int64(200)
	innerQuantityDestroyed := int64(300)
	innerQuantityDropped := int64(400)

	getKillmail = func(ctx context.Context, userAgent string, killmailID int64, killmailHash string) (*esiapi.KillmailsKillmailIdKillmailHashGet, error) {
		return &esiapi.KillmailsKillmailIdKillmailHashGet{
			KillmailId:    999,
			KillmailTime:  killmailTime,
			SolarSystemId: 30000142,
			MoonId:        &moonID,
			WarId:         &warID,
			Attackers: []esiapi.KillmailsKillmailIdKillmailHashGetAttackersInner{{
				AllianceId: &attackerAllianceID, CharacterId: &attackerCharacterID, CorporationId: &attackerCorporationID,
				DamageDone: 5000, FactionId: &attackerFactionID, FinalBlow: true, SecurityStatus: -1.5,
				ShipTypeId: &attackerShipTypeID, WeaponTypeId: &attackerWeaponTypeID,
			}},
			Victim: esiapi.KillmailsKillmailIdKillmailHashGetVictim{
				AllianceId: &victimAllianceID, CharacterId: &victimCharacterID, CorporationId: &victimCorporationID,
				DamageTaken: 12000, FactionId: &victimFactionID, ShipTypeId: 670,
				Items: []esiapi.KillmailsKillmailIdKillmailHashGetVictimItemsInner{{
					Flag: 1, ItemTypeId: 2, Singleton: 0,
					QuantityDestroyed: &outerQuantityDestroyed, QuantityDropped: &outerQuantityDropped,
					Items: []esiapi.KillmailsKillmailIdKillmailHashGetVictimItemsInnerItemsInner{{
						Flag: 3, ItemTypeId: 4, Singleton: 1,
						QuantityDestroyed: &innerQuantityDestroyed, QuantityDropped: &innerQuantityDropped,
					}},
				}},
				Position: &esiapi.KillmailsKillmailIdKillmailHashGetVictimPosition{X: 1.1, Y: 2.2, Z: 3.3},
			},
		}, nil
	}

	handler := getKillmailHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, GetKillmailInput{KillmailID: 999, KillmailHash: "somehash"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if out.KillmailID != 999 || !out.KillmailTime.Equal(killmailTime) || out.SolarSystemID != 30000142 ||
		out.MoonID == nil || *out.MoonID != moonID || out.WarID == nil || *out.WarID != warID {
		t.Fatalf("out = %+v, want KillmailID=999 KillmailTime=%v SolarSystemID=30000142 MoonID=%d WarID=%d", out, killmailTime, moonID, warID)
	}

	if len(out.Attackers) != 1 {
		t.Fatalf("out.Attackers = %+v, want one attacker", out.Attackers)
	}
	a := out.Attackers[0]
	if a.AllianceID == nil || *a.AllianceID != attackerAllianceID ||
		a.CharacterID == nil || *a.CharacterID != attackerCharacterID ||
		a.CorporationID == nil || *a.CorporationID != attackerCorporationID ||
		a.DamageDone != 5000 ||
		a.FactionID == nil || *a.FactionID != attackerFactionID ||
		!a.FinalBlow || a.SecurityStatus != -1.5 ||
		a.ShipTypeID == nil || *a.ShipTypeID != attackerShipTypeID ||
		a.WeaponTypeID == nil || *a.WeaponTypeID != attackerWeaponTypeID {
		t.Errorf("attacker = %+v, want AllianceID=%d CharacterID=%d CorporationID=%d DamageDone=5000 FactionID=%d FinalBlow=true SecurityStatus=-1.5 ShipTypeID=%d WeaponTypeID=%d",
			a, attackerAllianceID, attackerCharacterID, attackerCorporationID, attackerFactionID, attackerShipTypeID, attackerWeaponTypeID)
	}

	v := out.Victim
	if v.AllianceID == nil || *v.AllianceID != victimAllianceID ||
		v.CharacterID == nil || *v.CharacterID != victimCharacterID ||
		v.CorporationID == nil || *v.CorporationID != victimCorporationID ||
		v.DamageTaken != 12000 ||
		v.FactionID == nil || *v.FactionID != victimFactionID ||
		v.ShipTypeID != 670 {
		t.Errorf("victim = %+v, want AllianceID=%d CharacterID=%d CorporationID=%d DamageTaken=12000 FactionID=%d ShipTypeID=670",
			v, victimAllianceID, victimCharacterID, victimCorporationID, victimFactionID)
	}
	if v.Position == nil || v.Position.X != 1.1 || v.Position.Y != 2.2 || v.Position.Z != 3.3 {
		t.Errorf("victim.Position = %+v, want X=1.1 Y=2.2 Z=3.3", v.Position)
	}

	if len(v.Items) != 1 {
		t.Fatalf("victim.Items = %+v, want one item", v.Items)
	}
	outer := v.Items[0]
	if outer.Flag != 1 || outer.ItemTypeID != 2 || outer.Singleton != 0 ||
		outer.QuantityDestroyed == nil || *outer.QuantityDestroyed != outerQuantityDestroyed ||
		outer.QuantityDropped == nil || *outer.QuantityDropped != outerQuantityDropped {
		t.Errorf("outer item = %+v, want Flag=1 ItemTypeID=2 Singleton=0 QuantityDestroyed=%d QuantityDropped=%d",
			outer, outerQuantityDestroyed, outerQuantityDropped)
	}
	if len(outer.Items) != 1 {
		t.Fatalf("outer.Items = %+v, want one nested item", outer.Items)
	}
	inner := outer.Items[0]
	if inner.Flag != 3 || inner.ItemTypeID != 4 || inner.Singleton != 1 ||
		inner.QuantityDestroyed == nil || *inner.QuantityDestroyed != innerQuantityDestroyed ||
		inner.QuantityDropped == nil || *inner.QuantityDropped != innerQuantityDropped {
		t.Errorf("inner item = %+v, want Flag=3 ItemTypeID=4 Singleton=1 QuantityDestroyed=%d QuantityDropped=%d",
			inner, innerQuantityDestroyed, innerQuantityDropped)
	}
}

func TestGetKillmailHandler_HandlesNilOptionalFields(t *testing.T) {
	orig := getKillmail
	t.Cleanup(func() { getKillmail = orig })

	killmailTime := time.Unix(2000, 0)

	getKillmail = func(ctx context.Context, userAgent string, killmailID int64, killmailHash string) (*esiapi.KillmailsKillmailIdKillmailHashGet, error) {
		return &esiapi.KillmailsKillmailIdKillmailHashGet{
			KillmailId:    111,
			KillmailTime:  killmailTime,
			SolarSystemId: 30000142,
			MoonId:        nil,
			WarId:         nil,
			Attackers: []esiapi.KillmailsKillmailIdKillmailHashGetAttackersInner{{
				DamageDone: 100, FinalBlow: false, SecurityStatus: 0,
			}},
			Victim: esiapi.KillmailsKillmailIdKillmailHashGetVictim{
				DamageTaken: 200, ShipTypeId: 587,
				Items:    nil,
				Position: nil,
			},
		}, nil
	}

	handler := getKillmailHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, GetKillmailInput{KillmailID: 111, KillmailHash: "nilhash"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if out.KillmailID != 111 || !out.KillmailTime.Equal(killmailTime) || out.SolarSystemID != 30000142 {
		t.Fatalf("out = %+v, want KillmailID=111 KillmailTime=%v SolarSystemID=30000142", out, killmailTime)
	}
	if out.MoonID != nil {
		t.Errorf("out.MoonID = %v, want nil", out.MoonID)
	}
	if out.WarID != nil {
		t.Errorf("out.WarID = %v, want nil", out.WarID)
	}

	if len(out.Attackers) != 1 {
		t.Fatalf("out.Attackers = %+v, want one attacker", out.Attackers)
	}
	a := out.Attackers[0]
	if a.AllianceID != nil || a.CharacterID != nil || a.CorporationID != nil || a.FactionID != nil ||
		a.ShipTypeID != nil || a.WeaponTypeID != nil || a.DamageDone != 100 || a.FinalBlow || a.SecurityStatus != 0 {
		t.Errorf("attacker = %+v, want all optional fields nil, DamageDone=100 FinalBlow=false SecurityStatus=0", a)
	}

	v := out.Victim
	if v.AllianceID != nil || v.CharacterID != nil || v.CorporationID != nil || v.FactionID != nil {
		t.Errorf("victim = %+v, want all optional ID fields nil", v)
	}
	if v.DamageTaken != 200 || v.ShipTypeID != 587 {
		t.Errorf("victim = %+v, want DamageTaken=200 ShipTypeID=587", v)
	}
	if v.Position != nil {
		t.Errorf("victim.Position = %+v, want nil", v.Position)
	}
	if len(v.Items) != 0 {
		t.Errorf("victim.Items = %+v, want empty", v.Items)
	}
}

func TestListCharacterKillmailsRecentHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterKillmailsRecent
	t.Cleanup(func() { listCharacterKillmailsRecent = orig })
	wantErr := errors.New("upstream failure")
	listCharacterKillmailsRecent = func(ctx context.Context, clientID string, characterID int32, userAgent string, page int32) ([]esiapi.CharactersCharacterIdKillmailsRecentGetInner, int32, error) {
		return nil, 0, wantErr
	}

	handler := listCharacterKillmailsRecentHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterKillmailsRecentInput{CharacterID: 7})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCorporationKillmailsRecentHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationKillmailsRecent
	t.Cleanup(func() { listCorporationKillmailsRecent = orig })
	wantErr := errors.New("upstream failure")
	listCorporationKillmailsRecent = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CharactersCharacterIdKillmailsRecentGetInner, int32, error) {
		return nil, 0, wantErr
	}

	handler := listCorporationKillmailsRecentHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationKillmailsRecentInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetKillmailHandler_PropagatesError(t *testing.T) {
	orig := getKillmail
	t.Cleanup(func() { getKillmail = orig })
	wantErr := errors.New("upstream failure")
	getKillmail = func(ctx context.Context, userAgent string, killmailID int64, killmailHash string) (*esiapi.KillmailsKillmailIdKillmailHashGet, error) {
		return nil, wantErr
	}

	handler := getKillmailHandler("v1", nil)
	_, _, err := handler(context.Background(), nil, GetKillmailInput{KillmailID: 999, KillmailHash: "somehash"})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetKillmailHandler_ResolvesTypeNames(t *testing.T) {
	orig := getKillmail
	t.Cleanup(func() { getKillmail = orig })
	getKillmail = func(ctx context.Context, userAgent string, killmailID int64, killmailHash string) (*esiapi.KillmailsKillmailIdKillmailHashGet, error) {
		shipTypeID, weaponTypeID := int64(587), int64(2456)
		return &esiapi.KillmailsKillmailIdKillmailHashGet{
			Attackers: []esiapi.KillmailsKillmailIdKillmailHashGetAttackersInner{{ShipTypeId: &shipTypeID, WeaponTypeId: &weaponTypeID, FinalBlow: true}},
			Victim: esiapi.KillmailsKillmailIdKillmailHashGetVictim{
				ShipTypeId: 587,
				Items: []esiapi.KillmailsKillmailIdKillmailHashGetVictimItemsInner{
					{ItemTypeId: 34, Items: []esiapi.KillmailsKillmailIdKillmailHashGetVictimItemsInnerItemsInner{{ItemTypeId: 34}}},
				},
			},
		}, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter", 2456: "125mm Gatling AutoCannon II", 34: "Tritanium"}
	}

	handler := getKillmailHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, GetKillmailInput{KillmailID: 1, KillmailHash: "abc"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Victim.ShipTypeName != "Rifter" {
		t.Errorf("Victim.ShipTypeName = %q, want %q", out.Victim.ShipTypeName, "Rifter")
	}
	if out.Attackers[0].ShipTypeName != "Rifter" {
		t.Errorf("Attackers[0].ShipTypeName = %q, want %q", out.Attackers[0].ShipTypeName, "Rifter")
	}
	if out.Attackers[0].WeaponTypeName != "125mm Gatling AutoCannon II" {
		t.Errorf("Attackers[0].WeaponTypeName = %q, want %q", out.Attackers[0].WeaponTypeName, "125mm Gatling AutoCannon II")
	}
	if len(out.Victim.Items) != 1 || out.Victim.Items[0].ItemTypeName != "Tritanium" {
		t.Fatalf("Victim.Items = %+v, want one item with ItemTypeName=Tritanium", out.Victim.Items)
	}
	if len(out.Victim.Items[0].Items) != 1 || out.Victim.Items[0].Items[0].ItemTypeName != "Tritanium" {
		t.Fatalf("Victim.Items[0].Items = %+v, want one item with ItemTypeName=Tritanium", out.Victim.Items[0].Items)
	}
}
