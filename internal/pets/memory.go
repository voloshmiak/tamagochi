package pets

import (
	"context"
	"sync"
)

type MemoryPetRepository struct {
	mu      sync.RWMutex
	byOwner map[string]*Pet
}

func NewMemoryPetRepository() *MemoryPetRepository {
	return &MemoryPetRepository{byOwner: make(map[string]*Pet)}
}

func (r *MemoryPetRepository) Create(_ context.Context, p *Pet) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byOwner[p.OwnerID]; ok {
		return ErrPetExists
	}
	r.byOwner[p.OwnerID] = p.Clone()
	return nil
}

func (r *MemoryPetRepository) ByOwner(_ context.Context, ownerID string) (*Pet, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.byOwner[ownerID]
	if !ok {
		return nil, ErrPetNotFound
	}
	return p.Clone(), nil
}

func (r *MemoryPetRepository) ByOwnerForUpdate(ctx context.Context, ownerID string) (*Pet, error) {
	return r.ByOwner(ctx, ownerID)
}

func (r *MemoryPetRepository) Save(_ context.Context, p *Pet) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byOwner[p.OwnerID]; !ok {
		return ErrPetNotFound
	}
	r.byOwner[p.OwnerID] = p.Clone()
	return nil
}

type MemoryInventoryRepository struct {
	mu      sync.RWMutex
	byOwner map[string]*Inventory
}

func NewMemoryInventoryRepository() *MemoryInventoryRepository {
	return &MemoryInventoryRepository{byOwner: make(map[string]*Inventory)}
}

func (r *MemoryInventoryRepository) Get(_ context.Context, ownerID string) (*Inventory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if inv, ok := r.byOwner[ownerID]; ok {
		return inv.Clone(), nil
	}
	return NewInventory(ownerID), nil
}

func (r *MemoryInventoryRepository) Save(_ context.Context, inv *Inventory) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byOwner[inv.OwnerID] = inv.Clone()
	return nil
}
