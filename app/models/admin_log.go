package models

import (
	"time"

	airwaysql "github.com/daqing/airway/lib/sql"
)

// AdminLog is one audit entry for a high-risk admin operation.
type AdminLog struct {
	ID        airwaysql.IdType `db:"id" json:"id"`
	AdminID   int64            `db:"admin_id" json:"admin_id"`
	Action    string           `db:"action" json:"action"`
	Entity    string           `db:"entity" json:"entity"`
	EntityID  string           `db:"entity_id" json:"entity_id"`
	Detail    string           `db:"detail" json:"detail"`
	CreatedAt time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt time.Time        `db:"updated_at" json:"updated_at"`
}

func (AdminLog) TableName() string {
	return "admin_logs"
}

func init() {
	registerREPLModel("AdminLog", AdminLog{})
}
