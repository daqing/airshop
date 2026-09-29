package services

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/daqing/airway/lib/repo"
	sql "github.com/daqing/airway/lib/sql"

	"github.com/daqing/airshop/app/models"
)

const (
	ShipmentStatusCreated   = "created"
	ShipmentStatusInTransit = "in_transit"
	ShipmentStatusDelivered = "delivered"
)

var (
	ErrShipmentCarrierRequired = errors.New("carrier is required")
	ErrShipmentNumberRequired  = errors.New("tracking number is required")
	ErrShipmentNotAllowed      = errors.New("only paid orders can be shipped")
	ErrShipmentAlreadyExists   = errors.New("this order is already shipped")
	ErrShipmentNotFound        = errors.New("shipment not found")
	ErrShipmentStatusInvalid   = errors.New("shipment status must be in_transit or delivered")
)

// shipmentEvent is one entry of the tracking timeline.
type shipmentEvent struct {
	Time        string `json:"time"`
	Description string `json:"description"`
}

func appendShipmentEvent(current, description string, at time.Time) (string, error) {
	events := []shipmentEvent{}
	if strings.TrimSpace(current) != "" {
		if err := json.Unmarshal([]byte(current), &events); err != nil {
			events = []shipmentEvent{}
		}
	}
	events = append(events, shipmentEvent{
		Time:        at.UTC().Format(time.RFC3339),
		Description: description,
	})

	updated, err := json.Marshal(events)
	if err != nil {
		return "", err
	}
	return string(updated), nil
}

// ListPaidOrders returns orders waiting to be shipped, oldest first.
func ListPaidOrders() ([]*models.Order, error) {
	b := sql.Select("*").From("orders").Where(sql.Eq("status", OrderStatusPaid)).OrderBy("id ASC")
	return repo.Find[models.Order](repo.CurrentDB(), b)
}

// ListShipments returns every shipment, newest first.
func ListShipments() ([]*models.Shipment, error) {
	b := sql.Select("*").From("shipments").OrderBy("id DESC")
	return repo.Find[models.Shipment](repo.CurrentDB(), b)
}

func FindShipmentByOrder(orderID int64) (*models.Shipment, error) {
	return repo.FindOneBy[models.Shipment](sql.H{"order_id": orderID})
}

// ShipOrder creates the shipment for a paid order and moves the order to
// shipped. The first tracking event records the label creation.
func ShipOrder(orderID int64, carrier, trackingNo string) error {
	carrier = strings.TrimSpace(carrier)
	trackingNo = strings.TrimSpace(trackingNo)
	if carrier == "" {
		return ErrShipmentCarrierRequired
	}
	if trackingNo == "" {
		return ErrShipmentNumberRequired
	}

	order, err := FindOrder(orderID)
	if err != nil {
		return err
	}
	if order == nil {
		return ErrOrderNotFound
	}
	if existing, err := FindShipmentByOrder(orderID); err != nil {
		return err
	} else if existing != nil {
		return ErrShipmentAlreadyExists
	}
	if order.Status != OrderStatusPaid {
		return ErrShipmentNotAllowed
	}

	events, err := appendShipmentEvent("", "Label created: "+carrier+" "+trackingNo, time.Now().UTC())
	if err != nil {
		return err
	}

	if _, err := repo.CreateFrom[models.Shipment](sql.H{
		"order_id":    orderID,
		"carrier":     carrier,
		"tracking_no": trackingNo,
		"status":      ShipmentStatusCreated,
		"events":      events,
	}); err != nil {
		return err
	}

	if err := TransitionOrder(orderID, OrderStatusShipped); err != nil {
		// Roll the shipment row back so the admin can retry.
		_ = repo.DeleteWhere[models.Shipment](sql.H{"order_id": orderID})
		return err
	}
	return nil
}

// UpdateShipmentStatus moves the shipment along its own lifecycle
// (created → in_transit → delivered) and appends a tracking event.
func UpdateShipmentStatus(shipmentID int64, status string) error {
	if status != ShipmentStatusInTransit && status != ShipmentStatusDelivered {
		return ErrShipmentStatusInvalid
	}

	shipment, err := repo.FindByID[models.Shipment](sql.IdType(shipmentID))
	if err != nil {
		return err
	}
	if shipment == nil {
		return ErrShipmentNotFound
	}
	if status == ShipmentStatusInTransit && shipment.Status != ShipmentStatusCreated {
		return ErrShipmentStatusInvalid
	}
	if status == ShipmentStatusDelivered && shipment.Status == ShipmentStatusDelivered {
		return ErrShipmentStatusInvalid
	}

	events, err := appendShipmentEvent(shipment.Events, "Status: "+status, time.Now().UTC())
	if err != nil {
		return err
	}

	return repo.UpdateByID[models.Shipment](shipment.ID, sql.H{
		"status":     status,
		"events":     events,
		"updated_at": time.Now().UTC(),
	})
}
