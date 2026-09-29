package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/daqing/airway/lib/repo"
	sql "github.com/daqing/airway/lib/sql"

	"github.com/daqing/airshop/app/models"
)

var (
	ErrCartQuantityInvalid = errors.New("quantity must be at least 1")
	ErrCartItemNotFound    = errors.New("cart item not found")
	ErrProductUnavailable  = errors.New("this product is not available")
	ErrVariantRequired     = errors.New("please choose a variant")
	ErrVariantUnavailable  = errors.New("this variant is not available")
)

func stockFor(product *models.Product, variant *models.ProductVariant) int {
	if variant != nil {
		return variant.Stock
	}
	return product.Stock
}

func priceFor(product *models.Product, variant *models.ProductVariant) int64 {
	if variant != nil {
		return variant.PriceCents
	}
	return product.PriceCents
}

// CartForUser returns the user's cart, creating it on first use.
func CartForUser(userID int64) (*models.Cart, error) {
	cart, err := repo.FindOneBy[models.Cart](sql.H{"user_id": userID})
	if err != nil {
		return nil, err
	}
	if cart != nil {
		return cart, nil
	}
	return repo.CreateFrom[models.Cart](sql.H{"user_id": userID})
}

// resolveCartLine resolves the sellable unit for an add-to-cart request:
// variant lines need an active variant, product lines need an active product.
func resolveCartLine(productID int64, variantID *int64) (*models.Product, *models.ProductVariant, int, error) {
	product, err := repo.FindByID[models.Product](sql.IdType(productID))
	if err != nil {
		return nil, nil, 0, err
	}
	if product == nil || !product.Active {
		return nil, nil, 0, ErrProductUnavailable
	}

	activeVariants, err := ActiveVariants(productID)
	if err != nil {
		return nil, nil, 0, err
	}
	if SellsThroughVariants(activeVariants) {
		if variantID == nil {
			return nil, nil, 0, ErrVariantRequired
		}
		variant, err := repo.FindByID[models.ProductVariant](sql.IdType(*variantID))
		if err != nil {
			return nil, nil, 0, err
		}
		if variant == nil || variant.ProductID != productID || !VariantSellable(variant) {
			return nil, nil, 0, ErrVariantUnavailable
		}
		return product, variant, stockFor(product, variant), nil
	}

	if variantID != nil {
		return nil, nil, 0, ErrVariantRequired
	}
	if !ProductSellable(product) {
		return nil, nil, 0, ErrProductUnavailable
	}
	return product, nil, stockFor(product, nil), nil
}

// AddToCart adds a quantity of the sellable unit to the user's cart,
// merging with an existing line for the same product and variant.
func AddToCart(userID, productID int64, variantID *int64, quantity int) error {
	if quantity < 1 {
		return ErrCartQuantityInvalid
	}

	product, variant, stock, err := resolveCartLine(productID, variantID)
	if err != nil {
		return err
	}

	cart, err := CartForUser(userID)
	if err != nil {
		return err
	}

	items, err := repo.FindBy[models.CartItem](sql.H{"cart_id": int64(cart.ID), "product_id": productID})
	if err != nil {
		return err
	}

	var line *models.CartItem
	for _, item := range items {
		sameVariant := item.VariantID == nil && variantID == nil ||
			item.VariantID != nil && variantID != nil && *item.VariantID == *variantID
		if sameVariant {
			line = item
			break
		}
	}

	if line == nil {
		if quantity > stock {
			return fmt.Errorf("only %d in stock", stock)
		}
		_, err = repo.CreateFrom[models.CartItem](sql.H{
			"cart_id":     int64(cart.ID),
			"product_id":  productID,
			"variant_id":  variantID,
			"quantity":    quantity,
			"price_cents": priceFor(product, variant),
		})
		return err
	}

	newQuantity := line.Quantity + quantity
	if newQuantity > stock {
		return fmt.Errorf("only %d in stock", stock)
	}
	return repo.UpdateByID[models.CartItem](line.ID, sql.H{
		"quantity":   newQuantity,
		"updated_at": time.Now().UTC(),
	})
}

// cartForUserOrErr returns the user's cart; a missing cart means the cart is
// empty.
func cartForUserOrErr(userID int64) (*models.Cart, error) {
	return repo.FindOneBy[models.Cart](sql.H{"user_id": userID})
}

// ownedCartItem resolves the item through the user's cart, so foreign items
// read as not-found.
func ownedCartItem(userID, itemID int64) (*models.CartItem, error) {
	cart, err := cartForUserOrErr(userID)
	if err != nil {
		return nil, err
	}
	if cart == nil {
		return nil, ErrCartItemNotFound
	}

	item, err := repo.FindByID[models.CartItem](sql.IdType(itemID))
	if err != nil {
		return nil, err
	}
	if item == nil || item.CartID != int64(cart.ID) {
		return nil, ErrCartItemNotFound
	}
	return item, nil
}

// UpdateCartItem sets a new quantity for an owned line, capped by stock.
func UpdateCartItem(userID, itemID int64, quantity int) error {
	if quantity < 1 {
		return ErrCartQuantityInvalid
	}

	item, err := ownedCartItem(userID, itemID)
	if err != nil {
		return err
	}

	product, variant, stock, err := resolveCartLine(item.ProductID, item.VariantID)
	if err != nil {
		return err
	}
	_ = product
	_ = variant
	if quantity > stock {
		return fmt.Errorf("only %d in stock", stock)
	}

	return repo.UpdateByID[models.CartItem](item.ID, sql.H{
		"quantity":   quantity,
		"updated_at": time.Now().UTC(),
	})
}

// RemoveCartItem drops one line from the user's cart.
func RemoveCartItem(userID, itemID int64) error {
	item, err := ownedCartItem(userID, itemID)
	if err != nil {
		return err
	}
	return repo.DeleteByID[models.CartItem](item.ID)
}

// ClearCart removes every line from the user's cart.
func ClearCart(userID int64) error {
	cart, err := cartForUserOrErr(userID)
	if err != nil {
		return err
	}
	if cart == nil {
		return nil
	}
	return repo.DeleteWhere[models.CartItem](sql.H{"cart_id": int64(cart.ID)})
}

// CartLine pairs an item with its product and variant for display.
type CartLine struct {
	Item      *models.CartItem
	Product   *models.Product
	Variant   *models.ProductVariant
	ImageURL  string
	VariantOK bool
	// Stock is the currently available stock for the line's sellable unit.
	Stock int
	// Available reports whether the line can be bought at all (product and
	// variant still active, stock not zero).
	Available bool
	// Shortage reports whether the requested quantity exceeds the stock.
	Shortage bool
}

// CartLines returns the user's cart lines with display data and the subtotal
// in minor units. The subtotal only counts lines that are purchasable as-is
// (available and not short of stock).
func CartLines(userID int64) ([]CartLine, int64, error) {
	cart, err := cartForUserOrErr(userID)
	if err != nil {
		return nil, 0, err
	}
	if cart == nil {
		return nil, 0, nil
	}

	items, err := repo.FindBy[models.CartItem](sql.H{"cart_id": int64(cart.ID)})
	if err != nil {
		return nil, 0, err
	}

	lines := make([]CartLine, 0, len(items))
	if len(items) == 0 {
		return lines, 0, nil
	}

	// Batch-load the products for all lines in one query.
	productIDs := make([]int64, 0, len(items))
	for _, item := range items {
		productIDs = append(productIDs, item.ProductID)
	}
	products, err := repo.Find[models.Product](repo.CurrentDB(),
		sql.Select("*").From("products").Where(sql.In("id", productIDs)))
	if err != nil {
		return nil, 0, err
	}
	productByID := make(map[int64]*models.Product, len(products))
	for _, p := range products {
		productByID[int64(p.ID)] = p
	}

	var subtotal int64
	for _, item := range items {
		line := CartLine{Item: item}
		line.Product = productByID[item.ProductID]

		if item.VariantID != nil {
			variant, err := repo.FindByID[models.ProductVariant](sql.IdType(*item.VariantID))
			if err != nil {
				return nil, 0, err
			}
			line.Variant = variant
			line.VariantOK = variant != nil && variant.Active
		}

		if line.Product != nil {
			if line.Product.MainImage != "" {
				line.ImageURL = ProductImageURL(line.Product.MainImage)
			}

			variantActive := item.VariantID == nil || line.VariantOK
			if line.Product.Active && variantActive {
				line.Stock = stockFor(line.Product, line.Variant)
				line.Available = line.Stock > 0
				line.Shortage = line.Stock < item.Quantity
			}
		}

		if line.Available && !line.Shortage {
			subtotal += item.PriceCents * int64(item.Quantity)
		}

		lines = append(lines, line)
	}
	return lines, subtotal, nil
}
