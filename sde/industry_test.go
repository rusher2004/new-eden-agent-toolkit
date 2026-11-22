package sde

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"
)

// industryFixtures are real rows copied from the fuzzwork dump on 2026-11-18:
// the Capital Siege Array blueprint, Compressed Gneiss IV-Grade, the Standup
// XL-Set Ship Manufacturing Efficiency I rig and its effects, the Nomad, the
// Sotiyo, Jita 4-4, and the Jita / Perimeter / Aunenen / NL6V-7 / PF-QHK systems.
const industryFixtures = `
INSERT INTO industryActivity (typeID, activityID, time) VALUES (21040,1,16000);
INSERT INTO industryActivityMaterials (typeID, activityID, materialTypeID, quantity) VALUES (21040,1,2876,1);
INSERT INTO industryActivityMaterials (typeID, activityID, materialTypeID, quantity) VALUES (21040,1,57457,100);
INSERT INTO industryActivityMaterials (typeID, activityID, materialTypeID, quantity) VALUES (21040,1,40,188);
INSERT INTO industryActivityMaterials (typeID, activityID, materialTypeID, quantity) VALUES (21040,1,39,390);
INSERT INTO industryActivityMaterials (typeID, activityID, materialTypeID, quantity) VALUES (21040,1,38,750);
INSERT INTO industryActivityMaterials (typeID, activityID, materialTypeID, quantity) VALUES (21040,1,37,7500);
INSERT INTO industryActivityMaterials (typeID, activityID, materialTypeID, quantity) VALUES (21040,1,34,27750);
INSERT INTO industryActivityMaterials (typeID, activityID, materialTypeID, quantity) VALUES (21040,1,36,28500);
INSERT INTO industryActivityMaterials (typeID, activityID, materialTypeID, quantity) VALUES (21040,1,35,101250);
INSERT INTO industryActivityProducts (typeID, activityID, productTypeID, quantity) VALUES (21040,1,21039,1);
INSERT INTO industryActivitySkills (typeID, activityID, skillID, level) VALUES (21040,1,22242,1);
INSERT INTO invTypeMaterials (typeID, materialTypeID, quantity) VALUES (62555,35,2300);
INSERT INTO invTypeMaterials (typeID, materialTypeID, quantity) VALUES (62555,36,1725);
INSERT INTO invTypeMaterials (typeID, materialTypeID, quantity) VALUES (62555,37,920);
INSERT INTO dgmEffects (effectID, effectName, modifierInfo) VALUES (2663,'rigSlot',NULL);
INSERT INTO dgmEffects (effectID, effectName, modifierInfo) VALUES (6840,'rigAllShipManufactureMaterialBonus','[{"domain": "structureID", "func": "ItemModifier", "modifiedAttributeID": 2592, "modifyingAttributeID": 2594, "operation": 6}]');
INSERT INTO dgmEffects (effectID, effectName, modifierInfo) VALUES (6841,'rigAllShipManufactureTimeBonus','[{"domain": "structureID", "func": "ItemModifier", "modifiedAttributeID": 2591, "modifyingAttributeID": 2593, "operation": 6}]');
INSERT INTO dgmEffects (effectID, effectName, modifierInfo) VALUES (6842,'structureEngineeringRigSecurityModification','[{"domain": "itemID", "func": "ItemModifier", "modifiedAttributeID": 2595, "modifyingAttributeID": 2358, "operation": 4}, {"domain": "itemID", "func": "ItemModifier", "modifiedAttributeID": 2594, "modifyingAttributeID": 2358, "operation": 4}, {"domain": "itemID", "func": "ItemModifier", "modifiedAttributeID": 2593, "modifyingAttributeID": 2358, "operation": 4}, {"domain": "itemID", "func": "ItemModifier", "modifiedAttributeID": 2653, "modifyingAttributeID": 2358, "operation": 4}]');
INSERT INTO dgmTypeEffects (typeID, effectID, isDefault) VALUES (37180,2663,1);
INSERT INTO dgmTypeEffects (typeID, effectID, isDefault) VALUES (37180,6840,0);
INSERT INTO dgmTypeEffects (typeID, effectID, isDefault) VALUES (37180,6841,0);
INSERT INTO dgmTypeEffects (typeID, effectID, isDefault) VALUES (37180,6842,0);
INSERT INTO dgmTypeAttributes (typeID, attributeID, valueInt, valueFloat) VALUES (28846,866,NULL,17889.0);
INSERT INTO dgmTypeAttributes (typeID, attributeID, valueInt, valueFloat) VALUES (28846,867,NULL,5.0);
INSERT INTO dgmTypeAttributes (typeID, attributeID, valueInt, valueFloat) VALUES (28846,868,NULL,8200.0);
INSERT INTO dgmTypeAttributes (typeID, attributeID, valueInt, valueFloat) VALUES (28846,1312,NULL,-10.0);
INSERT INTO dgmTypeAttributes (typeID, attributeID, valueInt, valueFloat) VALUES (28846,1547,NULL,4.0);
INSERT INTO dgmTypeAttributes (typeID, attributeID, valueInt, valueFloat) VALUES (28846,1971,NULL,0.1);
INSERT INTO dgmTypeAttributes (typeID, attributeID, valueInt, valueFloat) VALUES (35827,1547,NULL,4.0);
INSERT INTO dgmTypeAttributes (typeID, attributeID, valueInt, valueFloat) VALUES (35827,1970,NULL,1.0);
INSERT INTO dgmTypeAttributes (typeID, attributeID, valueInt, valueFloat) VALUES (35827,2600,NULL,0.99);
INSERT INTO dgmTypeAttributes (typeID, attributeID, valueInt, valueFloat) VALUES (35827,2601,NULL,0.95);
INSERT INTO dgmTypeAttributes (typeID, attributeID, valueInt, valueFloat) VALUES (35827,2602,NULL,0.7);
INSERT INTO dgmTypeAttributes (typeID, attributeID, valueInt, valueFloat) VALUES (37180,1547,NULL,4.0);
INSERT INTO dgmTypeAttributes (typeID, attributeID, valueInt, valueFloat) VALUES (37180,2355,NULL,1.0);
INSERT INTO dgmTypeAttributes (typeID, attributeID, valueInt, valueFloat) VALUES (37180,2356,NULL,1.9);
INSERT INTO dgmTypeAttributes (typeID, attributeID, valueInt, valueFloat) VALUES (37180,2357,NULL,2.1);
INSERT INTO dgmTypeAttributes (typeID, attributeID, valueInt, valueFloat) VALUES (37180,2358,NULL,1.0);
INSERT INTO dgmTypeAttributes (typeID, attributeID, valueInt, valueFloat) VALUES (37180,2593,NULL,-20.0);
INSERT INTO dgmTypeAttributes (typeID, attributeID, valueInt, valueFloat) VALUES (37180,2594,NULL,-2.0);
INSERT INTO dgmTypeAttributes (typeID, attributeID, valueInt, valueFloat) VALUES (37180,2595,NULL,0.0);
INSERT INTO dgmTypeAttributes (typeID, attributeID, valueInt, valueFloat) VALUES (62555,790,NULL,60379.0);
INSERT INTO invTypes (typeID, groupID, typeName, published, marketGroupID, volume, portionSize, packagedVolume) VALUES (34,18,'Tritanium',1,1857,0.01,1,0.01);
INSERT INTO invTypes (typeID, groupID, typeName, published, marketGroupID, volume, portionSize, packagedVolume) VALUES (35,18,'Pyerite',1,1857,0.01,1,0.01);
INSERT INTO invTypes (typeID, groupID, typeName, published, marketGroupID, volume, portionSize, packagedVolume) VALUES (36,18,'Mexallon',1,1857,0.01,1,0.01);
INSERT INTO invTypes (typeID, groupID, typeName, published, marketGroupID, volume, portionSize, packagedVolume) VALUES (37,18,'Isogen',1,1857,0.01,1,0.01);
INSERT INTO invTypes (typeID, groupID, typeName, published, marketGroupID, volume, portionSize, packagedVolume) VALUES (38,18,'Nocxium',1,1857,0.01,1,0.01);
INSERT INTO invTypes (typeID, groupID, typeName, published, marketGroupID, volume, portionSize, packagedVolume) VALUES (39,18,'Zydrine',1,1857,0.01,1,0.01);
INSERT INTO invTypes (typeID, groupID, typeName, published, marketGroupID, volume, portionSize, packagedVolume) VALUES (40,18,'Megacyte',1,1857,0.01,1,0.01);
INSERT INTO invTypes (typeID, groupID, typeName, published, marketGroupID, volume, portionSize, packagedVolume) VALUES (2876,1041,'Wetware Mainframe',1,1337,50.0,1,50.0);
INSERT INTO invTypes (typeID, groupID, typeName, published, marketGroupID, volume, portionSize, packagedVolume) VALUES (21039,873,'Capital Siege Array',1,781,2000.0,1,2000.0);
INSERT INTO invTypes (typeID, groupID, typeName, published, marketGroupID, volume, portionSize, packagedVolume) VALUES (21040,915,'Capital Siege Array Blueprint',1,796,0.01,1,0.01);
INSERT INTO invTypes (typeID, groupID, typeName, published, marketGroupID, volume, portionSize, packagedVolume) VALUES (28846,902,'Nomad',1,1093,15500000.0,1,1300000.0);
INSERT INTO invTypes (typeID, groupID, typeName, published, marketGroupID, volume, portionSize, packagedVolume) VALUES (35827,1404,'Sotiyo',1,2324,800000.0,1,800000.0);
INSERT INTO invTypes (typeID, groupID, typeName, published, marketGroupID, volume, portionSize, packagedVolume) VALUES (37180,1868,'Standup XL-Set Ship Manufacturing Efficiency I',1,2349,40.0,1,40.0);
INSERT INTO invTypes (typeID, groupID, typeName, published, marketGroupID, volume, portionSize, packagedVolume) VALUES (57457,429,'Reinforced Carbon Fiber',1,499,0.1,1,0.1);
INSERT INTO invTypes (typeID, groupID, typeName, published, marketGroupID, volume, portionSize, packagedVolume) VALUES (62555,467,'Compressed Gneiss IV-Grade',1,525,0.05,100,0.05);
INSERT INTO invGroups (groupID, categoryID, groupName, published) VALUES (18,4,'Mineral',1);
INSERT INTO invGroups (groupID, categoryID, groupName, published) VALUES (429,4,'Composite',1);
INSERT INTO invGroups (groupID, categoryID, groupName, published) VALUES (467,25,'Gneiss',1);
INSERT INTO invGroups (groupID, categoryID, groupName, published) VALUES (873,17,'Capital Construction Components',1);
INSERT INTO invGroups (groupID, categoryID, groupName, published) VALUES (902,6,'Jump Freighter',1);
INSERT INTO invGroups (groupID, categoryID, groupName, published) VALUES (915,9,'Capital Construction Blueprints',1);
INSERT INTO invGroups (groupID, categoryID, groupName, published) VALUES (1041,43,'Advanced Commodities - Tier 4',1);
INSERT INTO invGroups (groupID, categoryID, groupName, published) VALUES (1404,65,'Engineering Complex',1);
INSERT INTO invGroups (groupID, categoryID, groupName, published) VALUES (1868,66,'Structure Engineering Rig XL - Ship Efficiency',1);
INSERT INTO mapSolarSystems (solarSystemID, constellationID, solarSystemName, x, y, z, security) VALUES (30000142,20000020,'Jita',-1.29064861735e+17,60755306910000000.0,1.1746922706e+17,0.945913);
INSERT INTO mapSolarSystems (solarSystemID, constellationID, solarSystemName, x, y, z, security) VALUES (30000144,20000020,'Perimeter',-1.432652330889429e+17,64923714928938896.0,1.0417862320674181e+17,0.949);
INSERT INTO mapSolarSystems (solarSystemID, constellationID, solarSystemName, x, y, z, security) VALUES (30001398,20000205,'Aunenen',-1.9065262629888752e+17,88927690357314880.0,1.7836417700017386e+17,0.44779);
INSERT INTO mapSolarSystems (solarSystemID, constellationID, solarSystemName, x, y, z, security) VALUES (30000855,20000125,'NL6V-7',-1.4903294260526365e+17,99542048543805280.0,2.3926934557504768e+17,-0.255748);
INSERT INTO mapSolarSystems (solarSystemID, constellationID, solarSystemName, x, y, z, security) VALUES (30001324,20000193,'PF-QHK',-1.305672291398673e+17,98759478627101390.0,3.297507456526137e+17,-0.134402);
INSERT INTO mapConstellations (constellationID, regionID, constellationName) VALUES (20000020,10000002,'Kimotoro');
INSERT INTO mapConstellations (constellationID, regionID, constellationName) VALUES (20000205,10000016,'Minnen');
INSERT INTO mapConstellations (constellationID, regionID, constellationName) VALUES (20000125,10000010,'X-VN63');
INSERT INTO mapConstellations (constellationID, regionID, constellationName) VALUES (20000193,10000015,'Y8HB-U');
INSERT INTO mapRegions (regionID, regionName) VALUES (10000002,'The Forge');
INSERT INTO mapRegions (regionID, regionName) VALUES (10000016,'Lonetrek');
INSERT INTO mapRegions (regionID, regionName) VALUES (10000010,'Tribute');
INSERT INTO mapRegions (regionID, regionName) VALUES (10000015,'Venal');
INSERT INTO mapSolarSystemJumps (fromRegionID, fromConstellationID, fromSolarSystemID, toSolarSystemID, toConstellationID, toRegionID) VALUES (10000002,20000020,30000142,30000144,20000020,10000002);
INSERT INTO mapSolarSystemJumps (fromRegionID, fromConstellationID, fromSolarSystemID, toSolarSystemID, toConstellationID, toRegionID) VALUES (10000002,20000020,30000144,30000142,20000020,10000002);
INSERT INTO staStations (stationID, stationName, solarSystemID, corporationID, operationID, reprocessingEfficiency, reprocessingStationsTake) VALUES (60003760,'Jita 4 - Moon 4 - Caldari Navy Assembly Plant',30000142,1000035,14,0.5,0.05);
INSERT INTO indModifierSources (typeID, activityName, modifierType, dogmaAttributeID, filterID) VALUES (35827,'manufacturing','material',2600,NULL);
INSERT INTO indModifierSources (typeID, activityName, modifierType, dogmaAttributeID, filterID) VALUES (35827,'manufacturing','time',2602,NULL);
INSERT INTO indModifierSources (typeID, activityName, modifierType, dogmaAttributeID, filterID) VALUES (37180,'manufacturing','material',2592,3);
INSERT INTO indModifierSources (typeID, activityName, modifierType, dogmaAttributeID, filterID) VALUES (37180,'manufacturing','time',2591,3);
INSERT INTO indTargetFilters (targetFilterID, name) VALUES (3,'Ships');
INSERT INTO indTargetFilters (targetFilterID, name) VALUES (13,'Capital Components');
INSERT INTO indTargetFilterCategories (targetFilterID, categoryID) VALUES (3,6);
INSERT INTO indTargetFilterCategories (targetFilterID, categoryID) VALUES (3,32);
INSERT INTO indTargetFilterGroups (targetFilterID, groupID) VALUES (13,873);
`

func seedIndustry(t *testing.T) *Store {
	t.Helper()
	store := seedStore(t)
	for _, stmt := range strings.Split(strings.TrimSpace(industryFixtures), "\n") {
		seedExec(t, store, stmt)
	}
	return store
}

func TestBlueprint_CapitalSiegeArray(t *testing.T) {
	store := seedIndustry(t)
	bp, ok, err := store.Blueprint(context.Background(), "test", 21040)
	if err != nil || !ok {
		t.Fatalf("Blueprint = ok %v, err %v", ok, err)
	}
	mfg, ok := bp.Activity(ActivityManufacturing)
	if !ok {
		t.Fatal("no manufacturing activity")
	}
	if mfg.TimeSeconds != 16000 {
		t.Errorf("time = %d, want 16000", mfg.TimeSeconds)
	}
	if len(mfg.Materials) != 9 {
		t.Errorf("materials = %d, want 9", len(mfg.Materials))
	}
	var pyerite int64
	for _, m := range mfg.Materials {
		if m.TypeID == 35 {
			pyerite = m.Quantity
		}
	}
	if pyerite != 101250 {
		t.Errorf("pyerite = %d, want 101250", pyerite)
	}
	if len(mfg.Products) != 1 || mfg.Products[0].TypeID != 21039 || mfg.Products[0].Quantity != 1 {
		t.Errorf("products = %+v", mfg.Products)
	}
	if len(mfg.Skills) != 1 || mfg.Skills[0].SkillID != 22242 || mfg.Skills[0].Level != 1 {
		t.Errorf("skills = %+v", mfg.Skills)
	}
}

func TestBlueprint_Unknown(t *testing.T) {
	store := seedIndustry(t)
	_, ok, err := store.Blueprint(context.Background(), "test", 1)
	if err != nil || ok {
		t.Errorf("Blueprint(unknown) = ok %v, err %v; want false, nil", ok, err)
	}
}

func TestBlueprintsForProduct(t *testing.T) {
	store := seedIndustry(t)
	got, err := store.BlueprintsForProduct(context.Background(), "test", 21039)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].BlueprintTypeID != 21040 || got[0].ActivityID != ActivityManufacturing || got[0].Quantity != 1 {
		t.Errorf("BlueprintsForProduct = %+v", got)
	}
}

func TestCompressedOres_Gneiss(t *testing.T) {
	store := seedIndustry(t)
	ores, err := store.CompressedOres(context.Background(), "test")
	if err != nil {
		t.Fatal(err)
	}
	if len(ores) != 1 {
		t.Fatalf("ores = %d, want 1", len(ores))
	}
	o := ores[0]
	if o.TypeID != 62555 || o.PortionSize != 100 || o.Volume != 0.05 || o.SkillID != 60379 || len(o.Materials) != 3 {
		t.Errorf("ore = %+v", o)
	}
}

func TestReprocessingMaterials(t *testing.T) {
	store := seedIndustry(t)
	got, err := store.ReprocessingMaterials(context.Background(), "test", 62555)
	if err != nil {
		t.Fatal(err)
	}
	want := []TypeQuantity{{35, 2300}, {36, 1725}, {37, 920}}
	if len(got) != len(want) {
		t.Fatalf("got %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestTypeBasics_PrefersPackagedVolume(t *testing.T) {
	store := seedIndustry(t)
	got, err := store.TypeBasics(context.Background(), "test", []int64{28846, 21039, 999})
	if err != nil {
		t.Fatal(err)
	}
	if got[28846].Volume != 1300000 {
		t.Errorf("Nomad volume = %v, want packaged 1300000", got[28846].Volume)
	}
	if got[21039].CategoryID == 0 || got[21039].GroupID != 873 {
		t.Errorf("siege array = %+v", got[21039])
	}
	if _, ok := got[999]; ok {
		t.Error("unknown type present")
	}
}

func TestTypesByName_CaseInsensitive(t *testing.T) {
	store := seedIndustry(t)
	got, err := store.TypesByName(context.Background(), "test", []string{"nomad", "No Such Thing"})
	if err != nil {
		t.Fatal(err)
	}
	if got["nomad"] != 28846 {
		t.Errorf("nomad = %d", got["nomad"])
	}
	if _, ok := got["No Such Thing"]; ok {
		t.Error("unknown name resolved")
	}
}

func TestNPCStationIndustryData_Jita(t *testing.T) {
	store := seedIndustry(t)
	st, ok, err := store.NPCStationIndustryData(context.Background(), "test", 60003760)
	if err != nil || !ok {
		t.Fatalf("ok %v err %v", ok, err)
	}
	if st.ReprocessingEfficiency != 0.5 || st.ReprocessingStationsTake != 0.05 || st.OwnerCorporationID != 1000035 || st.SystemID != 30000142 || st.Name != "Jita IV - Moon 4 - Caldari Navy Assembly Plant" {
		t.Errorf("station = %+v", st)
	}
}

func TestRigInfo_XLShipRig(t *testing.T) {
	store := seedIndustry(t)
	rig, ok, err := store.RigInfo(context.Background(), "test", 37180)
	if err != nil || !ok {
		t.Fatalf("ok %v err %v", ok, err)
	}
	want := []RigModifier{{2591, 2593}, {2592, 2594}}
	if len(rig.Modifiers) != 2 || rig.Modifiers[0] != want[0] || rig.Modifiers[1] != want[1] {
		t.Errorf("modifiers = %+v, want %+v", rig.Modifiers, want)
	}
	if rig.Attributes[AttrEngRigMatBonus] != -2 || rig.Attributes[AttrEngRigTimeBonus] != -20 || rig.Attributes[AttrNullSecModifier] != 2.1 || rig.Attributes[AttrLowSecModifier] != 1.9 {
		t.Errorf("attributes = %+v", rig.Attributes)
	}
	if rig.Size != 4 {
		t.Errorf("size = %d, want 4", rig.Size)
	}
}

func TestTypeAttributes_SotiyoRoleBonuses(t *testing.T) {
	store := seedIndustry(t)
	got, err := store.TypeAttributes(context.Background(), "test", 35827)
	if err != nil {
		t.Fatal(err)
	}
	if got[AttrStrEngMatBonus] != 0.99 || got[AttrStrEngTimeBonus] != 0.7 || got[AttrStrEngCostBonus] != 0.95 {
		t.Errorf("sotiyo = %+v", got)
	}
}

func TestRoutingSystems(t *testing.T) {
	store := seedIndustry(t)
	got, err := store.RoutingSystems(context.Background(), "test")
	if err != nil {
		t.Fatal(err)
	}
	by := map[string]RoutingSystem{}
	for _, s := range got {
		by[s.Name] = s
	}
	j, a := by["Jita"], by["Aunenen"]
	if !j.HasNPCStation || j.RegionName != "The Forge" || j.RegionID != 10000002 {
		t.Errorf("Jita = %+v", j)
	}
	ly := math.Sqrt((j.X-a.X)*(j.X-a.X)+(j.Y-a.Y)*(j.Y-a.Y)+(j.Z-a.Z)*(j.Z-a.Z)) / 9.46e15
	if math.Abs(ly-9.6275) > 0.0001 {
		t.Errorf("Jita-Aunenen = %.4f LY, want 9.628", ly)
	}
}

func TestStargateConnections(t *testing.T) {
	store := seedIndustry(t)
	got, err := store.StargateConnections(context.Background(), "test")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Errorf("connections = %+v, want Jita<->Perimeter both ways", got)
	}
}

func TestIndustryAccessors_NilStoreUnavailable(t *testing.T) {
	var store *Store
	if _, _, err := store.Blueprint(context.Background(), "test", 1); !errors.Is(err, ErrUnavailable) {
		t.Errorf("nil store err = %v, want ErrUnavailable", err)
	}
	if _, err := store.RoutingSystems(context.Background(), "test"); !errors.Is(err, ErrUnavailable) {
		t.Errorf("nil store err = %v, want ErrUnavailable", err)
	}
}

func TestIndustryModifiers_SotiyoAndRig(t *testing.T) {
	store := seedIndustry(t)
	sot, err := store.IndustryModifiers(context.Background(), "test", 35827)
	if err != nil {
		t.Fatal(err)
	}
	if len(sot) != 2 || sot[0] != (IndustryModifierSource{"manufacturing", "material", 2600, 0}) {
		t.Errorf("sotiyo = %+v", sot)
	}
	rig, err := store.IndustryModifiers(context.Background(), "test", 37180)
	if err != nil {
		t.Fatal(err)
	}
	if len(rig) != 2 || rig[0].FilterID != 3 {
		t.Errorf("rig = %+v", rig)
	}
}

func TestTargetFilters(t *testing.T) {
	store := seedIndustry(t)
	got, err := store.TargetFilters(context.Background(), "test")
	if err != nil {
		t.Fatal(err)
	}
	if f := got[3]; f.Name != "Ships" || len(f.Categories) != 2 || len(f.Groups) != 0 {
		t.Errorf("filter 3 = %+v", f)
	}
	if f := got[13]; len(f.Groups) != 1 || f.Groups[0] != 873 {
		t.Errorf("filter 13 = %+v", f)
	}
}
