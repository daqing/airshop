package services

import (
	"os"
	"testing"

	"github.com/daqing/airway/lib/migrate"
	"github.com/daqing/airway/lib/repo"
	sql "github.com/daqing/airway/lib/sql"

	"github.com/daqing/airshop/app/models"
)

// TestMarkOrderPaid runs only when a scratch Postgres is provided:
//
//	AIRWAY_PG_TEST_DSN=postgres://...airshop_test go test ./app/services -run TestMarkOrderPaid
func TestMarkOrderPaid(t *testing.T) {
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

	user, err := repo.CreateFrom[models.User](sql.H{"phone_number": "13400007777", "status": "active"})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	product, err := repo.CreateFrom[models.Product](sql.H{
		"name": "markpaid-widget", "slug": "markpaid-widget", "price_cents": 100, "stock": 10, "active": true,
	})
	if err != nil {
		t.Fatalf("seed product: %v", err)
	}
	t.Cleanup(func() {
		_ = repo.DeleteWhere[models.OrderItem](sql.H{})
		_ = repo.DeleteWhere[models.Order](sql.H{"user_id": int64(user.ID)})
		_ = repo.DeleteWhere[models.Product](sql.H{"slug": "markpaid-widget"})
		_ = repo.DeleteWhere[models.User](sql.H{"phone_number": "13400007777"})
	})

	order, err := repo.CreateFrom[models.Order](sql.H{
		"order_no":       "SO-MARK-1",
		"user_id":        int64(user.ID),
		"ship_recipient": "T", "ship_phone": "13400007777", "ship_address": "x",
		"subtotal_cents": 300, "total_cents": 300, "status": OrderStatusPending,
	})
	if err != nil {
		t.Fatalf("seed order: %v", err)
	}
	if _, err := repo.CreateFrom[models.OrderItem](sql.H{
		"order_id":         int64(order.ID),
		"product_id":       int64(product.ID),
		"product_name":     "x",
		"unit_price_cents": 100,
		"quantity":         3,
	}); err != nil {
		t.Fatalf("seed order item: %v", err)
	}

	// First success notification pays the order and deducts stock once.
	if err := MarkOrderPaid(int64(order.ID)); err != nil {
		t.Fatalf("mark paid: %v", err)
	}
	assertStock(t, "after first notification", int64(product.ID), 7, false)

	// Duplicate notifications are idempotent: no error, no double deduction.
	if err := MarkOrderPaid(int64(order.ID)); err != nil {
		t.Fatalf("expected a duplicate notification to succeed quietly, got %v", err)
	}
	assertStock(t, "after duplicate notification", int64(product.ID), 7, false)

	// Unknown orders surface as not-found for gateway logging.
	if err := MarkOrderPaid(999999); err != ErrOrderNotFound {
		t.Fatalf("expected not-found, got %v", err)
	}

	// Refunded orders refuse payment.
	if err := TransitionOrder(int64(order.ID), OrderStatusRefunded); err != nil {
		t.Fatalf("refund: %v", err)
	}
	if err := MarkOrderPaid(int64(order.ID)); err == nil {
		t.Fatal("expected paying a refunded order to be rejected")
	}
}
