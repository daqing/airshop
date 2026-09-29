package products_api

import (
	"github.com/gin-gonic/gin"

	"github.com/daqing/airway/lib/render"
	sql "github.com/daqing/airway/lib/sql"

	"github.com/daqing/airshop/app/services"
	"github.com/daqing/airshop/app/views/errors"
	"github.com/daqing/airshop/app/views/products"
)

func ShowAction(c *gin.Context) {
	slug := c.Param("slug")

	p, err := services.FindActiveProductBySlug(slug)
	if err != nil {
		render.ErrorMessage(c, err.Error())
		return
	}
	if p == nil {
		render.HTMLStatus(c, 404, errors.NotFound())
		return
	}

	categoryName := ""
	if p.CategoryID != nil {
		if cat, err := services.FindCategory(sql.IdType(*p.CategoryID)); err == nil && cat != nil {
			categoryName = cat.Name
		}
	}

	images, err := services.ProductImageEntries(int64(p.ID))
	if err != nil {
		render.ErrorMessage(c, err.Error())
		return
	}

	variants, err := services.ActiveVariants(int64(p.ID))
	if err != nil {
		render.ErrorMessage(c, err.Error())
		return
	}
	entries := make([]products.VariantEntry, 0, len(variants))
	for _, v := range variants {
		entries = append(entries, products.VariantEntry{
			Variant:  v,
			Price:    services.FormatCents(v.PriceCents),
			Sellable: services.VariantSellable(v),
		})
	}

	render.HTML(c, products.Detail(products.DetailData{
		Product:      p,
		CategoryName: categoryName,
		Images:       images,
		Variants:     entries,
	}))
}
