package kvstore

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/st3v3nmw/lsfr/pkg/attest"
)

func Persistence() *attest.Suite {
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
		Test("Basic Persistence Setup", func(do *attest.Do) {
			// Store initial data
			testData := map[string]string{
				"persistent:key1": "value1",
				"persistent:key2": "value with spaces",
				"persistent:key3": "🌍 unicode value",
				"persistent:key4": strings.Repeat("long_value_", 50),
			}

			for key, value := range testData {
				do.HTTP("primary", "PUT", fmt.Sprintf("/kv/%s", key), value).
					Returns().Status(http.StatusOK).
					Assert("Your server should accept PUT requests and store data.\n" +
						"Ensure your HTTP handler processes PUT requests correctly.")
			}

			// Verify data is accessible before restart
			for key, expectedValue := range testData {
				do.HTTP("primary", "GET", fmt.Sprintf("/kv/%s", key)).
					Returns().Status(http.StatusOK).Body(expectedValue).
					Assert("Your server should return stored values before persistence test.\n" +
						"Ensure basic storage functionality works correctly.")
			}
		}).

		// 2
		Test("Clean Shutdown Persistence", func(do *attest.Do) {
			do.Restart("primary")

			// Verify data survived the restart
			testData := map[string]string{
				"persistent:key1": "value1",
				"persistent:key2": "value with spaces",
				"persistent:key3": "🌍 unicode value",
				"persistent:key4": strings.Repeat("long_value_", 50),
			}

			for key, expectedValue := range testData {
				do.HTTP("primary", "GET", fmt.Sprintf("/kv/%s", key)).
					Returns().Status(http.StatusOK).Body(expectedValue).
					Assert("Your server should persist data across clean shutdowns.\n" +
						"Implement data persistence to disk (file-based storage, database, etc.).\n" +
						"Ensure data is written to persistent storage on PUT operations.")
			}
		}).

		// 3
		Test("SIGTERM Signal Handling", func(do *attest.Do) {
			// Add new data that should be persisted
			newTestData := map[string]string{
				"sigterm:key1": "sigterm_value1",
				"sigterm:key2": "data before signal",
				"sigterm:key3": "critical business data",
			}

			for key, value := range newTestData {
				do.HTTP("primary", "PUT", fmt.Sprintf("/kv/%s", key), value).
					Returns().Status(http.StatusOK).
					Assert("Your server should store new data for SIGTERM test.\n" +
						"Ensure PUT operations work correctly.")
			}

			// Send SIGTERM signal to simulate production shutdown
			do.Restart("primary")

			// Verify all data (old and new) survived SIGTERM
			allTestData := map[string]string{
				"persistent:key1": "value1",
				"persistent:key2": "value with spaces",
				"persistent:key3": "🌍 unicode value",
				"persistent:key4": strings.Repeat("long_value_", 50),
				"sigterm:key1":    "sigterm_value1",
				"sigterm:key2":    "data before signal",
				"sigterm:key3":    "critical business data",
			}

			for key, expectedValue := range allTestData {
				do.HTTP("primary", "GET", fmt.Sprintf("/kv/%s", key)).
					Returns().Status(http.StatusOK).Body(expectedValue).
					Assert("Your server should persist all data when handling SIGTERM.\n" +
						"Implement proper signal handling with graceful shutdown.\n" +
						"Ensure data is flushed to disk before process termination.")
			}
		}).

		// 4
		Test("Data Integrity After Multiple Restarts", func(do *attest.Do) {
			// Perform multiple cycles of data operations and restarts
			for cycle := 1; cycle <= 3; cycle++ {
				// Add cycle-specific data
				cycleKey := fmt.Sprintf("cycle:restart_%d", cycle)
				cycleValue := fmt.Sprintf("restart_data_%d", cycle)

				do.HTTP("primary", "PUT", fmt.Sprintf("/kv/%s", cycleKey), cycleValue).
					Returns().Status(http.StatusOK).
					Assert("Your server should store data for integrity test cycle.\n" +
						"Ensure PUT operations work correctly during multiple restart cycles.")

				// Restart the server
				do.Restart("primary")

				// Verify cycle data persisted
				do.HTTP("primary", "GET", fmt.Sprintf("/kv/%s", cycleKey)).
					Returns().Status(http.StatusOK).Body(cycleValue).
					Assert("Your server should maintain data integrity across multiple restarts.\n" +
						"Ensure persistent storage remains consistent and uncorrupted.")
			}

			// Verify all historical data still exists
			allHistoricalData := map[string]string{
				"persistent:key1": "value1",
				"persistent:key2": "value with spaces",
				"persistent:key3": "🌍 unicode value",
				"persistent:key4": strings.Repeat("long_value_", 50),
				"sigterm:key1":    "sigterm_value1",
				"sigterm:key2":    "data before signal",
				"sigterm:key3":    "critical business data",
				"cycle:restart_1": "restart_data_1",
				"cycle:restart_2": "restart_data_2",
				"cycle:restart_3": "restart_data_3",
			}

			for key, expectedValue := range allHistoricalData {
				do.HTTP("primary", "GET", fmt.Sprintf("/kv/%s", key)).
					Returns().Status(http.StatusOK).Body(expectedValue).
					Assert("Your server should preserve all historical data across restarts.\n" +
						"Ensure no data corruption or loss occurs during persistence operations.")
			}
		}).

		// 5
		Test("Persistence Under Load", func(do *attest.Do) {
			// Store data concurrently to test persistence under load
			putKV := func(key, value string) func() {
				return func() {
					do.HTTP("primary", "PUT", "/kv/load:"+key, value).
						Returns().Status(http.StatusOK).
						Assert("Your server should handle concurrent PUT requests under load.\n" +
							"Ensure persistence works correctly during high-traffic scenarios.")
				}
			}

			// Generate concurrent load
			do.Concurrently(
				putKV("concurrent1", "load_value1"),
				putKV("concurrent2", "load_value2"),
				putKV("concurrent3", "load_value3"),
				putKV("concurrent4", "load_value4"),
				putKV("concurrent5", "load_value5"),
				putKV("concurrent6", "load_value6"),
				putKV("concurrent7", "load_value7"),
				putKV("concurrent8", "load_value8"),
			)

			// Immediately restart to test persistence under concurrent load
			do.Restart("primary")

			// Verify all concurrent data was persisted
			for i := 1; i <= 8; i++ {
				do.HTTP("primary", "GET", fmt.Sprintf("/kv/load:concurrent%d", i)).
					Returns().Status(http.StatusOK).Body(fmt.Sprintf("load_value%d", i)).
					Assert("Your server should persist all concurrent writes correctly.\n" +
						"Ensure thread-safe persistence and no data loss under load.")
			}
		}).

		// 6
		Test("Empty Store Persistence", func(do *attest.Do) {
			// Clear all data
			do.HTTP("primary", "DELETE", "/clear").
				Returns().Status(http.StatusOK).
				Assert("Your server should implement a /clear endpoint.\n" +
					"Add a DELETE /clear method that deletes all key-value pairs.")

			// Verify store is empty
			do.HTTP("primary", "GET", "/kv/any:key").
				Returns().Status(http.StatusNotFound).Body("key not found\n").
				Assert("Your server should return 404 for non-existent keys after clear.\n" +
					"Ensure /clear endpoint removes all data.")

			// Restart with empty store
			do.Restart("primary")

			// Verify store remains empty after restart
			do.HTTP("primary", "GET", "/kv/any:key").
				Returns().Status(http.StatusNotFound).Body("key not found\n").
				Assert("Your server should handle empty store persistence correctly.\n" +
					"Ensure persistence layer handles empty state gracefully.")

			// Add data to empty store and verify it persists
			do.HTTP("primary", "PUT", "/kv/after:empty", "new_data").
				Returns().Status(http.StatusOK).
				Assert("Your server should accept new data after empty state restart.\n" +
					"Ensure persistence layer reinitializes correctly.")

			do.Restart("primary")

			do.HTTP("primary", "GET", "/kv/after:empty").
				Returns().Status(http.StatusOK).Body("new_data").
				Assert("Your server should persist data added after empty state restart.\n" +
					"Ensure persistence works correctly in all scenarios.")
		})
}
