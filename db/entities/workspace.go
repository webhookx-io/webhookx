package entities

import (
	"github.com/webhookx-io/webhookx/pkg/types"
)

type Workspace struct {
	ID          string   `json:"id" db:"id"`
	Name        *string  `json:"name" db:"name"`
	Description *string  `json:"description" db:"description"`
	Metadata    Metadata `json:"metadata" db:"metadata"`

	CreatedAt types.Time `db:"created_at" json:"created_at"`
	UpdatedAt types.Time `db:"updated_at" json:"updated_at"`
}
