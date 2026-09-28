package models

import (
	"time"

	airwaysql "github.com/daqing/airway/lib/sql"
)

type ProductVariant struct {
	ID         airwaysql.IdType `db:"id" json:"id"`
	ProductID  int64            `db:"product_id" json:"product_id"`
	Name       string           `db:"name" json:"name"`
	PriceCents int64            `db:"price_cents" json:"price_cents"`
	Stock      int              `db:"stock" json:"stock"`
	Active     bool             `db:"active" json:"active"`
	SortOrder  int              `db:"sort_order" json:"sort_order"`
	CreatedAt  time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time        `db:"updated_at" json:"updated_at"`
}

func (ProductVariant) TableName() string {
	return "product_variants"
}

func init() {
	registerREPLModel("ProductVariant", ProductVariant{})
}
