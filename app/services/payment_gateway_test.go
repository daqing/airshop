package services

import (
	"testing"

	"github.com/daqing/airshop/app/models"
)

type fakeGateway struct {
	name string
	link string
}

func (g fakeGateway) Name() string { return g.name }

func (g fakeGateway) PayLink(order *models.Order) (string, error) {
	return g.link + order.OrderNo, nil
}

func TestGatewayRegistry(t *testing.T) {
	RegisterGateway(fakeGateway{name: "alpha", link: "/pay/alpha?order="})
	RegisterGateway(fakeGateway{name: "beta", link: "/pay/beta?order="})
	t.Cleanup(func() {
		gatewayMu.Lock()
		delete(gatewayRegistry, "alpha")
		delete(gatewayRegistry, "beta")
		gatewayMu.Unlock()
	})

	names := GatewayNames()
	found := map[string]bool{}
	for _, n := range names {
		found[n] = true
	}
	if !found["alpha"] || !found["beta"] {
		t.Fatalf("expected alpha and beta registered, got %v", names)
	}
	for i := 1; i < len(names); i++ {
		if names[i-1] > names[i] {
			t.Fatalf("expected sorted gateway names, got %v", names)
		}
	}

	gateway, ok := Gateway("alpha")
	if !ok {
		t.Fatal("expected to find the alpha gateway")
	}
	link, err := gateway.PayLink(&models.Order{OrderNo: "SO-1"})
	if err != nil {
		t.Fatalf("pay link: %v", err)
	}
	if link != "/pay/alpha?order=SO-1" {
		t.Fatalf("unexpected pay link %q", link)
	}

	// Re-registering a name replaces the gateway.
	RegisterGateway(fakeGateway{name: "alpha", link: "/pay/alpha2?order="})
	gateway, _ = Gateway("alpha")
	link, _ = gateway.PayLink(&models.Order{OrderNo: "SO-1"})
	if link != "/pay/alpha2?order=SO-1" {
		t.Fatalf("expected the replacement gateway to win, got %q", link)
	}

	if _, ok := Gateway("missing"); ok {
		t.Fatal("expected an unknown gateway lookup to miss")
	}
}
