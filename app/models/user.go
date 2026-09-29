package models

import (
	"time"

	airwaysql "github.com/daqing/airway/lib/sql"
)

type User struct {
	ID           airwaysql.IdType `db:"id" json:"id"`
	PhoneNumber  string           `db:"phone_number" json:"phone_number"`
	PasswordHash string           `db:"password_hash" json:"-"`
	DisplayName  string           `db:"display_name" json:"display_name"`
	Status       string           `db:"status" json:"status"`
	CreatedAt    time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time        `db:"updated_at" json:"updated_at"`
}

func (User) TableName() string {
	return "users"
}

func init() {
	registerREPLModel("User", User{})
}
