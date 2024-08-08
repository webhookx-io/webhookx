package entities

import "github.com/webhookx-io/webhookx/utils"

type Workspace struct {
	ID          string  `json:"id" db:"id"`
	Name        *string `json:"name" db:"name"`
	Description *string `json:"description" db:"description"`

	CreatedAt UnixTime `db:"created_at" json:"created_at"`
	UpdatedAt UnixTime `db:"updated_at" json:"updated_at"`
}

func (m *Workspace) Validate() error {
	return utils.Validate(m)
}
