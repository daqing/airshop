package models

import (
	"time"

	airwaysql "github.com/daqing/airway/lib/sql"
)

type Cart struct {
	ID        airwaysql.IdType `db:"id" json:"id"`
	UserID    int64            `db:"user_id" json:"user_id"`
	CreatedAt time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt time.Time        `db:"updated_at" json:"updated_at"`
}

func (Cart) TableName() string {
	return "carts"
}

type CartItem struct {
	ID         airwaysql.IdType `db:"id" json:"id"`
	CartID     int64            `db:"cart_id" json:"cart_id"`
	ProductID  int64            `db:"product_id" json:"product_id"`
	VariantID  *int64           `db:"variant_id" json:"variant_id"`
	Quantity   int              `db:"quantity" json:"quantity"`
	PriceCents int64            `db:"price_cents" json:"price_cents"`
	CreatedAt  time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time        `db:"updated_at" json:"updated_at"`
}

func (CartItem) TableName() string {
	return "cart_items"
}

func init() {
	registerREPLModel("Cart", Cart{})
	registerREPLModel("CartItem", CartItem{})
}
