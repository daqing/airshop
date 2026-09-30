package services

import (
	"testing"

	"github.com/daqing/airshop/app/models"
)

// modelsCouponForTest builds a models.Coupon without repeating the struct.
type modelsCouponForTest models.Coupon

func couponAsCoupon(c *modelsCouponForTest) *models.Coupon {
	out := models.Coupon(*c)
	return &out
}

func TestCanTransitionOrder(t *testing.T) {
	// The full happy chain is legal.
	chain := map[string][]string{
		OrderStatusPending:   {OrderStatusPaid, OrderStatusCancelled},
		OrderStatusPaid:      {OrderStatusShipped, OrderStatusRefunded},
		OrderStatusShipped:   {OrderStatusCompleted, OrderStatusRefunded},
		OrderStatusCompleted: {OrderStatusRefunded},
	}
	for from, allowed := range chain {
		for _, to := range allowed {
			if !CanTransitionOrder(from, to) {
				t.Fatalf("expected %s -> %s to be allowed", from, to)
			}
		}
	}

	// Everything else is illegal: sampled jumps plus terminal statuses.
	illegal := []struct{ from, to string }{
		{OrderStatusPending, OrderStatusShipped},
		{OrderStatusPending, OrderStatusCompleted},
		{OrderStatusPending, OrderStatusRefunded},
		{OrderStatusPaid, OrderStatusCancelled},
		{OrderStatusPaid, OrderStatusCompleted},
		{OrderStatusShipped, OrderStatusPaid},
		{OrderStatusCompleted, OrderStatusPaid},
		{OrderStatusCancelled, OrderStatusPaid},
		{OrderStatusCancelled, OrderStatusRefunded},
		{OrderStatusRefunded, OrderStatusPaid},
		{OrderStatusRefunded, OrderStatusCancelled},
		{"unknown", OrderStatusPaid},
		{OrderStatusPaid, "unknown"},
	}
	for _, c := range illegal {
		if CanTransitionOrder(c.from, c.to) {
			t.Fatalf("expected %s -> %s to be rejected", c.from, c.to)
		}
	}
}

func TestCouponDiscountRounding(t *testing.T) {
	// Percent discounts round half-up once: 10% of 999 = 99.9 -> 100.
	coupon := &modelsCouponForTest{Type: CouponTypePercent, PercentOff: 10}
	if got := couponDiscountFor(couponAsCoupon(coupon), 999); got != 100 {
		t.Fatalf("expected rounded discount 100, got %d", got)
	}

	// 25% of 555 = 138.75 -> 139.
	coupon.PercentOff = 25
	if got := couponDiscountFor(couponAsCoupon(coupon), 555); got != 139 {
		t.Fatalf("expected rounded discount 139, got %d", got)
	}

	// Fixed discounts clamp to the subtotal.
	fixed := &modelsCouponForTest{Type: CouponTypeFixed, ValueCents: 5000}
	if got := couponDiscountFor(couponAsCoupon(fixed), 400); got != 400 {
		t.Fatalf("expected fixed discount clamped to 400, got %d", got)
	}
}
