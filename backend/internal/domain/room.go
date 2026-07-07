package domain

import (
	"time"

	"github.com/google/uuid"
)

type RoomKind string

const (
	RoomKindRoom    RoomKind = "room"
	RoomKindZone    RoomKind = "zone"
	RoomKindOutdoor RoomKind = "outdoor"
)

func (k RoomKind) Valid() bool {
	switch k {
	case RoomKindRoom, RoomKindZone, RoomKindOutdoor:
		return true
	}
	return false
}

type Room struct {
	ID          uuid.UUID `json:"id"`
	ProjectID   uuid.UUID `json:"projectId"`
	Name        string    `json:"name"`
	Kind        RoomKind  `json:"kind"`
	Description string    `json:"description"`
	AreaM2      *float64  `json:"areaM2"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
