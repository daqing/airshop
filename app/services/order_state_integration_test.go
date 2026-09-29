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

// TestOrderStockFollowsPayment verifies the T5.4 strategy: stock is deducted
// when the order becomes paid and restored when it is refunded.
func TestOrderStockFollowsPayment(t *testing.T) {
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

	user, err := repo.CreateFrom[models.User](sql.H{"phone_number": "13400006666", "status": "active"})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	product, err := repo.CreateFrom[models.Product](sql.H{
		"name": "stock-test-widget", "slug": "stock-test-widget", "price_cents": 100, "stock": 5, "active": true,
	})
	if err != nil {
		t.Fatalf("seed product: %v", err)
	}
	variantProduct, err := repo.CreateFrom[models.Product](sql.H{
		"name": "stock-test-tee", "slug": "stock-test-tee", "price_cents": 100, "stock": 0, "active": true,
	})
	if err != nil {
		t.Fatalf("seed variant product: %v", err)
	}
	variant, err := repo.CreateFrom[models.ProductVariant](sql.H{
		"product_id": int64(variantProduct.ID), "name": "One Size", "price_cents": 120, "stock": 4, "active": true,
	})
	if err != nil {
		t.Fatalf("seed variant: %v", err)
	}
	t.Cleanup(func() {
		_ = repo.DeleteWhere[models.OrderItem](sql.H{})
		_ = repo.DeleteWhere[models.Order](sql.H{"user_id": int64(user.ID)})
		_ = repo.DeleteWhere[models.ProductVariant](sql.H{})
		_ = repo.DeleteWhere[models.Product](sql.H{"slug": sql.Like("slug", "stock-test-%")})
		_ = repo.DeleteWhere[models.User](sql.H{"phone_number": "13400006666"})
	})

	order, err := repo.CreateFrom[models.Order](sql.H{
		"order_no":       "SO-STOCK-1",
		"user_id":        int64(user.ID),
		"ship_recipient": "T", "ship_phone": "13400006666", "ship_address": "x",
		"subtotal_cents": 320, "total_cents": 320, "status": OrderStatusPending,
	})
	if err != nil {
		t.Fatalf("seed order: %v", err)
	}
	variantID := int64(variant.ID)
	for _, item := range []struct {
		productID int64
		variantID *int64
		quantity  int
	}{
		{int64(product.ID), nil, 2},
		{int64(variantProduct.ID), &variantID, 1},
	} {
		if _, err := repo.CreateFrom[models.OrderItem](sql.H{
			"order_id":         int64(order.ID),
			"product_id":       item.productID,
			"variant_id":       item.variantID,
			"product_name":     "x",
			"unit_price_cents": 100,
			"quantity":         item.quantity,
		}); err != nil {
			t.Fatalf("seed order item: %v", err)
		}
	}

	// Pending: stock untouched.
	assertStock(t, "pending/product", int64(product.ID), 5, false)
	assertStock(t, "pending/variant", int64(variant.ID), 4, true)

	// Paying deducts both.
	if err := TransitionOrder(int64(order.ID), OrderStatusPaid); err != nil {
		t.Fatalf("pay: %v", err)
	}
	assertStock(t, "paid/product", int64(product.ID), 3, false)
	assertStock(t, "paid/variant", int64(variant.ID), 3, true)

	// Cancelling is unreachable from paid, so refunds restore instead.
	if err := TransitionOrder(int64(order.ID), OrderStatusRefunded); err != nil {
		t.Fatalf("refund: %v", err)
	}
	assertStock(t, "refunded/product", int64(product.ID), 5, false)
	assertStock(t, "refunded/variant", int64(variant.ID), 4, true)

	// A cancelled pending order never touched stock.
	order2, err := repo.CreateFrom[models.Order](sql.H{
		"order_no":       "SO-STOCK-2",
		"user_id":        int64(user.ID),
		"ship_recipient": "T", "ship_phone": "13400006666", "ship_address": "x",
		"subtotal_cents": 100, "total_cents": 100, "status": OrderStatusPending,
	})
	if err != nil {
		t.Fatalf("seed order 2: %v", err)
	}
	if _, err := repo.CreateFrom[models.OrderItem](sql.H{
		"order_id":         int64(order2.ID),
		"product_id":       int64(product.ID),
		"product_name":     "x",
		"unit_price_cents": 100,
		"quantity":         2,
	}); err != nil {
		t.Fatalf("seed order item 2: %v", err)
	}
	if err := TransitionOrder(int64(order2.ID), OrderStatusCancelled); err != nil {
		t.Fatalf("cancel pending: %v", err)
	}
	assertStock(t, "cancelled/product", int64(product.ID), 5, false)
}

func assertStock(t *testing.T, label string, productOrVariantID int64, want int, wantVariant bool) {
	t.Helper()

	if wantVariant {
		v, err := repo.FindByID[models.ProductVariant](sql.IdType(productOrVariantID))
		if err != nil || v == nil {
			t.Fatalf("%s: variant lookup failed: %v", label, err)
		}
		if v.Stock != want {
			t.Fatalf("%s: expected variant stock %d, got %d", label, want, v.Stock)
		}
		return
	}

	p, err := repo.FindByID[models.Product](sql.IdType(productOrVariantID))
	if err != nil || p == nil {
		t.Fatalf("%s: product lookup failed: %v", label, err)
	}
	if p.Stock != want {
		t.Fatalf("%s: expected product stock %d, got %d", label, want, p.Stock)
	}
}
