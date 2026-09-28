package layouts

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func renderToString(t *testing.T, c templ.Component) string {
	t.Helper()

	var buf bytes.Buffer
	if err := c.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render failed: %v", err)
	}
	return buf.String()
}

func TestStorefrontLayoutRendersShell(t *testing.T) {
	html := renderToString(t, Storefront("Test Page", "Test description"))

	for _, marker := range []string{
		"<title>Test Page</title>",
		`name="description" content="Test description"`,
		"aw-storefront-header",
		"aw-storefront-brand",
		">AirShop</a>",
		`href="/products"`,
		"aw-storefront-main",
		"aw-storefront-footer",
		"© 2026 AirShop",
		"--sf-bg",
		"prefers-color-scheme: light",
	} {
		if !strings.Contains(html, marker) {
			t.Fatalf("expected storefront HTML to contain %q, got:\n%s", marker, html)
		}
	}
}

func TestAdminLayoutRendersShell(t *testing.T) {
	html := renderToString(t, Admin("Admin Page"))

	for _, marker := range []string{
		"<title>Admin Page</title>",
		"aw-admin-topbar",
		"aw-admin-brand",
		">AirShop Admin</a>",
		`href="/admin/products"`,
		"aw-admin-main",
		"--ad-bg",
		"prefers-color-scheme: dark",
	} {
		if !strings.Contains(html, marker) {
			t.Fatalf("expected admin HTML to contain %q, got:\n%s", marker, html)
		}
	}
}
