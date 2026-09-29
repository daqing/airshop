package services

import (
	"fmt"

	"github.com/daqing/airshop/app/models"
)

// FakeGatewayName is the local-test gateway: it renders an internal cashier
// page where one click marks the order paid. No real money moves.
const FakeGatewayName = "fake"

func init() {
	RegisterGateway(FakeGateway{})
}

// FakeGateway implements PaymentGateway for local development.
type FakeGateway struct{}

func (FakeGateway) Name() string { return FakeGatewayName }

// PayLink sends the buyer to the internal fake cashier page.
func (FakeGateway) PayLink(order *models.Order) (string, error) {
	if order == nil {
		return "", fmt.Errorf("order is required")
	}
	return "/pay/fake?order=" + order.OrderNo, nil
}
