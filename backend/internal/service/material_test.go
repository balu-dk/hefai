package service

import (
	"testing"

	"github.com/google/uuid"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

func TestBuildShoppingList(t *testing.T) {
	supA, supB := uuid.New(), uuid.New()
	names := map[uuid.UUID]string{supA: "Stark", supB: "Bygma"}
	price := func(ore int64) *int64 { return &ore }

	materials := []*domain.Material{
		{Name: "Spær", SupplierID: &supA, Status: domain.MaterialNeeded, Quantity: 10, UnitPriceOre: price(45000)},
		{Name: "Skruer", SupplierID: &supA, Status: domain.MaterialNeeded, Quantity: 3, UnitPriceOre: price(9900)},
		{Name: "Mursten", SupplierID: &supB, Status: domain.MaterialOrdered, Quantity: 500}, // ordered: not on list
		{Name: "Isolering", Status: domain.MaterialNeeded, Quantity: 20},                    // no supplier
		{Name: "Maling", SupplierID: &supB, Status: domain.MaterialNeeded, Quantity: 4},     // no price
	}

	groups := buildShoppingList(materials, names)

	if len(groups) != 3 {
		t.Fatalf("expected 3 groups (Stark, Bygma, uden leverandør), got %d", len(groups))
	}
	byName := map[string]*domain.ShoppingListGroup{}
	for _, g := range groups {
		byName[g.SupplierName] = g
	}

	stark := byName["Stark"]
	if len(stark.Materials) != 2 {
		t.Errorf("Stark should have 2 materials, got %d", len(stark.Materials))
	}
	if want := int64(10*45000 + 3*9900); stark.TotalOre != want {
		t.Errorf("Stark total = %d øre, want %d", stark.TotalOre, want)
	}
	if len(byName["Bygma"].Materials) != 1 {
		t.Errorf("Bygma should only list needed materials (Maling), got %d", len(byName["Bygma"].Materials))
	}
	if byName["Bygma"].TotalOre != 0 {
		t.Errorf("Bygma total should be 0 without prices, got %d", byName["Bygma"].TotalOre)
	}
	if len(byName["Uden leverandør"].Materials) != 1 {
		t.Error("materials without supplier should group under 'Uden leverandør'")
	}
}
