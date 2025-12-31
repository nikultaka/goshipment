package models

import (
	"database/sql"
	"time"
)

// BaseModel contains common fields for all models
type BaseModel struct {
	ID        string       `json:"id" db:"id"`
	CreatedAt time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt time.Time    `json:"updated_at" db:"updated_at"`
	DeletedAt sql.NullTime `json:"deleted_at,omitempty" db:"deleted_at"`
}

// IsDeleted returns true if the model is soft deleted
func (b *BaseModel) IsDeleted() bool {
	return b.DeletedAt.Valid
}
