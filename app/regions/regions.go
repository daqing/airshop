// Package regions embeds the China administrative divisions dataset
// (province / city / district, Ministry of Civil Affairs codes) used by the
// address forms. Dataset: modood/Administrative-divisions-of-China (MIT),
// covering the 31 mainland provinces; Hong Kong, Macao and Taiwan are not
// included yet.
package regions

import (
	_ "embed"
	"encoding/json"
	"sync"
)

//go:embed pca-code.json
var raw []byte

// Region is one node of the three-level division tree.
type Region struct {
	Code     string   `json:"code"`
	Name     string   `json:"name"`
	Children []Region `json:"children,omitempty"`
}

var (
	once    sync.Once
	tree    []Region
	byCode  map[string]Region
	loadErr error
)

func load() {
	once.Do(func() {
		if err := json.Unmarshal(raw, &tree); err != nil {
			loadErr = err
			return
		}
		byCode = make(map[string]Region)
		var index func(nodes []Region)
		index = func(nodes []Region) {
			for _, n := range nodes {
				byCode[n.Code] = n
				index(n.Children)
			}
		}
		index(tree)
	})
}

// Provinces returns the top-level provinces.
func Provinces() ([]Region, error) {
	load()
	return tree, loadErr
}

// Children returns the divisions directly under the given code; provinces
// have cities, cities have districts, districts have none.
func Children(code string) ([]Region, error) {
	load()
	if loadErr != nil {
		return nil, loadErr
	}
	if node, ok := byCode[code]; ok {
		return node.Children, nil
	}
	return nil, nil
}

// ChildrenForNames resolves the city and district option lists for stored
// region names, so an edit form can pre-select them server-side.
func ChildrenForNames(provinceName, cityName string) (cities, districts []Region, err error) {
	load()
	if loadErr != nil {
		return nil, nil, loadErr
	}

	var province *Region
	for i := range tree {
		if tree[i].Name == provinceName {
			province = &tree[i]
			break
		}
	}
	if province == nil {
		return nil, nil, nil
	}
	cities = province.Children

	var city *Region
	for i := range cities {
		if cities[i].Name == cityName {
			city = &cities[i]
			break
		}
	}
	if city != nil {
		districts = city.Children
	}
	return cities, districts, nil
}
