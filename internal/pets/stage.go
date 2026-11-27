package pets

import "time"

type Stage string

const (
	StageEgg   Stage = "egg"
	StageBaby  Stage = "baby"
	StageChild Stage = "child"
	StageTeen  Stage = "teen"
	StageAdult Stage = "adult"
)

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

func (p *Pet) Stage(now time.Time) Stage { return StageAt(p.BornAt, now) }
