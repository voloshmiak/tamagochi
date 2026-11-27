package pets

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/voloshmiak/tamagochi/internal/brain"
	"github.com/voloshmiak/tamagochi/internal/platform/pg"
	"github.com/voloshmiak/tamagochi/internal/platform/pg/pgdb"
)

type PgPetRepository struct {
	Pool *pgxpool.Pool
}

func (r PgPetRepository) Create(ctx context.Context, p *Pet) error {
	lastActions, err := json.Marshal(p.LastActions)
	if err != nil {
		return err
	}
	q := pg.Queries(ctx, r.Pool)
	err = q.CreatePet(ctx, pgdb.CreatePetParams{
		ID:          p.ID,
		OwnerID:     p.OwnerID,
		Name:        p.Name,
		Appearance:  p.Appearance,
		Satiety:     p.Stats.Satiety,
		Mood:        p.Stats.Mood,
		Energy:      p.Stats.Energy,
		Cleanliness: p.Stats.Cleanliness,
		BornAt:      p.BornAt,
		UpdatedAt:   p.UpdatedAt,
		LastActions: lastActions,
	})
	if pg.UniqueViolation(err) != "" {
		return ErrPetExists
	}
	if err != nil {
		return fmt.Errorf("create pet: %w", err)
	}
	traits := slices.Sorted(maps.Keys(p.Traits))
	weights := make([]float64, len(traits))
	names := make([]string, len(traits))
	for i, t := range traits {
		names[i], weights[i] = string(t), p.Traits[t]
	}
	if err := q.InsertPetTraits(ctx, pgdb.InsertPetTraitsParams{PetID: p.ID, Traits: names, Weights: weights}); err != nil {
		return fmt.Errorf("create pet traits: %w", err)
	}
	return nil
}

func (r PgPetRepository) ByOwner(ctx context.Context, ownerID string) (*Pet, error) {
	return r.load(ctx, ownerID, false)
}

func (r PgPetRepository) ByOwnerForUpdate(ctx context.Context, ownerID string) (*Pet, error) {
	return r.load(ctx, ownerID, true)
}

func (r PgPetRepository) load(ctx context.Context, ownerID string, forUpdate bool) (*Pet, error) {
	q := pg.Queries(ctx, r.Pool)
	var row pgdb.Pet
	var err error
	if forUpdate {
		row, err = q.PetByOwnerForUpdate(ctx, ownerID)
	} else {
		row, err = q.PetByOwner(ctx, ownerID)
	}
	if pg.IsNoRows(err) {
		return nil, ErrPetNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("load pet: %w", err)
	}
	traitRows, err := q.PetTraits(ctx, row.ID)
	if err != nil {
		return nil, fmt.Errorf("load pet traits: %w", err)
	}
	traits := make(brain.Traits, len(traitRows))
	for _, t := range traitRows {
		traits[brain.Trait(t.Trait)] = t.Weight
	}
	lastActions := make(map[Action]time.Time)
	if err := json.Unmarshal(row.LastActions, &lastActions); err != nil {
		return nil, fmt.Errorf("decode last actions of pet %s: %w", row.ID, err)
	}
	return &Pet{
		ID:          row.ID,
		OwnerID:     row.OwnerID,
		Name:        row.Name,
		Appearance:  row.Appearance,
		Stats:       Stats{Satiety: row.Satiety, Mood: row.Mood, Energy: row.Energy, Cleanliness: row.Cleanliness},
		Traits:      traits,
		BornAt:      row.BornAt,
		UpdatedAt:   row.UpdatedAt,
		LastActions: lastActions,
	}, nil
}

func (r PgPetRepository) Save(ctx context.Context, p *Pet) error {
	lastActions, err := json.Marshal(p.LastActions)
	if err != nil {
		return err
	}
	n, err := pg.Queries(ctx, r.Pool).UpdatePet(ctx, pgdb.UpdatePetParams{
		ID:          p.ID,
		Name:        p.Name,
		Appearance:  p.Appearance,
		Satiety:     p.Stats.Satiety,
		Mood:        p.Stats.Mood,
		Energy:      p.Stats.Energy,
		Cleanliness: p.Stats.Cleanliness,
		UpdatedAt:   p.UpdatedAt,
		LastActions: lastActions,
	})
	if err != nil {
		return fmt.Errorf("save pet: %w", err)
	}
	if n == 0 {
		return ErrPetNotFound
	}
	return nil
}

type PgInventoryRepository struct {
	Pool *pgxpool.Pool
}

func (r PgInventoryRepository) Get(ctx context.Context, ownerID string) (*Inventory, error) {
	q := pg.Queries(ctx, r.Pool)
	inv := NewInventory(ownerID)
	switch row, err := q.InventoryByOwner(ctx, ownerID); {
	case pg.IsNoRows(err):
		return inv, nil
	case err != nil:
		return nil, fmt.Errorf("load inventory: %w", err)
	default:
		if row.DailyRewardAt != nil {
			inv.DailyRewardAt = *row.DailyRewardAt
		}
	}
	return inv, r.loadItems(ctx, q, inv)
}

func (r PgInventoryRepository) GetForUpdate(ctx context.Context, ownerID string) (*Inventory, error) {
	q := pg.Queries(ctx, r.Pool)
	if err := q.EnsureInventory(ctx, ownerID); err != nil {
		return nil, fmt.Errorf("ensure inventory: %w", err)
	}
	row, err := q.InventoryByOwnerForUpdate(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("lock inventory: %w", err)
	}
	inv := NewInventory(ownerID)
	if row.DailyRewardAt != nil {
		inv.DailyRewardAt = *row.DailyRewardAt
	}
	return inv, r.loadItems(ctx, q, inv)
}

func (r PgInventoryRepository) loadItems(ctx context.Context, q *pgdb.Queries, inv *Inventory) error {
	items, err := q.InventoryItems(ctx, inv.OwnerID)
	if err != nil {
		return fmt.Errorf("load inventory items: %w", err)
	}
	for _, it := range items {
		inv.Items[it.ItemID] = int(it.Quantity)
	}
	return nil
}

func (r PgInventoryRepository) Save(ctx context.Context, inv *Inventory) error {
	q := pg.Queries(ctx, r.Pool)
	if err := q.EnsureInventory(ctx, inv.OwnerID); err != nil {
		return fmt.Errorf("ensure inventory: %w", err)
	}
	var rewardAt *time.Time
	if !inv.DailyRewardAt.IsZero() {
		rewardAt = &inv.DailyRewardAt
	}
	if err := q.SetDailyRewardAt(ctx, pgdb.SetDailyRewardAtParams{OwnerID: inv.OwnerID, DailyRewardAt: rewardAt}); err != nil {
		return fmt.Errorf("save daily reward: %w", err)
	}
	if err := q.DeleteInventoryItems(ctx, inv.OwnerID); err != nil {
		return fmt.Errorf("clear inventory items: %w", err)
	}
	counts := inv.Counts()
	ids := slices.Sorted(maps.Keys(counts))
	qty := make([]int32, len(ids))
	for i, itemID := range ids {
		qty[i] = int32(counts[itemID])
	}
	err := q.InsertInventoryItems(ctx, pgdb.InsertInventoryItemsParams{OwnerID: inv.OwnerID, ItemIds: ids, Quantities: qty})
	if err != nil {
		return fmt.Errorf("save inventory items: %w", err)
	}
	return nil
}
