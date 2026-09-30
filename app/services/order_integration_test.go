package services

import (
	"os"
	"testing"

	"github.com/daqing/airway/lib/migrate"
	"github.com/daqing/airway/lib/repo"
	sql "github.com/daqing/airway/lib/sql"

	"github.com/daqing/airshop/app/models"
)

// TestPlaceOrder runs only when a scratch Postgres is provided:
//
//	AIRWAY_PG_TEST_DSN=postgres://...airshop_test go test ./app/services -run TestPlaceOrder
func TestPlaceOrder(t *testing.T) {
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

	user, err := repo.CreateFrom[models.User](sql.H{"phone_number": "13500001111", "status": "active"})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	t.Cleanup(func() {
		_ = repo.DeleteWhere[models.OrderItem](sql.H{})
		_ = repo.DeleteWhere[models.Order](sql.H{})
		_ = repo.DeleteWhere[models.CartItem](sql.H{})
		_ = repo.DeleteWhere[models.Cart](sql.H{})
		_ = repo.DeleteWhere[models.Address](sql.H{})
		_ = repo.DeleteWhere[models.Product](sql.H{"slug": sql.Like("slug", "order-test-%")})
		_ = repo.DeleteWhere[models.User](sql.H{"phone_number": "13500001111"})
	})

	address, err := CreateAddress(int64(user.ID), AddressInput{
		Recipient: "Order User", Phone: "13500001111",
		Province: "Shanghai", City: "Shanghai", Street: "No. 9 Order Rd", IsDefault: true,
	})
	if err != nil {
		t.Fatalf("seed address: %v", err)
	}

	product, err := repo.CreateFrom[models.Product](sql.H{
		"name": "order-test-widget", "slug": "order-test-widget", "price_cents": int64(2500), "stock": 4, "active": true,
	})
	if err != nil {
		t.Fatalf("seed product: %v", err)
	}

	// Empty cart cannot check out.
	if _, err := PlaceOrder(int64(user.ID), int64(address.ID), "fake", ""); err != ErrCartEmpty {
		t.Fatalf("expected empty-cart error, got %v", err)
	}

	if err := AddToCart(int64(user.ID), int64(product.ID), nil, 2); err != nil {
		t.Fatalf("add to cart: %v", err)
	}

	// A missing payment method is rejected.
	if _, err := PlaceOrder(int64(user.ID), int64(address.ID), " ", ""); err != ErrPaymentMethodNeeded {
		t.Fatalf("expected payment-method error, got %v", err)
	}

	// An unregistered gateway is rejected.
	if _, err := PlaceOrder(int64(user.ID), int64(address.ID), "bitcoin", ""); err != ErrPaymentMethodUnknown {
		t.Fatalf("expected unknown payment method, got %v", err)
	}

	// A foreign address is rejected.
	if _, err := PlaceOrder(int64(user.ID), 999999, "fake", ""); err != ErrAddressNotFound {
		t.Fatalf("expected address-not-found, got %v", err)
	}

	order, err := PlaceOrder(int64(user.ID), int64(address.ID), "fake", "")
	if err != nil {
		t.Fatalf("place order: %v", err)
	}

	if order.Status != "pending" {
		t.Fatalf("expected pending status, got %q", order.Status)
	}
	if order.SubtotalCents != 5000 || order.TotalCents != 5000 {
		t.Fatalf("expected amounts recomputed server-side at 5000, got %d/%d", order.SubtotalCents, order.TotalCents)
	}
	if !stringsHasPrefix(order.OrderNo, "SO") {
		t.Fatalf("expected an SO-prefixed order number, got %q", order.OrderNo)
	}
	if order.ShipRecipient != "Order User" || order.ShipAddress == "" {
		t.Fatalf("expected an address snapshot, got %q / %q", order.ShipRecipient, order.ShipAddress)
	}

	items, err := OrderItems(int64(order.ID))
	if err != nil {
		t.Fatalf("order items: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 order item, got %d", len(items))
	}
	if items[0].ProductName != "order-test-widget" || items[0].UnitPriceCents != 2500 || items[0].Quantity != 2 {
		t.Fatalf("expected a product snapshot, got %#v", items[0])
	}

	// The cart is cleared after placing the order.
	lines, _, err := CartLines(int64(user.ID))
	if err != nil {
		t.Fatalf("cart lines: %v", err)
	}
	if len(lines) != 0 {
		t.Fatalf("expected the cart to be cleared, got %d lines", len(lines))
	}

	// Ownership: another user cannot resolve the order number.
	if _, err := FindOrderByNo(int64(user.ID)+1, order.OrderNo); err != ErrOrderNotFound {
		t.Fatalf("expected cross-user order lookup to be not-found, got %v", err)
	}
	if _, err := FindOrderByNo(int64(user.ID), order.OrderNo); err != nil {
		t.Fatalf("owner lookup failed: %v", err)
	}
}

func stringsHasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
