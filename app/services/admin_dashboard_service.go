package services

import (
	"time"

	"github.com/daqing/airway/lib/repo"
	sql "github.com/daqing/airway/lib/sql"

	"github.com/daqing/airshop/app/models"
)

// Paid statuses count toward sales figures.
var salesStatuses = []string{OrderStatusPaid, OrderStatusShipped, OrderStatusCompleted}

// DashboardStats aggregates the admin landing-page numbers.
type DashboardStats struct {
	OrdersToday int64
	SalesToday  int64
	OrdersWeek  int64
	SalesWeek   int64

	AwaitingPayment int64 // status pending
	ToShip          int64 // status paid
	Shipped         int64 // status shipped
	Refunded        int64 // status refunded
}

// ordersSince counts paid orders created since the given local wall-clock
// time; created_at is filled by CURRENT_TIMESTAMP, so the cutoff uses the
// local wall clock (see CONVENTIONS.md).
func ordersSince(cutoff time.Time) (int64, error) {
	return repo.Count(repo.CurrentDB(), sql.SelectColumns("count(*)").From("orders").Where(sql.AllOf(
		sql.In("status", salesStatuses),
		sql.Gte("created_at", cutoff),
	)))
}

// salesSince sums the totals of paid orders created since the cutoff.
func salesSince(cutoff time.Time) (int64, error) {
	type totalRow struct {
		Total int64 `db:"total"`
	}
	b := sql.SelectColumns("COALESCE(SUM(total_cents), 0) AS total").From("orders").Where(sql.AllOf(
		sql.In("status", salesStatuses),
		sql.Gte("created_at", cutoff),
	))
	row, err := repo.FindOne[totalRow](repo.CurrentDB(), b)
	if err != nil {
		return 0, err
	}
	if row == nil {
		return 0, nil
	}
	return row.Total, nil
}

func countByStatus(status string) (int64, error) {
	return repo.CountWhere[models.Order](sql.H{"status": status})
}

// DashboardData assembles every number the admin dashboard shows.
func DashboardData() (DashboardStats, error) {
	stats := DashboardStats{}

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekStart := todayStart.AddDate(0, 0, -6)

	var err error
	if stats.OrdersToday, err = ordersSince(todayStart); err != nil {
		return stats, err
	}
	if stats.SalesToday, err = salesSince(todayStart); err != nil {
		return stats, err
	}
	if stats.OrdersWeek, err = ordersSince(weekStart); err != nil {
		return stats, err
	}
	if stats.SalesWeek, err = salesSince(weekStart); err != nil {
		return stats, err
	}
	if stats.AwaitingPayment, err = countByStatus(OrderStatusPending); err != nil {
		return stats, err
	}
	if stats.ToShip, err = countByStatus(OrderStatusPaid); err != nil {
		return stats, err
	}
	if stats.Shipped, err = countByStatus(OrderStatusShipped); err != nil {
		return stats, err
	}
	if stats.Refunded, err = countByStatus(OrderStatusRefunded); err != nil {
		return stats, err
	}
	return stats, nil
}
