package entities

import (
	"github.com/webhookx-io/webhookx/pkg/types"
	"github.com/webhookx-io/webhookx/utils"
)

type Workspace struct {
	ID          string  `json:"id" db:"id"`
	Name        *string `json:"name" db:"name"`
	Description *string `json:"description" db:"description"`

	CreatedAt types.Time `db:"created_at" json:"created_at"`
	UpdatedAt types.Time `db:"updated_at" json:"updated_at"`
}

func (m *Workspace) Validate() error {
	return utils.Validate(m)
}
