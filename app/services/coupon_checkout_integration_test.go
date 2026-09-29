package services

import (
	"os"
	"testing"

	"github.com/daqing/airway/lib/migrate"
	"github.com/daqing/airway/lib/repo"
	sql "github.com/daqing/airway/lib/sql"

	"github.com/daqing/airshop/app/models"
)

// TestCouponCheckout runs only when a scratch Postgres is provided:
//
//	AIRWAY_PG_TEST_DSN=postgres://...airshop_test go test ./app/services -run TestCouponCheckout
func TestCouponCheckout(t *testing.T) {
	dsn := os.Getenv("AIRWAY_PG_TEST_DSN")
	if dsn == "" {
		t.Skip("set AIRWAY_PG_TEST_DSN to run this integration test")
	}

	if err := migrate.Run(migrate.Options{
		DSN:          dsn,
		Migrations:   os.DirFS("../../db/migrate"),
		SnapshotPath: "",
	}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := repo.SetupDB(dsn); err != nil {
		t.Fatalf("setup db: %v", err)
	}

	user, err := repo.CreateFrom[models.User](sql.H{"phone_number": "13600009900", "status": "active"})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	address, err := CreateAddress(int64(user.ID), AddressInput{
		Recipient: "Coupon User", Phone: "13600009900", City: "Beijing", Street: "Coupon Rd 1", IsDefault: true,
	})
	if err != nil {
		t.Fatalf("seed address: %v", err)
	}
	product, err := repo.CreateFrom[models.Product](sql.H{
		"name": "coupon-test-item", "slug": "coupon-test-item", "price_cents": int64(4000), "stock": 9, "active": true,
	})
	if err != nil {
		t.Fatalf("seed product: %v", err)
	}
	t.Cleanup(func() {
		_ = repo.DeleteWhere[models.CouponRedemption](sql.H{})
		_ = repo.DeleteWhere[models.OrderItem](sql.H{})
		_ = repo.DeleteWhere[models.Order](sql.H{})
		_ = repo.DeleteWhere[models.CartItem](sql.H{})
		_ = repo.DeleteWhere[models.Cart](sql.H{})
		_ = repo.DeleteWhere[models.Address](sql.H{})
		_ = repo.DeleteWhere[models.Product](sql.H{"slug": "coupon-test-item"})
		_ = repo.DeleteWhere[models.Coupon](sql.H{"code": sql.Like("code", "CPN-TEST%")})
		_ = repo.DeleteWhere[models.User](sql.H{"phone_number": "13600009900"})
	})

	fixed, err := AdminCreateCoupon(CouponInput{
		Code: "CPN-TEST-FIX10", Type: CouponTypeFixed, ValueCents: 1000, ThresholdCents: 5000, Enabled: true,
	})
	if err != nil {
		t.Fatalf("seed fixed coupon: %v", err)
	}
	once, err := AdminCreateCoupon(CouponInput{
		Code: "CPN-TEST-ONCE", Type: CouponTypePercent, PercentOff: 10, Enabled: true,
	})
	if err != nil {
		t.Fatalf("seed once coupon: %v", err)
	}

	// Unknown code -> not found; known code under threshold -> not applicable.
	if _, _, err := ApplicableCoupon(int64(user.ID), "CPN-NOPE", 6000); err != ErrCouponNotFound {
		t.Fatalf("expected not-found, got %v", err)
	}
	if _, _, err := ApplicableCoupon(int64(user.ID), fixed.Code, 1000); err != ErrCouponNotApplicable {
		t.Fatalf("expected below-threshold to be not applicable, got %v", err)
	}

	// Fill the cart above the threshold and apply the fixed coupon.
	if err := AddToCart(int64(user.ID), int64(product.ID), nil, 2); err != nil {
		t.Fatalf("add to cart: %v", err)
	}

	order, err := PlaceOrder(int64(user.ID), int64(address.ID), "fake", fixed.Code)
	if err != nil {
		t.Fatalf("place order with coupon: %v", err)
	}
	if order.DiscountCents != 1000 {
		t.Fatalf("expected 1000 discount, got %d", order.DiscountCents)
	}
	if order.TotalCents != 7000 {
		t.Fatalf("expected total 7000 (8000-1000), got %d", order.TotalCents)
	}

	redemptions, err := repo.FindBy[models.CouponRedemption](sql.H{"coupon_id": int64(fixed.ID)})
	if err != nil {
		t.Fatalf("redemptions: %v", err)
	}
	if len(redemptions) != 1 || redemptions[0].DiscountCents != 1000 || redemptions[0].OrderID != int64(order.ID) {
		t.Fatalf("expected one redemption snapshot, got %#v", redemptions)
	}

	// The same user cannot reuse the fixed coupon on a second order; placing
	// the order without the coupon code works normally.
	if err := AddToCart(int64(user.ID), int64(product.ID), nil, 2); err != nil {
		t.Fatalf("add to cart again: %v", err)
	}
	if _, _, err := ApplicableCoupon(int64(user.ID), fixed.Code, 8000); err != ErrCouponNotApplicable {
		t.Fatalf("expected reuse to be not applicable, got %v", err)
	}
	orderPlain, err := PlaceOrder(int64(user.ID), int64(address.ID), "fake", "")
	if err != nil {
		t.Fatalf("second order without coupon: %v", err)
	}
	if orderPlain.DiscountCents != 0 {
		t.Fatalf("expected no discount without a coupon, got %d", orderPlain.DiscountCents)
	}

	// The percent coupon applies once per user too; total_count=1 exhausts it
	// after the first redemption, and the order still total-checks.
	if err := AddToCart(int64(user.ID), int64(product.ID), nil, 2); err != nil {
		t.Fatalf("add to cart for percent order: %v", err)
	}
	if _, _, err := ApplicableCoupon(int64(user.ID), once.Code, 8000); err != nil {
		t.Fatalf("expected once coupon applicable, got %v", err)
	}
	order2, err := PlaceOrder(int64(user.ID), int64(address.ID), "fake", once.Code)
	if err != nil {
		t.Fatalf("place order with percent coupon: %v", err)
	}
	if order2.DiscountCents != 800 {
		t.Fatalf("expected 10%% of 8000 = 800 discount, got %d", order2.DiscountCents)
	}
	if _, _, err := ApplicableCoupon(int64(user.ID), once.Code, 8000); err != ErrCouponNotApplicable {
		t.Fatalf("expected once coupon exhausted for this user, got %v", err)
	}
	assertOrderStatus(t, order2.OrderNo, OrderStatusPending)
}
