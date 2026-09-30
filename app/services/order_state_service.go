package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/daqing/airway/lib/repo"
	sql "github.com/daqing/airway/lib/sql"

	"github.com/daqing/airshop/app/models"
)

// Order status values and the allowed transitions between them. Every
// status change goes through TransitionOrder — nothing else writes the
// status column.
const (
	OrderStatusPending   = "pending"
	OrderStatusPaid      = "paid"
	OrderStatusShipped   = "shipped"
	OrderStatusCompleted = "completed"
	OrderStatusCancelled = "cancelled"
	OrderStatusRefunded  = "refunded"
)

var orderTransitions = map[string][]string{
	OrderStatusPending:   {OrderStatusPaid, OrderStatusCancelled},
	OrderStatusPaid:      {OrderStatusShipped, OrderStatusRefunded},
	OrderStatusShipped:   {OrderStatusCompleted, OrderStatusRefunded},
	OrderStatusCompleted: {OrderStatusRefunded},
	OrderStatusCancelled: {},
	OrderStatusRefunded:  {},
}

var ErrOrderTransitionInvalid = errors.New("this order status cannot change that way")

func knownOrderStatus(status string) bool {
	_, ok := orderTransitions[status]
	return ok
}

func CanTransitionOrder(from, to string) bool {
	if !knownOrderStatus(from) || !knownOrderStatus(to) {
		return false
	}
	for _, allowed := range orderTransitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}

// FindOrder returns the order by id.
func FindOrder(orderID int64) (*models.Order, error) {
	return repo.FindByID[models.Order](sql.IdType(orderID))
}

// TransitionOrder moves the order to the target status when the transition
// is allowed, returning a descriptive error otherwise. Stock follows the
// T5.4 strategy: deducted when the order becomes paid, restored when it is
// refunded.
func TransitionOrder(orderID int64, to string) error {
	if !knownOrderStatus(to) {
		return fmt.Errorf("unknown order status %q", to)
	}

	order, err := FindOrder(orderID)
	if err != nil {
		return err
	}
	if order == nil {
		return ErrOrderNotFound
	}

	if !CanTransitionOrder(order.Status, to) {
		return fmt.Errorf("%w: %s -> %s", ErrOrderTransitionInvalid, order.Status, to)
	}

	items, err := OrderItems(orderID)
	if err != nil {
		return err
	}

	err = repo.WithTx(repo.CurrentDB(), func(tx *repo.Tx) error {
		if err := repo.UpdateWith(tx.Executor(), sql.UpdateTable(sql.TableFor(models.Order{})).Set(sql.H{
			"status":     to,
			"updated_at": time.Now().UTC(),
		}).Where(sql.Eq("id", int64(order.ID)))); err != nil {
			return err
		}

		switch to {
		case OrderStatusPaid:
			return adjustOrderItemsStock(items, -1)
		case OrderStatusRefunded:
			if err := adjustOrderItemsStock(items, +1); err != nil {
				return err
			}
			return restoreOrderCoupon(tx, orderID)
		case OrderStatusCancelled:
			// Stock was never deducted for pending orders, but the coupon
			// redemption is returned so the user can use it again.
			return restoreOrderCoupon(tx, orderID)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

// restoreOrderCoupon returns a redeemed coupon to the user after a
// cancellation or refund, per the T7.4 policy.
func restoreOrderCoupon(tx *repo.Tx, orderID int64) error {
	return repo.DeleteWith(tx.Executor(),
		sql.DeleteFrom(sql.TableFor(models.CouponRedemption{})).Where(sql.Eq("order_id", orderID)))
}

// adjustOrderItemsStock moves every ordered line's stock by delta (clamped
// at zero; a negative stock can only appear if stock was changed manually
// between checkout and payment).
func adjustOrderItemsStock(items []*models.OrderItem, delta int) error {
	for _, item := range items {
		if item.VariantID != nil {
			variant, err := repo.FindByID[models.ProductVariant](sql.IdType(*item.VariantID))
			if err != nil {
				return err
			}
			if variant == nil {
				continue
			}
			if err := repo.UpdateByID[models.ProductVariant](variant.ID, sql.H{
				"stock":      clampStock(variant.Stock + delta*item.Quantity),
				"updated_at": time.Now().UTC(),
			}); err != nil {
				return err
			}
			continue
		}

		product, err := repo.FindByID[models.Product](sql.IdType(item.ProductID))
		if err != nil {
			return err
		}
		if product == nil {
			continue
		}
		if err := repo.UpdateByID[models.Product](product.ID, sql.H{
			"stock":      clampStock(product.Stock + delta*item.Quantity),
			"updated_at": time.Now().UTC(),
		}); err != nil {
			return err
		}
	}
	return nil
}

func clampStock(stock int) int {
	if stock < 0 {
		return 0
	}
	return stock
}
