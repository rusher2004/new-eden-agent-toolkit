package mcp

import (
	"context"
	"errors"
	"testing"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// fakeDogmaESI fakes the dogma tools' ESI calls; err is returned by every call.
type fakeDogmaESI struct {
	attributeIDs []int64
	attribute    *esiapi.DogmaAttributesAttributeIdGet
	dynamicItem  *esiapi.DogmaDynamicItemsTypeIdItemIdGet
	effectIDs    []int64
	effect       *esiapi.DogmaEffectsEffectIdGet
	err          error
}

func (f fakeDogmaESI) DogmaAttributes(context.Context) ([]int64, error) {
	return f.attributeIDs, f.err
}

func (f fakeDogmaESI) DogmaAttributeDetail(context.Context, int64) (*esiapi.DogmaAttributesAttributeIdGet, error) {
	return f.attribute, f.err
}

func (f fakeDogmaESI) DogmaDynamicItemDetail(context.Context, int64, int64) (*esiapi.DogmaDynamicItemsTypeIdItemIdGet, error) {
	return f.dynamicItem, f.err
}

func (f fakeDogmaESI) DogmaEffects(context.Context) ([]int64, error) {
	return f.effectIDs, f.err
}

func (f fakeDogmaESI) DogmaEffectDetail(context.Context, int64) (*esiapi.DogmaEffectsEffectIdGet, error) {
	return f.effect, f.err
}

func openDogmaFake(c dogmaESI) func() dogmaESI { return func() dogmaESI { return c } }

func TestListDogmaAttributesHandler_MapsFields(t *testing.T) {
	handler := listDogmaAttributesHandler(openDogmaFake(fakeDogmaESI{attributeIDs: []int64{1, 2, 3}}))
	_, out, err := handler(context.Background(), nil, ListDogmaAttributesInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.AttributeIDs) != 3 || out.AttributeIDs[0] != 1 || out.AttributeIDs[1] != 2 || out.AttributeIDs[2] != 3 {
		t.Errorf("out.AttributeIDs = %v, want [1 2 3]", out.AttributeIDs)
	}
}

func TestGetDogmaAttributeHandler_MapsFields(t *testing.T) {

	defaultVal := 1.5
	description := "Test description"
	displayName := "Test Display"
	highIsGood := true
	iconID := int64(123)
	name := "TestAttr"
	published := true
	stackable := false
	unitID := int64(456)

	fake := fakeDogmaESI{attribute: &esiapi.DogmaAttributesAttributeIdGet{
		AttributeId:  42,
		DefaultValue: &defaultVal,
		Description:  &description,
		DisplayName:  &displayName,
		HighIsGood:   &highIsGood,
		IconId:       &iconID,
		Name:         &name,
		Published:    &published,
		Stackable:    &stackable,
		UnitId:       &unitID,
	}}

	handler := getDogmaAttributeHandler(openDogmaFake(fake))
	_, out, err := handler(context.Background(), nil, GetDogmaAttributeInput{AttributeID: 42})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.AttributeID != 42 {
		t.Errorf("out.AttributeID = %d, want 42", out.AttributeID)
	}
	if out.DefaultValue == nil || *out.DefaultValue != 1.5 {
		t.Errorf("out.DefaultValue = %v, want *1.5", out.DefaultValue)
	}
	if out.Description == nil || *out.Description != "Test description" {
		t.Errorf("out.Description = %v, want *\"Test description\"", out.Description)
	}
	if out.DisplayName == nil || *out.DisplayName != "Test Display" {
		t.Errorf("out.DisplayName = %v, want *\"Test Display\"", out.DisplayName)
	}
	if out.HighIsGood == nil || *out.HighIsGood != true {
		t.Errorf("out.HighIsGood = %v, want *true", out.HighIsGood)
	}
	if out.IconID == nil || *out.IconID != 123 {
		t.Errorf("out.IconID = %v, want *123", out.IconID)
	}
	if out.Name == nil || *out.Name != "TestAttr" {
		t.Errorf("out.Name = %v, want *\"TestAttr\"", out.Name)
	}
	if out.Published == nil || *out.Published != true {
		t.Errorf("out.Published = %v, want *true", out.Published)
	}
	if out.Stackable == nil || *out.Stackable != false {
		t.Errorf("out.Stackable = %v, want *false", out.Stackable)
	}
	if out.UnitID == nil || *out.UnitID != 456 {
		t.Errorf("out.UnitID = %v, want *456", out.UnitID)
	}
}

func TestGetDogmaDynamicItemHandler_MapsFields(t *testing.T) {

	fake := fakeDogmaESI{dynamicItem: &esiapi.DogmaDynamicItemsTypeIdItemIdGet{
		CreatedBy:     100,
		MutatorTypeId: 200,
		SourceTypeId:  300,
		DogmaAttributes: []esiapi.DogmaDynamicItemsTypeIdItemIdGetDogmaAttributesInner{
			{AttributeId: 1, Value: 10.5},
			{AttributeId: 2, Value: 20.3},
		},
		DogmaEffects: []esiapi.DogmaDynamicItemsTypeIdItemIdGetDogmaEffectsInner{
			{EffectId: 5, IsDefault: true},
			{EffectId: 6, IsDefault: false},
		},
	}}

	handler := getDogmaDynamicItemHandler("v1", openDogmaFake(fake), nil)
	_, out, err := handler(context.Background(), nil, GetDogmaDynamicItemInput{TypeID: 1001, ItemID: 2002})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.TypeID != 1001 {
		t.Errorf("out.TypeID = %d, want 1001", out.TypeID)
	}
	if out.ItemID != 2002 {
		t.Errorf("out.ItemID = %d, want 2002", out.ItemID)
	}
	if out.CreatedBy != 100 {
		t.Errorf("out.CreatedBy = %d, want 100", out.CreatedBy)
	}
	if out.MutatorTypeID != 200 {
		t.Errorf("out.MutatorTypeID = %d, want 200", out.MutatorTypeID)
	}
	if out.SourceTypeID != 300 {
		t.Errorf("out.SourceTypeID = %d, want 300", out.SourceTypeID)
	}
	if len(out.DogmaAttributes) != 2 {
		t.Errorf("out.DogmaAttributes length = %d, want 2", len(out.DogmaAttributes))
	}
	if len(out.DogmaAttributes) >= 2 {
		if out.DogmaAttributes[0].AttributeID != 1 || out.DogmaAttributes[0].Value != 10.5 {
			t.Errorf("out.DogmaAttributes[0] = %+v, want {AttributeID: 1, Value: 10.5}", out.DogmaAttributes[0])
		}
		if out.DogmaAttributes[1].AttributeID != 2 || out.DogmaAttributes[1].Value != 20.3 {
			t.Errorf("out.DogmaAttributes[1] = %+v, want {AttributeID: 2, Value: 20.3}", out.DogmaAttributes[1])
		}
	}
	if len(out.DogmaEffects) != 2 {
		t.Errorf("out.DogmaEffects length = %d, want 2", len(out.DogmaEffects))
	}
	if len(out.DogmaEffects) >= 2 {
		if out.DogmaEffects[0].EffectID != 5 || out.DogmaEffects[0].IsDefault != true {
			t.Errorf("out.DogmaEffects[0] = %+v, want {EffectID: 5, IsDefault: true}", out.DogmaEffects[0])
		}
		if out.DogmaEffects[1].EffectID != 6 || out.DogmaEffects[1].IsDefault != false {
			t.Errorf("out.DogmaEffects[1] = %+v, want {EffectID: 6, IsDefault: false}", out.DogmaEffects[1])
		}
	}
}

func TestListDogmaEffectsHandler_MapsFields(t *testing.T) {
	handler := listDogmaEffectsHandler(openDogmaFake(fakeDogmaESI{effectIDs: []int64{10, 20}}))
	_, out, err := handler(context.Background(), nil, ListDogmaEffectsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.EffectIDs) != 2 || out.EffectIDs[0] != 10 || out.EffectIDs[1] != 20 {
		t.Errorf("out.EffectIDs = %v, want [10 20]", out.EffectIDs)
	}
}

func TestGetDogmaEffectHandler_MapsFields(t *testing.T) {

	description := "Effect description"
	displayName := "Effect Display"
	name := "TestEffect"
	disallowAutoRepeat := true
	electronicChance := false
	isAssistance := true
	isOffensive := false
	isWarpSafe := true
	published := true
	rangeChance := false
	dischargeAttributeID := int64(111)
	durationAttributeID := int64(222)
	effectCategory := int64(333)
	falloffAttributeID := int64(444)
	iconID := int64(555)
	postExpression := int64(666)
	preExpression := int64(777)
	rangeAttributeID := int64(888)
	trackingSpeedAttributeID := int64(999)

	fake := fakeDogmaESI{effect: &esiapi.DogmaEffectsEffectIdGet{
		EffectId:                 50,
		Description:              &description,
		DisplayName:              &displayName,
		Name:                     &name,
		DisallowAutoRepeat:       &disallowAutoRepeat,
		ElectronicChance:         &electronicChance,
		IsAssistance:             &isAssistance,
		IsOffensive:              &isOffensive,
		IsWarpSafe:               &isWarpSafe,
		Published:                &published,
		RangeChance:              &rangeChance,
		DischargeAttributeId:     &dischargeAttributeID,
		DurationAttributeId:      &durationAttributeID,
		EffectCategory:           &effectCategory,
		FalloffAttributeId:       &falloffAttributeID,
		IconId:                   &iconID,
		PostExpression:           &postExpression,
		PreExpression:            &preExpression,
		RangeAttributeId:         &rangeAttributeID,
		TrackingSpeedAttributeId: &trackingSpeedAttributeID,
		Modifiers: []esiapi.DogmaEffectsEffectIdGetModifiersInner{
			{Func: "mod1"},
			{Func: "mod2", Domain: new("domain1"), EffectId: new(int64(100)), ModifiedAttributeId: new(int64(200)), ModifyingAttributeId: new(int64(300)), Operator: new(int64(1))},
		},
	}}

	handler := getDogmaEffectHandler(openDogmaFake(fake))
	_, out, err := handler(context.Background(), nil, GetDogmaEffectInput{EffectID: 50})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.EffectID != 50 {
		t.Errorf("out.EffectID = %d, want 50", out.EffectID)
	}
	if out.Description == nil || *out.Description != "Effect description" {
		t.Errorf("out.Description = %v, want *\"Effect description\"", out.Description)
	}
	if out.DisplayName == nil || *out.DisplayName != "Effect Display" {
		t.Errorf("out.DisplayName = %v, want *\"Effect Display\"", out.DisplayName)
	}
	if out.Name == nil || *out.Name != "TestEffect" {
		t.Errorf("out.Name = %v, want *\"TestEffect\"", out.Name)
	}
	if out.DisallowAutoRepeat == nil || *out.DisallowAutoRepeat != true {
		t.Errorf("out.DisallowAutoRepeat = %v, want *true", out.DisallowAutoRepeat)
	}
	if out.ElectronicChance == nil || *out.ElectronicChance != false {
		t.Errorf("out.ElectronicChance = %v, want *false", out.ElectronicChance)
	}
	if out.IsAssistance == nil || *out.IsAssistance != true {
		t.Errorf("out.IsAssistance = %v, want *true", out.IsAssistance)
	}
	if out.IsOffensive == nil || *out.IsOffensive != false {
		t.Errorf("out.IsOffensive = %v, want *false", out.IsOffensive)
	}
	if out.IsWarpSafe == nil || *out.IsWarpSafe != true {
		t.Errorf("out.IsWarpSafe = %v, want *true", out.IsWarpSafe)
	}
	if out.Published == nil || *out.Published != true {
		t.Errorf("out.Published = %v, want *true", out.Published)
	}
	if out.RangeChance == nil || *out.RangeChance != false {
		t.Errorf("out.RangeChance = %v, want *false", out.RangeChance)
	}
	if out.DischargeAttributeID == nil || *out.DischargeAttributeID != 111 {
		t.Errorf("out.DischargeAttributeID = %v, want *111", out.DischargeAttributeID)
	}
	if out.DurationAttributeID == nil || *out.DurationAttributeID != 222 {
		t.Errorf("out.DurationAttributeID = %v, want *222", out.DurationAttributeID)
	}
	if out.EffectCategory == nil || *out.EffectCategory != 333 {
		t.Errorf("out.EffectCategory = %v, want *333", out.EffectCategory)
	}
	if out.FalloffAttributeID == nil || *out.FalloffAttributeID != 444 {
		t.Errorf("out.FalloffAttributeID = %v, want *444", out.FalloffAttributeID)
	}
	if out.IconID == nil || *out.IconID != 555 {
		t.Errorf("out.IconID = %v, want *555", out.IconID)
	}
	if out.PostExpression == nil || *out.PostExpression != 666 {
		t.Errorf("out.PostExpression = %v, want *666", out.PostExpression)
	}
	if out.PreExpression == nil || *out.PreExpression != 777 {
		t.Errorf("out.PreExpression = %v, want *777", out.PreExpression)
	}
	if out.RangeAttributeID == nil || *out.RangeAttributeID != 888 {
		t.Errorf("out.RangeAttributeID = %v, want *888", out.RangeAttributeID)
	}
	if out.TrackingSpeedAttributeID == nil || *out.TrackingSpeedAttributeID != 999 {
		t.Errorf("out.TrackingSpeedAttributeID = %v, want *999", out.TrackingSpeedAttributeID)
	}
	if len(out.Modifiers) != 2 {
		t.Errorf("out.Modifiers length = %d, want 2", len(out.Modifiers))
	}
	if len(out.Modifiers) >= 2 {
		if out.Modifiers[0].Func != "mod1" || out.Modifiers[0].Domain != nil {
			t.Errorf("out.Modifiers[0] = %+v, want {Func: \"mod1\", Domain: nil}", out.Modifiers[0])
		}
		if out.Modifiers[0].EffectID != nil {
			t.Errorf("out.Modifiers[0].EffectID = %v, want nil", out.Modifiers[0].EffectID)
		}
		if out.Modifiers[0].ModifiedAttributeID != nil {
			t.Errorf("out.Modifiers[0].ModifiedAttributeID = %v, want nil", out.Modifiers[0].ModifiedAttributeID)
		}
		if out.Modifiers[0].ModifyingAttributeID != nil {
			t.Errorf("out.Modifiers[0].ModifyingAttributeID = %v, want nil", out.Modifiers[0].ModifyingAttributeID)
		}
		if out.Modifiers[0].Operator != nil {
			t.Errorf("out.Modifiers[0].Operator = %v, want nil", out.Modifiers[0].Operator)
		}
		if out.Modifiers[1].Func != "mod2" || out.Modifiers[1].Domain == nil || *out.Modifiers[1].Domain != "domain1" {
			t.Errorf("out.Modifiers[1].Func/Domain = %s/%v, want \"mod2\"/\"domain1\"", out.Modifiers[1].Func, out.Modifiers[1].Domain)
		}
		if out.Modifiers[1].EffectID == nil || *out.Modifiers[1].EffectID != 100 {
			t.Errorf("out.Modifiers[1].EffectID = %v, want *100", out.Modifiers[1].EffectID)
		}
		if out.Modifiers[1].ModifiedAttributeID == nil || *out.Modifiers[1].ModifiedAttributeID != 200 {
			t.Errorf("out.Modifiers[1].ModifiedAttributeID = %v, want *200", out.Modifiers[1].ModifiedAttributeID)
		}
		if out.Modifiers[1].ModifyingAttributeID == nil || *out.Modifiers[1].ModifyingAttributeID != 300 {
			t.Errorf("out.Modifiers[1].ModifyingAttributeID = %v, want *300", out.Modifiers[1].ModifyingAttributeID)
		}
		if out.Modifiers[1].Operator == nil || *out.Modifiers[1].Operator != 1 {
			t.Errorf("out.Modifiers[1].Operator = %v, want *1", out.Modifiers[1].Operator)
		}
	}
}

// Helper functions to create pointers
//
//go:fix inline
func pStr(s string) *string {
	return new(s)
}

//go:fix inline
func pInt64(i int64) *int64 {
	return new(i)
}

func TestListDogmaAttributesHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")

	handler := listDogmaAttributesHandler(openDogmaFake(fakeDogmaESI{err: wantErr}))
	_, _, err := handler(context.Background(), nil, ListDogmaAttributesInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetDogmaAttributeHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")

	handler := getDogmaAttributeHandler(openDogmaFake(fakeDogmaESI{err: wantErr}))
	_, _, err := handler(context.Background(), nil, GetDogmaAttributeInput{AttributeID: 42})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetDogmaDynamicItemHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")

	handler := getDogmaDynamicItemHandler("v1", openDogmaFake(fakeDogmaESI{err: wantErr}), nil)
	_, _, err := handler(context.Background(), nil, GetDogmaDynamicItemInput{TypeID: 1001, ItemID: 2002})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetDogmaDynamicItemHandler_ResolvesTypeNames(t *testing.T) {
	fake := fakeDogmaESI{dynamicItem: &esiapi.DogmaDynamicItemsTypeIdItemIdGet{CreatedBy: 100, MutatorTypeId: 588, SourceTypeId: 587}}

	origResolveTypeNames := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolveTypeNames })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter", 588: "Abyssal Mutaplasmid"}
	}

	handler := getDogmaDynamicItemHandler("v1", openDogmaFake(fake), nil)
	_, out, err := handler(context.Background(), nil, GetDogmaDynamicItemInput{TypeID: 587, ItemID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.TypeName != "Rifter" {
		t.Errorf("TypeName = %q, want %q", out.TypeName, "Rifter")
	}
	if out.MutatorTypeName != "Abyssal Mutaplasmid" {
		t.Errorf("MutatorTypeName = %q, want %q", out.MutatorTypeName, "Abyssal Mutaplasmid")
	}
	if out.SourceTypeName != "Rifter" {
		t.Errorf("SourceTypeName = %q, want %q", out.SourceTypeName, "Rifter")
	}
}

func TestListDogmaEffectsHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")

	handler := listDogmaEffectsHandler(openDogmaFake(fakeDogmaESI{err: wantErr}))
	_, _, err := handler(context.Background(), nil, ListDogmaEffectsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetDogmaEffectHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")

	handler := getDogmaEffectHandler(openDogmaFake(fakeDogmaESI{err: wantErr}))
	_, _, err := handler(context.Background(), nil, GetDogmaEffectInput{EffectID: 50})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}
