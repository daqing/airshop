package models

import (
	"time"

	airwaysql "github.com/daqing/airway/lib/sql"
)

type Order struct {
	ID            airwaysql.IdType `db:"id" json:"id"`
	OrderNo       string           `db:"order_no" json:"order_no"`
	UserID        int64            `db:"user_id" json:"user_id"`
	ShipRecipient string           `db:"ship_recipient" json:"ship_recipient"`
	ShipPhone     string           `db:"ship_phone" json:"ship_phone"`
	ShipAddress   string           `db:"ship_address" json:"ship_address"`
	SubtotalCents int64            `db:"subtotal_cents" json:"subtotal_cents"`
	DiscountCents int64            `db:"discount_cents" json:"discount_cents"`
	ShippingCents int64            `db:"shipping_cents" json:"shipping_cents"`
	TotalCents    int64            `db:"total_cents" json:"total_cents"`
	Status        string           `db:"status" json:"status"`
	PaymentMethod string           `db:"payment_method" json:"payment_method"`
	CreatedAt     time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time        `db:"updated_at" json:"updated_at"`
}

func (Order) TableName() string {
	return "orders"
}

type OrderItem struct {
	ID             airwaysql.IdType `db:"id" json:"id"`
	OrderID        int64            `db:"order_id" json:"order_id"`
	ProductID      int64            `db:"product_id" json:"product_id"`
	VariantID      *int64           `db:"variant_id" json:"variant_id"`
	ProductName    string           `db:"product_name" json:"product_name"`
	VariantName    string           `db:"variant_name" json:"variant_name"`
	ImageKey       string           `db:"image_key" json:"image_key"`
	UnitPriceCents int64            `db:"unit_price_cents" json:"unit_price_cents"`
	Quantity       int              `db:"quantity" json:"quantity"`
	CreatedAt      time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time        `db:"updated_at" json:"updated_at"`
}

func (OrderItem) TableName() string {
	return "order_items"
}

func init() {
	registerREPLModel("Order", Order{})
	registerREPLModel("OrderItem", OrderItem{})
}
