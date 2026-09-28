package admin_api

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/daqing/airway/lib/render"
	sql "github.com/daqing/airway/lib/sql"

	"github.com/daqing/airshop/app/models"
	"github.com/daqing/airshop/app/services"
	"github.com/daqing/airshop/app/views/admin/categories"
)

func CategoriesIndexAction(c *gin.Context) {
	cats, err := services.ListCategories()
	if err != nil {
		render.ErrorMessage(c, err.Error())
		return
	}

	render.HTML(c, categories.Index(cats, c.Query("error")))
}

func NewCategoryAction(c *gin.Context) {
	parents, err := services.ListCategories()
	if err != nil {
		render.ErrorMessage(c, err.Error())
		return
	}

	render.HTML(c, categories.Form("New category", "/admin/categories", nil, parents, ""))
}

func CreateCategoryAction(c *gin.Context) {
	in := categoryInputFromForm(c)

	_, err := services.CreateCategory(in)
	if err != nil {
		renderCategoryFormError(c, "New category", "/admin/categories", nil, in, err.Error())
		return
	}

	render.Found(c, "/admin/categories")
}

func EditCategoryAction(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		render.ErrorMessage(c, "invalid id")
		return
	}

	cat, err := services.FindCategory(id)
	if err != nil {
		render.ErrorMessage(c, err.Error())
		return
	}
	if cat == nil {
		render.Found(c, "/admin/categories?error="+categoryMissing)
		return
	}

	parents, err := services.ListCategories()
	if err != nil {
		render.ErrorMessage(c, err.Error())
		return
	}

	action := "/admin/categories/" + strconv.FormatInt(int64(id), 10) + "/update"
	render.HTML(c, categories.Form("Edit category", action, cat, excludeCategory(parents, id), ""))
}

func UpdateCategoryAction(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		render.ErrorMessage(c, "invalid id")
		return
	}

	in := categoryInputFromForm(c)
	if err := services.UpdateCategory(id, in); err != nil {
		display := &models.Category{
			ID: id, Name: in.Name, Slug: in.Slug,
			ParentID: in.ParentID, SortOrder: in.SortOrder, Enabled: in.Enabled,
		}
		action := "/admin/categories/" + strconv.FormatInt(int64(id), 10) + "/update"
		renderCategoryFormError(c, "Edit category", action, display, in, err.Error())
		return
	}

	render.Found(c, "/admin/categories")
}

func DestroyCategoryAction(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		render.ErrorMessage(c, "invalid id")
		return
	}

	if err := services.DeleteCategory(id); err != nil {
		render.Found(c, "/admin/categories?error="+err.Error())
		return
	}

	render.Found(c, "/admin/categories")
}

const categoryMissing = "category not found"

func parseID(c *gin.Context) (sql.IdType, error) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	return sql.IdType(id), err
}

func categoryInputFromForm(c *gin.Context) services.CategoryInput {
	in := services.CategoryInput{
		Name: c.PostForm("name"),
		Slug: c.PostForm("slug"),
	}

	if raw := c.PostForm("parent_id"); raw != "" {
		if parentID, err := strconv.ParseInt(raw, 10, 64); err == nil {
			in.ParentID = &parentID
		}
	}

	if raw := c.PostForm("sort_order"); raw != "" {
		if sort, err := strconv.Atoi(raw); err == nil {
			in.SortOrder = sort
		}
	}

	in.Enabled = c.PostForm("enabled") == "true"
	return in
}

func renderCategoryFormError(c *gin.Context, title, action string, cat *models.Category, in services.CategoryInput, msg string) {
	parents, err := services.ListCategories()
	if err != nil {
		render.ErrorMessage(c, err.Error())
		return
	}

	display := cat
	if display == nil {
		display = &models.Category{Name: in.Name, Slug: in.Slug, ParentID: in.ParentID, SortOrder: in.SortOrder, Enabled: in.Enabled}
	}

	render.HTMLStatus(c, 422, categories.Form(title, action, display, excludeCategory(parents, display.ID), msg))
}

func excludeCategory(cats []*models.Category, id sql.IdType) []*models.Category {
	filtered := make([]*models.Category, 0, len(cats))
	for _, cat := range cats {
		if cat.ID != id {
			filtered = append(filtered, cat)
		}
	}
	return filtered
}
