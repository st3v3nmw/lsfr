package registry

import (
	"fmt"

	"github.com/st3v3nmw/lsfr/internal/suite"
)

type StageFunc func() suite.Suite

var stages = make(map[string]map[int]StageFunc)

func RegisterStage(challenge string, stageNum int, fn StageFunc) {
	if stages[challenge] == nil {
		stages[challenge] = make(map[int]StageFunc)
	}

	stages[challenge][stageNum] = fn
}

func GetStage(challenge string, stageNum int) (StageFunc, error) {
	challengeStages, exists := stages[challenge]
	if !exists {
		return nil, fmt.Errorf("challenge %q not found", challenge)
	}

	stageFn, exists := challengeStages[stageNum]
	if !exists {
		return nil, fmt.Errorf("stage %d not found for challenge %q", stageNum, challenge)
	}

	return stageFn, nil
}
