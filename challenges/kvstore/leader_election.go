package kvstore

// Notes:
//
// Timing assumptions:
//   - Election timeout: 500-1,000ms (randomized)
//   - Heartbeat interval: 100ms
//   - Elections should complete within 2 seconds under normal conditions
//   - Wait ≥2 seconds to verify "no leader elected" scenarios
//
// Observability (black-box testing via APIs):
//   - GET /cluster/info: role, term, leader, votedFor
//   - GET/PUT/DELETE /kv/*: 307 redirect to leader, 503 if no leader
//   - POST /cluster/partition: isolate nodes (persists across restarts)
//   - POST /cluster/heal: restore connectivity
//
// Possible test scenarios:
//   1. Leader Election Completes
//   2. Exactly One Leader Per Term
//   3. Leader Maintains Authority via Heartbeats
//   4. Follower Redirects Clients to Leader
//   5. State Survives Crashes
//   6. Minority Partition Cannot Elect Leader
//   7. Majority Partition Elects Leader
//   8. Healing After Partition

import (
	"fmt"

	. "github.com/st3v3nmw/lsfr/internal/attest"
)

func LeaderElection() *Suite {
	return New().
		// 0
		Setup(func(do *Do) {
			for i := range 5 {
				do.Start(fmt.Sprintf("node-%d", i+1))
			}
		})
}
