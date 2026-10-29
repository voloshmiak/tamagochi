package brain

import (
	"cmp"
	"maps"
	"math/rand/v2"
	"slices"
)

type Trait string

const (
	TraitPlayful Trait = "playful"
	TraitLazy    Trait = "lazy"
	TraitGlutton Trait = "glutton"
	TraitShy     Trait = "shy"
)

var AllTraits = []Trait{TraitGlutton, TraitLazy, TraitPlayful, TraitShy}

type Traits map[Trait]float64

func GenerateTraits(seed uint64) Traits {
	rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	t := make(Traits, len(AllTraits))
	for _, tr := range AllTraits {
		t[tr] = float64(rng.IntN(101)) / 100
	}
	return t
}

func (t Traits) Weight(tr Trait) float64 { return t[tr] }

func (t Traits) Dominant() Trait {
	keys := slices.Sorted(maps.Keys(t))
	if len(keys) == 0 {
		return ""
	}
	best := keys[0]
	for _, k := range keys[1:] {
		if cmp.Compare(t[k], t[best]) > 0 {
			best = k
		}
	}
	return best
}
