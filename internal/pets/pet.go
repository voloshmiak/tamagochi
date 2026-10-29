package pets

import (
	"fmt"
	"maps"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/voloshmiak/tamagochi/internal/brain"
	"github.com/voloshmiak/tamagochi/internal/platform/apperr"
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
	name = strings.TrimSpace(name)
	if n := utf8.RuneCountInString(name); n < 2 || n > 20 {
		return nil, fmt.Errorf("%w: pet name must be 2-20 characters", apperr.ErrInvalid)
	}
	if appearance == "" {
		return nil, fmt.Errorf("%w: appearance is required", apperr.ErrInvalid)
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
