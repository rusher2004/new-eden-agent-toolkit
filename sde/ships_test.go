package sde

import (
	"context"
	"testing"
)

func TestPublishedTypesInGroupsAndWithEffect(t *testing.T) {
	store := seedStore(t)
	for _, stmt := range []string{
		"INSERT INTO invGroups (groupID, categoryID, groupName, published) VALUES (547, 6, 'Carrier', 1)",
		"INSERT INTO invGroups (groupID, categoryID, groupName, published) VALUES (898, 6, 'Black Ops', 1)",
		"INSERT INTO invTypes (typeID, groupID, typeName, published, volume, packagedVolume) VALUES (23757, 547, 'Archon', 1, 11250000, 1300000)",
		"INSERT INTO invTypes (typeID, groupID, typeName, published, volume) VALUES (23911, 547, 'Thanatos', 1, 11250000)",
		"INSERT INTO invTypes (typeID, groupID, typeName, published, volume) VALUES (99999, 547, 'Unreleased Carrier', 0, 1)",
		"INSERT INTO invTypes (typeID, groupID, typeName, published, volume) VALUES (22430, 898, 'Sin', 1, 50000)",
		"INSERT INTO dgmEffects (effectID, effectName) VALUES (6684, 'jumpPortalPassengerBonusModAddSkill')",
		"INSERT INTO dgmTypeEffects (typeID, effectID, isDefault) VALUES (23757, 6684, 0)",
		"INSERT INTO dgmTypeEffects (typeID, effectID, isDefault) VALUES (99999, 6684, 0)",
	} {
		seedExec(t, store, stmt)
	}
	got, err := store.PublishedTypesInGroups(context.Background(), "test", []int64{547, 898})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].Name != "Archon" || got[2].Name != "Thanatos" || got[0].CategoryID != 6 || got[0].AssembledVolume != 11250000 || got[0].Volume != 1300000 {
		t.Errorf("in groups = %+v", got)
	}
	with, err := store.PublishedTypesWithEffect(context.Background(), "test", "jumpPortalPassengerBonusModAddSkill")
	if err != nil || len(with) != 1 || with[0].TypeID != 23757 {
		t.Errorf("with effect = %+v, %v", with, err)
	}
	if none, err := store.PublishedTypesInGroups(context.Background(), "test", nil); err != nil || none != nil {
		t.Errorf("no groups = %+v, %v", none, err)
	}
}
