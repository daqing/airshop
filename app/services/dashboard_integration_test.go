package services

import (
	"os"
	"testing"
	"time"

	"github.com/daqing/airway/lib/migrate"
	"github.com/daqing/airway/lib/repo"
	sql "github.com/daqing/airway/lib/sql"

	"github.com/daqing/airshop/app/models"
)

// TestDashboardData runs only when a scratch Postgres is provided:
//
//	AIRWAY_PG_TEST_DSN=postgres://...airshop_test go test ./app/services -run TestDashboardData
func TestDashboardData(t *testing.T) {
	dsn := os.Getenv("AIRWAY_PG_TEST_DSN")
	if dsn == "" {
		t.Skip("set AIRWAY_PG_TEST_DSN to run this integration test")
	}

	if err := migrate.Run(migrate.Options{
		DSN:          dsn,
		Migrations:   os.DirFS("../../db/migrate"),
		SnapshotPath: "",
	}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := repo.SetupDB(dsn); err != nil {
		t.Fatalf("setup db: %v", err)
	}

	user, err := repo.CreateFrom[models.User](sql.H{"phone_number": "13100009999", "status": "active"})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	t.Cleanup(func() {
		_ = repo.DeleteWhere[models.Order](sql.H{"user_id": int64(user.ID)})
		_ = repo.DeleteWhere[models.User](sql.H{"phone_number": "13100009999"})
	})

	seedOrder := func(orderNo string, total int64, status string, createdAt time.Time) {
		t.Helper()
		if _, err := repo.CreateFrom[models.Order](sql.H{
			"order_no": orderNo, "user_id": int64(user.ID),
			"ship_recipient": "T", "ship_phone": "13100009999", "ship_address": "x",
			"subtotal_cents": total, "total_cents": total, "status": status,
			"created_at": createdAt, "updated_at": createdAt,
		}); err != nil {
			t.Fatalf("seed %s: %v", orderNo, err)
		}
	}

	localNow := time.Now()
	todayStart := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, localNow.Location())

	// Today: two paid orders (sales 300) and one pending (not counted).
	seedOrder("SO-DSH-T1", 100, OrderStatusPaid, todayStart.Add(time.Hour))
	seedOrder("SO-DSH-T2", 200, OrderStatusPaid, todayStart.Add(2*time.Hour))
	seedOrder("SO-DSH-P", 999, OrderStatusPending, todayStart.Add(3*time.Hour))

	// Earlier this week: one paid (sales 400) and one refunded (not counted).
	seedOrder("SO-DSH-W1", 400, OrderStatusPaid, todayStart.AddDate(0, 0, -2))
	seedOrder("SO-DSH-R", 500, OrderStatusRefunded, todayStart.AddDate(0, 0, -3))

	// Older than a week: not counted in the week window.
	seedOrder("SO-DSH-OLD", 800, OrderStatusPaid, todayStart.AddDate(0, 0, -9))

	stats, err := DashboardData()
	if err != nil {
		t.Fatalf("dashboard data: %v", err)
	}

	if stats.OrdersToday != 2 {
		t.Fatalf("expected 2 orders today, got %d", stats.OrdersToday)
	}
	if stats.SalesToday != 300 {
		t.Fatalf("expected sales today 300, got %d", stats.SalesToday)
	}
	if stats.OrdersWeek != 3 {
		t.Fatalf("expected 3 orders in the week window, got %d", stats.OrdersWeek)
	}
	if stats.SalesWeek != 700 {
		t.Fatalf("expected week sales 700, got %d", stats.SalesWeek)
	}
	if stats.AwaitingPayment != 1 {
		t.Fatalf("expected 1 awaiting payment, got %d", stats.AwaitingPayment)
	}
	// ToShip counts every paid order regardless of age — the 9-day-old one
	// still needs shipping.
	if stats.ToShip != 4 {
		t.Fatalf("expected 4 paid orders to ship, got %d", stats.ToShip)
	}
	if stats.Refunded != 1 {
		t.Fatalf("expected 1 refunded, got %d", stats.Refunded)
	}
}
