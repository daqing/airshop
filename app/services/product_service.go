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

// LatestProducts returns up to limit active products, newest first, for the
// storefront homepage section.
func LatestProducts(limit int) ([]*models.Product, error) {
	b := sql.Select("*").
		From("products").
		Where(sql.Eq("active", true)).
		OrderBy("id DESC").
		Limit(limit)
	return repo.Find[models.Product](repo.CurrentDB(), b)
}

const StorefrontPageSize = 12

// StorefrontListProducts returns one page of active products for the public
// catalog, newest first, optionally scoped to a category slug. The resolved
// category is nil when no filter is set or the slug is unknown.
func StorefrontListProducts(categorySlug string, page, size int) ([]*models.Product, *models.Category, int64, error) {
	var category *models.Category
	if categorySlug != "" {
		cat, err := repo.FindOneBy[models.Category](sql.H{"slug": categorySlug, "enabled": true})
		if err != nil {
			return nil, nil, 0, err
		}
		category = cat
	}

	conds := []sql.CondBuilder{sql.Eq("active", true)}
	if category != nil {
		conds = append(conds, sql.Eq("category_id", int64(category.ID)))
	}
	cond := sql.AllOf(conds...)

	b := sql.Select("*").
		From("products").
		Where(cond).
		OrderBy("id DESC").
		Limit(size).
		Offset((page - 1) * size)
	products, err := repo.Find[models.Product](repo.CurrentDB(), b)
	if err != nil {
		return nil, nil, 0, err
	}

	cb := sql.SelectColumns("count(*)").From("products").Where(cond)
	total, err := repo.Count(repo.CurrentDB(), cb)
	if err != nil {
		return nil, nil, 0, err
	}

	return products, category, total, nil
}
