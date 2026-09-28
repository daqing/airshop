package services

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/daqing/airway/lib/repo"
	sql "github.com/daqing/airway/lib/sql"

	"github.com/daqing/airshop/app/models"
)

var (
	slugifyNonAlnum           = regexp.MustCompile(`[^a-z0-9]+`)
	slugifyTrimDashes         = regexp.MustCompile(`^-+|-+$`)
	ErrProductNameRequired    = errors.New("name is required")
	ErrProductSlugInvalid     = errors.New("slug must be lowercase letters, digits and dashes")
	ErrProductSlugTaken       = errors.New("slug is already taken")
	ErrProductPriceInvalid    = errors.New("price must be zero or positive")
	ErrProductStockInvalid    = errors.New("stock must be zero or positive")
	ErrProductCategoryMissing = errors.New("category does not exist")
)

type ProductInput struct {
	Name        string
	Slug        string
	CategoryID  *int64
	Description string
	PriceCents  int64
	Stock       int
	Active      bool
	MainImage   string
}

func (in ProductInput) normalizeSlug() string {
	slug := strings.ToLower(strings.TrimSpace(in.Slug))
	if slug == "" {
		slug = slugifyNonAlnum.ReplaceAllString(strings.ToLower(strings.TrimSpace(in.Name)), "-")
		slug = slugifyTrimDashes.ReplaceAllString(slug, "")
	}
	return slug
}

func (in ProductInput) validate(selfID int64) error {
	if strings.TrimSpace(in.Name) == "" {
		return ErrProductNameRequired
	}
	if !slugPattern.MatchString(in.normalizeSlug()) {
		return ErrProductSlugInvalid
	}
	if in.PriceCents < 0 {
		return ErrProductPriceInvalid
	}
	if in.Stock < 0 {
		return ErrProductStockInvalid
	}
	if in.CategoryID != nil {
		parent, err := repo.FindByID[models.Category](sql.IdType(*in.CategoryID))
		if err != nil {
			return err
		}
		if parent == nil {
			return ErrProductCategoryMissing
		}
	}
	return nil
}

func productSlugTaken(slug string, selfID int64) (bool, error) {
	conflict, err := repo.FindOneBy[models.Product](sql.H{"slug": slug})
	if err != nil {
		return false, err
	}
	return conflict != nil && conflict.ID != sql.IdType(selfID), nil
}

type ProductFilter struct {
	Query  string
	Status string // "" all, "active", "inactive"
	Page   int    // 1-based
	Size   int
}

const ProductsPageSize = 20

func productFilterCond(f ProductFilter) sql.CondBuilder {
	conds := []sql.CondBuilder{}

	if q := strings.TrimSpace(f.Query); q != "" {
		conds = append(conds, sql.ILike("name", "%"+q+"%"))
	}
	switch f.Status {
	case "active":
		conds = append(conds, sql.Eq("active", true))
	case "inactive":
		conds = append(conds, sql.Eq("active", false))
	}

	if len(conds) == 0 {
		return nil
	}
	return sql.AllOf(conds...)
}

func ListProducts(f ProductFilter) ([]*models.Product, int64, error) {
	cond := productFilterCond(f)

	b := sql.Select("*").From("products")
	if cond != nil {
		b = b.Where(cond)
	}
	b = b.OrderBy("id DESC").Limit(f.Size).Offset((f.Page - 1) * f.Size)

	products, err := repo.Find[models.Product](repo.CurrentDB(), b)
	if err != nil {
		return nil, 0, err
	}

	cb := sql.SelectColumns("count(*)").From("products")
	if cond != nil {
		cb = cb.Where(cond)
	}
	total, err := repo.Count(repo.CurrentDB(), cb)
	if err != nil {
		return nil, 0, err
	}

	return products, total, nil
}

func FindProduct(id sql.IdType) (*models.Product, error) {
	return repo.FindByID[models.Product](id)
}

func CreateProduct(in ProductInput) (*models.Product, error) {
	slug := in.normalizeSlug()
	if err := in.validate(0); err != nil {
		return nil, err
	}
	if taken, err := productSlugTaken(slug, 0); err != nil {
		return nil, err
	} else if taken {
		return nil, ErrProductSlugTaken
	}

	return repo.CreateFrom[models.Product](sql.H{
		"category_id": in.CategoryID,
		"name":        in.Name,
		"slug":        slug,
		"description": in.Description,
		"price_cents": in.PriceCents,
		"stock":       in.Stock,
		"active":      in.Active,
		"main_image":  in.MainImage,
	})
}

func UpdateProduct(id sql.IdType, in ProductInput) error {
	slug := in.normalizeSlug()
	if err := in.validate(int64(id)); err != nil {
		return err
	}
	if taken, err := productSlugTaken(slug, int64(id)); err != nil {
		return err
	} else if taken {
		return ErrProductSlugTaken
	}

	return repo.UpdateByID[models.Product](id, sql.H{
		"category_id": in.CategoryID,
		"name":        in.Name,
		"slug":        slug,
		"description": in.Description,
		"price_cents": in.PriceCents,
		"stock":       in.Stock,
		"active":      in.Active,
		"main_image":  in.MainImage,
		"updated_at":  time.Now(),
	})
}

func SetProductActive(id sql.IdType, active bool) error {
	return repo.UpdateByID[models.Product](id, sql.H{
		"active":     active,
		"updated_at": time.Now(),
	})
}
