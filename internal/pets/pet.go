package pets

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"time"
	"unicode/utf8"

	"github.com/voloshmiak/tamagochi/internal/brain"
	"github.com/voloshmiak/tamagochi/internal/platform/apperr"
	"github.com/voloshmiak/tamagochi/internal/platform/text"
)

type Pet struct {
	ID          string
	OwnerID     string
	Name        string
	Appearance  string
	Stats       Stats
	Traits      brain.Traits
	BornAt      time.Time
	UpdatedAt   time.Time
	LastActions map[Action]time.Time
}

func NewPet(id, ownerID, name, appearance string, traits brain.Traits, now time.Time) (*Pet, error) {
	name, ok := text.Clean(name)
	if !ok {
		return nil, fmt.Errorf("%w: pet name contains forbidden characters", apperr.ErrInvalid)
	}
	if n := utf8.RuneCountInString(name); n < 2 || n > 20 {
		return nil, fmt.Errorf("%w: pet name must be 2-20 characters", apperr.ErrInvalid)
	}
	if !validAppearance(appearance) {
		return nil, fmt.Errorf("%w: appearance must be 1-32 latin letters, digits or dashes", apperr.ErrInvalid)
	}
	return &Pet{
		ID:          id,
		OwnerID:     ownerID,
		Name:        name,
		Appearance:  appearance,
		Stats:       InitialStats,
		Traits:      traits,
		BornAt:      now,
		UpdatedAt:   now,
		LastActions: make(map[Action]time.Time),
	}, nil
}

func (p *Pet) Clone() *Pet {
	c := *p
	c.Traits = maps.Clone(p.Traits)
	c.LastActions = maps.Clone(p.LastActions)
	return &c
}

func (p *Pet) Recompute(now time.Time) {
	elapsed := now.Sub(p.UpdatedAt)
	if elapsed <= 0 {
		return
	}
	p.Stats = p.Stats.add(DecayPerHour.scale(-elapsed.Hours())).clamp()
	p.UpdatedAt = now
}

func (p *Pet) TimeToThreshold(threshold float64) (time.Duration, StatName, bool) {
	var (
		best  time.Duration
		stat  StatName
		found bool
	)
	for _, n := range StatNames {
		v, rate := p.Stats.Get(n), DecayPerHour.Get(n)
		if v < threshold || rate <= 0 {
			continue
		}
		d := time.Duration((v - threshold) / rate * float64(time.Hour))
		if !found || d < best {
			best, stat, found = d, n, true
		}
	}
	return best, stat, found
}

func (p *Pet) LowStats(threshold float64) []StatName {
	var out []StatName
	for _, n := range StatNames {
		if p.Stats.Get(n) < threshold {
			out = append(out, n)
		}
	}
	slices.SortStableFunc(out, func(a, b StatName) int { return cmp.Compare(p.Stats.Get(a), p.Stats.Get(b)) })
	return out
}

func (p *Pet) CooldownLeft(rule ActionRule, now time.Time) time.Duration {
	last, ok := p.LastActions[rule.Action()]
	if !ok {
		return 0
	}
	return max(last.Add(rule.Cooldown()).Sub(now), 0)
}

func (p *Pet) Apply(rule ActionRule, now time.Time) error {
	if left := p.CooldownLeft(rule, now); left > 0 {
		return &CooldownError{Action: rule.Action(), Left: left}
	}
	p.Stats = rule.Effect(p.Stats).clamp()
	p.LastActions[rule.Action()] = now
	p.UpdatedAt = now
	return nil
}

func (p *Pet) BrainInput() brain.Input {
	return brain.Input{
		Satiety:     p.Stats.Satiety,
		Mood:        p.Stats.Mood,
		Energy:      p.Stats.Energy,
		Cleanliness: p.Stats.Cleanliness,
		Traits:      p.Traits,
	}
}

type CooldownError struct {
	Action Action
	Left   time.Duration
}

func (e *CooldownError) Error() string {
	return fmt.Sprintf("action %q is on cooldown for %s", e.Action, e.Left.Round(time.Second))
}

func (e *CooldownError) Unwrap() error { return apperr.ErrTooEarly }

func (e *CooldownError) RetryAfter() time.Duration { return e.Left }

func validAppearance(s string) bool {
	if len(s) == 0 || len(s) > 32 {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
			return false
		}
	}
	return true
}
