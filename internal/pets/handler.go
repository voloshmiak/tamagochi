package pets

import (
	"math"
	"net/http"
	"time"

	"github.com/voloshmiak/tamagochi/internal/api/oapi"
	"github.com/voloshmiak/tamagochi/internal/brain"
	"github.com/voloshmiak/tamagochi/internal/platform/clock"
	"github.com/voloshmiak/tamagochi/internal/platform/httpx"
)

// Handler implements the pets operations of oapi.ServerInterface.
type Handler struct {
	svc   *Service
	clock clock.Clock
}

func NewHandler(svc *Service, c clock.Clock) *Handler { return &Handler{svc: svc, clock: c} }

func NewPetView(p *Pet, rules Rules, now time.Time) oapi.Pet {
	r := func(v float64) int { return int(math.Round(v)) }
	cd := make(map[string]int, len(rules))
	for a, rule := range rules {
		cd[string(a)] = int(math.Ceil(p.CooldownLeft(rule, now).Seconds()))
	}
	traits := make(map[string]float32, len(p.Traits))
	for t, w := range p.Traits {
		traits[string(t)] = float32(w)
	}
	return oapi.Pet{
		Id:         p.ID,
		Name:       p.Name,
		Appearance: p.Appearance,
		Stage:      oapi.Stage(p.Stage(now)),
		Stats: oapi.Stats{
			Satiety:     r(p.Stats.Satiety),
			Mood:        r(p.Stats.Mood),
			Energy:      r(p.Stats.Energy),
			Cleanliness: r(p.Stats.Cleanliness),
		},
		Traits:    traits,
		Cooldowns: cd,
		BornAt:    p.BornAt,
		UpdatedAt: p.UpdatedAt,
	}
}

func newInventoryView(inv *Inventory, now time.Time) oapi.Inventory {
	return oapi.Inventory{
		Items:         inv.Counts(),
		DailyRewardIn: int(math.Ceil(inv.DailyRewardIn(now).Seconds())),
	}
}

func reactionView(r brain.Reaction) oapi.Reaction {
	return oapi.Reaction{
		Event:    string(r.Event),
		Emotion:  oapi.ReactionEmotion(r.Emotion),
		Behavior: string(r.Behavior),
		Trait:    string(r.Trait),
	}
}

func (h *Handler) CreatePet(w http.ResponseWriter, r *http.Request) {
	var req oapi.CreatePetJSONRequestBody
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

func (h *Handler) GetMyPet(w http.ResponseWriter, r *http.Request) {
	pet, err := h.svc.Get(r.Context(), httpx.UserID(r.Context()))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, NewPetView(pet, h.svc.Rules(), h.clock.Now()))
}

func (h *Handler) PerformAction(w http.ResponseWriter, r *http.Request) {
	var req oapi.PerformActionJSONRequestBody
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	res, err := h.svc.Perform(r.Context(), httpx.UserID(r.Context()), Action(req.Action))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	out := oapi.ActionResult{
		Pet:      NewPetView(res.Pet, h.svc.Rules(), h.clock.Now()),
		Points:   res.Points,
		Reaction: reactionView(res.Reaction),
	}
	if res.Used != nil {
		out.UsedItem = &oapi.Item{Id: res.Used.ID, Name: res.Used.Name, Kind: oapi.ItemKind(res.Used.Kind)}
	}
	httpx.JSON(w, http.StatusOK, out)
}

func (h *Handler) GetInventory(w http.ResponseWriter, r *http.Request) {
	inv, err := h.svc.Inventory(r.Context(), httpx.UserID(r.Context()))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, newInventoryView(inv, h.clock.Now()))
}

func (h *Handler) ClaimDailyReward(w http.ResponseWriter, r *http.Request) {
	granted, inv, err := h.svc.ClaimDailyReward(r.Context(), httpx.UserID(r.Context()))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, oapi.DailyReward{
		Granted:   granted,
		Inventory: newInventoryView(inv, h.clock.Now()),
	})
}
