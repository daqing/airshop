package models

import (
	"reflect"
	"testing"
)

func TestCommerceModelsRegisteredForREPL(t *testing.T) {
	expected := map[string]reflect.Type{
		"Category":       reflect.TypeOf(Category{}),
		"Product":        reflect.TypeOf(Product{}),
		"ProductVariant": reflect.TypeOf(ProductVariant{}),
	}

	namespace := REPLNamespace()
	for name, want := range expected {
		got, ok := namespace[name]
		if !ok {
			t.Fatalf("expected %s in REPL namespace, got %#v", name, namespace)
		}
		if got != want {
			t.Fatalf("expected %s to be %v, got %v", name, want, got)
		}
	}
}

func TestCommerceModelTableNames(t *testing.T) {
	cases := map[string]string{
		reflect.TypeOf(Category{}).Name():       "categories",
		reflect.TypeOf(Product{}).Name():        "products",
		reflect.TypeOf(ProductVariant{}).Name(): "product_variants",
	}

	for modelName, want := range cases {
		var model interface{ TableName() string }
		switch modelName {
		case "Category":
			model = Category{}
		case "Product":
			model = Product{}
		case "ProductVariant":
			model = ProductVariant{}
		}

		if got := model.TableName(); got != want {
			t.Fatalf("expected %s table name %q, got %q", modelName, want, got)
		}
	}
}
