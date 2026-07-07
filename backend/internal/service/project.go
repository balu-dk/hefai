package service

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

type ProjectRepo interface {
	ProjectAccess
	Create(ctx context.Context, p *domain.Project) (*domain.Project, error)
	Get(ctx context.Context, id uuid.UUID) (*domain.Project, error)
	ListForUser(ctx context.Context, userID uuid.UUID) ([]*domain.Project, error)
	Update(ctx context.Context, p *domain.Project) (*domain.Project, error)
	Delete(ctx context.Context, id uuid.UUID) error
	ListMembers(ctx context.Context, projectID uuid.UUID) ([]*domain.ProjectMember, error)
	AddMember(ctx context.Context, projectID, userID uuid.UUID, role domain.ProjectRole) error
	RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error
}

type Projects struct {
	repo  ProjectRepo
	users UserRepo
}

func NewProjects(repo ProjectRepo, users UserRepo) *Projects {
	return &Projects{repo: repo, users: users}
}

type ProjectInput struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Kind         string   `json:"kind"`
	Status       string   `json:"status"`
	Address      string   `json:"address"`
	Municipality string   `json:"municipality"`
	CadastralID  string   `json:"cadastralId"`
	PlotAreaM2   *float64 `json:"plotAreaM2"`
	Latitude     *float64 `json:"latitude"`
	Longitude    *float64 `json:"longitude"`
	UTMX         *float64 `json:"utmX"`
	UTMY         *float64 `json:"utmY"`
}

func (s *Projects) Create(ctx context.Context, userID uuid.UUID, in ProjectInput) (*domain.Project, error) {
	p, err := projectFromInput(in)
	if err != nil {
		return nil, err
	}
	p.CreatedBy = userID
	return s.repo.Create(ctx, p)
}

func (s *Projects) Get(ctx context.Context, userID, projectID uuid.UUID) (*domain.Project, error) {
	if _, err := requireRead(ctx, s.repo, projectID, userID); err != nil {
		return nil, err
	}
	return s.repo.Get(ctx, projectID)
}

func (s *Projects) List(ctx context.Context, userID uuid.UUID) ([]*domain.Project, error) {
	return s.repo.ListForUser(ctx, userID)
}

func (s *Projects) Update(ctx context.Context, userID, projectID uuid.UUID, in ProjectInput) (*domain.Project, error) {
	if err := requireWrite(ctx, s.repo, projectID, userID); err != nil {
		return nil, err
	}
	p, err := projectFromInput(in)
	if err != nil {
		return nil, err
	}
	p.ID = projectID
	return s.repo.Update(ctx, p)
}

func (s *Projects) Delete(ctx context.Context, userID, projectID uuid.UUID) error {
	if err := requireManage(ctx, s.repo, projectID, userID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, projectID)
}

func (s *Projects) ListMembers(ctx context.Context, userID, projectID uuid.UUID) ([]*domain.ProjectMember, error) {
	if _, err := requireRead(ctx, s.repo, projectID, userID); err != nil {
		return nil, err
	}
	return s.repo.ListMembers(ctx, projectID)
}

// AddMember invites an existing user by email.
func (s *Projects) AddMember(ctx context.Context, userID, projectID uuid.UUID, email string, role domain.ProjectRole) error {
	if err := requireManage(ctx, s.repo, projectID, userID); err != nil {
		return err
	}
	if !role.Valid() {
		return domain.Validation("ugyldig rolle")
	}
	invited, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if err == domain.ErrNotFound {
			return domain.Validation("ingen bruger med den e-mail — brugeren skal registrere sig først")
		}
		return err
	}
	if err := s.repo.AddMember(ctx, projectID, invited.ID, role); err != nil {
		if err == domain.ErrConflict {
			return domain.Validation("brugeren er allerede medlem")
		}
		return err
	}
	return nil
}

func (s *Projects) RemoveMember(ctx context.Context, userID, projectID, memberID uuid.UUID) error {
	if err := requireManage(ctx, s.repo, projectID, userID); err != nil {
		return err
	}
	// An owner cannot remove themselves; ownership must be transferred by
	// adding another owner first, which keeps the project reachable.
	if memberID == userID {
		members, err := s.repo.ListMembers(ctx, projectID)
		if err != nil {
			return err
		}
		owners := 0
		for _, m := range members {
			if m.Role == domain.RoleOwner {
				owners++
			}
		}
		if owners <= 1 {
			return domain.Validation("projektet skal have mindst én ejer")
		}
	}
	return s.repo.RemoveMember(ctx, projectID, memberID)
}

func projectFromInput(in ProjectInput) (*domain.Project, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, domain.Validation("projektnavn kræves")
	}
	kind := domain.ProjectKind(in.Kind)
	if in.Kind == "" {
		kind = domain.ProjectKindOther
	}
	if !kind.Valid() {
		return nil, domain.Validation("ugyldig projekttype")
	}
	status := domain.ProjectStatus(in.Status)
	if in.Status == "" {
		status = domain.ProjectStatusPlanning
	}
	if !status.Valid() {
		return nil, domain.Validation("ugyldig projektstatus")
	}
	if in.PlotAreaM2 != nil && *in.PlotAreaM2 < 0 {
		return nil, domain.Validation("grundareal kan ikke være negativt")
	}
	return &domain.Project{
		Name:         name,
		Description:  in.Description,
		Kind:         kind,
		Status:       status,
		Address:      strings.TrimSpace(in.Address),
		Municipality: strings.TrimSpace(in.Municipality),
		CadastralID:  strings.TrimSpace(in.CadastralID),
		PlotAreaM2:   in.PlotAreaM2,
		Latitude:     in.Latitude,
		Longitude:    in.Longitude,
		UTMX:         in.UTMX,
		UTMY:         in.UTMY,
	}, nil
}
