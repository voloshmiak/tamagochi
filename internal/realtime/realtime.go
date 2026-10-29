package realtime

import "context"

type Broker interface {
	Publish(ctx context.Context, topic string, msg []byte) error
	Subscribe(ctx context.Context, topic string) (<-chan []byte, func())
}

func PetTopic(ownerID string) string { return "pet:" + ownerID }

type Message struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

const (
	TypePetUpdated = "pet.updated"
	TypeReplica    = "pet.replica"
)

type AccessChecker interface {
	CanView(ctx context.Context, viewer, owner string) (bool, error)
}
