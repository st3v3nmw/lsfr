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
		Test("Store Initial Testing Data", func(do *attest.Do) {
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
		Test("Verify Data Survives Restart", func(do *attest.Do) {
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
		Test("Check Data Integrity After Multiple Restarts", func(do *attest.Do) {
			// Perform multiple cycles of data operations and restarts
			for cycle := 1; cycle <= 4; cycle++ {
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

		// 4
		Test("Test Persistence When Under Load", func(do *attest.Do) {
			// Store data concurrently to test persistence under load
			putKV := func(key, value string) func() {
				return func() {
					do.HTTP("primary", "PUT", "/kv/load:"+key, value).
						Returns().Status(http.StatusOK).
						Assert("Your server should handle concurrent PUT requests under load (up to 10,000 requests).\n" +
							"Ensure persistence works correctly during high-traffic scenarios.")
				}
			}

			fns := []func(){}
			for i := 1; i <= 10_000; i++ {
				fns = append(fns, putKV(fmt.Sprintf("concurrent%d", i), fmt.Sprintf("value%d", i)))
			}

			// Generate concurrent load
			do.Concurrently(fns...)

			// Immediately restart to test persistence under concurrent load
			do.Restart("primary")

			// Verify all concurrent data was persisted
			for i := 1; i <= 10_000; i++ {
				do.HTTP("primary", "GET", fmt.Sprintf("/kv/load:concurrent%d", i)).
					Returns().Status(http.StatusOK).Body(fmt.Sprintf("value%d", i)).
					Assert("Your server should persist all concurrent writes correctly.\n" +
						"Ensure thread-safe persistence and no data loss under load (up to 10,000 concurrent requests).")
			}
		})
}
