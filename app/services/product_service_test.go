package services

import (
	"testing"

	"github.com/daqing/airshop/app/models"
)

func TestSellsThroughVariants(t *testing.T) {
	if SellsThroughVariants(nil) {
		t.Fatal("expected no variants to mean product-level selling")
	}

	inactive := []*models.ProductVariant{{Active: false}}
	if SellsThroughVariants(inactive) {
		t.Fatal("expected only-inactive variants to mean product-level selling")
	}

	mixed := []*models.ProductVariant{{Active: false}, {Active: true}}
	if !SellsThroughVariants(mixed) {
		t.Fatal("expected any active variant to mean variant selling")
	}
}

func TestSellableChecks(t *testing.T) {
	if ProductSellable(&models.Product{Active: true, Stock: 0}) {
		t.Fatal("expected out-of-stock product to be unsellable")
	}
	if ProductSellable(&models.Product{Active: false, Stock: 5}) {
		t.Fatal("expected inactive product to be unsellable")
	}
	if !ProductSellable(&models.Product{Active: true, Stock: 5}) {
		t.Fatal("expected active in-stock product to be sellable")
	}

	if VariantSellable(&models.ProductVariant{Active: true, Stock: 0}) {
		t.Fatal("expected out-of-stock variant to be unsellable")
	}
	if VariantSellable(&models.ProductVariant{Active: false, Stock: 5}) {
		t.Fatal("expected inactive variant to be unsellable")
	}
	if !VariantSellable(&models.ProductVariant{Active: true, Stock: 5}) {
		t.Fatal("expected active in-stock variant to be sellable")
	}
}
