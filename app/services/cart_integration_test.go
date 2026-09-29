package services

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/daqing/airway/lib/migrate"
	"github.com/daqing/airway/lib/repo"
	sql "github.com/daqing/airway/lib/sql"

	"github.com/daqing/airshop/app/models"
)

// TestCartService runs only when a scratch Postgres is provided:
//
//	AIRWAY_PG_TEST_DSN=postgres://...airshop_test go test ./app/services -run TestCartService
func TestCartService(t *testing.T) {
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

	userA, err := repo.CreateFrom[models.User](sql.H{"phone_number": "13600000001", "status": "active"})
	if err != nil {
		t.Fatalf("seed user A: %v", err)
	}
	userB, err := repo.CreateFrom[models.User](sql.H{"phone_number": "13600000002", "status": "active"})
	if err != nil {
		t.Fatalf("seed user B: %v", err)
	}
	t.Cleanup(func() {
		_ = repo.DeleteWhere[models.CartItem](sql.H{})
		_ = repo.DeleteWhere[models.Cart](sql.H{})
		_ = repo.DeleteWhere[models.ProductVariant](sql.H{})
		_ = repo.DeleteWhere[models.Product](sql.H{"slug": sql.Like("slug", "cart-test-%")})
		_ = repo.DeleteWhere[models.User](sql.H{"phone_number": sql.Like("phone_number", "136000000%")})
	})

	product, err := repo.CreateFrom[models.Product](sql.H{
		"name": "cart-test-widget", "slug": "cart-test-widget", "price_cents": int64(1500), "stock": 3, "active": true,
	})
	if err != nil {
		t.Fatalf("seed product: %v", err)
	}

	// Add to cart and merge on repeat adds.
	if err := AddToCart(int64(userA.ID), int64(product.ID), nil, 2); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := AddToCart(int64(userA.ID), int64(product.ID), nil, 1); err != nil {
		t.Fatalf("add again: %v", err)
	}

	lines, subtotal, err := CartLines(int64(userA.ID))
	if err != nil {
		t.Fatalf("lines: %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("expected 1 merged line, got %d", len(lines))
	}
	if lines[0].Item.Quantity != 3 {
		t.Fatalf("expected merged quantity 3, got %d", lines[0].Item.Quantity)
	}
	if subtotal != 4500 {
		t.Fatalf("expected subtotal 4500, got %d", subtotal)
	}

	// Stock caps quantity, both on add and on update.
	if err := AddToCart(int64(userA.ID), int64(product.ID), nil, 1); err == nil {
		t.Fatal("expected a stock cap error when adding beyond stock")
	}
	if err := UpdateCartItem(int64(userA.ID), int64(lines[0].Item.ID), 2); err != nil {
		t.Fatalf("update quantity: %v", err)
	}
	if err := UpdateCartItem(int64(userA.ID), int64(lines[0].Item.ID), 10); err == nil {
		t.Fatal("expected an update beyond stock to fail")
	}

	// Inactive products cannot be added.
	if err := repo.UpdateByID[models.Product](product.ID, sql.H{"active": false, "updated_at": time.Now().UTC()}); err != nil {
		t.Fatalf("deactivate product: %v", err)
	}
	if err := AddToCart(int64(userA.ID), int64(product.ID), nil, 1); err != ErrProductUnavailable {
		t.Fatalf("expected unavailable product error, got %v", err)
	}
	if err := repo.UpdateByID[models.Product](product.ID, sql.H{"active": true, "updated_at": time.Now().UTC()}); err != nil {
		t.Fatalf("reactivate product: %v", err)
	}

	// Variant products require a variant and snapshot the variant price.
	variantProduct, err := repo.CreateFrom[models.Product](sql.H{
		"name": "cart-test-tee", "slug": "cart-test-tee", "price_cents": int64(1000), "stock": 0, "active": true,
	})
	if err != nil {
		t.Fatalf("seed variant product: %v", err)
	}
	variant, err := repo.CreateFrom[models.ProductVariant](sql.H{
		"product_id": int64(variantProduct.ID), "name": "Red / M", "price_cents": int64(1100), "stock": 2, "active": true,
	})
	if err != nil {
		t.Fatalf("seed variant: %v", err)
	}

	if err := AddToCart(int64(userA.ID), int64(variantProduct.ID), nil, 1); err != ErrVariantRequired {
		t.Fatalf("expected variant-required error, got %v", err)
	}
	variantID := int64(variant.ID)
	if err := AddToCart(int64(userA.ID), int64(variantProduct.ID), &variantID, 1); err != nil {
		t.Fatalf("add variant line: %v", err)
	}

	lines, subtotal, err = CartLines(int64(userA.ID))
	if err != nil {
		t.Fatalf("lines: %v", err)
	}
	if len(lines) != 2 || subtotal != 4100 {
		t.Fatalf("expected 2 lines and subtotal 4100, got %d lines and %d", len(lines), subtotal)
	}
	if !strings.Contains(lines[1].Variant.Name, "Red") {
		t.Fatalf("expected variant display data, got %#v", lines[1].Variant)
	}

	// Ownership: user B sees an empty cart and cannot touch user A's items.
	bLines, _, err := CartLines(int64(userB.ID))
	if err != nil {
		t.Fatalf("user B lines: %v", err)
	}
	if len(bLines) != 0 {
		t.Fatalf("expected user B to have no lines, got %d", len(bLines))
	}
	if err := UpdateCartItem(int64(userB.ID), int64(lines[0].Item.ID), 1); err != ErrCartItemNotFound {
		t.Fatalf("expected cross-user update to be not-found, got %v", err)
	}
	if err := RemoveCartItem(int64(userB.ID), int64(lines[0].Item.ID)); err != ErrCartItemNotFound {
		t.Fatalf("expected cross-user remove to be not-found, got %v", err)
	}

	// Remove one line, then clear the rest.
	if err := RemoveCartItem(int64(userA.ID), int64(lines[0].Item.ID)); err != nil {
		t.Fatalf("remove line: %v", err)
	}
	if err := ClearCart(int64(userA.ID)); err != nil {
		t.Fatalf("clear cart: %v", err)
	}
	lines, _, err = CartLines(int64(userA.ID))
	if err != nil {
		t.Fatalf("lines after clear: %v", err)
	}
	if len(lines) != 0 {
		t.Fatalf("expected an empty cart after clearing, got %d lines", len(lines))
	}
}
