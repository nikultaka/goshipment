package models

import (
	"database/sql"
	"time"
)

// TeamMember represents a team member in the directory
type TeamMember struct {
	ID          string         `json:"id" db:"id"`
	UserID      string         `json:"user_id" db:"user_id"`
	Department  string         `json:"department" db:"department"`
	Position    string         `json:"position" db:"position"`
	Bio         sql.NullString `json:"bio,omitempty" db:"bio"`
	Skills      sql.NullString `json:"skills,omitempty" db:"skills"` // JSON array
	LinkedIn    sql.NullString `json:"linkedin,omitempty" db:"linkedin"`
	Twitter     sql.NullString `json:"twitter,omitempty" db:"twitter"`
	GitHub      sql.NullString `json:"github,omitempty" db:"github"`
	Visibility  string         `json:"visibility" db:"visibility"` // public, internal, private
	IsActive    bool           `json:"is_active" db:"is_active"`
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at" db:"updated_at"`
	DeletedAt   sql.NullTime   `json:"deleted_at,omitempty" db:"deleted_at"`
}

// IsVisible returns true if the team member profile is visible
func (tm *TeamMember) IsVisible() bool {
	return tm.IsActive && !tm.DeletedAt.Valid
}

// IsPublic returns true if the profile is publicly visible
func (tm *TeamMember) IsPublic() bool {
	return tm.Visibility == "public"
}
