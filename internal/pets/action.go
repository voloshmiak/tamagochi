package pets

import "time"

type Action string

const (
	ActionFeed  Action = "feed"
	ActionPlay  Action = "play"
	ActionSleep Action = "sleep"
	ActionWash  Action = "wash"
)

type ActionRule interface {
	Action() Action
	Cooldown() time.Duration
	RequiredItem() ItemKind
	Points() int
	Effect(s Stats) Stats
}

type StatRule struct {
	Kind  Action
	Wait  time.Duration
	Item  ItemKind
	Score int
	Delta Stats
}

func (r StatRule) Action() Action          { return r.Kind }
func (r StatRule) Cooldown() time.Duration { return r.Wait }
func (r StatRule) RequiredItem() ItemKind  { return r.Item }
func (r StatRule) Points() int             { return r.Score }
func (r StatRule) Effect(s Stats) Stats    { return s.add(r.Delta) }

type Rules map[Action]ActionRule

func DefaultRules() Rules {
	rules := []ActionRule{
		StatRule{Kind: ActionFeed, Wait: 5 * time.Minute, Item: ItemFood, Score: 10,
			Delta: Stats{Satiety: 30, Mood: 5}},
		StatRule{Kind: ActionPlay, Wait: 3 * time.Minute, Score: 10,
			Delta: Stats{Mood: 25, Energy: -10, Satiety: -5, Cleanliness: -5}},
		StatRule{Kind: ActionSleep, Wait: 30 * time.Minute, Score: 5,
			Delta: Stats{Energy: 50, Satiety: -5}},
		StatRule{Kind: ActionWash, Wait: 10 * time.Minute, Item: ItemSoap, Score: 5,
			Delta: Stats{Cleanliness: 40, Mood: -5}},
	}
	out := make(Rules, len(rules))
	for _, r := range rules {
		out[r.Action()] = r
	}
	return out
}
