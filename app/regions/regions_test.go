package regions

import "testing"

func TestProvincesLoaded(t *testing.T) {
	provinces, err := Provinces()
	if err != nil {
		t.Fatalf("load provinces: %v", err)
	}
	if len(provinces) != 31 {
		t.Fatalf("expected 31 provinces, got %d", len(provinces))
	}
}

func TestChildren(t *testing.T) {
	shanghai, err := Children("31")
	if err != nil {
		t.Fatalf("children of 31: %v", err)
	}
	if len(shanghai) == 0 || shanghai[0].Name != "市辖区" {
		t.Fatalf("expected Shanghai's 市辖区 child, got %#v", shanghai)
	}

	districts, err := Children(shanghai[0].Code)
	if err != nil {
		t.Fatalf("children of city: %v", err)
	}
	if len(districts) == 0 {
		t.Fatal("expected districts under the city")
	}

	if kids, _ := Children("9999"); kids != nil {
		t.Fatalf("expected no children for an unknown code, got %#v", kids)
	}
}

func TestChildrenForNames(t *testing.T) {
	cities, districts, err := ChildrenForNames("上海市", "市辖区")
	if err != nil {
		t.Fatalf("resolve by names: %v", err)
	}
	if len(cities) == 0 {
		t.Fatal("expected cities for 上海市")
	}
	if len(districts) == 0 {
		t.Fatal("expected districts for 市辖区")
	}

	cities, districts, err = ChildrenForNames("不存在", "也不存在")
	if err != nil {
		t.Fatalf("unknown names should not error: %v", err)
	}
	if cities != nil || districts != nil {
		t.Fatalf("expected empty lists for unknown names, got %#v / %#v", cities, districts)
	}
}
