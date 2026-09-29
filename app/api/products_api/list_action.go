package products_api

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/daqing/airshop/app/models"
	"github.com/daqing/airshop/app/services"
	productsview "github.com/daqing/airshop/app/views/products"
	"github.com/daqing/airway/lib/render"
)

func ListAction(c *gin.Context) {
	page := 1
	if raw := c.Query("page"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			page = parsed
		}
	}

	products, _, total, err := services.StorefrontListProducts(c.Query("category"), page, services.StorefrontPageSize)
	if err != nil {
		render.ErrorMessage(c, err.Error())
		return
	}

	cats, err := services.ListCategories()
	if err != nil {
		render.ErrorMessage(c, err.Error())
		return
	}

	enabled := make([]*models.Category, 0, len(cats))
	for _, cat := range cats {
		if cat.Enabled {
			enabled = append(enabled, cat)
		}
	}

	totalPages := int((total + services.StorefrontPageSize - 1) / services.StorefrontPageSize)
	if totalPages < 1 {
		totalPages = 1
	}

	render.HTML(c, productsview.Listing(productsview.ListingData{
		Products:     products,
		Categories:   enabled,
		CategorySlug: c.Query("category"),
		Page:         page,
		TotalPages:   totalPages,
		Total:        total,
	}))
}
