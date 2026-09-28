package admin_api

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/daqing/airway/lib/render"

	"github.com/daqing/airshop/app/models"
	"github.com/daqing/airshop/app/services"
	productsviews "github.com/daqing/airshop/app/views/admin/products"
)

func ProductsIndexAction(c *gin.Context) {
	f := productFilterFromQuery(c)

	products, total, err := services.ListProducts(f)
	if err != nil {
		render.ErrorMessage(c, err.Error())
		return
	}

	cats, err := services.ListCategories()
	if err != nil {
		render.ErrorMessage(c, err.Error())
		return
	}

	totalPages := int((total + services.ProductsPageSize - 1) / services.ProductsPageSize)
	if totalPages < 1 {
		totalPages = 1
	}

	render.HTML(c, productsviews.Index(productsviews.IndexData{
		Products:   products,
		Categories: cats,
		Query:      f.Query,
		Status:     f.Status,
		Page:       f.Page,
		TotalPages: totalPages,
		Total:      total,
		Flash:      c.Query("error"),
	}))
}

func NewProductAction(c *gin.Context) {
	cats, err := services.ListCategories()
	if err != nil {
		render.ErrorMessage(c, err.Error())
		return
	}

	render.HTML(c, productsviews.Form("New product", "/admin/products", nil, cats, ""))
}

func CreateProductAction(c *gin.Context) {
	in := productInputFromForm(c)

	_, err := services.CreateProduct(in)
	if err != nil {
		renderProductFormError(c, "New product", "/admin/products", nil, in, err.Error())
		return
	}

	render.Found(c, "/admin/products")
}

func EditProductAction(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		render.ErrorMessage(c, "invalid id")
		return
	}

	p, err := services.FindProduct(id)
	if err != nil {
		render.ErrorMessage(c, err.Error())
		return
	}
	if p == nil {
		render.Found(c, "/admin/products?error="+productMissing)
		return
	}

	cats, err := services.ListCategories()
	if err != nil {
		render.ErrorMessage(c, err.Error())
		return
	}

	action := "/admin/products/" + strconv.FormatInt(int64(id), 10) + "/update"
	render.HTML(c, productsviews.Form("Edit product", action, p, cats, ""))
}

func UpdateProductAction(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		render.ErrorMessage(c, "invalid id")
		return
	}

	in := productInputFromForm(c)
	if err := services.UpdateProduct(id, in); err != nil {
		display := &models.Product{
			ID: id, CategoryID: in.CategoryID, Name: in.Name, Slug: in.Slug,
			Description: in.Description, PriceCents: in.PriceCents, Stock: in.Stock,
			Active: in.Active, MainImage: in.MainImage,
		}
		action := "/admin/products/" + strconv.FormatInt(int64(id), 10) + "/update"
		renderProductFormError(c, "Edit product", action, display, in, err.Error())
		return
	}

	render.Found(c, "/admin/products")
}

func ActivateProductAction(c *gin.Context) {
	toggleProductActive(c, true)
}

func DeactivateProductAction(c *gin.Context) {
	toggleProductActive(c, false)
}

func toggleProductActive(c *gin.Context, active bool) {
	id, err := parseID(c)
	if err != nil {
		render.ErrorMessage(c, "invalid id")
		return
	}

	if err := services.SetProductActive(id, active); err != nil {
		render.Found(c, "/admin/products?error="+err.Error())
		return
	}

	render.Found(c, "/admin/products")
}

const productMissing = "product not found"

func productFilterFromQuery(c *gin.Context) services.ProductFilter {
	f := services.ProductFilter{
		Query:  c.Query("q"),
		Status: c.Query("status"),
		Page:   1,
		Size:   services.ProductsPageSize,
	}

	if page, err := strconv.Atoi(c.Query("page")); err == nil && page > 0 {
		f.Page = page
	}
	if f.Status != "active" && f.Status != "inactive" {
		f.Status = ""
	}
	return f
}

func productInputFromForm(c *gin.Context) services.ProductInput {
	in := services.ProductInput{
		Name:        c.PostForm("name"),
		Slug:        c.PostForm("slug"),
		Description: c.PostForm("description"),
		MainImage:   strings.TrimSpace(c.PostForm("main_image")),
		Active:      c.PostForm("active") == "true",
	}

	if raw := c.PostForm("category_id"); raw != "" {
		if catID, err := strconv.ParseInt(raw, 10, 64); err == nil {
			in.CategoryID = &catID
		}
	}

	if cents, err := services.ParsePriceToCents(c.PostForm("price")); err == nil {
		in.PriceCents = cents
	} else {
		in.PriceCents = -1
	}

	if stock, err := strconv.Atoi(c.PostForm("stock")); err == nil {
		in.Stock = stock
	} else {
		in.Stock = -1
	}

	return in
}

func renderProductFormError(c *gin.Context, title, action string, p *models.Product, in services.ProductInput, msg string) {
	cats, err := services.ListCategories()
	if err != nil {
		render.ErrorMessage(c, err.Error())
		return
	}

	display := p
	if display == nil {
		display = &models.Product{
			CategoryID: in.CategoryID, Name: in.Name, Slug: in.Slug,
			Description: in.Description, PriceCents: in.PriceCents, Stock: in.Stock,
			Active: in.Active, MainImage: in.MainImage,
		}
	}

	render.HTMLStatus(c, 422, productsviews.Form(title, action, display, cats, msg))
}
