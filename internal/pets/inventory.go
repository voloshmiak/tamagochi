package pets

import (
	"fmt"
	"maps"
	"slices"
	"time"

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

// Catalog lists the items that exist in the game. The same items are seeded
// into the items table by the database migrations.
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

// DailyRewardItems is what the player gets once per calendar day (FR-PET-06).
var DailyRewardItems = map[string]int{"apple": 3, "fish": 1, "soap": 1}

type Inventory struct {
	OwnerID string
	Items   map[string]int
	// DailyRewardAt is when the daily reward was last claimed; zero if never.
	DailyRewardAt time.Time
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

// Counts returns the items the player actually has, without zero entries.
func (inv *Inventory) Counts() map[string]int {
	out := make(map[string]int, len(inv.Items))
	for itemID, n := range inv.Items {
		if n > 0 {
			out[itemID] = n
		}
	}
	return out
}

// ErrNoItem is returned when an action needs an item the player does not have.
var ErrNoItem = fmt.Errorf("%w: no item", apperr.ErrConflict)

func (inv *Inventory) ConsumeKind(kind ItemKind, catalog Catalog) (Item, error) {
	for _, itemID := range slices.Sorted(maps.Keys(inv.Items)) {
		item, ok := catalog[itemID]
		if !ok || item.Kind != kind || inv.Items[itemID] <= 0 {
			continue
		}
		inv.Items[itemID]--
		return item, nil
	}
	return Item{}, fmt.Errorf("%w: no %s in inventory", ErrNoItem, kind)
}

// DailyRewardIn returns how long until the daily reward can be claimed again;
// zero means it is available now. The reward resets at midnight UTC.
func (inv *Inventory) DailyRewardIn(now time.Time) time.Duration {
	if inv.DailyRewardAt.IsZero() {
		return 0
	}
	next := startOfDay(inv.DailyRewardAt).AddDate(0, 0, 1)
	return max(next.Sub(now), 0)
}

// ClaimDaily adds the daily reward to the inventory. It fails with a
// *RewardTooEarlyError if the reward was already claimed today.
func (inv *Inventory) ClaimDaily(reward map[string]int, now time.Time) error {
	if left := inv.DailyRewardIn(now); left > 0 {
		return &RewardTooEarlyError{Left: left}
	}
	for itemID, n := range reward {
		inv.Add(itemID, n)
	}
	inv.DailyRewardAt = now
	return nil
}

func startOfDay(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

type RewardTooEarlyError struct {
	Left time.Duration
}

func (e *RewardTooEarlyError) Error() string {
	return fmt.Sprintf("daily reward already claimed; next one in %s", e.Left.Round(time.Second))
}

func (e *RewardTooEarlyError) Unwrap() error { return apperr.ErrTooEarly }

func (e *RewardTooEarlyError) RetryAfter() time.Duration { return e.Left }
