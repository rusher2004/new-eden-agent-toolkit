package industry_test

import (
	"math"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/industry"
)

// Real SDE values, copied 2026-11-18.
var (
	filters = map[int64]industry.TargetFilter{
		3:  {ID: 3, Name: "Ships", Categories: []int64{6, 32}},
		13: {ID: 13, Name: "Capital Components", Groups: []int64{873}},
		18: {ID: 18, Name: "Composite Reactions", Groups: []int64{428, 429, 4932}},
	}
	engAll = func(mat, cost, time int64) []industry.ModifierSource {
		var out []industry.ModifierSource
		for _, a := range []string{industry.ActManufacturing, industry.ActCopying, industry.ActInvention, industry.ActResearchMaterial, industry.ActResearchTime} {
			if a == industry.ActManufacturing {
				out = append(out, industry.ModifierSource{Activity: a, Kind: industry.KindMaterial, AttributeID: mat})
			}
			out = append(out, industry.ModifierSource{Activity: a, Kind: industry.KindCost, AttributeID: cost}, industry.ModifierSource{Activity: a, Kind: industry.KindTime, AttributeID: time})
		}
		return out
	}
	sotiyo   = &industry.StructureInput{TypeID: 35827, Name: "Sotiyo", Attributes: map[int64]float64{1547: 4, 1970: 1, 2600: 0.99, 2601: 0.95, 2602: 0.7}, Sources: engAll(2600, 2601, 2602)}
	raitaru  = &industry.StructureInput{TypeID: 35825, Name: "Raitaru", Attributes: map[int64]float64{1547: 2, 2600: 0.99, 2601: 0.97, 2602: 0.85}, Sources: engAll(2600, 2601, 2602)}
	azbel    = &industry.StructureInput{TypeID: 35826, Name: "Azbel", Attributes: map[int64]float64{1547: 3, 2600: 0.99, 2601: 0.96, 2602: 0.8}, Sources: engAll(2600, 2601, 2602)}
	astrahus = &industry.StructureInput{TypeID: 35832, Name: "Astrahus", Attributes: map[int64]float64{1547: 2}}
	tatara   = &industry.StructureInput{TypeID: 35836, Name: "Tatara", Attributes: map[int64]float64{1547: 3, 2721: 0.75}, Sources: []industry.ModifierSource{{Activity: industry.ActReaction, Kind: industry.KindTime, AttributeID: 2721}}}

	xlShipRig = func(mat, time float64) industry.RigInput {
		return industry.RigInput{Name: "Standup XL-Set Ship Manufacturing Efficiency", Attributes: map[int64]float64{1547: 4, 2355: 1, 2356: 1.9, 2357: 2.1, 2593: time, 2594: mat},
			Sources:        []industry.ModifierSource{{industry.ActManufacturing, industry.KindMaterial, 2592, 3}, {industry.ActManufacturing, industry.KindTime, 2591, 3}},
			TargetToSource: map[int64]int64{2592: 2594, 2591: 2593}}
	}
	xlCompRig = industry.RigInput{Name: "Standup XL-Set Structure and Component Manufacturing Efficiency I", Attributes: map[int64]float64{1547: 4, 2355: 1, 2356: 1.9, 2357: 2.1, 2593: -20, 2594: -2},
		Sources:        []industry.ModifierSource{{industry.ActManufacturing, industry.KindMaterial, 2559, 13}, {industry.ActManufacturing, industry.KindTime, 2560, 13}},
		TargetToSource: map[int64]int64{2559: 2594, 2560: 2593}}
	reactorRig = industry.RigInput{Name: "Standup L-Set Reactor Efficiency I", Attributes: map[int64]float64{1547: 3, 1970: 1, 2356: 1.0, 2357: 1.1, 2713: -20, 2714: -2},
		Sources:        []industry.ModifierSource{{industry.ActReaction, industry.KindMaterial, 2718, 18}, {industry.ActReaction, industry.KindTime, 2717, 18}},
		TargetToSource: map[int64]int64{2718: 2714, 2717: 2713}}
)

const (
	nullSecurity = -0.134402 // PF-QHK
	nullRegion   = 10000015  // Venal
	minokawa     = 1538      // Force Auxiliary group, category 6
	capComp      = 873       // Capital Construction Components, category 17
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestFacility_SotiyoNullsec_Minokawa(t *testing.T) {
	f := industry.ResolveFacility(industry.ResolveFacilityInput{Name: "Nullsec Sotiyo", Kind: industry.KindUpwell, Structure: sotiyo, Rigs: []industry.RigInput{xlShipRig(-2, -20), xlCompRig}, Filters: filters, Security: nullSecurity, RegionID: nullRegion})
	hull := f.Multipliers(industry.ActManufacturing, 6, minokawa)
	if !near(hull.Material, 0.99*0.958) || !near(hull.Time, 0.7*0.58) || !near(hull.Cost, 0.95) {
		t.Errorf("hull = %+v", hull)
	}
	comp := f.Multipliers(industry.ActManufacturing, 17, capComp)
	if !near(comp.Material, 0.99*0.958) || !near(comp.Time, 0.7*0.58) {
		t.Errorf("component = %+v", comp)
	}
	if len(f.Warnings) != 0 {
		t.Errorf("warnings = %v", f.Warnings)
	}
}

func TestFacility_RigSecurityScaling(t *testing.T) {
	tests := []struct {
		name     string
		sec      float64
		region   int64
		rigMat   float64
		wantMat  float64
		warnings int
	}{
		{"T1 lowsec", 0.3, 10000016, -2, 0.99 * (1 - 0.038), 0},
		{"T2 nullsec", -0.5, nullRegion, -2.4, 0.99 * (1 - 0.0504), 0},
		{"T1 wormhole uses nullsec", -1, 11000001, -2, 0.99 * 0.958, 0},
		{"T1 highsec, Sotiyo warns", 0.9, 10000002, -2, 0.99 * 0.98, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := industry.ResolveFacility(industry.ResolveFacilityInput{Kind: industry.KindUpwell, Structure: sotiyo, Rigs: []industry.RigInput{xlShipRig(tt.rigMat, -20)}, Filters: filters, Security: tt.sec, RegionID: tt.region})
			if got := f.Multipliers(industry.ActManufacturing, 6, minokawa).Material; !near(got, tt.wantMat) {
				t.Errorf("material = %v, want %v", got, tt.wantMat)
			}
			if len(f.Warnings) != tt.warnings {
				t.Errorf("warnings = %v", f.Warnings)
			}
		})
	}
}

func TestFacility_RigOnlyHitsItsFilter(t *testing.T) {
	f := industry.ResolveFacility(industry.ResolveFacilityInput{Kind: industry.KindUpwell, Structure: sotiyo, Rigs: []industry.RigInput{xlShipRig(-2, -20)}, Filters: filters, Security: nullSecurity, RegionID: nullRegion})
	comp := f.Multipliers(industry.ActManufacturing, 17, capComp)
	if !near(comp.Material, 0.99) || !near(comp.Time, 0.7) {
		t.Errorf("ship rig leaked onto components: %+v", comp)
	}
}

func TestFacility_TataraReactions(t *testing.T) {
	f := industry.ResolveFacility(industry.ResolveFacilityInput{Name: "Nullsec Tatara", Kind: industry.KindUpwell, Structure: tatara, Rigs: []industry.RigInput{reactorRig}, Filters: filters, Security: nullSecurity, RegionID: nullRegion, Services: []string{industry.SvcCompositeReactor}})
	m := f.Multipliers(industry.ActReaction, 4, 429)
	if !near(m.Material, 0.978) || !near(m.Time, 0.75*0.78) {
		t.Errorf("reaction = %+v, want material 0.978 (the rig's -2.2%% in nullsec) and time 0.585", m)
	}
	if err := f.CanRun(industry.ActReaction, industry.Product{Name: "Titanium Carbide", CategoryID: 4, GroupID: 429}); err != nil {
		t.Errorf("CanRun = %v", err)
	}
}

func TestFacility_ReactorRigInHighsecIgnored(t *testing.T) {
	f := industry.ResolveFacility(industry.ResolveFacilityInput{Kind: industry.KindUpwell, Structure: tatara, Rigs: []industry.RigInput{reactorRig}, Filters: filters, Security: 0.8, RegionID: 10000002})
	if len(f.Warnings) != 1 {
		t.Errorf("warnings = %v", f.Warnings)
	}
	if err := f.CanRun(industry.ActReaction, industry.Product{Name: "Titanium Carbide", GroupID: 429}); err == nil {
		t.Error("highsec reaction allowed")
	}
}

func TestFacility_OtherStructures(t *testing.T) {
	tests := []struct {
		name      string
		structure *industry.StructureInput
		sec       float64
		wantMat   float64
		wantTime  float64
		wantCost  float64
	}{
		{"Raitaru highsec", raitaru, 0.9, 0.99, 0.85, 0.97},
		{"Azbel lowsec", azbel, 0.3, 0.99, 0.8, 0.96},
		{"Astrahus has no role bonus", astrahus, 0.9, 1, 1, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := industry.ResolveFacility(industry.ResolveFacilityInput{Kind: industry.KindUpwell, Structure: tt.structure, Filters: filters, Security: tt.sec, RegionID: 10000002})
			m := f.Multipliers(industry.ActManufacturing, 7, 60)
			if !near(m.Material, tt.wantMat) || !near(m.Time, tt.wantTime) || !near(m.Cost, tt.wantCost) {
				t.Errorf("multipliers = %+v", m)
			}
		})
	}
}

func TestFacility_NPCStation(t *testing.T) {
	f := industry.ResolveFacility(industry.ResolveFacilityInput{Name: "Jita 4-4", Kind: industry.KindNPCStation, Security: 0.9459, RegionID: 10000002, FacilityTax: 0.5})
	m := f.Multipliers(industry.ActManufacturing, 6, 25)
	if m.Material != 1 || m.Time != 1 || m.Cost != 1 || f.FacilityTax != industry.NPCFacilityTax {
		t.Errorf("npc = %+v tax %v", m, f.FacilityTax)
	}
}

func TestFacility_CanRun_Capitals(t *testing.T) {
	carrier := industry.Product{Name: "Thanatos", CategoryID: 6, GroupID: 547}
	titan := industry.Product{Name: "Avatar", CategoryID: 6, GroupID: 30}
	freighter := industry.Product{Name: "Charon", CategoryID: 6, GroupID: 513}
	tests := []struct {
		name    string
		in      industry.ResolveFacilityInput
		product industry.Product
		ok      bool
	}{
		{"Sotiyo nullsec with shipyard", industry.ResolveFacilityInput{Kind: industry.KindUpwell, Structure: sotiyo, Security: -0.5, RegionID: nullRegion, Services: []string{industry.SvcManufacturing, industry.SvcCapitalShipyard}}, carrier, true},
		{"Sotiyo without capital shipyard", industry.ResolveFacilityInput{Kind: industry.KindUpwell, Structure: sotiyo, Security: -0.5, RegionID: nullRegion, Services: []string{industry.SvcManufacturing}}, carrier, false},
		{"Raitaru too small", industry.ResolveFacilityInput{Kind: industry.KindUpwell, Structure: raitaru, Security: -0.5, RegionID: nullRegion, Services: []string{industry.SvcManufacturing, industry.SvcCapitalShipyard}}, carrier, false},
		{"Azbel highsec", industry.ResolveFacilityInput{Kind: industry.KindUpwell, Structure: azbel, Security: 0.9, RegionID: 10000002, Services: []string{industry.SvcManufacturing, industry.SvcCapitalShipyard}}, carrier, false},
		{"lowsec NPC station", industry.ResolveFacilityInput{Kind: industry.KindNPCStation, Security: 0.3, RegionID: 10000016}, carrier, true},
		{"titan needs sov upgrade", industry.ResolveFacilityInput{Kind: industry.KindUpwell, Structure: sotiyo, Security: -0.5, RegionID: nullRegion, Services: []string{industry.SvcManufacturing, industry.SvcSupercapShipyard}}, titan, false},
		{"titan with everything", industry.ResolveFacilityInput{Kind: industry.KindUpwell, Structure: sotiyo, Security: -0.5, RegionID: nullRegion, Services: []string{industry.SvcManufacturing, industry.SvcSupercapShipyard}, SupercapUpgrade: true}, titan, true},
		{"freighter in highsec Raitaru", industry.ResolveFacilityInput{Kind: industry.KindUpwell, Structure: raitaru, Security: 0.9, RegionID: 10000002, Services: []string{industry.SvcManufacturing}}, freighter, true},
		{"freighter at highsec starbase", industry.ResolveFacilityInput{Kind: industry.KindStarbase, Security: 0.9, RegionID: 10000002}, freighter, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.in.Name = tt.name
			err := industry.ResolveFacility(tt.in).CanRun(industry.ActManufacturing, tt.product)
			if (err == nil) != tt.ok {
				t.Errorf("CanRun = %v, want ok=%v", err, tt.ok)
			}
		})
	}
}

func TestFacility_TaxCap(t *testing.T) {
	f := industry.ResolveFacility(industry.ResolveFacilityInput{Kind: industry.KindUpwell, Structure: raitaru, Security: 0.9, RegionID: 10000002, FacilityTax: 0.2})
	if f.FacilityTax != industry.IndustryTaxCap || len(f.Warnings) != 1 {
		t.Errorf("tax = %v warnings %v", f.FacilityTax, f.Warnings)
	}
}
