package models

import (
	"time"

	airwaysql "github.com/daqing/airway/lib/sql"
)

// Shipment is the one-per-order delivery record; Events holds a JSON array
// of {time, description} entries forming the tracking timeline.
type Shipment struct {
	ID         airwaysql.IdType `db:"id" json:"id"`
	OrderID    int64            `db:"order_id" json:"order_id"`
	Carrier    string           `db:"carrier" json:"carrier"`
	TrackingNo string           `db:"tracking_no" json:"tracking_no"`
	Status     string           `db:"status" json:"status"`
	Events     string           `db:"events" json:"events"`
	CreatedAt  time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time        `db:"updated_at" json:"updated_at"`
}

func (Shipment) TableName() string {
	return "shipments"
}

func init() {
	registerREPLModel("Shipment", Shipment{})
}
