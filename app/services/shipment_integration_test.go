package services

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/daqing/airway/lib/migrate"
	"github.com/daqing/airway/lib/repo"
	sql "github.com/daqing/airway/lib/sql"

	"github.com/daqing/airshop/app/models"
)

// TestShipOrder runs only when a scratch Postgres is provided:
//
//	AIRWAY_PG_TEST_DSN=postgres://...airshop_test go test ./app/services -run TestShipOrder
func TestShipOrder(t *testing.T) {
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

	user, err := repo.CreateFrom[models.User](sql.H{"phone_number": "13200006666", "status": "active"})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	t.Cleanup(func() {
		_ = repo.DeleteWhere[models.Shipment](sql.H{})
		_ = repo.DeleteWhere[models.Order](sql.H{"user_id": int64(user.ID)})
		_ = repo.DeleteWhere[models.User](sql.H{"phone_number": "13200006666"})
	})

	newOrder := func(orderNo, status string) *models.Order {
		t.Helper()
		order, err := repo.CreateFrom[models.Order](sql.H{
			"order_no": orderNo, "user_id": int64(user.ID),
			"ship_recipient": "T", "ship_phone": "13200006666", "ship_address": "x",
			"subtotal_cents": 100, "total_cents": 100, "status": status,
		})
		if err != nil {
			t.Fatalf("seed %s: %v", orderNo, err)
		}
		return order
	}

	pending := newOrder("SO-SHP-PENDING", OrderStatusPending)
	paid := newOrder("SO-SHP-PAID", OrderStatusPaid)

	// Input validation.
	if err := ShipOrder(int64(paid.ID), "  ", "SF123"); err != ErrShipmentCarrierRequired {
		t.Fatalf("expected carrier-required, got %v", err)
	}
	if err := ShipOrder(int64(paid.ID), "SF Express", " "); err == nil {
		t.Fatal("expected tracking-required error")
	}

	// Only paid orders can ship.
	if err := ShipOrder(int64(pending.ID), "SF Express", "SF001"); err != ErrShipmentNotAllowed {
		t.Fatalf("expected not-allowed for pending, got %v", err)
	}

	// Happy path: shipment created with a first event, order becomes shipped.
	if err := ShipOrder(int64(paid.ID), "SF Express", "SF001"); err != nil {
		t.Fatalf("ship: %v", err)
	}
	check, err := FindOrder(int64(paid.ID))
	if err != nil || check.Status != OrderStatusShipped {
		t.Fatalf("expected shipped, got %#v / %v", check, err)
	}

	shipment, err := FindShipmentByOrder(int64(paid.ID))
	if err != nil || shipment == nil {
		t.Fatalf("shipment lookup: %v", err)
	}
	if shipment.Carrier != "SF Express" || shipment.TrackingNo != "SF001" || shipment.Status != ShipmentStatusCreated {
		t.Fatalf("unexpected shipment %#v", shipment)
	}

	var events []map[string]string
	if err := json.Unmarshal([]byte(shipment.Events), &events); err != nil {
		t.Fatalf("events json: %v", err)
	}
	if len(events) != 1 || events[0]["description"] != "Label created: SF Express SF001" {
		t.Fatalf("expected one label-created event, got %#v", events)
	}

	// Double shipping is rejected.
	if err := ShipOrder(int64(paid.ID), "SF Express", "SF002"); err != ErrShipmentAlreadyExists {
		t.Fatalf("expected already-shipped, got %v", err)
	}

	// Status lifecycle appends events.
	if err := UpdateShipmentStatus(int64(shipment.ID), ShipmentStatusInTransit); err != nil {
		t.Fatalf("in transit: %v", err)
	}
	if err := UpdateShipmentStatus(int64(shipment.ID), ShipmentStatusDelivered); err != nil {
		t.Fatalf("delivered: %v", err)
	}
	if err := UpdateShipmentStatus(int64(shipment.ID), ShipmentStatusDelivered); err == nil {
		t.Fatal("expected a repeated delivered update to be rejected")
	}

	shipment, _ = FindShipmentByOrder(int64(paid.ID))
	if err := json.Unmarshal([]byte(shipment.Events), &events); err != nil {
		t.Fatalf("events json: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("expected 3 tracking events, got %d", len(events))
	}
	if shipment.Status != ShipmentStatusDelivered {
		t.Fatalf("expected delivered, got %q", shipment.Status)
	}
}
