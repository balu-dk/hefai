// Package service holds the business logic. Services consume repository
// interfaces (defined here, implemented in package repository), enforce
// authorization per project role, and validate input.
package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

// ProjectAccess answers "what role does this user have on this project".
// Implemented by repository.Projects.
type ProjectAccess interface {
	GetMemberRole(ctx context.Context, projectID, userID uuid.UUID) (domain.ProjectRole, error)
}

// requireRead returns the caller's role or ErrForbidden.
func requireRead(ctx context.Context, access ProjectAccess, projectID, userID uuid.UUID) (domain.ProjectRole, error) {
	return access.GetMemberRole(ctx, projectID, userID)
}

// requireWrite ensures the caller may modify project content.
func requireWrite(ctx context.Context, access ProjectAccess, projectID, userID uuid.UUID) error {
	role, err := access.GetMemberRole(ctx, projectID, userID)
	if err != nil {
		return err
	}
	if !role.CanWrite() {
		return domain.ErrForbidden
	}
	return nil
}

// requireManage ensures the caller owns the project.
func requireManage(ctx context.Context, access ProjectAccess, projectID, userID uuid.UUID) error {
	role, err := access.GetMemberRole(ctx, projectID, userID)
	if err != nil {
		return err
	}
	if !role.CanManage() {
		return domain.ErrForbidden
	}
	return nil
}
