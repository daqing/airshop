package models

import (
	"time"

	airwaysql "github.com/daqing/airway/lib/sql"
)

// CouponClaim records that a user claimed a coupon at the coupon center; a
// claim is required before the coupon can be applied at checkout.
type CouponClaim struct {
	ID        airwaysql.IdType `db:"id" json:"id"`
	CouponID  int64            `db:"coupon_id" json:"coupon_id"`
	UserID    int64            `db:"user_id" json:"user_id"`
	CreatedAt time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt time.Time        `db:"updated_at" json:"updated_at"`
}

func (CouponClaim) TableName() string {
	return "coupon_claims"
}

func init() {
	registerREPLModel("CouponClaim", CouponClaim{})
}
