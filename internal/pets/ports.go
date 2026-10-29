package pets

import (
	"context"
	"fmt"

	"github.com/voloshmiak/tamagochi/internal/brain"
	"github.com/voloshmiak/tamagochi/internal/platform/apperr"
)

var (
	ErrPetNotFound   = fmt.Errorf("%w: pet", apperr.ErrNotFound)
	ErrPetExists     = fmt.Errorf("%w: player already has a pet", apperr.ErrConflict)
	ErrUnknownAction = fmt.Errorf("%w: unknown action", apperr.ErrInvalid)
)

type PetRepository interface {
	Create(ctx context.Context, p *Pet) error
	ByOwner(ctx context.Context, ownerID string) (*Pet, error)
	ByOwnerForUpdate(ctx context.Context, ownerID string) (*Pet, error)
	Save(ctx context.Context, p *Pet) error
}

type InventoryRepository interface {
	Get(ctx context.Context, ownerID string) (*Inventory, error)
	Save(ctx context.Context, inv *Inventory) error
}

type Decider interface {
	ReactionFor(in brain.Input, ev brain.Event) brain.Reaction
}
