package services

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/daqing/airway/lib/repo"
	sql "github.com/daqing/airway/lib/sql"
	"github.com/daqing/airway/lib/storage"
	"github.com/daqing/airway/lib/utils"

	"github.com/daqing/airshop/app/models"
)

const ProductImageMaxSize = 5 << 20 // 5 MB

var productImageExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true,
}

var ErrProductImageTooLarge = errors.New("image exceeds the 5 MB limit")
var ErrProductImageTypeUnsupported = errors.New("image type not supported (jpg, png, webp, gif)")
var ErrProductImageNotFound = errors.New("image not found")

// ProductImageEntry pairs an image row with its browser-ready URL.
type ProductImageEntry struct {
	Image  *models.ProductImage
	URL    string
	IsMain bool
}

// ProductImageURL resolves a stored image key to its browser-ready URL.
// Falls back to the local download path when storage is not configured
// (e.g. view unit tests).
func ProductImageURL(key string) string {
	u := ""
	if store := currentStorage(); store != nil {
		if v, err := store.URL(context.Background(), key, 24*time.Hour); err == nil {
			u = v
		}
	}
	if u == "" {
		u = "/api/v1/storage/" + key
	}
	if strings.HasPrefix(u, "/") {
		u = utils.URLPrefix() + u
	}
	return u
}

func currentStorage() (s storage.Storage) {
	defer func() {
		if recover() != nil {
			s = nil
		}
	}()
	return storage.Current()
}

// ProductImageEntries lists a product's images ordered main-first.
func ProductImageEntries(productID int64) ([]ProductImageEntry, error) {
	images, err := repo.FindBy[models.ProductImage](sql.H{"product_id": productID})
	if err != nil {
		return nil, err
	}

	ordered := sortProductImages(images)
	entries := make([]ProductImageEntry, 0, len(ordered))
	for i, img := range ordered {
		entries = append(entries, ProductImageEntry{
			Image:  img,
			URL:    ProductImageURL(img.Key),
			IsMain: i == 0,
		})
	}
	return entries, nil
}

func sortProductImages(images []*models.ProductImage) []*models.ProductImage {
	ordered := append([]*models.ProductImage(nil), images...)
	for i := 1; i < len(ordered); i++ {
		for j := i; j > 0 && productImageLess(ordered[j], ordered[j-1]); j-- {
			ordered[j], ordered[j-1] = ordered[j-1], ordered[j]
		}
	}
	return ordered
}

func productImageLess(a, b *models.ProductImage) bool {
	if a.SortOrder != b.SortOrder {
		return a.SortOrder < b.SortOrder
	}
	return a.ID < b.ID
}

// AddProductImages stores uploaded files and appends them to the product's
// image list. The first image on the list doubles as the product's main
// image (products.main_image).
func AddProductImages(productID int64, files []*multipart.FileHeader) error {
	if len(files) == 0 {
		return nil
	}

	if _, err := repo.FindByID[models.Product](sql.IdType(productID)); err != nil {
		return err
	}

	existing, err := repo.FindBy[models.ProductImage](sql.H{"product_id": productID})
	if err != nil {
		return err
	}
	nextSort := len(existing)

	for _, fh := range files {
		if err := validateProductImageFile(fh); err != nil {
			return err
		}

		key, err := putProductImage(fh)
		if err != nil {
			return err
		}

		nextSort++
		if _, err := repo.CreateFrom[models.ProductImage](sql.H{
			"product_id": productID,
			"key":        key,
			"sort_order": nextSort,
		}); err != nil {
			return err
		}
	}

	return refreshMainImage(productID)
}

func validateProductImageFile(fh *multipart.FileHeader) error {
	if fh.Size > ProductImageMaxSize {
		return ErrProductImageTooLarge
	}
	if !productImageExts[strings.ToLower(filepath.Ext(fh.Filename))] {
		return ErrProductImageTypeUnsupported
	}
	return nil
}

func putProductImage(fh *multipart.FileHeader) (string, error) {
	src, err := fh.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	ext := strings.ToLower(filepath.Ext(fh.Filename))
	key := fmt.Sprintf("products/%s/%s%s", time.Now().Format("200601"), utils.RandomHex(16), ext)

	contentType := fh.Header.Get("Content-Type")
	if contentType == "" {
		contentType = mime.TypeByExtension(ext)
	}

	err = storage.Current().Put(context.Background(), key, storage.Object{
		Reader:      src,
		Size:        fh.Size,
		ContentType: contentType,
	})
	if err != nil {
		return "", err
	}
	return key, nil
}

// DeleteProductImage removes the image row and its stored object. When the
// main image is deleted, the next image in line becomes the main one.
func DeleteProductImage(productID, imageID int64) error {
	img, err := repo.FindByID[models.ProductImage](sql.IdType(imageID))
	if err != nil {
		return err
	}
	if img == nil || img.ProductID != productID {
		return ErrProductImageNotFound
	}

	if err := repo.DeleteByID[models.ProductImage](sql.IdType(imageID)); err != nil {
		return err
	}

	_ = storage.Current().Delete(context.Background(), img.Key)

	return refreshMainImage(productID)
}

// MakeProductImageMain moves the image to the front of the list, which makes
// it the main image.
func MakeProductImageMain(productID, imageID int64) error {
	images, err := repo.FindBy[models.ProductImage](sql.H{"product_id": productID})
	if err != nil {
		return err
	}

	target := -1
	for i, img := range images {
		if int64(img.ID) == imageID {
			target = i
			break
		}
	}
	if target == -1 {
		return ErrProductImageNotFound
	}

	reordered := append([]*models.ProductImage(nil), images[target])
	reordered = append(reordered, images[:target]...)
	reordered = append(reordered, images[target+1:]...)

	for i, img := range reordered {
		if img.SortOrder == i {
			continue
		}
		if err := repo.UpdateByID[models.ProductImage](img.ID, sql.H{
			"sort_order": i,
			"updated_at": time.Now().UTC(),
		}); err != nil {
			return err
		}
	}

	return refreshMainImage(productID)
}

// refreshMainImage keeps products.main_image in sync with the first image;
// an empty list clears it.
func refreshMainImage(productID int64) error {
	images, err := repo.FindBy[models.ProductImage](sql.H{"product_id": productID})
	if err != nil {
		return err
	}

	main := ""
	if ordered := sortProductImages(images); len(ordered) > 0 {
		main = ordered[0].Key
	}

	return repo.UpdateByID[models.Product](sql.IdType(productID), sql.H{
		"main_image": main,
		"updated_at": time.Now().UTC(),
	})
}
