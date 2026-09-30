package categories

import (
	"fmt"
	"strconv"

	"github.com/daqing/airshop/app/models"
)

func parentName(cats []*models.Category, id int64) string {
	for _, c := range cats {
		if int64(c.ID) == id {
			return c.Name
		}
	}
	return "—"
}

func fmtID(id int64) string {
	return strconv.FormatInt(id, 10)
}

func fmtInt(n int) string {
	return fmt.Sprintf("%d", n)
}

func formValue(s string) string {
	return s
}

func catName(c *models.Category) string {
	if c == nil {
		return ""
	}
	return c.Name
}

func catSlug(c *models.Category) string {
	if c == nil {
		return ""
	}
	return c.Slug
}

func catSortOrder(c *models.Category) string {
	if c == nil {
		return "0"
	}
	return fmtInt(c.SortOrder)
}

func enabledBadge(enabled bool) string {
	if enabled {
		return "on"
	}
	return "off"
}
