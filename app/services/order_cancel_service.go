package services

import (
	"log"
	"time"

	"github.com/daqing/airway/lib/repo"
	sql "github.com/daqing/airway/lib/sql"

	"github.com/daqing/airshop/app/models"
)

// UnpaidOrderExpiry is how long a pending order may wait for payment before
// the expiry sweeper cancels it.
const UnpaidOrderExpiry = 30 * time.Minute

// CancelOrder cancels the user's own order. Only pending orders can be
// cancelled — the state machine rejects everything else.
func CancelOrder(userID int64, orderNo string) error {
	order, err := FindOrderByNo(userID, orderNo)
	if err != nil {
		return err
	}
	return TransitionOrder(int64(order.ID), OrderStatusCancelled)
}

// CancelExpiredOrders cancels pending orders that have waited longer than
// maxAge for payment. The cutoff uses the local wall clock on purpose:
// created_at is filled by the database's CURRENT_TIMESTAMP, which writes the
// server-local time (see the note in CONVENTIONS.md).
func CancelExpiredOrders(maxAge time.Duration) (int, error) {
	cutoff := time.Now().Add(-maxAge)

	b := sql.Select("*").From("orders").Where(sql.AllOf(
		sql.Eq("status", OrderStatusPending),
		sql.Lt("created_at", cutoff),
	))
	expired, err := repo.Find[models.Order](repo.CurrentDB(), b)
	if err != nil {
		return 0, err
	}

	cancelled := 0
	for _, order := range expired {
		if err := TransitionOrder(int64(order.ID), OrderStatusCancelled); err != nil {
			log.Printf("order expiry sweep: order %s: %v", order.OrderNo, err)
			continue
		}
		cancelled++
	}
	return cancelled, nil
}

// RunOrderExpirySweeper runs CancelExpiredOrders once and then on every
// tick until the process exits. Intended as a background goroutine from
// main.
func RunOrderExpirySweeper(maxAge, interval time.Duration) {
	if _, err := CancelExpiredOrders(maxAge); err != nil {
		log.Printf("order expiry sweep: %v", err)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		cancelled, err := CancelExpiredOrders(maxAge)
		if err != nil {
			log.Printf("order expiry sweep: %v", err)
			continue
		}
		if cancelled > 0 {
			log.Printf("order expiry sweep: cancelled %d order(s)", cancelled)
		}
	}
}
