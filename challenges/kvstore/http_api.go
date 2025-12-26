package kvstore

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/st3v3nmw/lsfr/internal/attest"
)

func HTTPAPI() *attest.Suite {
	return attest.New().
		// 0
		Setup(func(do *attest.Do) {
			do.Start("primary")

			// Clear key-value store
			do.HTTP("primary", "DELETE", "/clear").
				Returns().Status(http.StatusOK).
				Assert("Your server should implement a /clear endpoint.\n" +
					"Add a DELETE /clear method that deletes all key-value pairs.")
		}).

		// 1
		Test("PUT Basic Operations", func(do *attest.Do) {
			// Set initial key-value pairs that subsequent tests can rely on
			capitals := map[string]string{
				"kenya":    "Nairobi",
				"uganda":   "Kampala",
				"tanzania": "Dar es Salaam",
			}
			for country, capital := range capitals {
				do.HTTP("primary", "PUT", fmt.Sprintf("/kv/%s:capital", country), capital).
					Returns().Status(http.StatusOK).
					Assert("Your server should accept PUT requests and return 200 OK.\n" +
						"Ensure your HTTP handler processes PUT requests to /kv/{key}.")
			}

			// Test overwrite behavior
			do.HTTP("primary", "PUT", "/kv/tanzania:capital", "Dodoma").
				Returns().Status(http.StatusOK).
				Assert("Your server should allow overwriting existing keys.\n" +
					"Ensure PUT requests update the value of existing keys.")

			// Verify overwrite worked
			do.HTTP("primary", "GET", "/kv/tanzania:capital").
				Returns().Status(http.StatusOK).Body("Dodoma").
				Assert("Your server should return the updated value after overwrite.\n" +
					"Ensure GET requests return the most recently stored value.")
		}).

		// 2
		Test("PUT Edge and Error Cases", func(do *attest.Do) {
			// Empty value
			do.HTTP("primary", "PUT", "/kv/empty").
				Returns().Status(http.StatusBadRequest).Body("value cannot be empty\n").
				Assert("Your server accepted an empty value when it should reject it.\n" +
					"Add validation to return 400 Bad Request for empty values.")

			// Empty key
			do.HTTP("primary", "PUT", "/kv/", "some_value").
				Returns().Status(http.StatusBadRequest).Body("key cannot be empty\n").
				Assert("Your server accepted an empty key when it should reject it.\n" +
					"Add validation to return 400 Bad Request for empty keys.")

			// Unicode handling
			do.HTTP("primary", "PUT", "/kv/unicode:key", "🌍 Nairobi").
				Returns().Status(http.StatusOK).
				Assert("Your server should handle Unicode characters in values.\n" +
					"Ensure your HTTP handler properly processes UTF-8 encoded data.")

			// Long key and value
			longKey := "long:" + strings.Repeat("k", 100)
			longValue := strings.Repeat("v", 10_000)
			do.HTTP("primary", "PUT", fmt.Sprintf("/kv/%s", longKey), longValue).
				Returns().Status(http.StatusOK).
				Assert("Your server should handle long keys and values.\n" +
					"Ensure your server doesn't have arbitrary key & value length limits.")

			// Special characters in key/value
			do.HTTP("primary", "PUT", "/kv/special:key-with_symbols.123", "value with spaces & symbols! \t").
				Returns().Status(http.StatusOK).
				Assert("Your server should handle special characters in keys and values.\n" +
					"Ensure proper URL path parsing and value encoding/decoding.")
		}).

		// 3
		Test("GET Basic Operations", func(do *attest.Do) {
			// Retrieve values we know exist from PUT tests
			do.HTTP("primary", "GET", "/kv/kenya:capital").
				Returns().Status(http.StatusOK).Body("Nairobi").
				Assert("Your server should return stored values with GET requests.\n" +
					"Ensure your key-value storage and retrieval logic is working correctly.")
			do.HTTP("primary", "GET", "/kv/uganda:capital").
				Returns().Status(http.StatusOK).Body("Kampala").
				Assert("Your server should return stored values with GET requests.\n" +
					"Ensure your key-value storage and retrieval logic is working correctly.")
			do.HTTP("primary", "GET", "/kv/tanzania:capital").
				Returns().Status(http.StatusOK).Body("Dodoma").
				Assert("Your server should return the most recently stored value.\n" +
					"Ensure overwrite operations update the stored value correctly.")

			// Verify Unicode handling
			do.HTTP("primary", "GET", "/kv/unicode:key").
				Returns().Status(http.StatusOK).Body("🌍 Nairobi").
				Assert("Your server should preserve Unicode characters in stored values.\n" +
					"Ensure proper UTF-8 handling in your storage and retrieval logic.")

			// Verify long values
			longKey := "long:" + strings.Repeat("k", 100)
			longValue := strings.Repeat("v", 10_000)
			do.HTTP("primary", "GET", fmt.Sprintf("/kv/%s", longKey)).
				Returns().Status(http.StatusOK).Body(longValue).
				Assert("Your server should handle retrieval of long keys and values.\n" +
					"Ensure your storage doesn't truncate or corrupt large data.")
		}).

		// 4
		Test("GET Edge and Error Cases", func(do *attest.Do) {
			// Non-existent key
			do.HTTP("primary", "GET", "/kv/nonexistent:key").
				Returns().Status(http.StatusNotFound).Body("key not found\n").
				Assert("Your server should return 404 Not Found when a key doesn't exist.\n" +
					"Check your key lookup logic and error handling.")

			// Case sensitivity test
			do.HTTP("primary", "GET", "/kv/KENYA:CAPITAL").
				Returns().Status(http.StatusNotFound).Body("key not found\n").
				Assert("Your server should return 404 Not Found when a key doesn't exist.\n" +
					"Check your key lookup logic and error handling.")

			// Empty key
			do.HTTP("primary", "GET", "/kv/").
				Returns().Status(http.StatusBadRequest).Body("key cannot be empty\n").
				Assert("Your server accepted an empty key when it should reject it.\n" +
					"Add validation to return 400 Bad Request for empty keys.")
		}).

		// 5
		Test("DELETE Basic Operations", func(do *attest.Do) {
			// Delete an existing key
			do.HTTP("primary", "DELETE", "/kv/tanzania:capital").
				Returns().Status(http.StatusOK).
				Assert("Your server should accept DELETE requests and return 200 OK.\n" +
					"Ensure your HTTP handler processes DELETE requests to /kv/{key}.")

			// Verify deletion worked
			do.HTTP("primary", "GET", "/kv/tanzania:capital").
				Returns().Status(http.StatusNotFound).Body("key not found\n").
				Assert("Your server should return 404 Not Found when a key doesn't exist.\n" +
					"Check your key lookup logic and error handling.")

			// Verify other keys still exist
			do.HTTP("primary", "GET", "/kv/kenya:capital").
				Returns().Status(http.StatusOK).Body("Nairobi").
				Assert("Your server should only delete the specified key, not affect others.\n" +
					"Ensure your delete operation doesn't remove unrelated data.")
		}).

		// 6
		Test("DELETE Edge and Error Cases", func(do *attest.Do) {
			// Delete non-existent key
			do.HTTP("primary", "DELETE", "/kv/nonexistent:key").
				Returns().Status(http.StatusOK).
				Assert("Your server should gracefully handle deletion of non-existent keys.\n" +
					"Returning 200 OK for missing keys is acceptable (idempotent).")

			// Delete same key twice
			do.HTTP("primary", "PUT", "/kv/delete:twice", "value").
				Returns().Status(http.StatusOK).
				Assert("Your server should accept PUT requests and return 200 OK.\n" +
					"Ensure your HTTP handler processes PUT requests to /kv/{key}.")
			do.HTTP("primary", "DELETE", "/kv/delete:twice").
				Returns().Status(http.StatusOK).
				Assert("Your server should successfully delete existing keys.\n" +
					"Implement proper key removal in your storage logic.")
			do.HTTP("primary", "DELETE", "/kv/delete:twice").
				Returns().Status(http.StatusOK).
				Assert("Your server should handle repeated deletions gracefully.\n" +
					"Deleting the same key twice should be idempotent (return 200 OK).")

			// Empty key
			do.HTTP("primary", "DELETE", "/kv/").
				Returns().Status(http.StatusBadRequest).Body("key cannot be empty\n").
				Assert("Your server accepted an empty key when it should reject it.\n" +
					"Add validation to return 400 Bad Request for empty keys.")
		}).

		// 7
		Test("Concurrent Operations", func(do *attest.Do) {
			// Test concurrent writes
			putKV := func(key, value string) func() {
				return func() {
					do.HTTP("primary", "PUT", "/kv/concurrent:"+key, value).
						Returns().Status(http.StatusOK).
						Assert("Your server should handle concurrent PUT requests correctly.\n" +
							"Ensure thread-safety in your storage implementation.")
				}
			}

			fns := []func(){}
			for i := 1; i <= 1_000; i++ {
				fns = append(fns, putKV(fmt.Sprintf("key%d", i), fmt.Sprintf("value%d", i)))
			}

			do.Concurrently(fns...)

			// Verify all concurrent writes succeeded
			for i := 1; i <= 1_000; i++ {
				do.HTTP("primary", "GET", fmt.Sprintf("/kv/concurrent:key%d", i)).
					Returns().Status(http.StatusOK).Body(fmt.Sprintf("value%d", i)).
					Assert("Your server should store all concurrent writes correctly.\n" +
						"Ensure no data corruption or loss occurs during concurrent operations.")
			}
		}).

		// 8
		Test("Check Allowed HTTP Methods", func(do *attest.Do) {
			// POST & PATCH /kv/{key} not allowed
			methods := []string{"POST", "PATCH"}
			for _, method := range methods {
				do.HTTP("primary", method, "/kv/test:key").
					Returns().Status(http.StatusMethodNotAllowed).Body("method not allowed\n").
					Assert("Your server should reject unsupported HTTP methods.\n" +
						"Add logic to return 405 Method Not Allowed for unsupported methods.")
			}
		})
}
