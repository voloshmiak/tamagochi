package brain

type Behavior string

const (
	BehaviorIdle  Behavior = "idle"
	BehaviorEat   Behavior = "want_eat"
	BehaviorPlay  Behavior = "want_play"
	BehaviorSleep Behavior = "want_sleep"
	BehaviorWash  Behavior = "want_wash"
)

type Input struct {
	Satiety     float64
	Mood        float64
	Energy      float64
	Cleanliness float64
	Traits      Traits
}

type Consideration interface {
	Behavior() Behavior
	Score(in Input) float64
}

func need(stat float64) float64 { return 1 - stat/100 }

type HungerConsideration struct{}

func (HungerConsideration) Behavior() Behavior { return BehaviorEat }
func (HungerConsideration) Score(in Input) float64 {
	n := need(in.Satiety)
	return n * n * (1 + 0.5*in.Traits.Weight(TraitGlutton))
}

type BoredomConsideration struct{}

func (BoredomConsideration) Behavior() Behavior { return BehaviorPlay }
func (BoredomConsideration) Score(in Input) float64 {
	return need(in.Mood) * (1 + 0.5*in.Traits.Weight(TraitPlayful))
}

type TirednessConsideration struct{}

func (TirednessConsideration) Behavior() Behavior { return BehaviorSleep }
func (TirednessConsideration) Score(in Input) float64 {
	return need(in.Energy) * (1 + 0.5*in.Traits.Weight(TraitLazy))
}

type DirtConsideration struct{}

func (DirtConsideration) Behavior() Behavior     { return BehaviorWash }
func (DirtConsideration) Score(in Input) float64 { return 0.8 * need(in.Cleanliness) }

func DefaultConsiderations() []Consideration {
	return []Consideration{
		HungerConsideration{},
		BoredomConsideration{},
		TirednessConsideration{},
		DirtConsideration{},
	}
}
