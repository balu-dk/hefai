package domain

import (
	"time"

	"github.com/google/uuid"
)

type Supplier struct {
	ID            uuid.UUID `json:"id"`
	ProjectID     uuid.UUID `json:"projectId"`
	CompanyName   string    `json:"companyName"`
	ContactPerson string    `json:"contactPerson"`
	Trade         string    `json:"trade"`
	Phone         string    `json:"phone"`
	Email         string    `json:"email"`
	Notes         string    `json:"notes"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}
