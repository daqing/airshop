package services

import (
	"errors"
	"strings"
	"time"

	"github.com/daqing/airway/lib/repo"
	sql "github.com/daqing/airway/lib/sql"
	"github.com/daqing/airway/lib/utils"

	"github.com/daqing/airshop/app/models"
)

var (
	ErrCartEmpty            = errors.New("your cart is empty")
	ErrCartHasUnavailable   = errors.New("some items in your cart are no longer available")
	ErrOrderNotFound        = errors.New("order not found")
	ErrPaymentMethodNeeded  = errors.New("choose a payment method")
	ErrPaymentMethodUnknown = errors.New("unknown payment method")
)

// PlaceOrder turns the user's purchasable cart lines into an order: amounts
// are recomputed server-side from the line snapshots, the address is copied
// into the order and the cart is cleared — all in one transaction.
func PlaceOrder(userID int64, addressID int64, paymentMethod string) (*models.Order, error) {
	paymentMethod = strings.TrimSpace(paymentMethod)
	if paymentMethod == "" {
		return nil, ErrPaymentMethodNeeded
	}
	if _, ok := Gateway(paymentMethod); !ok {
		return nil, ErrPaymentMethodUnknown
	}

	lines, _, err := CartLines(userID)
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, ErrCartEmpty
	}
	for _, line := range lines {
		if !line.Available || line.Shortage {
			return nil, ErrCartHasUnavailable
		}
	}

	address, err := FindAddress(userID, addressID)
	if err != nil {
		return nil, err
	}

	subtotal := int64(0)
	for _, line := range lines {
		subtotal += line.Item.PriceCents * int64(line.Item.Quantity)
	}
	discount := int64(0)
	shipping := int64(0)

	orderNo := "SO" + time.Now().UTC().Format("20060102150405") + "-" + strings.ToUpper(utils.RandomHex(4))

	var order *models.Order
	err = repo.WithTx(repo.CurrentDB(), func(tx *repo.Tx) error {
		created, err := repo.CreateWith[models.Order](tx.Executor(), sql.Create(models.Order{}, sql.H{
			"order_no":       orderNo,
			"user_id":        userID,
			"ship_recipient": address.Recipient,
			"ship_phone":     address.Phone,
			"ship_address":   shipAddressString(address),
			"subtotal_cents": subtotal,
			"discount_cents": discount,
			"shipping_cents": shipping,
			"total_cents":    subtotal - discount + shipping,
			"status":         "pending",
			"payment_method": paymentMethod,
		}))
		if err != nil {
			return err
		}
		order = created

		for _, line := range lines {
			var variantID any
			if line.Item.VariantID != nil {
				variantID = *line.Item.VariantID
			}
			productName := ""
			imageKey := ""
			if line.Product != nil {
				productName = line.Product.Name
				imageKey = line.Product.MainImage
			}
			variantName := ""
			if line.Variant != nil {
				variantName = line.Variant.Name
			}

			if _, err := repo.CreateWith[models.OrderItem](tx.Executor(), sql.Create(models.OrderItem{}, sql.H{
				"order_id":         int64(order.ID),
				"product_id":       line.Item.ProductID,
				"variant_id":       variantID,
				"product_name":     productName,
				"variant_name":     variantName,
				"image_key":        imageKey,
				"unit_price_cents": line.Item.PriceCents,
				"quantity":         line.Item.Quantity,
			})); err != nil {
				return err
			}
		}

		return repo.DeleteWith(tx.Executor(),
			sql.DeleteFrom(sql.TableFor(models.CartItem{})).Where(sql.Eq("cart_id", lines[0].Item.CartID)))
	})
	if err != nil {
		return nil, err
	}

	return order, nil
}

func shipAddressString(a *models.Address) string {
	parts := make([]string, 0, 4)
	for _, p := range []string{a.Province, a.City, a.District, a.Street} {
		if strings.TrimSpace(p) != "" {
			parts = append(parts, strings.TrimSpace(p))
		}
	}
	return strings.Join(parts, " ")
}

// FindOrderByNo returns the user's order with the given order number, or
// ErrOrderNotFound when it does not exist or belongs to someone else.
func FindOrderByNo(userID int64, orderNo string) (*models.Order, error) {
	order, err := repo.FindOneBy[models.Order](sql.H{"order_no": orderNo})
	if err != nil {
		return nil, err
	}
	if order == nil || order.UserID != userID {
		return nil, ErrOrderNotFound
	}
	return order, nil
}

const OrdersPageSize = 10

// KnownOrderStatus reports whether the status filter value is a real order
// status.
func KnownOrderStatus(status string) bool {
	return knownOrderStatus(status)
}

// ListOrders returns one page of the user's orders, newest first, optionally
// filtered by status, with the total count.
func ListOrders(userID int64, status string, page, size int) ([]*models.Order, int64, error) {
	conds := []sql.CondBuilder{sql.Eq("user_id", userID)}
	if KnownOrderStatus(status) {
		conds = append(conds, sql.Eq("status", status))
	}
	cond := sql.AllOf(conds...)

	b := sql.Select("*").From("orders").Where(cond).OrderBy("id DESC").Limit(size).Offset((page - 1) * size)
	orders, err := repo.Find[models.Order](repo.CurrentDB(), b)
	if err != nil {
		return nil, 0, err
	}

	cb := sql.SelectColumns("count(*)").From("orders").Where(cond)
	total, err := repo.Count(repo.CurrentDB(), cb)
	if err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

// OrderItems returns the lines of an order.
func OrderItems(orderID int64) ([]*models.OrderItem, error) {
	b := sql.Select("*").From("order_items").Where(sql.Eq("order_id", orderID)).OrderBy("id ASC")
	return repo.Find[models.OrderItem](repo.CurrentDB(), b)
}
