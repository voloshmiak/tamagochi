package pets

const (
	MinStat = 0.0
	MaxStat = 100.0
)

type Stats struct {
	Satiety     float64
	Mood        float64
	Energy      float64
	Cleanliness float64
}

var DecayPerHour = Stats{Satiety: 6, Mood: 4, Energy: 3, Cleanliness: 2}

var InitialStats = Stats{Satiety: 80, Mood: 80, Energy: 80, Cleanliness: 80}

func (s Stats) add(d Stats) Stats {
	return Stats{
		Satiety:     s.Satiety + d.Satiety,
		Mood:        s.Mood + d.Mood,
		Energy:      s.Energy + d.Energy,
		Cleanliness: s.Cleanliness + d.Cleanliness,
	}
}

func (s Stats) scale(k float64) Stats {
	return Stats{Satiety: s.Satiety * k, Mood: s.Mood * k, Energy: s.Energy * k, Cleanliness: s.Cleanliness * k}
}

func (s Stats) clamp() Stats {
	c := func(v float64) float64 { return min(max(v, MinStat), MaxStat) }
	return Stats{Satiety: c(s.Satiety), Mood: c(s.Mood), Energy: c(s.Energy), Cleanliness: c(s.Cleanliness)}
}
