package kvstore

import "github.com/st3v3nmw/lsfr/internal/registry"

func init() {
	challenge := &registry.Challenge{
		Name: "Distributed Key-Value Store",
		Summary: `In this challenge, you'll build a distributed key-value store from scratch.
You'll start with a simple HTTP API and progressively add persistence, crash recovery,
clustering, replication, and consensus mechanisms.`,
	}

	challenge.AddStage("http-api", "HTTP API with GET/PUT/DELETE Operations", HTTPAPI)
	challenge.AddStage("persistence", "Data Survives SIGTERM", Persistence)
	challenge.AddStage("crash-recovery", "Data Survives SIGKILL", CrashRecovery)

	registry.RegisterChallenge("kv-store", challenge)
}
