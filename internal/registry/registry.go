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

// Challenge represents a coding challenge
type Challenge struct {
	Key        string
	Name       string
	Summary    string
	Concepts   []string
	Stages     map[string]*Stage
	StageOrder []string
}

// Stage represents a single stage within a challenge
type Stage struct {
	Name string
	Fn   StageFunc
}

// StageFunc is a function that returns a test suite for a stage
type StageFunc func() suite.Suite

// AddStage adds a new stage to the challenge
func (c *Challenge) AddStage(key, name string, fn StageFunc) {
	if c.Stages == nil {
		c.Stages = make(map[string]*Stage)
	}

	c.Stages[key] = &Stage{Name: name, Fn: fn}
	c.StageOrder = append(c.StageOrder, key)
}

// GetStage retrieves a stage by key
func (c *Challenge) GetStage(key string) (*Stage, error) {
	stage, exists := c.Stages[key]
	if !exists {
		return nil, fmt.Errorf("Stage %q not found for challenge %s.", key, c.Key)
	}

	return stage, nil
}

// StageIndex returns the index of a stage in the order, or -1 if not found
func (c *Challenge) StageIndex(key string) int {
	for i, stageKey := range c.StageOrder {
		if stageKey == key {
			return i
		}
	}

	return -1
}

// Len returns the number of stages in the challenge
func (c *Challenge) Len() int {
	return len(c.StageOrder)
}

// README generates the README content for the challenge
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

1. Edit _run.sh_ to start your implementation.
2. Run _lsfr test_ to test the current stage.
3. Run _lsfr next_ when ready to advance.

Good luck! 🚀
`, c.Name, c.Summary, stages)
}

// RegisterChallenge registers a challenge in the global registry
func RegisterChallenge(key string, challenge *Challenge) {
	if challenge.Len() == 0 {
		log.Fatalf("Cannot register empty challenge %s.", key)
	}

	challenge.Key = key
	challenges[key] = challenge
}

// GetChallenge retrieves a registered challenge by key
func GetChallenge(key string) (*Challenge, error) {
	challenge, exists := challenges[key]
	if !exists {
		return nil, fmt.Errorf("Challenge %s not found", key)
	}

	return challenge, nil
}

// GetAllChallenges returns all registered challenges
func GetAllChallenges() map[string]*Challenge {
	return challenges
}
