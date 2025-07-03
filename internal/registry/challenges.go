package registry

import (
	"fmt"

	"github.com/st3v3nmw/lsfr/internal/suite"
)

var challenges = make(map[string]*Challenge)

type Challenge struct {
	Name     string
	Concepts []string
	Stages   []*Stage
}

type Stage struct {
	Name    string
	Summary string
	Fn      StageFunc
}

type StageFunc func() suite.Suite

func (c *Challenge) AddStage(name, summary string, fn StageFunc) {
	c.Stages = append(c.Stages, &Stage{
		Name:    name,
		Summary: summary,
		Fn:      fn,
	})
}

func (c *Challenge) GetStage(number int) (*Stage, error) {

	if number > len(c.Stages) || number < 1 {
		return nil, fmt.Errorf("stage %d not found for challenge %q", number, c.Name)
	}

	number -= 1
	return c.Stages[number], nil
}

func (c *Challenge) Len() int {
	return len(c.Stages)
}
func RegisterChallenge(key string, challenge *Challenge) {
	challenges[key] = challenge
}

func GetChallenge(key string) (*Challenge, error) {
	challenge, exists := challenges[key]
	if !exists {
		return nil, fmt.Errorf("challenge %q not found", key)
	}

	return challenge, nil
}
