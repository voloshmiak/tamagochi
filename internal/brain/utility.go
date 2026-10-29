package brain

import (
	"encoding/binary"
	"hash/fnv"
	"maps"
	"math"
	"math/rand/v2"
	"slices"
)

type Event string

type Emotion string

const (
	EmotionHappy  Emotion = "happy"
	EmotionCalm   Emotion = "calm"
	EmotionGrumpy Emotion = "grumpy"
	EmotionShy    Emotion = "shy"
)

type Reaction struct {
	Event    Event    `json:"event"`
	Emotion  Emotion  `json:"emotion"`
	Behavior Behavior `json:"behavior"`
	Trait    Trait    `json:"trait"`
}

type UtilityBrain struct {
	Seed           uint64
	Jitter         float64
	IdleThreshold  float64
	considerations []Consideration
}

func NewUtilityBrain(seed uint64, cs ...Consideration) *UtilityBrain {
	if len(cs) == 0 {
		cs = DefaultConsiderations()
	}
	return &UtilityBrain{Seed: seed, Jitter: 0.05, IdleThreshold: 0.25, considerations: cs}
}

func (b *UtilityBrain) Decide(in Input) Behavior {
	rng := b.rngFor(in)
	best, bestScore := BehaviorIdle, b.IdleThreshold
	for _, c := range b.considerations {
		score := c.Score(in) + rng.Float64()*b.Jitter
		if score > bestScore {
			best, bestScore = c.Behavior(), score
		}
	}
	return best
}

func (b *UtilityBrain) ReactionFor(in Input, ev Event) Reaction {
	dominant := in.Traits.Dominant()
	emotion := EmotionCalm
	switch {
	case in.Mood >= 60:
		emotion = EmotionHappy
	case in.Mood < 30:
		emotion = EmotionGrumpy
	}
	if dominant == TraitShy && emotion == EmotionCalm {
		emotion = EmotionShy
	}
	return Reaction{Event: ev, Emotion: emotion, Behavior: b.Decide(in), Trait: dominant}
}

func (b *UtilityBrain) rngFor(in Input) *rand.Rand {
	h := fnv.New64a()
	var buf [8]byte
	write := func(f float64) {
		binary.LittleEndian.PutUint64(buf[:], math.Float64bits(f))
		_, _ = h.Write(buf[:])
	}
	write(in.Satiety)
	write(in.Mood)
	write(in.Energy)
	write(in.Cleanliness)
	for _, tr := range slices.Sorted(maps.Keys(in.Traits)) {
		_, _ = h.Write([]byte(tr))
		write(in.Traits[tr])
	}
	return rand.New(rand.NewPCG(b.Seed, h.Sum64()))
}
