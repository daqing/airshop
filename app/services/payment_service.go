package services

// MarkOrderPaid is the single payment-success entry point for every gateway
// (the fake cashier today, signed async callbacks in T6.3a/b). Idempotent by
// contract: duplicate notifications for an already-paid order succeed
// quietly so gateways stop retrying, while orders in other states (refunded,
// cancelled) are rejected. Paying deducts stock via the state machine.
func MarkOrderPaid(orderID int64) error {
	order, err := FindOrder(orderID)
	if err != nil {
		return err
	}
	if order == nil {
		return ErrOrderNotFound
	}
	if order.Status == OrderStatusPaid {
		return nil
	}
	return TransitionOrder(orderID, OrderStatusPaid)
}
