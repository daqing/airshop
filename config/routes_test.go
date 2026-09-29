package config

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRoutesRegistersCoreEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	Routes(r)

	registered := map[string]bool{}
	for _, route := range r.Routes() {
		registered[route.Method+" "+route.Path] = true
	}

	expected := []string{
		"GET /",
		"GET /products",
		"GET /products/:slug",
		"GET /signin",
		"POST /signin/code",
		"POST /signin",
		"POST /signout",
		"GET /account",
		"GET /cart",
		"POST /cart/add",
		"POST /cart/items/:id",
		"POST /cart/items/:id/delete",
		"POST /cart/clear",
		"GET /checkout",
		"POST /checkout",
		"GET /orders/:orderNo",
		"GET /orders",
		"GET /account/addresses",
		"GET /account/addresses/new",
		"POST /account/addresses",
		"GET /account/addresses/:id/edit",
		"POST /account/addresses/:id/update",
		"POST /account/addresses/:id/delete",
		"POST /account/addresses/:id/default",
		"GET /admin",
		"GET /admin/categories",
		"GET /admin/categories/new",
		"POST /admin/categories",
		"GET /admin/categories/:id/edit",
		"POST /admin/categories/:id/update",
		"POST /admin/categories/:id/delete",
		"GET /admin/products",
		"GET /admin/products/new",
		"POST /admin/products",
		"GET /admin/products/:id/edit",
		"POST /admin/products/:id/update",
		"POST /admin/products/:id/activate",
		"POST /admin/products/:id/deactivate",
		"GET /health",
		"GET /openapi.json",
		"GET /ws",
		"POST /ws/publish",
		"POST /api/v1/storage",
		"GET /api/v1/storage/*key",
		"DELETE /api/v1/storage/*key",
	}

	for _, route := range expected {
		if !registered[route] {
			t.Fatalf("expected route %s to be registered, got %#v", route, registered)
		}
	}
}

func TestNoRouteRenders404Page(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	Routes(r)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/definitely-missing", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}

	body := w.Body.String()
	for _, want := range []string{"404", "does not exist", "aw-storefront-header"} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected 404 body to contain %q, got %q", want, body)
		}
	}
}

func TestForbiddenHandlerRenders403Page(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/private", ForbiddenHandler)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", w.Code)
	}

	body := w.Body.String()
	for _, want := range []string{"403", "permission", "aw-storefront-header"} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected 403 body to contain %q, got %q", want, body)
		}
	}
}
