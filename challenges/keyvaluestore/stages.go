package keyvaluestore

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/st3v3nmw/lsfr/internal/registry"
	"github.com/st3v3nmw/lsfr/internal/suite"
)

func init() {
	challenge := &registry.Challenge{
		Name:     "Distributed Key-Value Store",
		Concepts: []string{"Storage Engines", "Replication", "Consensus", "Fault Tolerance"},
		README: `# Distributed Key-Value Store

Build a distributed key-value database from scratch. You'll start with a simple HTTP API and progressively add persistence, clustering, and fault tolerance.

## Stages

1. **http-api** - Basic GET/PUT/DELETE operations
2. **persistence** - Data survives restarts and crashes
3. **clustering** - Multi-node replication
4. **fault-tolerance** - Handle network partitions

Your server should listen on port 8888 and implement:
- ` + "`PUT /kv/{key}`" + ` - Store a value
- ` + "`GET /kv/{key}`" + ` - Retrieve a value
- ` + "`DELETE /kv/{key}`" + ` - Delete a value` + `

## Getting Started

1. Edit ` + "`run.sh`" + ` to start your implementation
2. Run ` + "`lsfr test`" + ` to test the current stage
3. Run ` + "`lsfr next`" + ` when ready to advance

Good luck! 🚀`,
	}

	challenge.AddStage(
		"Basic Operations",
		"HTTP API with GET/PUT/DELETE",
		Stage1,
	)

	registry.RegisterChallenge("key-value-store", challenge)
}

func Stage1() suite.Suite {
	return *suite.New().
		// 0
		Setup(func(do *suite.Do) error {
			do.Run("node", 8888)
			do.WaitForPort("node")

			cleanupKeys := []string{
				"kenya:capital", "uganda:capital", "tanzania:capital",
				"test:key", "empty", "unicode:key", "long:" + strings.Repeat("k", 100),
				"special:key-with_symbols.123", "delete:twice",
				"concurrent:key1", "concurrent:key2", "concurrent:key3",
			}
			for _, key := range cleanupKeys {
				do.HTTP("node", "DELETE", fmt.Sprintf("/kv/%s", key))
			}

			return nil
		}).

		// 1
		Test("PUT Basic Operations", func(do *suite.Do) {
			// Set initial key-value pairs that subsequent tests can rely on
			do.HTTP("node", "PUT", "/kv/kenya:capital", "Nairobi").
				Got().Status(http.StatusOK)
			do.HTTP("node", "PUT", "/kv/uganda:capital", "Kampala").
				Got().Status(http.StatusOK)
			do.HTTP("node", "PUT", "/kv/tanzania:capital", "Dar es Salaam").
				Got().Status(http.StatusOK)

			// Test overwrite behavior
			do.HTTP("node", "PUT", "/kv/tanzania:capital", "Dodoma").
				Got().Status(http.StatusOK)

			// Verify overwrite worked
			do.HTTP("node", "GET", "/kv/tanzania:capital").
				Got().Status(http.StatusOK).Body("Dodoma")
		}).

		// 2
		Test("PUT Edge and Error Cases", func(do *suite.Do) {
			// Empty value
			do.HTTP("node", "PUT", "/kv/empty").
				Got().Status(http.StatusBadRequest).Body("value cannot be empty\n")

			// Empty key
			do.HTTP("node", "PUT", "/kv/", "some_value").
				Got().Status(http.StatusBadRequest).Body("key cannot be empty\n")

			// Unicode handling
			do.HTTP("node", "PUT", "/kv/unicode:key", "🌍 Nairobi").
				Got().Status(http.StatusOK)

			// Long key and value
			longKey := "long:" + strings.Repeat("k", 100)
			longValue := strings.Repeat("v", 1000)
			do.HTTP("node", "PUT", fmt.Sprintf("/kv/%s", longKey), longValue).
				Got().Status(http.StatusOK)

			// Special characters in key/value
			do.HTTP("node", "PUT", "/kv/special:key-with_symbols.123", "value with spaces & symbols!").
				Got().Status(http.StatusOK)
		}).

		// 3
		Test("GET Basic Operations", func(do *suite.Do) {
			// Retrieve values we know exist from PUT tests
			do.HTTP("node", "GET", "/kv/kenya:capital").
				Got().Status(http.StatusOK).Body("Nairobi")
			do.HTTP("node", "GET", "/kv/uganda:capital").
				Got().Status(http.StatusOK).Body("Kampala")
			do.HTTP("node", "GET", "/kv/tanzania:capital").
				Got().Status(http.StatusOK).Body("Dodoma") // Should be overwritten value

			// Verify Unicode handling
			do.HTTP("node", "GET", "/kv/unicode:key").
				Got().Status(http.StatusOK).Body("🌍 Nairobi")

			// Verify long values
			longKey := "long:" + strings.Repeat("k", 100)
			longValue := strings.Repeat("v", 1000)
			do.HTTP("node", "GET", fmt.Sprintf("/kv/%s", longKey)).
				Got().Status(http.StatusOK).Body(longValue)
		}).

		// 4
		Test("GET Edge and Error Cases", func(do *suite.Do) {
			// Non-existent key
			do.HTTP("node", "GET", "/kv/nonexistent:key").
				Got().Status(http.StatusNotFound).Body("key not found\n")

			// Case sensitivity test
			do.HTTP("node", "GET", "/kv/KENYA:CAPITAL").
				Got().Status(http.StatusNotFound).Body("key not found\n")

			// Empty key
			do.HTTP("node", "GET", "/kv/").
				Got().Status(http.StatusBadRequest).Body("key cannot be empty\n")
		}).

		// 5
		Test("DELETE Basic Operations", func(do *suite.Do) {
			// Delete an existing key
			do.HTTP("node", "DELETE", "/kv/tanzania:capital").
				Got().Status(http.StatusOK)

			// Verify deletion worked
			do.HTTP("node", "GET", "/kv/tanzania:capital").
				Got().Status(http.StatusNotFound).Body("key not found\n")

			// Verify other keys still exist
			do.HTTP("node", "GET", "/kv/kenya:capital").
				Got().Status(http.StatusOK).Body("Nairobi")
		}).

		// 6
		Test("DELETE Edge and Error Cases", func(do *suite.Do) {
			// Delete non-existent key
			do.HTTP("node", "DELETE", "/kv/nonexistent:key").
				Got().Status(http.StatusOK)

			// Delete same key twice
			do.HTTP("node", "PUT", "/kv/delete:twice", "value").
				Got().Status(http.StatusOK)
			do.HTTP("node", "DELETE", "/kv/delete:twice").
				Got().Status(http.StatusOK)
			do.HTTP("node", "DELETE", "/kv/delete:twice").
				Got().Status(http.StatusOK)

			// Empty key
			do.HTTP("node", "DELETE", "/kv/").
				Got().Status(http.StatusBadRequest).Body("key cannot be empty\n")
		}).

		// 7
		Test("Concurrent Operations", func(do *suite.Do) {
			// Test concurrent writes
			do.Concurrently(
				func() {
					do.HTTP("node", "PUT", "/kv/concurrent:key1", "value1").
						Got().Status(http.StatusOK)
				},
				func() {
					do.HTTP("node", "PUT", "/kv/concurrent:key2", "value2").
						Got().Status(http.StatusOK)
				},
				func() {
					do.HTTP("node", "PUT", "/kv/concurrent:key3", "value3").
						Got().Status(http.StatusOK)
				},
			)

			// Verify all concurrent writes succeeded
			do.HTTP("node", "GET", "/kv/concurrent:key1").
				Got().Status(http.StatusOK).Body("value1")
			do.HTTP("node", "GET", "/kv/concurrent:key2").
				Got().Status(http.StatusOK).Body("value2")
			do.HTTP("node", "GET", "/kv/concurrent:key3").
				Got().Status(http.StatusOK).Body("value3")
		}).

		// 8
		Test("Check Allowed HTTP Methods", func(do *suite.Do) {
			// POST not allowed
			do.HTTP("node", "POST", "/kv/test:key").
				Got().Status(http.StatusMethodNotAllowed).Body("method not allowed\n")

			// PATCH not allowed
			do.HTTP("node", "PATCH", "/kv/test:key").
				Got().Status(http.StatusMethodNotAllowed).Body("method not allowed\n")
		})
}
