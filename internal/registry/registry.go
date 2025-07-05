package registry

import (
	"fmt"
	"log"

	"github.com/st3v3nmw/lsfr/internal/suite"
)

func init() {
	log.SetFlags(0)
}

var challenges = make(map[string]*Challenge)

type Challenge struct {
	Key        string
	Name       string
	Concepts   []string
	Summary    string
	Stages     map[string]*Stage
	StageOrder []string
}

type Stage struct {
	Name string
	Fn   StageFunc
}

type StageFunc func() suite.Suite

func (c *Challenge) AddStage(key, name string, fn StageFunc) {
	if c.Stages == nil {
		c.Stages = make(map[string]*Stage)
	}

	c.Stages[key] = &Stage{Name: name, Fn: fn}
	c.StageOrder = append(c.StageOrder, key)
}

func (c *Challenge) GetStage(key string) (*Stage, error) {
	stage, exists := c.Stages[key]
	if !exists {
		return nil, fmt.Errorf("Stage %q not found for challenge %s.", key, c.Key)
	}

	return stage, nil
}

func (c *Challenge) Len() int {
	return len(c.StageOrder)
}

func (c *Challenge) README() string {
	stages := ""
	for i, key := range c.StageOrder {
		stages += fmt.Sprintf("%d. **%s** - %s\n", i+1, key, c.Stages[key].Name)
	}

	return fmt.Sprintf(`# %s Challenge

%s

## Stages

%s
## Getting Started

1. Edit _run.sh_ to start your implementation
2. Run _lsfr test_ to test the current stage
3. Run _lsfr next_ when ready to advance

Good luck! 🚀
`, c.Name, c.Summary, stages)
}

func RegisterChallenge(key string, challenge *Challenge) {
	if len(challenge.Stages) == 0 {
		log.Fatalf("Cannot register empty challenge %s.", key)
	}

	challenge.Key = key
	challenges[key] = challenge
}

func GetChallenge(key string) (*Challenge, error) {
	challenge, exists := challenges[key]
	if !exists {
		return nil, fmt.Errorf("Challenge %s not found", key)
	}

	return challenge, nil
}

func GetAllChallenges() map[string]*Challenge {
	return challenges
}
