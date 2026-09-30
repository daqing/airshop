package products_api

import (
	"strings"
	"testing"

	"github.com/daqing/airshop/app/models"
	"github.com/daqing/airshop/app/services"
	productsview "github.com/daqing/airshop/app/views/products"
)

func renderDetail(t *testing.T, data productsview.DetailData) string {
	t.Helper()

	var buf strings.Builder
	if err := productsview.Detail(data).Render(t.Context(), &buf); err != nil {
		t.Fatalf("render failed: %v", err)
	}
	return buf.String()
}

func TestDetailRendersSimpleProduct(t *testing.T) {
	body := renderDetail(t, productsview.DetailData{
		Product: &models.Product{
			Name: "Photo Tee", Slug: "photo-tee", PriceCents: 1990,
			Stock: 5, Active: true, Description: "Soft cotton.",
		},
		CategoryName: "Apparel",
		Images: []services.ProductImageEntry{
			{URL: "/api/v1/storage/products/a.png", IsMain: true},
			{URL: "/api/v1/storage/products/b.png"},
		},
	})

	for _, want := range []string{
		"<title>Photo Tee</title>",
		"19.90",
		"In stock",
		"Soft cotton.",
		"Apparel",
		"Add to cart",
		"/api/v1/storage/products/a.png",
		"/api/v1/storage/products/b.png",
		"gallery-thumbs",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected detail HTML to contain %q, got:\n%s", want, body)
		}
	}
}

func TestDetailShowsVariantsWhenPresent(t *testing.T) {
	body := renderDetail(t, productsview.DetailData{
		Product: &models.Product{Name: "Tee", Slug: "tee", PriceCents: 1000, Stock: 5, Active: true},
		Variants: []productsview.VariantEntry{
			{Variant: &models.ProductVariant{ID: 1, Name: "Red / M", PriceCents: 1100, Active: true}, Price: "11.00", Sellable: true},
			{Variant: &models.ProductVariant{ID: 2, Name: "Blue / L", PriceCents: 1200, Active: true}, Price: "12.00", Sellable: false},
		},
	})

	for _, want := range []string{
		"Red / M",
		"11.00",
		"Blue / L",
		"Out of stock",
		`type="radio"`,
		"variants",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected variant detail HTML to contain %q, got:\n%s", want, body)
		}
	}

	if strings.Contains(body, `class="detail-price"`) {
		t.Fatal("variant-selling product must not show the flat price block")
	}
}

func TestDetailShowsOutOfStock(t *testing.T) {
	body := renderDetail(t, productsview.DetailData{
		Product: &models.Product{Name: "Gone", Slug: "gone", PriceCents: 500, Stock: 0, Active: true},
	})

	if !strings.Contains(body, "Out of stock") {
		t.Fatalf("expected out-of-stock label, got:\n%s", body)
	}
}
