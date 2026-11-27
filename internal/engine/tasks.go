package engine

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/voloshmiak/tamagochi/internal/notify"
	"github.com/voloshmiak/tamagochi/internal/queue"
)

type PingPayload struct {
	SentAt time.Time `json:"sentAt"`
}

type LowStatPayload struct {
	OwnerID string    `json:"ownerId"`
	Version time.Time `json:"version"`
}

type NotifyPayload struct {
	Kind       notify.Kind `json:"kind"`
	UserID     string      `json:"userId"`
	FromUserID string      `json:"fromUserId,omitempty"`
	OfferID    string      `json:"offerId,omitempty"`
	Status     string      `json:"status,omitempty"`
}

func PingTask(now time.Time) queue.Task {
	return queue.Task{Type: queue.TaskPing, Payload: mustJSON(PingPayload{SentAt: now})}
}

func LowStatTask(ownerID string, version, at time.Time) queue.Task {
	return queue.Task{
		ID:      fmt.Sprintf("lowstat:%s:%d:%d", ownerID, version.UnixNano(), at.Unix()),
		Type:    queue.TaskLowStat,
		Payload: mustJSON(LowStatPayload{OwnerID: ownerID, Version: version}),
		RunAt:   at,
	}
}

func NotifyTask(p NotifyPayload) queue.Task {
	return queue.Task{Type: queue.TaskNotify, Payload: mustJSON(p)}
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func decode(t queue.Task, v any) error {
	if err := json.Unmarshal(t.Payload, v); err != nil {
		return fmt.Errorf("%w: bad %s payload: %v", queue.ErrSkipRetry, t.Type, err)
	}
	return nil
}
