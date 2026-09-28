package models

import (
	"time"

	airwaysql "github.com/daqing/airway/lib/sql"
)

type Product struct {
	ID          airwaysql.IdType `db:"id" json:"id"`
	CategoryID  *int64           `db:"category_id" json:"category_id"`
	Name        string           `db:"name" json:"name"`
	Slug        string           `db:"slug" json:"slug"`
	Description string           `db:"description" json:"description"`
	PriceCents  int64            `db:"price_cents" json:"price_cents"`
	Stock       int              `db:"stock" json:"stock"`
	Active      bool             `db:"active" json:"active"`
	MainImage   string           `db:"main_image" json:"main_image"`
	CreatedAt   time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time        `db:"updated_at" json:"updated_at"`
}

func (Product) TableName() string {
	return "products"
}

func init() {
	registerREPLModel("Product", Product{})
}
