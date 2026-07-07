package service

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

type RoomRepo interface {
	Create(ctx context.Context, m *domain.Room) (*domain.Room, error)
	Get(ctx context.Context, id uuid.UUID) (*domain.Room, error)
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]*domain.Room, error)
	Update(ctx context.Context, m *domain.Room) (*domain.Room, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type Rooms struct {
	repo   RoomRepo
	access ProjectAccess
}

func NewRooms(repo RoomRepo, access ProjectAccess) *Rooms {
	return &Rooms{repo: repo, access: access}
}

type RoomPatch struct {
	Name        *string  `json:"name"`
	Kind        *string  `json:"kind"`
	Description *string  `json:"description"`
	AreaM2      *float64 `json:"areaM2"`
}

func (s *Rooms) Create(ctx context.Context, userID, projectID uuid.UUID, patch RoomPatch) (*domain.Room, error) {
	if err := requireWrite(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	m := &domain.Room{ProjectID: projectID, Kind: domain.RoomKindRoom}
	if err := applyRoomPatch(m, patch); err != nil {
		return nil, err
	}
	if m.Name == "" {
		return nil, domain.Validation("rumnavn kræves")
	}
	return s.repo.Create(ctx, m)
}

func (s *Rooms) List(ctx context.Context, userID, projectID uuid.UUID) ([]*domain.Room, error) {
	if _, err := requireRead(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	return s.repo.ListByProject(ctx, projectID)
}

func (s *Rooms) Update(ctx context.Context, userID, roomID uuid.UUID, patch RoomPatch) (*domain.Room, error) {
	m, err := s.repo.Get(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if err := requireWrite(ctx, s.access, m.ProjectID, userID); err != nil {
		return nil, err
	}
	if err := applyRoomPatch(m, patch); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, m)
}

func (s *Rooms) Delete(ctx context.Context, userID, roomID uuid.UUID) error {
	m, err := s.repo.Get(ctx, roomID)
	if err != nil {
		return err
	}
	if err := requireWrite(ctx, s.access, m.ProjectID, userID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, roomID)
}

func applyRoomPatch(m *domain.Room, patch RoomPatch) error {
	if patch.Name != nil {
		m.Name = strings.TrimSpace(*patch.Name)
		if m.Name == "" {
			return domain.Validation("rumnavn kan ikke være tomt")
		}
	}
	if patch.Kind != nil {
		kind := domain.RoomKind(*patch.Kind)
		if !kind.Valid() {
			return domain.Validation("ugyldig rumtype")
		}
		m.Kind = kind
	}
	if patch.Description != nil {
		m.Description = *patch.Description
	}
	if patch.AreaM2 != nil {
		if *patch.AreaM2 < 0 {
			return domain.Validation("areal kan ikke være negativt")
		}
		m.AreaM2 = patch.AreaM2
	}
	return nil
}
