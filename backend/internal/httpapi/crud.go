package httpapi

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// Generic handler factories for the recurring CRUD shapes: create/list under
// a parent resource, patch/delete by own ID. Services keep the full
// authorization and validation logic; these only adapt HTTP.

// createUnder handles POST /..{param}../sub with a JSON patch body.
func createUnder[P, R any](fn func(context.Context, uuid.UUID, uuid.UUID, P) (R, error), param string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parentID, err := pathUUID(r, param)
		if err != nil {
			writeError(w, err)
			return
		}
		var patch P
		if err := decode(r, &patch); err != nil {
			writeError(w, err)
			return
		}
		result, err := fn(r.Context(), userID(r), parentID, patch)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, result)
	}
}

// getUnder handles GET /..{param}../sub (lists, summaries, exports).
func getUnder[R any](fn func(context.Context, uuid.UUID, uuid.UUID) (R, error), param string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parentID, err := pathUUID(r, param)
		if err != nil {
			writeError(w, err)
			return
		}
		result, err := fn(r.Context(), userID(r), parentID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

// patchByID handles PATCH /resource/{param} with a JSON patch body.
func patchByID[P, R any](fn func(context.Context, uuid.UUID, uuid.UUID, P) (R, error), param string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := pathUUID(r, param)
		if err != nil {
			writeError(w, err)
			return
		}
		var patch P
		if err := decode(r, &patch); err != nil {
			writeError(w, err)
			return
		}
		result, err := fn(r.Context(), userID(r), id, patch)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

// deleteByID handles DELETE /resource/{param}.
func deleteByID(fn func(context.Context, uuid.UUID, uuid.UUID) error, param string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := pathUUID(r, param)
		if err != nil {
			writeError(w, err)
			return
		}
		if err := fn(r.Context(), userID(r), id); err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusNoContent, nil)
	}
}
