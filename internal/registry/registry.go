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
	Stages     map[string]*Stage
	StageOrder []string
	README     string
}

type Stage struct {
	Name    string
	Summary string
	Fn      StageFunc
}

type StageFunc func() suite.Suite

func (c *Challenge) AddStage(key, name, summary string, fn StageFunc) {
	if c.Stages == nil {
		c.Stages = make(map[string]*Stage)
	}

	c.Stages[key] = &Stage{
		Name:    name,
		Summary: summary,
		Fn:      fn,
	}

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
		return nil, fmt.Errorf("challenge %s not found", key)
	}

	return challenge, nil
}
