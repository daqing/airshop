package services

import (
	"os"
	"testing"

	"github.com/daqing/airway/lib/migrate"
	"github.com/daqing/airway/lib/repo"
	sql "github.com/daqing/airway/lib/sql"

	"github.com/daqing/airshop/app/models"
)

// TestAudit runs only when a scratch Postgres is provided:
//
//	AIRWAY_PG_TEST_DSN=postgres://...airshop_test go test ./app/services -run TestAudit
func TestAudit(t *testing.T) {
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
	t.Cleanup(func() {
		_ = repo.DeleteWhere[models.AdminLog](sql.H{})
	})

	Audit(7, "order.refund", "order", "SO-X", "detail trail")
	Audit(7, "product.price", "product", "42", "100 -> 90")

	logs, err := AdminLogs(10)
	if err != nil {
		t.Fatalf("admin logs: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("expected 2 audit entries, got %d", len(logs))
	}
	if logs[0].Action != "product.price" {
		t.Fatalf("expected newest first, got %q", logs[0].Action)
	}
	if logs[1].AdminID != 7 || logs[1].EntityID != "SO-X" || logs[1].Detail != "detail trail" {
		t.Fatalf("unexpected entry %#v", logs[1])
	}
}
