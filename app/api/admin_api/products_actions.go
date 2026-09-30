package admin_api

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/daqing/airway/lib/render"

	"github.com/daqing/airshop/app/middlewares"
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

	render.HTML(c, productsviews.Index(middlewares.CurrentAdmin(c), productsviews.IndexData{
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

	render.HTML(c, productsviews.Form(middlewares.CurrentAdmin(c), "New product", "/admin/products", nil, cats, nil, ""))
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

	entries, err := services.ProductImageEntries(int64(id))
	if err != nil {
		render.ErrorMessage(c, err.Error())
		return
	}

	render.HTML(c, productsviews.Form(middlewares.CurrentAdmin(c), "Edit product", action, p, cats, entries, ""))
}

func UpdateProductAction(c *gin.Context) {
	admin := middlewares.CurrentAdmin(c)

	id, err := parseID(c)
	if err != nil {
		render.ErrorMessage(c, "invalid id")
		return
	}

	in := productInputFromForm(c)

	// Audit only real price changes; the previous price is needed for the
	// detail trail.
	existing, err := services.FindProduct(id)
	if err != nil {
		render.ErrorMessage(c, err.Error())
		return
	}

	if err := services.UpdateProduct(id, in); err != nil {
		display := &models.Product{
			ID: id, CategoryID: in.CategoryID, Name: in.Name, Slug: in.Slug,
			Description: in.Description, PriceCents: in.PriceCents, Stock: in.Stock,
			Active: in.Active,
		}
		action := "/admin/products/" + strconv.FormatInt(int64(id), 10) + "/update"
		renderProductFormError(c, "Edit product", action, display, in, err.Error())
		return
	}

	if existing != nil && existing.PriceCents != in.PriceCents {
		services.Audit(int64(admin.ID), "product.price", "product",
			strconv.FormatInt(int64(id), 10),
			services.FormatCents(existing.PriceCents)+" -> "+services.FormatCents(in.PriceCents))
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
	admin := middlewares.CurrentAdmin(c)
	id, err := parseID(c)
	if err != nil {
		render.ErrorMessage(c, "invalid id")
		return
	}

	if err := services.SetProductActive(id, active); err != nil {
		render.Found(c, "/admin/products?error="+err.Error())
		return
	}

	verb := "product.deactivate"
	if active {
		verb = "product.activate"
	}
	services.Audit(int64(admin.ID), verb, "product", strconv.FormatInt(int64(id), 10), "")
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
			Active: in.Active,
		}
	}

	render.HTMLStatus(c, 422, productsviews.Form(middlewares.CurrentAdmin(c), title, action, display, cats, nil, msg))
}

func UploadProductImagesAction(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		render.ErrorMessage(c, "invalid id")
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		render.Found(c, "/admin/products/"+strconv.FormatInt(int64(id), 10)+"/edit?error=upload+failed")
		return
	}

	if err := services.AddProductImages(int64(id), form.File["images"]); err != nil {
		render.Found(c, "/admin/products/"+strconv.FormatInt(int64(id), 10)+"/edit?error="+err.Error())
		return
	}

	render.Found(c, "/admin/products/"+strconv.FormatInt(int64(id), 10)+"/edit")
}

func DeleteProductImageAction(c *gin.Context) {
	productImageAction(c, services.DeleteProductImage)
}

func MakeProductImageMainAction(c *gin.Context) {
	productImageAction(c, services.MakeProductImageMain)
}

func productImageAction(c *gin.Context, apply func(productID, imageID int64) error) {
	id, err := parseID(c)
	if err != nil {
		render.ErrorMessage(c, "invalid id")
		return
	}

	imageID, err := strconv.ParseInt(c.Param("imageId"), 10, 64)
	if err != nil {
		render.ErrorMessage(c, "invalid image id")
		return
	}

	editURL := "/admin/products/" + strconv.FormatInt(int64(id), 10) + "/edit"
	if err := apply(int64(id), imageID); err != nil {
		render.Found(c, editURL+"?error="+err.Error())
		return
	}

	render.Found(c, editURL)
}
