package models

import (
	"time"

	airwaysql "github.com/daqing/airway/lib/sql"
)

type Category struct {
	ID        airwaysql.IdType `db:"id" json:"id"`
	Name      string           `db:"name" json:"name"`
	Slug      string           `db:"slug" json:"slug"`
	ParentID  *int64           `db:"parent_id" json:"parent_id"`
	SortOrder int              `db:"sort_order" json:"sort_order"`
	Enabled   bool             `db:"enabled" json:"enabled"`
	CreatedAt time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt time.Time        `db:"updated_at" json:"updated_at"`
}

func (Category) TableName() string {
	return "categories"
}

func init() {
	registerREPLModel("Category", Category{})
}
