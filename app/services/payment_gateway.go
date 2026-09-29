package services

import (
	"sort"
	"sync"

	"github.com/daqing/airshop/app/models"
)

// PaymentGateway is one checkout channel. The checkout page lists registered
// gateways by name and sends the buyer to PayLink; each gateway handles its
// own payment confirmation route (the fake gateway posts to itself, real
// channels verify async callbacks in M6+).
type PaymentGateway interface {
	// Name identifies the gateway; it is stored as orders.payment_method.
	Name() string
	// PayLink returns the URL the buyer is redirected to in order to pay
	// the given order.
	PayLink(order *models.Order) (string, error)
}

var (
	gatewayMu       sync.RWMutex
	gatewayRegistry = map[string]PaymentGateway{}
)

// RegisterGateway adds (or replaces) a gateway in the registry. Intended to
// be called from init() by each gateway implementation.
func RegisterGateway(gateway PaymentGateway) {
	gatewayMu.Lock()
	defer gatewayMu.Unlock()
	gatewayRegistry[gateway.Name()] = gateway
}

// Gateway looks up a registered gateway by name.
func Gateway(name string) (PaymentGateway, bool) {
	gatewayMu.RLock()
	defer gatewayMu.RUnlock()
	gateway, ok := gatewayRegistry[name]
	return gateway, ok
}

// GatewayNames returns the registered gateway names in sorted order.
func GatewayNames() []string {
	gatewayMu.RLock()
	defer gatewayMu.RUnlock()

	names := make([]string, 0, len(gatewayRegistry))
	for name := range gatewayRegistry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
