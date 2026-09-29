package models

import (
	"time"

	airwaysql "github.com/daqing/airway/lib/sql"
)

type Address struct {
	ID        airwaysql.IdType `db:"id" json:"id"`
	UserID    int64            `db:"user_id" json:"user_id"`
	Recipient string           `db:"recipient" json:"recipient"`
	Phone     string           `db:"phone" json:"phone"`
	Province  string           `db:"province" json:"province"`
	City      string           `db:"city" json:"city"`
	District  string           `db:"district" json:"district"`
	Street    string           `db:"street" json:"street"`
	IsDefault bool             `db:"is_default" json:"is_default"`
	CreatedAt time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt time.Time        `db:"updated_at" json:"updated_at"`
}

func (Address) TableName() string {
	return "addresses"
}

func init() {
	registerREPLModel("Address", Address{})
}
