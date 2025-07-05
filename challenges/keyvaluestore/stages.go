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
		README: `# Key-Value Store Challenge

Build a distributed key-value database from scratch. You'll start with a simple HTTP API and progressively add persistence, clustering, and fault tolerance.

## Stages

1. **http-api** - Basic GET/PUT/DELETE operations
2. **persistence** - Data survives restarts and crashes
3. **clustering** - Multi-node replication
4. **fault-tolerance** - Handle network partitions

## Getting Started

1. Edit _run.sh_ to start your implementation
2. Run _lsfr test_ to test the current stage
3. Run _lsfr next_ when ready to advance

Your server should listen on port 8888 and implement:
1. PUT /kv/{key} - Store a value
2. GET /kv/{key} - Retrieve a value
3. DELETE /kv/{key} - Delete a value

Good luck! 🚀
`,
	}

	challenge.AddStage(
		"http-api",
		"Basic HTTP API",
		"HTTP API with GET/PUT/DELETE",
		HTTPAPIStage,
	)

	registry.RegisterChallenge("key-value-store", challenge)
}

func HTTPAPIStage() suite.Suite {
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
			putHelp := "Your server should accept PUT requests and return 200 OK.\nEnsure your HTTP handler processes PUT requests to /kv/{key}."
			do.HTTP("node", "PUT", "/kv/kenya:capital", "Nairobi").
				WithHelp(putHelp).
				Got().Status(http.StatusOK)
			do.HTTP("node", "PUT", "/kv/uganda:capital", "Kampala").
				WithHelp(putHelp).
				Got().Status(http.StatusOK)
			do.HTTP("node", "PUT", "/kv/tanzania:capital", "Dar es Salaam").
				WithHelp(putHelp).
				Got().Status(http.StatusOK)

			// Test overwrite behavior
			do.HTTP("node", "PUT", "/kv/tanzania:capital", "Dodoma").
				WithHelp("Your server should allow overwriting existing keys.\nPUT requests should update the value of existing keys.").
				Got().Status(http.StatusOK)

			// Verify overwrite worked
			do.HTTP("node", "GET", "/kv/tanzania:capital").
				WithHelp("Your server should return the updated value after overwrite.\nGET requests should return the most recently stored value.").
				Got().Status(http.StatusOK).Body("Dodoma")
		}).

		// 2
		Test("PUT Edge and Error Cases", func(do *suite.Do) {
			// Empty value
			do.HTTP("node", "PUT", "/kv/empty").
				WithHelp("Your server accepted an empty value when it should reject it.\nAdd validation to return 400 Bad Request for empty values.").
				Got().Status(http.StatusBadRequest).Body("value cannot be empty\n")

			// Empty key
			do.HTTP("node", "PUT", "/kv/", "some_value").
				WithHelp("Your server accepted an empty key when it should reject it.\nAdd validation to return 400 Bad Request for empty keys.").
				Got().Status(http.StatusBadRequest).Body("key cannot be empty\n")

			// Unicode handling
			do.HTTP("node", "PUT", "/kv/unicode:key", "🌍 Nairobi").
				WithHelp("Your server should handle Unicode characters in values.\nEnsure your HTTP handler properly processes UTF-8 encoded data.").
				Got().Status(http.StatusOK)

			// Long key and value
			longKey := "long:" + strings.Repeat("k", 100)
			longValue := strings.Repeat("v", 1000)
			do.HTTP("node", "PUT", fmt.Sprintf("/kv/%s", longKey), longValue).
				WithHelp("Your server should handle long keys and values.\nEnsure your implementation doesn't have arbitrary length limits.").
				Got().Status(http.StatusOK)

			// Special characters in key/value
			do.HTTP("node", "PUT", "/kv/special:key-with_symbols.123", "value with spaces & symbols!").
				WithHelp("Your server should handle special characters in keys and values.\nEnsure proper URL path parsing and value encoding/decoding.").
				Got().Status(http.StatusOK)
		}).

		// 3
		Test("GET Basic Operations", func(do *suite.Do) {
			// Retrieve values we know exist from PUT tests
			do.HTTP("node", "GET", "/kv/kenya:capital").
				WithHelp("Your server should return stored values with GET requests.\nEnsure your key-value storage and retrieval logic is working correctly.").
				Got().Status(http.StatusOK).Body("Nairobi")
			do.HTTP("node", "GET", "/kv/uganda:capital").
				WithHelp("Your server should return stored values with GET requests.\nEnsure your key-value storage and retrieval logic is working correctly.").
				Got().Status(http.StatusOK).Body("Kampala")
			do.HTTP("node", "GET", "/kv/tanzania:capital").
				WithHelp("Your server should return the most recently stored value.\nEnsure overwrite operations update the stored value correctly.").
				Got().Status(http.StatusOK).Body("Dodoma")

			// Verify Unicode handling
			do.HTTP("node", "GET", "/kv/unicode:key").
				WithHelp("Your server should preserve Unicode characters in stored values.\nEnsure proper UTF-8 handling in your storage and retrieval logic.").
				Got().Status(http.StatusOK).Body("🌍 Nairobi")

			// Verify long values
			longKey := "long:" + strings.Repeat("k", 100)
			longValue := strings.Repeat("v", 1000)
			do.HTTP("node", "GET", fmt.Sprintf("/kv/%s", longKey)).
				WithHelp("Your server should handle retrieval of long keys and values.\nEnsure your storage doesn't truncate or corrupt large data.").
				Got().Status(http.StatusOK).Body(longValue)
		}).

		// 4
		Test("GET Edge and Error Cases", func(do *suite.Do) {
			// Non-existent key
			do.HTTP("node", "GET", "/kv/nonexistent:key").
				WithHelp("Your server should return 404 Not Found when a key doesn't exist.\nCheck your key lookup logic and error handling.").
				Got().Status(http.StatusNotFound).Body("key not found\n")

			// Case sensitivity test
			do.HTTP("node", "GET", "/kv/KENYA:CAPITAL").
				WithHelp("Your server should return 404 Not Found when a key doesn't exist.\nCheck your key lookup logic and error handling.").
				Got().Status(http.StatusNotFound).Body("key not found\n")

			// Empty key
			do.HTTP("node", "GET", "/kv/").
				WithHelp("Your server accepted an empty key when it should reject it.\nAdd validation to return 400 Bad Request for empty keys.").
				Got().Status(http.StatusBadRequest).Body("key cannot be empty\n")
		}).

		// 5
		Test("DELETE Basic Operations", func(do *suite.Do) {
			// Delete an existing key
			do.HTTP("node", "DELETE", "/kv/tanzania:capital").
				WithHelp("Your server should accept DELETE requests and return 200 OK.\nEnsure your HTTP handler processes DELETE requests to /kv/{key}.").
				Got().Status(http.StatusOK)

			// Verify deletion worked
			do.HTTP("node", "GET", "/kv/tanzania:capital").
				WithHelp("Your server should return 404 Not Found when a key doesn't exist.\nCheck your key lookup logic and error handling.").
				Got().Status(http.StatusNotFound).Body("key not found\n")

			// Verify other keys still exist
			do.HTTP("node", "GET", "/kv/kenya:capital").
				WithHelp("Your server should only delete the specified key, not affect others.\nEnsure your delete operation doesn't remove unrelated data.").
				Got().Status(http.StatusOK).Body("Nairobi")
		}).

		// 6
		Test("DELETE Edge and Error Cases", func(do *suite.Do) {
			// Delete non-existent key
			do.HTTP("node", "DELETE", "/kv/nonexistent:key").
				WithHelp("Your server should gracefully handle deletion of non-existent keys.\nReturning 200 OK for missing keys is acceptable (idempotent).").
				Got().Status(http.StatusOK)

			// Delete same key twice
			do.HTTP("node", "PUT", "/kv/delete:twice", "value").
				WithHelp("Your server should accept PUT requests and return 200 OK.\nEnsure your HTTP handler processes PUT requests to /kv/{key}.").
				Got().Status(http.StatusOK)
			do.HTTP("node", "DELETE", "/kv/delete:twice").
				WithHelp("Your server should successfully delete existing keys.\nImplement proper key removal in your storage logic.").
				Got().Status(http.StatusOK)
			do.HTTP("node", "DELETE", "/kv/delete:twice").
				WithHelp("Your server should handle repeated deletions gracefully.\nDeleting the same key twice should remain idempotent (return 200 OK).").
				Got().Status(http.StatusOK)

			// Empty key
			do.HTTP("node", "DELETE", "/kv/").
				WithHelp("Your server accepted an empty key when it should reject it.\nAdd validation to return 400 Bad Request for empty keys.").
				Got().Status(http.StatusBadRequest).Body("key cannot be empty\n")
		}).

		// 7
		Test("Concurrent Operations", func(do *suite.Do) {
			// Test concurrent writes
			putHelp := "Your server should handle concurrent PUT requests correctly.\nEnsure thread-safety in your storage implementation."
			do.Concurrently(
				func() {
					do.HTTP("node", "PUT", "/kv/concurrent:key1", "value1").
						WithHelp(putHelp).
						Got().Status(http.StatusOK)
				},
				func() {
					do.HTTP("node", "PUT", "/kv/concurrent:key2", "value2").
						WithHelp(putHelp).
						Got().Status(http.StatusOK)
				},
				func() {
					do.HTTP("node", "PUT", "/kv/concurrent:key3", "value3").
						WithHelp(putHelp).
						Got().Status(http.StatusOK)
				},
			)

			// Verify all concurrent writes succeeded
			getHelp := "Your server should store all concurrent writes correctly.\nEnsure no data corruption or loss occurs during concurrent operations."
			do.HTTP("node", "GET", "/kv/concurrent:key1").
				WithHelp(getHelp).
				Got().Status(http.StatusOK).Body("value1")
			do.HTTP("node", "GET", "/kv/concurrent:key2").
				WithHelp(getHelp).
				Got().Status(http.StatusOK).Body("value2")
			do.HTTP("node", "GET", "/kv/concurrent:key3").
				WithHelp(getHelp).
				Got().Status(http.StatusOK).Body("value3")
		}).

		// 8
		Test("Check Allowed HTTP Methods", func(do *suite.Do) {
			notAllowedHelp := "Your server should reject unsupported HTTP methods.\nAdd logic to return 405 Method Not Allowed for unsupported methods."

			// POST not allowed
			do.HTTP("node", "POST", "/kv/test:key").
				WithHelp(notAllowedHelp).
				Got().Status(http.StatusMethodNotAllowed).Body("method not allowed\n")

			// PATCH not allowed
			do.HTTP("node", "PATCH", "/kv/test:key").
				WithHelp(notAllowedHelp).
				Got().Status(http.StatusMethodNotAllowed).Body("method not allowed\n")
		})
}
