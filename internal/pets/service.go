package pets

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"log/slog"
	"maps"
	"slices"
	"time"

	"github.com/voloshmiak/tamagochi/internal/brain"
	"github.com/voloshmiak/tamagochi/internal/platform/clock"
	"github.com/voloshmiak/tamagochi/internal/platform/id"
	"github.com/voloshmiak/tamagochi/internal/platform/tx"
)

type Deps struct {
	Pets        PetRepository
	Inventory   InventoryRepository
	Tx          tx.Manager
	Brain       Decider
	Clock       clock.Clock
	Rules       Rules
	Catalog     Catalog
	DailyReward map[string]int
	LowStat     LowStatScheduler
	Publisher   Publisher
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
	if d.DailyReward == nil {
		d.DailyReward = DailyRewardItems
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
		inv.Put(StarterItems)
		if err := s.d.Inventory.Save(ctx, inv); err != nil {
			return err
		}
		s.afterSave(ctx, pet)
		return nil
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
			inv, err := s.d.Inventory.GetForUpdate(ctx, ownerID)
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
		s.afterSave(ctx, pet)
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

func (s *Service) ClaimDailyReward(ctx context.Context, ownerID string) (map[string]int, *Inventory, error) {
	now := s.d.Clock.Now()
	var inv *Inventory
	err := s.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		if inv, err = s.d.Inventory.GetForUpdate(ctx, ownerID); err != nil {
			return err
		}
		if err := inv.ClaimDaily(s.d.DailyReward, now); err != nil {
			return err
		}
		return s.d.Inventory.Save(ctx, inv)
	})
	if err != nil {
		return nil, nil, err
	}
	return maps.Clone(s.d.DailyReward), inv, nil
}

func (s *Service) Now() time.Time { return s.d.Clock.Now() }

func (s *Service) afterSave(ctx context.Context, pet *Pet) {
	snapshot := pet.Clone()
	tx.AfterCommit(ctx, func(ctx context.Context) {
		if s.d.LowStat != nil {
			if wait, _, ok := snapshot.TimeToThreshold(LowStatThreshold); ok {
				at := snapshot.UpdatedAt.Add(wait)
				if err := s.d.LowStat.ScheduleLowStatCheck(ctx, snapshot.OwnerID, snapshot.UpdatedAt, at); err != nil {
					slog.ErrorContext(ctx, "schedule low stat check", "owner", snapshot.OwnerID, "err", err)
				}
			}
		}
		if s.d.Publisher != nil {
			s.d.Publisher.PetUpdated(ctx, snapshot)
		}
	})
}

func (s *Service) ReserveItems(ctx context.Context, owner string, items map[string]int) error {
	if err := s.d.Catalog.Validate(items); err != nil {
		return err
	}
	return s.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		inv, err := s.d.Inventory.GetForUpdate(ctx, owner)
		if err != nil {
			return err
		}
		if err := inv.Take(items); err != nil {
			return err
		}
		return s.d.Inventory.Save(ctx, inv)
	})
}

func (s *Service) ReleaseItems(ctx context.Context, owner string, items map[string]int) error {
	if err := s.d.Catalog.Validate(items); err != nil {
		return err
	}
	return s.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		inv, err := s.d.Inventory.GetForUpdate(ctx, owner)
		if err != nil {
			return err
		}
		inv.Put(items)
		return s.d.Inventory.Save(ctx, inv)
	})
}

func (s *Service) ExchangeItems(ctx context.Context, from, to string, give, want map[string]int) error {
	if from == to {
		return fmt.Errorf("%w: cannot exchange with yourself", ErrBadItems)
	}
	if err := s.d.Catalog.Validate(give); err != nil {
		return err
	}
	if err := s.d.Catalog.Validate(want); err != nil {
		return err
	}
	return s.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		invs := make(map[string]*Inventory, 2)
		for _, owner := range slices.Sorted(slices.Values([]string{from, to})) {
			inv, err := s.d.Inventory.GetForUpdate(ctx, owner)
			if err != nil {
				return err
			}
			invs[owner] = inv
		}
		if err := invs[to].Take(want); err != nil {
			return err
		}
		invs[to].Put(give)
		invs[from].Put(want)
		for _, owner := range slices.Sorted(maps.Keys(invs)) {
			if err := s.d.Inventory.Save(ctx, invs[owner]); err != nil {
				return err
			}
		}
		return nil
	})
}

func seedOf(s string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return h.Sum64()
}
