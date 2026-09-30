package services

import (
	"os"
	"testing"

	"github.com/daqing/airway/lib/migrate"
	"github.com/daqing/airway/lib/repo"
	sql "github.com/daqing/airway/lib/sql"

	"github.com/daqing/airshop/app/models"
)

// TestLatestProducts runs only when a scratch Postgres is provided, following
// the airway integration-test convention:
//
//	AIRWAY_PG_TEST_DSN=postgres://...airshop_test go test ./app/services -run TestLatestProducts
func TestLatestProducts(t *testing.T) {
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

	seed := []models.Product{
		{Name: "lp-test-old-active", Slug: "lp-test-old-active", PriceCents: 100, Active: true},
		{Name: "lp-test-new-active", Slug: "lp-test-new-active", PriceCents: 200, Active: true},
		{Name: "lp-test-inactive", Slug: "lp-test-inactive", PriceCents: 300, Active: false},
	}
	for _, p := range seed {
		if _, err := repo.CreateFrom[models.Product](sql.H{
			"name": p.Name, "slug": p.Slug, "price_cents": p.PriceCents, "active": p.Active,
		}); err != nil {
			t.Fatalf("seed %s: %v", p.Slug, err)
		}
	}
	t.Cleanup(func() {
		_ = repo.DeleteWhere[models.Product](sql.H{"slug": sql.Like("slug", "lp-test-%")})
	})

	products, err := LatestProducts(1)
	if err != nil {
		t.Fatalf("LatestProducts: %v", err)
	}
	if len(products) != 1 {
		t.Fatalf("expected 1 product with limit 1, got %d", len(products))
	}
	if products[0].Slug != "lp-test-new-active" {
		t.Fatalf("expected newest active product, got %q", products[0].Slug)
	}

	all, err := LatestProducts(10)
	if err != nil {
		t.Fatalf("LatestProducts(10): %v", err)
	}
	if len(all) < 2 {
		t.Fatalf("expected both active test products, got %d", len(all))
	}
	for _, p := range all {
		if p.Slug == "lp-test-inactive" {
			t.Fatal("inactive product must not appear on the homepage")
		}
	}
}
