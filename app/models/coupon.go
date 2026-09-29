package models

import (
	"time"

	airwaysql "github.com/daqing/airway/lib/sql"
)

type Coupon struct {
	ID             airwaysql.IdType `db:"id" json:"id"`
	Code           string           `db:"code" json:"code"`
	Type           string           `db:"type" json:"type"`
	ValueCents     int64            `db:"value_cents" json:"value_cents"`
	PercentOff     int              `db:"percent_off" json:"percent_off"`
	ThresholdCents int64            `db:"threshold_cents" json:"threshold_cents"`
	TotalCount     int              `db:"total_count" json:"total_count"`
	StartsAt       *time.Time       `db:"starts_at" json:"starts_at"`
	ExpiresAt      *time.Time       `db:"expires_at" json:"expires_at"`
	Enabled        bool             `db:"enabled" json:"enabled"`
	CreatedAt      time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time        `db:"updated_at" json:"updated_at"`
}

func (Coupon) TableName() string {
	return "coupons"
}

type CouponRedemption struct {
	ID            airwaysql.IdType `db:"id" json:"id"`
	CouponID      int64            `db:"coupon_id" json:"coupon_id"`
	UserID        int64            `db:"user_id" json:"user_id"`
	OrderID       int64            `db:"order_id" json:"order_id"`
	DiscountCents int64            `db:"discount_cents" json:"discount_cents"`
	CreatedAt     time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time        `db:"updated_at" json:"updated_at"`
}

func (CouponRedemption) TableName() string {
	return "coupon_redemptions"
}

func init() {
	registerREPLModel("Coupon", Coupon{})
	registerREPLModel("CouponRedemption", CouponRedemption{})
}
