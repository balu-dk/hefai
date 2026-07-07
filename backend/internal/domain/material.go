package domain

import (
	"time"

	"github.com/google/uuid"
)

type MaterialStatus string

const (
	MaterialNeeded    MaterialStatus = "needed"
	MaterialOrdered   MaterialStatus = "ordered"
	MaterialDelivered MaterialStatus = "delivered"
	MaterialInStock   MaterialStatus = "in_stock"
	MaterialUsed      MaterialStatus = "used"
)

func (s MaterialStatus) Valid() bool {
	switch s {
	case MaterialNeeded, MaterialOrdered, MaterialDelivered, MaterialInStock, MaterialUsed:
		return true
	}
	return false
}

type Material struct {
	ID           uuid.UUID      `json:"id"`
	ProjectID    uuid.UUID      `json:"projectId"`
	PhaseID      *uuid.UUID     `json:"phaseId"`
	TaskID       *uuid.UUID     `json:"taskId"`
	RoomID       *uuid.UUID     `json:"roomId"`
	SupplierID   *uuid.UUID     `json:"supplierId"`
	Name         string         `json:"name"`
	Spec         string         `json:"spec"`
	Quantity     float64        `json:"quantity"`
	Unit         string         `json:"unit"`
	UnitPriceOre *int64         `json:"unitPriceOre"`
	Currency     string         `json:"currency"`
	Status       MaterialStatus `json:"status"`
	Notes        string         `json:"notes"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
}

// ShoppingListGroup is the purchase list for one supplier (nil SupplierID =
// materials with no supplier chosen yet).
type ShoppingListGroup struct {
	SupplierID   *uuid.UUID  `json:"supplierId"`
	SupplierName string      `json:"supplierName"`
	Materials    []*Material `json:"materials"`
	TotalOre     int64       `json:"totalOre"` // sum of quantity × unit price where known
}
