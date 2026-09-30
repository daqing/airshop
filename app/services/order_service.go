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
func PlaceOrder(userID int64, addressID int64, paymentMethod, couponCode string) (*models.Order, error) {
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
	shipping := int64(0)

	// The coupon discount is recomputed inside the transaction with the
	// coupon row locked FOR UPDATE, so concurrent checkouts cannot exceed a
	// coupon's total count.
	discount := int64(0)
	var appliedCoupon *models.Coupon

	orderNo := "SO" + time.Now().UTC().Format("20060102150405") + "-" + strings.ToUpper(utils.RandomHex(4))

	var order *models.Order
	err = repo.WithTx(repo.CurrentDB(), func(tx *repo.Tx) error {
		if strings.TrimSpace(couponCode) != "" {
			locked, err := repo.FindOneWith[models.Coupon](tx.Executor(),
				sql.Select("*").From("coupons").Where(sql.Eq("code", strings.ToUpper(strings.TrimSpace(couponCode)))).ForUpdate())
			if err != nil {
				return err
			}
			if locked == nil {
				return ErrCouponNotFound
			}
			var discountErr error
			discount, discountErr = CouponDiscountForLoaded(userID, locked, subtotal)
			if discountErr != nil {
				return discountErr
			}
			appliedCoupon = locked
		}

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

		if appliedCoupon != nil {
			if _, err := repo.CreateWith[models.CouponRedemption](tx.Executor(), sql.Create(models.CouponRedemption{}, sql.H{
				"coupon_id":      int64(appliedCoupon.ID),
				"user_id":        userID,
				"order_id":       int64(order.ID),
				"discount_cents": discount,
			})); err != nil {
				return err
			}
		}

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

const AdminOrdersPageSize = 20

// AdminListOrders returns one page of all orders for the admin console,
// newest first, optionally filtered by status and searched by order number.
func AdminListOrders(status, search string, page, size int) ([]*models.Order, int64, error) {
	conds := []sql.CondBuilder{}
	if KnownOrderStatus(status) {
		conds = append(conds, sql.Eq("status", status))
	}
	if trimmed := strings.TrimSpace(search); trimmed != "" {
		conds = append(conds, sql.ILike("order_no", "%"+trimmed+"%"))
	}

	var b *sql.Builder
	if len(conds) == 0 {
		b = sql.Select("*").From("orders")
	} else {
		b = sql.Select("*").From("orders").Where(sql.AllOf(conds...))
	}
	b = b.OrderBy("id DESC").Limit(size).Offset((page - 1) * size)

	orders, err := repo.Find[models.Order](repo.CurrentDB(), b)
	if err != nil {
		return nil, 0, err
	}

	cb := sql.SelectColumns("count(*)").From("orders")
	if len(conds) > 0 {
		cb = cb.Where(sql.AllOf(conds...))
	}
	total, err := repo.Count(repo.CurrentDB(), cb)
	if err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

// AdminFindOrderByNo returns any order by number without user scoping.
func AdminFindOrderByNo(orderNo string) (*models.Order, error) {
	order, err := repo.FindOneBy[models.Order](sql.H{"order_no": orderNo})
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, ErrOrderNotFound
	}
	return order, nil
}

// AdminRefundOrder moves an order to refunded through the state machine;
// stock and coupon restore happen inside the transition.
func AdminRefundOrder(orderNo string) error {
	order, err := AdminFindOrderByNo(orderNo)
	if err != nil {
		return err
	}
	return TransitionOrder(int64(order.ID), OrderStatusRefunded)
}

// OrderItems returns the lines of an order.
func OrderItems(orderID int64) ([]*models.OrderItem, error) {
	b := sql.Select("*").From("order_items").Where(sql.Eq("order_id", orderID)).OrderBy("id ASC")
	return repo.Find[models.OrderItem](repo.CurrentDB(), b)
}
