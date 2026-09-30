package models

import (
	"time"

	airwaysql "github.com/daqing/airway/lib/sql"
)

type AdminUser struct {
	ID           airwaysql.IdType `db:"id" json:"id"`
	Username     string           `db:"username" json:"username"`
	PasswordHash string           `db:"password_hash" json:"-"`
	DisplayName  string           `db:"display_name" json:"display_name"`
	Status       string           `db:"status" json:"status"`
	CreatedAt    time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time        `db:"updated_at" json:"updated_at"`
}

func (AdminUser) TableName() string {
	return "admin_users"
}

type AdminSession struct {
	ID        airwaysql.IdType `db:"id" json:"id"`
	Token     string           `db:"token" json:"-"`
	AdminID   int64            `db:"admin_id" json:"admin_id"`
	ExpiresAt time.Time        `db:"expires_at" json:"expires_at"`
	CreatedAt time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt time.Time        `db:"updated_at" json:"updated_at"`
}

func (AdminSession) TableName() string {
	return "admin_sessions"
}

func init() {
	registerREPLModel("AdminUser", AdminUser{})
	registerREPLModel("AdminSession", AdminSession{})
}
