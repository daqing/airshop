package home_api

import (
	"strings"
	"testing"

	"github.com/daqing/airshop/app/models"
	"github.com/daqing/airshop/app/views/home"
)

func TestIndexViewRendersEmptyStorefront(t *testing.T) {
	var buf strings.Builder
	if err := home.Index(nil).Render(t.Context(), &buf); err != nil {
		t.Fatalf("render failed: %v", err)
	}
	body := buf.String()

	for _, want := range []string{
		"<title>AirShop</title>",
		"Latest products",
		"No products yet",
		"aw-storefront-header",
		"aw-storefront-footer",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected home HTML to contain %q, got:\n%s", want, body)
		}
	}
}

func TestIndexViewShowsProductsAndHidesNothing(t *testing.T) {
	var buf strings.Builder
	if err := home.Index([]*models.Product{
		{ID: 1, Name: "Mug", Slug: "mug", PriceCents: 4500, Active: true},
		{ID: 2, Name: "Tee", Slug: "tee", PriceCents: 1990, Active: true, MainImage: "products/202609/abc.png"},
	}).Render(t.Context(), &buf); err != nil {
		t.Fatalf("render failed: %v", err)
	}
	body := buf.String()

	for _, want := range []string{
		">Mug<",
		"45.00",
		">Tee<",
		"19.90",
		"/products/mug",
		"/products/tee",
		"/api/v1/storage/products/202609/abc.png",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected home HTML to contain %q, got:\n%s", want, body)
		}
	}
}
