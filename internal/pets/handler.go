package pets

import (
	"math"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/voloshmiak/tamagochi/internal/brain"
	"github.com/voloshmiak/tamagochi/internal/platform/clock"
	"github.com/voloshmiak/tamagochi/internal/platform/httpx"
)

type Handler struct {
	svc   *Service
	clock clock.Clock
}

func NewHandler(svc *Service, c clock.Clock) *Handler { return &Handler{svc: svc, clock: c} }

func (h *Handler) Routes(r chi.Router) {
	r.Post("/pets", h.create)
	r.Get("/pets/me", h.get)
	r.Post("/pets/me/actions", h.perform)
	r.Get("/inventory", h.inventory)
}

type StatsView struct {
	Satiety     int `json:"satiety"`
	Mood        int `json:"mood"`
	Energy      int `json:"energy"`
	Cleanliness int `json:"cleanliness"`
}

type PetView struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Appearance string         `json:"appearance"`
	Stats      StatsView      `json:"stats"`
	Traits     brain.Traits   `json:"traits"`
	Cooldowns  map[Action]int `json:"cooldowns"`
	UpdatedAt  time.Time      `json:"updatedAt"`
}

func NewPetView(p *Pet, rules Rules, now time.Time) PetView {
	r := func(v float64) int { return int(math.Round(v)) }
	cd := make(map[Action]int, len(rules))
	for a, rule := range rules {
		cd[a] = int(math.Ceil(p.CooldownLeft(rule, now).Seconds()))
	}
	return PetView{
		ID:         p.ID,
		Name:       p.Name,
		Appearance: p.Appearance,
		Stats:      StatsView{r(p.Stats.Satiety), r(p.Stats.Mood), r(p.Stats.Energy), r(p.Stats.Cleanliness)},
		Traits:     p.Traits,
		Cooldowns:  cd,
		UpdatedAt:  p.UpdatedAt,
	}
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name       string `json:"name"`
		Appearance string `json:"appearance"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	pet, err := h.svc.Create(r.Context(), httpx.UserID(r.Context()), req.Name, req.Appearance)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, NewPetView(pet, h.svc.Rules(), h.clock.Now()))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	pet, err := h.svc.Get(r.Context(), httpx.UserID(r.Context()))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, NewPetView(pet, h.svc.Rules(), h.clock.Now()))
}

func (h *Handler) perform(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action Action `json:"action"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	res, err := h.svc.Perform(r.Context(), httpx.UserID(r.Context()), req.Action)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, struct {
		Pet      PetView        `json:"pet"`
		Points   int            `json:"points"`
		Reaction brain.Reaction `json:"reaction"`
		Used     *Item          `json:"usedItem,omitempty"`
	}{NewPetView(res.Pet, h.svc.Rules(), h.clock.Now()), res.Points, res.Reaction, res.Used})
}

func (h *Handler) inventory(w http.ResponseWriter, r *http.Request) {
	inv, err := h.svc.Inventory(r.Context(), httpx.UserID(r.Context()))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]map[string]int{"items": inv.Items})
}
