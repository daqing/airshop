package services

import (
	"github.com/daqing/airway/lib/repo"
	sql "github.com/daqing/airway/lib/sql"

	"github.com/daqing/airshop/app/models"
)

// A product sells through variants when it has at least one active variant
// (see docs/CONVENTIONS.md "Product variants"); otherwise the product row
// itself is the sellable unit.

func ActiveVariants(productID int64) ([]*models.ProductVariant, error) {
	return repo.FindBy[models.ProductVariant](sql.H{"product_id": productID, "active": true})
}

func SellsThroughVariants(variants []*models.ProductVariant) bool {
	for _, v := range variants {
		if v.Active {
			return true
		}
	}
	return false
}

func ProductSellable(product *models.Product) bool {
	return product.Active && product.Stock > 0
}

func VariantSellable(variant *models.ProductVariant) bool {
	return variant.Active && variant.Stock > 0
}
