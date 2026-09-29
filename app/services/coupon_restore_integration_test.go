package services

import (
	"os"
	"testing"

	_ "github.com/daqing/airshop/db/migrate"
	"github.com/daqing/airway/lib/migrate"
	"github.com/daqing/airway/lib/repo"
	sql "github.com/daqing/airway/lib/sql"

	"github.com/daqing/airshop/app/models"
)

// TestCouponRestoreOnCancelAndRefund runs only when a scratch Postgres is
// provided:
//
//	AIRWAY_PG_TEST_DSN=postgres://...airshop_test go test ./app/services -run TestCouponRestoreOnCancelAndRefund
func TestCouponRestoreOnCancelAndRefund(t *testing.T) {
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

	user, err := repo.CreateFrom[models.User](sql.H{"phone_number": "13600008811", "status": "active"})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	address, err := CreateAddress(int64(user.ID), AddressInput{
		Recipient: "R", Phone: "13600008811", City: "Beijing", Street: "Restore Rd 1", IsDefault: true,
	})
	if err != nil {
		t.Fatalf("seed address: %v", err)
	}
	product, err := repo.CreateFrom[models.Product](sql.H{
		"name": "restore-test-item", "slug": "restore-test-item", "price_cents": int64(5000), "stock": 9, "active": true,
	})
	if err != nil {
		t.Fatalf("seed product: %v", err)
	}
	coupon, err := AdminCreateCoupon(CouponInput{
		Code: "CPN-RESTORE", Type: CouponTypeFixed, ValueCents: 500, ThresholdCents: 4000, Enabled: true,
	})
	if err != nil {
		t.Fatalf("seed coupon: %v", err)
	}
	t.Cleanup(func() {
		_ = repo.DeleteWhere[models.CouponRedemption](sql.H{})
		_ = repo.DeleteWhere[models.OrderItem](sql.H{})
		_ = repo.DeleteWhere[models.Order](sql.H{})
		_ = repo.DeleteWhere[models.Address](sql.H{})
		_ = repo.DeleteWhere[models.Product](sql.H{"slug": "restore-test-item"})
		_ = repo.DeleteWhere[models.Coupon](sql.H{"code": "CPN-RESTORE"})
		_ = repo.DeleteWhere[models.User](sql.H{"phone_number": "13600008811"})
	})

	placeWithCoupon := func(orderNo string) *models.Order {
		t.Helper()
		if err := AddToCart(int64(user.ID), int64(product.ID), nil, 1); err != nil {
			t.Fatalf("add to cart: %v", err)
		}
		// Claim first (once per user), then place the order. Refunds and
		// cancellations only remove redemptions; the claim stays, so the
		// coupon keeps working across this user's orders.
		if err := ClaimCoupon(int64(user.ID), int64(coupon.ID)); err != nil {
			t.Fatalf("claim coupon: %v", err)
		}
		order, err := PlaceOrder(int64(user.ID), int64(address.ID), "fake", coupon.Code)
		if err != nil {
			t.Fatalf("place %s: %v", orderNo, err)
		}
		return order
	}

	// Cancelled pending order returns the coupon.
	cancelled := placeWithCoupon("SO-RESTORE-1")
	if err := TransitionOrder(int64(cancelled.ID), OrderStatusCancelled); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if _, _, err := ApplicableCoupon(int64(user.ID), coupon.Code, 5000); err != nil {
		t.Fatalf("expected the coupon restored after cancellation, got %v", err)
	}

	// Refunded paid order returns the coupon too (stock is restored as well).
	paid := placeWithCoupon("SO-RESTORE-2")
	if err := TransitionOrder(int64(paid.ID), OrderStatusPaid); err != nil {
		t.Fatalf("pay: %v", err)
	}
	if _, _, err := ApplicableCoupon(int64(user.ID), coupon.Code, 5000); err != ErrCouponNotApplicable {
		t.Fatalf("expected the coupon consumed while paid, got %v", err)
	}
	if err := TransitionOrder(int64(paid.ID), OrderStatusRefunded); err != nil {
		t.Fatalf("refund: %v", err)
	}
	if _, _, err := ApplicableCoupon(int64(user.ID), coupon.Code, 5000); err != nil {
		t.Fatalf("expected the coupon restored after refund, got %v", err)
	}
	assertStock(t, "refund restores stock", int64(product.ID), 9, false)

	// The redemption rows for both orders are gone.
	remaining, err := repo.CountWhere[models.CouponRedemption](sql.H{"coupon_id": int64(coupon.ID)})
	if err != nil {
		t.Fatalf("count redemptions: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("expected all redemptions removed, got %d", remaining)
	}
}
