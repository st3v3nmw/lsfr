package kvstore

import "github.com/st3v3nmw/lsfr/internal/registry"

func init() {
	challenge := &registry.Challenge{
		Name: "Distributed Key-Value Store",
		Summary: `Build a distributed key-value database from scratch.
You'll start with a simple HTTP API and progressively add persistence, clustering, and fault tolerance.`,
		Concepts: []string{"Storage Engines", "Replication", "Consensus", "Fault Tolerance"},
	}

	challenge.AddStage("http-api", "HTTP API with GET/PUT/DELETE Operations", HTTPAPIStage)

	registry.RegisterChallenge("kv-store", challenge)
}
