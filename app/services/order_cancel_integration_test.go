package services

import (
	"os"
	"testing"
	"time"

	"github.com/daqing/airway/lib/migrate"
	"github.com/daqing/airway/lib/repo"
	sql "github.com/daqing/airway/lib/sql"

	"github.com/daqing/airshop/app/models"
)

// TestOrderCancelAndExpiry runs only when a scratch Postgres is provided:
//
//	AIRWAY_PG_TEST_DSN=postgres://...airshop_test go test ./app/services -run TestOrderCancelAndExpiry
func TestOrderCancelAndExpiry(t *testing.T) {
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

	userA, err := repo.CreateFrom[models.User](sql.H{"phone_number": "13300007001", "status": "active"})
	if err != nil {
		t.Fatalf("seed user A: %v", err)
	}
	userB, err := repo.CreateFrom[models.User](sql.H{"phone_number": "13300007002", "status": "active"})
	if err != nil {
		t.Fatalf("seed user B: %v", err)
	}
	t.Cleanup(func() {
		_ = repo.DeleteWhere[models.Order](sql.H{})
		_ = repo.DeleteWhere[models.User](sql.H{"phone_number": sql.Like("phone_number", "133000070%")})
	})

	newPendingOrder := func(orderNo string) *models.Order {
		t.Helper()
		order, err := repo.CreateFrom[models.Order](sql.H{
			"order_no": orderNo, "user_id": int64(userA.ID),
			"ship_recipient": "T", "ship_phone": "13300007001", "ship_address": "x",
			"subtotal_cents": 100, "total_cents": 100, "status": OrderStatusPending,
		})
		if err != nil {
			t.Fatalf("seed %s: %v", orderNo, err)
		}
		return order
	}

	// Manual cancel: owner succeeds, strangers read not-found.
	order := newPendingOrder("SO-CXL-1")
	if err := CancelOrder(int64(userB.ID), order.OrderNo); err != ErrOrderNotFound {
		t.Fatalf("expected cross-user cancel to be not-found, got %v", err)
	}
	if err := CancelOrder(int64(userA.ID), order.OrderNo); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if check, _ := FindOrder(int64(order.ID)); check.Status != OrderStatusCancelled {
		t.Fatalf("expected cancelled, got %q", check.Status)
	}
	// Cancelling twice fails through the state machine.
	if err := CancelOrder(int64(userA.ID), order.OrderNo); err == nil {
		t.Fatal("expected a second cancel to be rejected")
	}

	// Non-pending orders cannot be cancelled.
	paid := newPendingOrder("SO-CXL-2")
	if err := TransitionOrder(int64(paid.ID), OrderStatusPaid); err != nil {
		t.Fatalf("mark paid: %v", err)
	}
	if err := CancelOrder(int64(userA.ID), paid.OrderNo); err == nil {
		t.Fatal("expected cancelling a paid order to be rejected")
	}

	// The expiry sweeper cancels only old pending orders, leaving fresh ones
	// and other statuses alone.
	old := newPendingOrder("SO-CXL-3")
	newPendingOrder("SO-CXL-4")
	paidOld := newPendingOrder("SO-CXL-5")
	if err := TransitionOrder(int64(paidOld.ID), OrderStatusPaid); err != nil {
		t.Fatalf("mark paid old: %v", err)
	}

	// Backdate through Go so the stored wall clock is local, matching the
	// database defaults that CURRENT_TIMESTAMP writes.
	backdate := time.Now().Add(-UnpaidOrderExpiry - time.Minute)
	if err := repo.UpdateByID[models.Order](old.ID, sql.H{"created_at": backdate}); err != nil {
		t.Fatalf("backdate old: %v", err)
	}
	if err := repo.UpdateByID[models.Order](paidOld.ID, sql.H{"created_at": backdate}); err != nil {
		t.Fatalf("backdate paid old: %v", err)
	}

	cancelled, err := CancelExpiredOrders(UnpaidOrderExpiry)
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if cancelled != 1 {
		t.Fatalf("expected the sweeper to cancel 1 order, got %d", cancelled)
	}
	assertOrderStatus(t, "SO-CXL-3", OrderStatusCancelled)
	assertOrderStatus(t, "SO-CXL-4", OrderStatusPending)
	assertOrderStatus(t, "SO-CXL-5", OrderStatusPaid)
}

func assertOrderStatus(t *testing.T, orderNo, want string) {
	t.Helper()

	order, err := repo.FindOneBy[models.Order](sql.H{"order_no": orderNo})
	if err != nil || order == nil {
		t.Fatalf("find %s: %v", orderNo, err)
	}
	if order.Status != want {
		t.Fatalf("expected %s to be %s, got %s", orderNo, want, order.Status)
	}
}
