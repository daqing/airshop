package services

import (
	"errors"
	"regexp"
	"time"

	"github.com/daqing/airway/lib/repo"
	sql "github.com/daqing/airway/lib/sql"

	"github.com/daqing/airshop/app/models"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

var (
	ErrCategoryNameRequired  = errors.New("name is required")
	ErrCategorySlugInvalid   = errors.New("slug must be lowercase letters, digits and dashes")
	ErrCategorySlugTaken     = errors.New("slug is already taken")
	ErrCategoryParentMissing = errors.New("parent category does not exist")
	ErrCategoryParentSelf    = errors.New("category cannot be its own parent")
	ErrCategoryHasChildren   = errors.New("category has subcategories; move or delete them first")
)

type CategoryInput struct {
	Name      string
	Slug      string
	ParentID  *int64
	SortOrder int
	Enabled   bool
}

func (in CategoryInput) validate(selfID int64) error {
	if in.Name == "" {
		return ErrCategoryNameRequired
	}
	if !slugPattern.MatchString(in.Slug) {
		return ErrCategorySlugInvalid
	}
	if in.ParentID != nil {
		if *in.ParentID == selfID {
			return ErrCategoryParentSelf
		}
		parent, err := repo.FindByID[models.Category](sql.IdType(*in.ParentID))
		if err != nil {
			return err
		}
		if parent == nil {
			return ErrCategoryParentMissing
		}
	}
	return nil
}

func slugTaken(slug string, selfID int64) (bool, error) {
	conflict, err := repo.FindOneBy[models.Category](sql.H{"slug": slug})
	if err != nil {
		return false, err
	}
	return conflict != nil && conflict.ID != sql.IdType(selfID), nil
}

func ListCategories() ([]*models.Category, error) {
	b := sql.Select("*").From("categories").OrderBy("sort_order ASC, id ASC")
	return repo.Find[models.Category](repo.CurrentDB(), b)
}

func FindCategory(id sql.IdType) (*models.Category, error) {
	return repo.FindByID[models.Category](id)
}

func CreateCategory(in CategoryInput) (*models.Category, error) {
	if err := in.validate(0); err != nil {
		return nil, err
	}
	if taken, err := slugTaken(in.Slug, 0); err != nil {
		return nil, err
	} else if taken {
		return nil, ErrCategorySlugTaken
	}

	return repo.CreateFrom[models.Category](sql.H{
		"name":       in.Name,
		"slug":       in.Slug,
		"parent_id":  in.ParentID,
		"sort_order": in.SortOrder,
		"enabled":    in.Enabled,
	})
}

func UpdateCategory(id sql.IdType, in CategoryInput) error {
	if err := in.validate(int64(id)); err != nil {
		return err
	}
	if taken, err := slugTaken(in.Slug, int64(id)); err != nil {
		return err
	} else if taken {
		return ErrCategorySlugTaken
	}

	return repo.UpdateByID[models.Category](id, sql.H{
		"name":       in.Name,
		"slug":       in.Slug,
		"parent_id":  in.ParentID,
		"sort_order": in.SortOrder,
		"enabled":    in.Enabled,
		"updated_at": time.Now().UTC(),
	})
}

func DeleteCategory(id sql.IdType) error {
	children, err := repo.CountWhere[models.Category](sql.H{"parent_id": int64(id)})
	if err != nil {
		return err
	}
	if children > 0 {
		return ErrCategoryHasChildren
	}
	return repo.DeleteByID[models.Category](id)
}
