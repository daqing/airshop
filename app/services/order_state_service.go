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
// is allowed, returning a descriptive error otherwise.
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

	return repo.UpdateByID[models.Order](order.ID, sql.H{
		"status":     to,
		"updated_at": time.Now().UTC(),
	})
}
