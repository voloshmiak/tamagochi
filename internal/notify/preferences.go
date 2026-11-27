package notify

import (
	"fmt"
	"maps"
	"strconv"
	"strings"
	"time"

	"github.com/voloshmiak/tamagochi/internal/platform/apperr"
)

const minutesPerDay = 24 * 60

type Preferences struct {
	Disabled     map[Kind]bool
	QuietEnabled bool
	QuietFrom    int
	QuietTo      int
	Timezone     string
	Lang         string
}

func DefaultPreferences() Preferences {
	return Preferences{
		Disabled:  map[Kind]bool{},
		QuietFrom: 23 * 60,
		QuietTo:   8 * 60,
		Timezone:  "UTC",
		Lang:      "uk",
	}
}

func (p Preferences) Clone() Preferences {
	p.Disabled = maps.Clone(p.Disabled)
	if p.Disabled == nil {
		p.Disabled = map[Kind]bool{}
	}
	return p
}

func (p Preferences) Allows(k Kind) bool { return !p.Disabled[k] }

func (p Preferences) Location() *time.Location {
	loc, err := time.LoadLocation(p.Timezone)
	if err != nil {
		return time.UTC
	}
	return loc
}

func (p Preferences) LocalDay(now time.Time) string {
	return now.In(p.Location()).Format(time.DateOnly)
}

func (p Preferences) InQuietHours(now time.Time) bool {
	if !p.QuietEnabled || p.QuietFrom == p.QuietTo {
		return false
	}
	local := now.In(p.Location())
	m := local.Hour()*60 + local.Minute()
	if p.QuietFrom < p.QuietTo {
		return m >= p.QuietFrom && m < p.QuietTo
	}
	return m >= p.QuietFrom || m < p.QuietTo
}

func (p Preferences) normalize() (Preferences, error) {
	p = p.Clone()
	for k := range p.Disabled {
		if !validKind(k) {
			return Preferences{}, fmt.Errorf("%w: unknown notification kind %q", apperr.ErrInvalid, k)
		}
		if !p.Disabled[k] {
			delete(p.Disabled, k)
		}
	}
	for _, m := range []int{p.QuietFrom, p.QuietTo} {
		if m < 0 || m >= minutesPerDay {
			return Preferences{}, fmt.Errorf("%w: quiet hours must be within a day", apperr.ErrInvalid)
		}
	}
	if p.Timezone == "" {
		p.Timezone = "UTC"
	}
	if _, err := time.LoadLocation(p.Timezone); err != nil || strings.HasPrefix(p.Timezone, "Local") {
		return Preferences{}, fmt.Errorf("%w: unknown time zone %q", apperr.ErrInvalid, p.Timezone)
	}
	if p.Lang != "en" {
		p.Lang = "uk"
	}
	return p, nil
}

func ParseClock(s string) (int, error) {
	h, m, ok := strings.Cut(s, ":")
	hh, err1 := strconv.Atoi(h)
	mm, err2 := strconv.Atoi(m)
	if !ok || err1 != nil || err2 != nil || len(h) != 2 || len(m) != 2 || hh < 0 || hh > 23 || mm < 0 || mm > 59 {
		return 0, fmt.Errorf("%w: time must look like HH:MM, got %q", apperr.ErrInvalid, s)
	}
	return hh*60 + mm, nil
}

func FormatClock(minutes int) string {
	return fmt.Sprintf("%02d:%02d", minutes/60, minutes%60)
}
