package models

import (
	"time"

	airwaysql "github.com/daqing/airway/lib/sql"
)

type ProductImage struct {
	ID        airwaysql.IdType `db:"id" json:"id"`
	ProductID int64            `db:"product_id" json:"product_id"`
	Key       string           `db:"key" json:"key"`
	SortOrder int              `db:"sort_order" json:"sort_order"`
	CreatedAt time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt time.Time        `db:"updated_at" json:"updated_at"`
}

func (ProductImage) TableName() string {
	return "product_images"
}

func init() {
	registerREPLModel("ProductImage", ProductImage{})
}
