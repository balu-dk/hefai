// Package repository implements persistence against PostgreSQL using pgx.
// Repositories translate database errors into domain sentinel errors and
// contain no business logic.
package repository

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

// mapErr translates pgx/PostgreSQL errors into domain errors.
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return domain.ErrConflict
		case "23503": // foreign_key_violation
			return domain.Validation("referenced entity does not exist")
		case "23514": // check_violation
			return domain.Validation("constraint violated: " + pgErr.ConstraintName)
		}
	}
	return err
}
