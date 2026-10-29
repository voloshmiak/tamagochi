package pets

import (
	"fmt"
	"maps"
	"slices"

	"github.com/voloshmiak/tamagochi/internal/platform/apperr"
)

type ItemKind string

const (
	ItemFood ItemKind = "food"
	ItemToy  ItemKind = "toy"
	ItemSoap ItemKind = "soap"
)

type Item struct {
	ID   string   `json:"id"`
	Name string   `json:"name"`
	Kind ItemKind `json:"kind"`
}

type Catalog map[string]Item

func DefaultCatalog() Catalog {
	return Catalog{
		"apple": {ID: "apple", Name: "Apple", Kind: ItemFood},
		"fish":  {ID: "fish", Name: "Fish", Kind: ItemFood},
		"ball":  {ID: "ball", Name: "Ball", Kind: ItemToy},
		"soap":  {ID: "soap", Name: "Soap", Kind: ItemSoap},
	}
}

var StarterItems = map[string]int{"apple": 5, "fish": 2, "soap": 2, "ball": 1}

type Inventory struct {
	OwnerID string
	Items   map[string]int
}

func NewInventory(ownerID string) *Inventory {
	return &Inventory{OwnerID: ownerID, Items: make(map[string]int)}
}

func (inv *Inventory) Clone() *Inventory {
	c := *inv
	c.Items = maps.Clone(inv.Items)
	return &c
}

func (inv *Inventory) Add(itemID string, n int) { inv.Items[itemID] += n }

func (inv *Inventory) ConsumeKind(kind ItemKind, catalog Catalog) (Item, error) {
	for _, itemID := range slices.Sorted(maps.Keys(inv.Items)) {
		item, ok := catalog[itemID]
		if !ok || item.Kind != kind || inv.Items[itemID] <= 0 {
			continue
		}
		inv.Items[itemID]--
		return item, nil
	}
	return Item{}, fmt.Errorf("%w: no %s in inventory", apperr.ErrConflict, kind)
}
