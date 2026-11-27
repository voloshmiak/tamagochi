package notify

import (
	"maps"
	"net/http"
	"slices"

	"github.com/voloshmiak/tamagochi/internal/api/oapi"
	"github.com/voloshmiak/tamagochi/internal/platform/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) GetPushKey(w http.ResponseWriter, r *http.Request) {
	key := h.svc.PublicKey()
	httpx.JSON(w, http.StatusOK, oapi.PushKey{PublicKey: key, Enabled: key != ""})
}

func (h *Handler) SubscribePush(w http.ResponseWriter, r *http.Request) {
	var req oapi.PushSubscription
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	err := h.svc.Subscribe(r.Context(), Subscription{
		UserID:   httpx.UserID(r.Context()),
		Endpoint: req.Endpoint,
		P256dh:   req.Keys.P256dh,
		Auth:     req.Keys.Auth,
	})
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) UnsubscribePush(w http.ResponseWriter, r *http.Request) {
	var req oapi.PushUnsubscribe
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if err := h.svc.Unsubscribe(r.Context(), httpx.UserID(r.Context()), req.Endpoint); err != nil {
		httpx.Error(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetNotificationPreferences(w http.ResponseWriter, r *http.Request) {
	p, err := h.svc.Preferences(r.Context(), httpx.UserID(r.Context()))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, preferencesView(p))
}

func (h *Handler) SetNotificationPreferences(w http.ResponseWriter, r *http.Request) {
	var req oapi.NotificationPreferences
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	from, err := ParseClock(req.QuietHours.From)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	to, err := ParseClock(req.QuietHours.To)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	p := Preferences{
		Disabled:     make(map[Kind]bool, len(req.Disabled)),
		QuietEnabled: req.QuietHours.Enabled,
		QuietFrom:    from,
		QuietTo:      to,
		Timezone:     req.Timezone,
		Lang:         string(req.Lang),
	}
	for _, k := range req.Disabled {
		p.Disabled[Kind(k)] = true
	}
	saved, err := h.svc.SetPreferences(r.Context(), httpx.UserID(r.Context()), p)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, preferencesView(saved))
}

func preferencesView(p Preferences) oapi.NotificationPreferences {
	disabled := make([]oapi.NotificationKind, 0, len(p.Disabled))
	for _, k := range slices.Sorted(maps.Keys(p.Disabled)) {
		if p.Disabled[k] {
			disabled = append(disabled, oapi.NotificationKind(k))
		}
	}
	return oapi.NotificationPreferences{
		Disabled: disabled,
		QuietHours: oapi.QuietHours{
			Enabled: p.QuietEnabled,
			From:    FormatClock(p.QuietFrom),
			To:      FormatClock(p.QuietTo),
		},
		Timezone: p.Timezone,
		Lang:     oapi.NotificationPreferencesLang(p.Lang),
	}
}
