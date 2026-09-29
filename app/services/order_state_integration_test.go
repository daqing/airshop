package services

import (
	"os"
	"testing"

	"github.com/daqing/airway/lib/migrate"
	"github.com/daqing/airway/lib/repo"
	sql "github.com/daqing/airway/lib/sql"

	"github.com/daqing/airshop/app/models"
)

// TestOrderTransitions runs only when a scratch Postgres is provided:
//
//	AIRWAY_PG_TEST_DSN=postgres://...airshop_test go test ./app/services -run TestOrderTransitions
func TestOrderTransitions(t *testing.T) {
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

	user, err := repo.CreateFrom[models.User](sql.H{"phone_number": "13400005555", "status": "active"})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	t.Cleanup(func() {
		_ = repo.DeleteWhere[models.Order](sql.H{"user_id": int64(user.ID)})
		_ = repo.DeleteWhere[models.User](sql.H{"phone_number": "13400005555"})
	})

	order, err := repo.CreateFrom[models.Order](sql.H{
		"order_no":       "SO-TEST-1",
		"user_id":        int64(user.ID),
		"ship_recipient": "T", "ship_phone": "13400005555", "ship_address": "x",
		"subtotal_cents": 100, "total_cents": 100, "status": OrderStatusPending,
	})
	if err != nil {
		t.Fatalf("seed order: %v", err)
	}

	// Happy path pending -> paid -> shipped -> completed -> refunded.
	for _, next := range []string{OrderStatusPaid, OrderStatusShipped, OrderStatusCompleted, OrderStatusRefunded} {
		if err := TransitionOrder(int64(order.ID), next); err != nil {
			t.Fatalf("transition to %s: %v", next, err)
		}
	}
	if check, err := FindOrder(int64(order.ID)); err != nil || check.Status != OrderStatusRefunded {
		t.Fatalf("expected refunded, got %#v / %v", check, err)
	}

	// Refunded is terminal.
	if err := TransitionOrder(int64(order.ID), OrderStatusPaid); err == nil {
		t.Fatal("expected the terminal refunded status to refuse transitions")
	}

	// Invalid jumps are rejected from a fresh pending order.
	order2, err := repo.CreateFrom[models.Order](sql.H{
		"order_no":       "SO-TEST-2",
		"user_id":        int64(user.ID),
		"ship_recipient": "T", "ship_phone": "13400005555", "ship_address": "x",
		"subtotal_cents": 100, "total_cents": 100, "status": OrderStatusPending,
	})
	if err != nil {
		t.Fatalf("seed order 2: %v", err)
	}
	if err := TransitionOrder(int64(order2.ID), OrderStatusShipped); err == nil {
		t.Fatal("expected pending -> shipped to be rejected")
	}
	if err := TransitionOrder(int64(order2.ID), "flying"); err == nil {
		t.Fatal("expected an unknown target status to be rejected")
	}

	// Cancelled is terminal too.
	if err := TransitionOrder(int64(order2.ID), OrderStatusCancelled); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if err := TransitionOrder(int64(order2.ID), OrderStatusPaid); err == nil {
		t.Fatal("expected the cancelled status to refuse transitions")
	}

	// Unknown orders are not-found.
	if err := TransitionOrder(999999, OrderStatusPaid); err != ErrOrderNotFound {
		t.Fatalf("expected not-found, got %v", err)
	}

	// The status column only ever holds known values.
	if knownOrderStatus("flying") {
		t.Fatal("expected unknown statuses to be unrecognized")
	}
}
