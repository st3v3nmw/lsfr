package attest_test

import (
	"context"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	. "github.com/st3v3nmw/lsfr/internal/attest"
)

func TestHTTP(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		config     *Config
		testFunc   func(*Do, string)
		cancel     func(*Do)
		shouldPass bool
	}{
		{
			name: "Basic OK",
			handler: func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/kv/kenya:capital":
					switch r.Method {
					case "PUT":
						w.WriteHeader(http.StatusOK)
					case "GET":
						w.WriteHeader(http.StatusOK)
						w.Write([]byte("Nairobi"))
					default:
						w.WriteHeader(http.StatusMethodNotAllowed)
					}
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			},
			testFunc: func(do *Do, serviceName string) {
				do.HTTP(serviceName, "PUT", "/kv/kenya:capital", "Nairobi").
					Returns().Status(Is(200)).
					Assert("Server should handle PUT requests properly")

				do.HTTP(serviceName, "GET", "/kv/kenya:capital").
					Returns().Status(Is(200)).Body(Is("Nairobi")).
					Assert("Server should handle GET requests properly")

				do.HTTP(serviceName, "PATCH", "/kv/kenya:capital").
					Returns().Status(Is(405)).
					Assert("Server should return 405 for unsupported methods")

				do.HTTP(serviceName, "GET", "/unknown").
					Returns().Status(Is(404)).
					Assert("Server should return 404 for non-existent endpoints")
			},
			shouldPass: true,
		},
		{
			name: "Status Code Mismatch",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			testFunc: func(do *Do, serviceName string) {
				do.HTTP(serviceName, "GET", "/").
					Returns().Status(Is(200)).
					Assert("Should fail when expecting 200 OK but server returns 404")
			},
			shouldPass: false,
		},
		{
			name: "Body Mismatch",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("Mombasa"))
			},
			testFunc: func(do *Do, serviceName string) {
				do.HTTP(serviceName, "GET", "/").
					Returns().Status(Is(200)).Body(Is("Nairobi")).
					Assert("Should fail when expecting 'Nairobi' but server returns 'Mombasa'")
			},
			shouldPass: false,
		},
		{
			name: "Timeout",
			handler: func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(500 * time.Millisecond)

				w.WriteHeader(http.StatusOK)
				w.Write([]byte("Done"))
			},
			config: &Config{ExecuteTimeout: 50 * time.Millisecond},
			testFunc: func(do *Do, serviceName string) {
				do.HTTP(serviceName, "GET", "/").
					Returns().Status(Is(200)).Body(Is("Done")).
					Assert("Should fail when request times out before server responds")
			},
			shouldPass: false,
		},
		{
			name: "Eventually OK",
			handler: func() http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					readyAfter := time.Now().Add(500 * time.Millisecond)
					if time.Now().Before(readyAfter) {
						w.WriteHeader(http.StatusOK)
						w.Write([]byte("Ready"))
					} else {
						w.WriteHeader(http.StatusServiceUnavailable)
						w.Write([]byte("Starting up..."))
					}
				}
			}(),
			testFunc: func(do *Do, serviceName string) {
				do.HTTP(serviceName, "GET", "/").
					Eventually().
					Returns().Status(Is(200)).Body(Is("Ready")).
					Assert("Service should eventually become ready")
			},
			shouldPass: true,
		},
		{
			name: "Eventually Timeout",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte("Starting up..."))
			},
			testFunc: func(do *Do, serviceName string) {
				do.HTTP(serviceName, "GET", "/").
					Eventually().Within(500 * time.Millisecond).
					Returns().Status(Is(200)).Body(Is("Ready")).
					Assert("Should fail when service never becomes ready within timeout")
			},
			shouldPass: false,
		},
		{
			name: "Eventually Cancellation",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte("Starting up..."))
			},
			testFunc: func(do *Do, serviceName string) {
				do.HTTP(serviceName, "GET", "/").
					Eventually().Within(time.Second).
					Returns().Status(Is(200)).Body(Is("Ready")).
					Assert("Should fail when operation is cancelled before completion")
			},
			cancel: func(do *Do) {
				go func() {
					time.Sleep(500 * time.Millisecond)
					do.Cancel()
				}()
			},
			shouldPass: false,
		},
		{
			name: "Consistently OK",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("Stable"))
			},
			testFunc: func(do *Do, serviceName string) {
				do.HTTP(serviceName, "GET", "/").
					Consistently().For(500 * time.Millisecond).
					Returns().Status(Is(200)).Body(Is("Stable")).
					Assert("Service should remain consistently available")
			},
			shouldPass: true,
		},
		{
			name: "Consistently Failure",
			handler: func() http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					if rand.IntN(2) == 1 {
						w.WriteHeader(http.StatusOK)
						w.Write([]byte("Stable"))
					} else {
						w.WriteHeader(http.StatusServiceUnavailable)
						w.Write([]byte("Unstable"))
					}
				}
			}(),
			testFunc: func(do *Do, serviceName string) {
				do.HTTP(serviceName, "GET", "/").
					Consistently().
					Returns().Status(Is(200)).Body(Is("Stable")).
					Assert("Should fail when service returns intermittent errors")
			},
			shouldPass: false,
		},
		{
			name: "Consistently Cancellation",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("Stable"))
			},
			testFunc: func(do *Do, serviceName string) {
				do.HTTP(serviceName, "GET", "/").
					Consistently().For(3 * time.Second).
					Returns().Status(Is(200)).Body(Is("Stable")).
					Assert("Should pass when cancelled during consistency check")
			},
			cancel: func(do *Do) {
				go func() {
					time.Sleep(500 * time.Millisecond)
					do.Cancel()
				}()
			},
			shouldPass: true,
		},
		{
			name: "Contains Matcher - matches substring",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("Error: file not found"))
			},
			testFunc: func(do *Do, serviceName string) {
				do.HTTP(serviceName, "GET", "/").
					Returns().Status(Is(200)).Body(Contains("file not found")).
					Assert("Should accept response containing the substring")
			},
			shouldPass: true,
		},
		{
			name: "Contains Matcher - fails when substring not present",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("Success"))
			},
			testFunc: func(do *Do, serviceName string) {
				do.HTTP(serviceName, "GET", "/").
					Returns().Status(Is(200)).Body(Contains("error")).
					Assert("Should fail when substring is not in response")
			},
			shouldPass: false,
		},
		{
			name: "Matches Matcher - matches regex pattern",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("User ID: 12345"))
			},
			testFunc: func(do *Do, serviceName string) {
				do.HTTP(serviceName, "GET", "/").
					Returns().Status(Is(200)).Body(Matches(`User ID: \d+`)).
					Assert("Should match regex pattern")
			},
			shouldPass: true,
		},
		{
			name: "Matches Matcher - fails when pattern doesn't match",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("User ID: abc"))
			},
			testFunc: func(do *Do, serviceName string) {
				do.HTTP(serviceName, "GET", "/").
					Returns().Status(Is(200)).Body(Matches(`User ID: \d+`)).
					Assert("Should fail when pattern doesn't match")
			},
			shouldPass: false,
		},
		{
			name: "OneOf Matcher - matches one of several values",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("value2"))
			},
			testFunc: func(do *Do, serviceName string) {
				do.HTTP(serviceName, "GET", "/").
					Returns().Status(Is(200)).Body(OneOf("value1", "value2", "value3")).
					Assert("Should accept value2 as one of the valid options")
			},
			shouldPass: true,
		},
		{
			name: "OneOf Matcher - fails when value not in list",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("invalid"))
			},
			testFunc: func(do *Do, serviceName string) {
				do.HTTP(serviceName, "GET", "/").
					Returns().Status(Is(200)).Body(OneOf("value1", "value2", "value3")).
					Assert("Should fail when response is not in the list of valid values")
			},
			shouldPass: false,
		},
		{
			name: "Not Matcher - negates another matcher",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("Success"))
			},
			testFunc: func(do *Do, serviceName string) {
				do.HTTP(serviceName, "GET", "/").
					Returns().Status(Is(200)).Body(Not(Contains("error"))).
					Assert("Should pass when negated matcher doesn't match")
			},
			shouldPass: true,
		},
		{
			name: "Not Matcher - fails when negated matcher matches",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("Error occurred"))
			},
			testFunc: func(do *Do, serviceName string) {
				do.HTTP(serviceName, "GET", "/").
					Returns().Status(Is(200)).Body(Not(Contains("Error"))).
					Assert("Should fail when negated matcher matches")
			},
			shouldPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			serviceName := "test-service"
			port := strings.Split(server.URL, ":")[2]

			if tt.config == nil {
				tt.config = &Config{}
			}
			tt.config.WorkingDir = t.TempDir()

			suite := New().WithConfig(tt.config)

			success := suite.
				Setup(func(do *Do) {
					do.MockProcess(serviceName, port)
					if tt.cancel != nil {
						tt.cancel(do)
					}
				}).
				Test(tt.name, func(do *Do) {
					tt.testFunc(do, serviceName)
				}).
				Run(context.Background())

			if success != tt.shouldPass {
				if tt.shouldPass {
					t.Errorf("%s test should pass but failed", tt.name)
				} else {
					t.Errorf("%s test should fail but passed", tt.name)
				}
			}
		})
	}
}
