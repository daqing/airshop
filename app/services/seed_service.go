package services

import (
	"log"

	"github.com/daqing/airway/lib/repo"
	sql "github.com/daqing/airway/lib/sql"

	"github.com/daqing/airshop/app/models"
)

// SeedDemoDataIfEmpty fills an empty catalog with demo categories, products
// and a welcome coupon so a fresh local install has something to browse.
// Idempotent: it only runs while the products table has no rows, and the
// caller gates it to local environments.
func SeedDemoDataIfEmpty() (bool, error) {
	count, err := repo.CountEvery[models.Product]()
	if err != nil {
		return false, err
	}
	if count > 0 {
		return false, nil
	}

	apparel, err := repo.CreateFrom[models.Category](sql.H{
		"name": "Apparel", "slug": "apparel", "sort_order": 1, "enabled": true,
	})
	if err != nil {
		return false, err
	}
	home, err := repo.CreateFrom[models.Category](sql.H{
		"name": "Home & Living", "slug": "home-living", "sort_order": 2, "enabled": true,
	})
	if err != nil {
		return false, err
	}

	demo := []struct {
		category int64
		name     string
		slug     string
		desc     string
		cents    int64
		stock    int
	}{
		{int64(apparel.ID), "Classic Tee", "classic-tee", "Soft cotton tee in a relaxed fit.", 1990, 20},
		{int64(apparel.ID), "Heavy Hoodie", "heavy-hoodie", "Brushed fleece hoodie for cold evenings.", 4900, 12},
		{int64(apparel.ID), "Canvas Tote", "canvas-tote", "Sturdy everyday tote bag.", 1490, 30},
		{int64(home.ID), "Ceramic Mug", "ceramic-mug", "Hand-glazed 350 ml mug.", 2590, 15},
		{int64(home.ID), "Soy Candle", "soy-candle", "Slow-burning soy candle, cedar scent.", 1890, 25},
	}
	for _, item := range demo {
		if _, err := repo.CreateFrom[models.Product](sql.H{
			"category_id": item.category,
			"name":        item.name,
			"slug":        item.slug,
			"description": item.desc,
			"price_cents": item.cents,
			"stock":       item.stock,
			"active":      true,
		}); err != nil {
			return false, err
		}
	}

	if _, err := repo.CreateFrom[models.Coupon](sql.H{
		"code":        "WELCOME10",
		"type":        CouponTypePercent,
		"percent_off": 10,
		"enabled":     true,
	}); err != nil {
		return false, err
	}

	log.Printf("seeded demo catalog: %d products in %d categories + WELCOME10 coupon", len(demo), 2)
	return true, nil
}
