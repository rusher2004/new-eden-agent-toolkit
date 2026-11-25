package esi_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	goesi "github.com/fnt-eve/goesi-openapi"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

func TestDogmaAttributes_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[6,9,20,21,22,23,24,25,26,27]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	ids, err := client.DogmaAttributes(context.Background())
	if err != nil {
		t.Fatalf("DogmaAttributes returned error: %v", err)
	}
	if len(ids) != 10 || ids[0] != 6 || ids[1] != 9 {
		t.Errorf("ids = %v, want [6 9 20 21 22 23 24 25 26 27]", ids)
	}
}

func TestDogmaAttributeDetail_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"attribute_id":6,
			"default_value":0,
			"description":"Base Shield Recharge Rate",
			"display_name":"Shield Recharge Rate",
			"high_is_good":true,
			"icon_id":123,
			"name":"shieldRechargeRate",
			"published":true,
			"stackable":true,
			"unit_id":5
		}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	detail, err := client.DogmaAttributeDetail(context.Background(), 6)
	if err != nil {
		t.Fatalf("DogmaAttributeDetail returned error: %v", err)
	}

	// Check required fields
	if detail.AttributeId != 6 {
		t.Errorf("detail.AttributeId = %v, want 6", detail.AttributeId)
	}
	if detail.DefaultValue == nil || *detail.DefaultValue != 0 {
		t.Errorf("detail.DefaultValue = %v, want 0", detail.DefaultValue)
	}
	if detail.Description == nil || *detail.Description != "Base Shield Recharge Rate" {
		t.Errorf("detail.Description = %v, want 'Base Shield Recharge Rate'", detail.Description)
	}
	if detail.DisplayName == nil || *detail.DisplayName != "Shield Recharge Rate" {
		t.Errorf("detail.DisplayName = %v, want 'Shield Recharge Rate'", detail.DisplayName)
	}
	if detail.HighIsGood == nil || *detail.HighIsGood != true {
		t.Errorf("detail.HighIsGood = %v, want true", detail.HighIsGood)
	}
	if detail.IconId == nil || *detail.IconId != 123 {
		t.Errorf("detail.IconId = %v, want 123", detail.IconId)
	}
	if detail.Name == nil || *detail.Name != "shieldRechargeRate" {
		t.Errorf("detail.Name = %v, want 'shieldRechargeRate'", detail.Name)
	}
	if detail.Published == nil || *detail.Published != true {
		t.Errorf("detail.Published = %v, want true", detail.Published)
	}
	if detail.Stackable == nil || *detail.Stackable != true {
		t.Errorf("detail.Stackable = %v, want true", detail.Stackable)
	}
	if detail.UnitId == nil || *detail.UnitId != 5 {
		t.Errorf("detail.UnitId = %v, want 5", detail.UnitId)
	}
}

func TestDogmaAttributeDetail_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.DogmaAttributeDetail(context.Background(), 999999); err == nil {
		t.Fatal("DogmaAttributeDetail returned nil error, want an error for HTTP 404")
	}
}

func TestDogmaDynamicItemDetail_ParsesResponseAndUsesCorrectArgumentOrder(t *testing.T) {
	typeID := int64(37001)
	itemID := int64(1000000001)
	requestPath := ""

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write(fmt.Appendf(nil, `{
			"created_by":2112625428,
			"dogma_attributes":[
				{"attribute_id":6,"value":100.5},
				{"attribute_id":37,"value":50.25}
			],
			"dogma_effects":[
				{"effect_id":4680,"is_default":true},
				{"effect_id":4681,"is_default":false}
			],
			"mutator_type_id":33468,
			"source_type_id":%d
		}`, typeID))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	detail, err := client.DogmaDynamicItemDetail(context.Background(), typeID, itemID)
	if err != nil {
		t.Fatalf("DogmaDynamicItemDetail returned error: %v", err)
	}

	// Check fields
	if detail.CreatedBy != 2112625428 {
		t.Errorf("detail.CreatedBy = %v, want 2112625428", detail.CreatedBy)
	}
	if detail.MutatorTypeId != 33468 {
		t.Errorf("detail.MutatorTypeId = %v, want 33468", detail.MutatorTypeId)
	}
	if detail.SourceTypeId != typeID {
		t.Errorf("detail.SourceTypeId = %v, want %d", detail.SourceTypeId, typeID)
	}

	// Check dogma_attributes
	if len(detail.DogmaAttributes) != 2 {
		t.Errorf("len(detail.DogmaAttributes) = %v, want 2", len(detail.DogmaAttributes))
	} else {
		if detail.DogmaAttributes[0].AttributeId != 6 || detail.DogmaAttributes[0].Value != 100.5 {
			t.Errorf("detail.DogmaAttributes[0] = %+v, want AttributeId=6 Value=100.5", detail.DogmaAttributes[0])
		}
		if detail.DogmaAttributes[1].AttributeId != 37 || detail.DogmaAttributes[1].Value != 50.25 {
			t.Errorf("detail.DogmaAttributes[1] = %+v, want AttributeId=37 Value=50.25", detail.DogmaAttributes[1])
		}
	}

	// Check dogma_effects
	if len(detail.DogmaEffects) != 2 {
		t.Errorf("len(detail.DogmaEffects) = %v, want 2", len(detail.DogmaEffects))
	} else {
		if detail.DogmaEffects[0].EffectId != 4680 || detail.DogmaEffects[0].IsDefault != true {
			t.Errorf("detail.DogmaEffects[0] = %+v, want EffectId=4680 IsDefault=true", detail.DogmaEffects[0])
		}
		if detail.DogmaEffects[1].EffectId != 4681 || detail.DogmaEffects[1].IsDefault != false {
			t.Errorf("detail.DogmaEffects[1] = %+v, want EffectId=4681 IsDefault=false", detail.DogmaEffects[1])
		}
	}

	// Verify correct URL path: /dogma/dynamic/items/{typeID}/{itemID}/
	expectedPath := fmt.Sprintf("/dogma/dynamic/items/%d/%d", typeID, itemID)
	if requestPath != expectedPath {
		t.Errorf("request path = %v, want %v (verifies argument order swap)", requestPath, expectedPath)
	}
}

func TestDogmaEffects_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[4,10,11,12,13,14,15,16,17,18]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	ids, err := client.DogmaEffects(context.Background())
	if err != nil {
		t.Fatalf("DogmaEffects returned error: %v", err)
	}
	if len(ids) != 10 || ids[0] != 4 || ids[1] != 10 {
		t.Errorf("ids = %v, want [4 10 11 12 13 14 15 16 17 18]", ids)
	}
}

func TestDogmaEffectDetail_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"description":"Rate of capacitor capacity depletion for active shield hardeners.",
			"disallow_auto_repeat":false,
			"display_name":"Shield Hardener",
			"duration_attribute_id":73,
			"effect_category":123,
			"effect_id":4680,
			"electronic_chance":false,
			"falloff_attribute_id":null,
			"icon_id":10001,
			"is_offensive":false,
			"is_assistance":false,
			"modifiers":[
				{
					"domain":"shipID",
					"effect_id":4680,
					"func":"ItemModifier",
					"modified_attribute_id":27,
					"modifying_attribute_id":6,
					"operator":10
				},
				{
					"domain":"target",
					"effect_id":4680,
					"func":"ItemModifier",
					"modified_attribute_id":28,
					"modifying_attribute_id":7,
					"operator":20
				}
			],
			"name":"shieldHardenerEffect",
			"post_expression":null,
			"pre_expression":null,
			"published":true,
			"range_attribute_id":null,
			"range_chance":false,
			"tracking_speed_attribute_id":null
		}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	detail, err := client.DogmaEffectDetail(context.Background(), 4680)
	if err != nil {
		t.Fatalf("DogmaEffectDetail returned error: %v", err)
	}

	// Check top-level fields
	if detail.Description == nil || *detail.Description != "Rate of capacitor capacity depletion for active shield hardeners." {
		t.Errorf("detail.Description = %v, want 'Rate of capacitor capacity depletion for active shield hardeners.'", detail.Description)
	}
	if detail.DisallowAutoRepeat == nil || *detail.DisallowAutoRepeat != false {
		t.Errorf("detail.DisallowAutoRepeat = %v, want false", detail.DisallowAutoRepeat)
	}
	if detail.DisplayName == nil || *detail.DisplayName != "Shield Hardener" {
		t.Errorf("detail.DisplayName = %v, want 'Shield Hardener'", detail.DisplayName)
	}
	if detail.DurationAttributeId == nil || *detail.DurationAttributeId != 73 {
		t.Errorf("detail.DurationAttributeId = %v, want 73", detail.DurationAttributeId)
	}
	if detail.EffectCategory == nil || *detail.EffectCategory != 123 {
		t.Errorf("detail.EffectCategory = %v, want 123", detail.EffectCategory)
	}
	if detail.EffectId != 4680 {
		t.Errorf("detail.EffectId = %v, want 4680", detail.EffectId)
	}
	if detail.ElectronicChance == nil || *detail.ElectronicChance != false {
		t.Errorf("detail.ElectronicChance = %v, want false", detail.ElectronicChance)
	}
	if detail.IconId == nil || *detail.IconId != 10001 {
		t.Errorf("detail.IconId = %v, want 10001", detail.IconId)
	}
	if detail.IsOffensive == nil || *detail.IsOffensive != false {
		t.Errorf("detail.IsOffensive = %v, want false", detail.IsOffensive)
	}
	if detail.IsAssistance == nil || *detail.IsAssistance != false {
		t.Errorf("detail.IsAssistance = %v, want false", detail.IsAssistance)
	}
	if detail.Name == nil || *detail.Name != "shieldHardenerEffect" {
		t.Errorf("detail.Name = %v, want 'shieldHardenerEffect'", detail.Name)
	}
	if detail.Published == nil || *detail.Published != true {
		t.Errorf("detail.Published = %v, want true", detail.Published)
	}

	// Check modifiers array
	if len(detail.Modifiers) != 2 {
		t.Errorf("len(detail.Modifiers) = %v, want 2", len(detail.Modifiers))
	} else {
		// Check first modifier
		if detail.Modifiers[0].Domain == nil || *detail.Modifiers[0].Domain != "shipID" {
			t.Errorf("detail.Modifiers[0].Domain = %v, want 'shipID'", detail.Modifiers[0].Domain)
		}
		if detail.Modifiers[0].EffectId == nil || *detail.Modifiers[0].EffectId != 4680 {
			t.Errorf("detail.Modifiers[0].EffectId = %v, want 4680", detail.Modifiers[0].EffectId)
		}
		if detail.Modifiers[0].Func != "ItemModifier" {
			t.Errorf("detail.Modifiers[0].Func = %v, want 'ItemModifier'", detail.Modifiers[0].Func)
		}
		if detail.Modifiers[0].ModifiedAttributeId == nil || *detail.Modifiers[0].ModifiedAttributeId != 27 {
			t.Errorf("detail.Modifiers[0].ModifiedAttributeId = %v, want 27", detail.Modifiers[0].ModifiedAttributeId)
		}
		if detail.Modifiers[0].ModifyingAttributeId == nil || *detail.Modifiers[0].ModifyingAttributeId != 6 {
			t.Errorf("detail.Modifiers[0].ModifyingAttributeId = %v, want 6", detail.Modifiers[0].ModifyingAttributeId)
		}
		if detail.Modifiers[0].Operator == nil || *detail.Modifiers[0].Operator != 10 {
			t.Errorf("detail.Modifiers[0].Operator = %v, want 10", detail.Modifiers[0].Operator)
		}

		// Check second modifier
		if detail.Modifiers[1].Domain == nil || *detail.Modifiers[1].Domain != "target" {
			t.Errorf("detail.Modifiers[1].Domain = %v, want 'target'", detail.Modifiers[1].Domain)
		}
		if detail.Modifiers[1].ModifyingAttributeId == nil || *detail.Modifiers[1].ModifyingAttributeId != 7 {
			t.Errorf("detail.Modifiers[1].ModifyingAttributeId = %v, want 7", detail.Modifiers[1].ModifyingAttributeId)
		}
		if detail.Modifiers[1].Operator == nil || *detail.Modifiers[1].Operator != 20 {
			t.Errorf("detail.Modifiers[1].Operator = %v, want 20", detail.Modifiers[1].Operator)
		}
	}
}
