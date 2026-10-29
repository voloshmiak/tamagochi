package pets

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"

	"github.com/voloshmiak/tamagochi/internal/brain"
	"github.com/voloshmiak/tamagochi/internal/platform/clock"
	"github.com/voloshmiak/tamagochi/internal/platform/id"
	"github.com/voloshmiak/tamagochi/internal/platform/tx"
)

type Deps struct {
	Pets      PetRepository
	Inventory InventoryRepository
	Tx        tx.Manager
	Brain     Decider
	Clock     clock.Clock
	Rules     Rules
	Catalog   Catalog
}

type Service struct {
	d Deps
}

func NewService(d Deps) *Service {
	if d.Rules == nil {
		d.Rules = DefaultRules()
	}
	if d.Catalog == nil {
		d.Catalog = DefaultCatalog()
	}
	return &Service{d: d}
}

func (s *Service) Rules() Rules { return s.d.Rules }

func (s *Service) Create(ctx context.Context, ownerID, name, appearance string) (*Pet, error) {
	now := s.d.Clock.Now()
	petID := id.New()
	pet, err := NewPet(petID, ownerID, name, appearance, brain.GenerateTraits(seedOf(petID)), now)
	if err != nil {
		return nil, err
	}
	err = s.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		switch _, err := s.d.Pets.ByOwnerForUpdate(ctx, ownerID); {
		case err == nil:
			return ErrPetExists
		case !errors.Is(err, ErrPetNotFound):
			return err
		}
		if err := s.d.Pets.Create(ctx, pet); err != nil {
			return err
		}
		inv := NewInventory(ownerID)
		for itemID, n := range StarterItems {
			inv.Add(itemID, n)
		}
		return s.d.Inventory.Save(ctx, inv)
	})
	if err != nil {
		return nil, err
	}
	return pet, nil
}

func (s *Service) Get(ctx context.Context, ownerID string) (*Pet, error) {
	pet, err := s.d.Pets.ByOwner(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	pet.Recompute(s.d.Clock.Now())
	return pet, nil
}

type ActionResult struct {
	Pet      *Pet
	Points   int
	Reaction brain.Reaction
	Used     *Item
}

func (s *Service) Perform(ctx context.Context, ownerID string, action Action) (ActionResult, error) {
	rule, ok := s.d.Rules[action]
	if !ok {
		return ActionResult{}, fmt.Errorf("%w: %q", ErrUnknownAction, action)
	}
	now := s.d.Clock.Now()
	var res ActionResult
	err := s.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		pet, err := s.d.Pets.ByOwnerForUpdate(ctx, ownerID)
		if err != nil {
			return err
		}
		pet.Recompute(now)
		if left := pet.CooldownLeft(rule, now); left > 0 {
			return &CooldownError{Action: action, Left: left}
		}
		if kind := rule.RequiredItem(); kind != "" {
			inv, err := s.d.Inventory.Get(ctx, ownerID)
			if err != nil {
				return err
			}
			item, err := inv.ConsumeKind(kind, s.d.Catalog)
			if err != nil {
				return err
			}
			if err := s.d.Inventory.Save(ctx, inv); err != nil {
				return err
			}
			res.Used = &item
		}
		if err := pet.Apply(rule, now); err != nil {
			return err
		}
		if err := s.d.Pets.Save(ctx, pet); err != nil {
			return err
		}
		res.Pet, res.Points = pet, rule.Points()
		return nil
	})
	if err != nil {
		return ActionResult{}, err
	}
	res.Reaction = s.d.Brain.ReactionFor(res.Pet.BrainInput(), brain.Event(action))
	return res, nil
}

func (s *Service) Inventory(ctx context.Context, ownerID string) (*Inventory, error) {
	return s.d.Inventory.Get(ctx, ownerID)
}

func seedOf(s string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return h.Sum64()
}
