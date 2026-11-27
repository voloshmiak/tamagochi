package pets

import "time"

// Stage is the growth stage of a pet (FR-PET-02). It is derived from the
// pet's age, so it never has to be stored or migrated.
type Stage string

const (
	StageEgg   Stage = "egg"
	StageBaby  Stage = "baby"
	StageChild Stage = "child"
	StageTeen  Stage = "teen"
	StageAdult Stage = "adult"
)

// stageAges lists the age at which each stage starts, in ascending order.
var stageAges = []struct {
	from  time.Duration
	stage Stage
}{
	{0, StageEgg},
	{10 * time.Minute, StageBaby},
	{24 * time.Hour, StageChild},
	{3 * 24 * time.Hour, StageTeen},
	{7 * 24 * time.Hour, StageAdult},
}

// StageAt returns the stage of a pet born at bornAt, as of now.
func StageAt(bornAt, now time.Time) Stage {
	age := now.Sub(bornAt)
	stage := StageEgg
	for _, s := range stageAges {
		if age >= s.from {
			stage = s.stage
		}
	}
	return stage
}

// Stage returns the pet's growth stage as of now.
func (p *Pet) Stage(now time.Time) Stage { return StageAt(p.BornAt, now) }
