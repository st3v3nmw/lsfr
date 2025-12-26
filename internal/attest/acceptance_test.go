package attest_test

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/st3v3nmw/lsfr/internal/attest"
)

func TestHTTP(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		config     *attest.Config
		testFunc   func(*attest.Do, string)
		cancel     func(*attest.Do)
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
			testFunc: func(do *attest.Do, serviceName string) {
				do.HTTP(serviceName, "PUT", "/kv/kenya:capital", "Nairobi").
					Returns().Status(http.StatusOK).
					Assert("Server should handle PUT requests properly")

				do.HTTP(serviceName, "GET", "/kv/kenya:capital").
					Returns().Status(http.StatusOK).Body("Nairobi").
					Assert("Server should handle GET requests properly")

				do.HTTP(serviceName, "PATCH", "/kv/kenya:capital").
					Returns().Status(http.StatusMethodNotAllowed).
					Assert("Server should return 405 for unsupported methods")

				do.HTTP(serviceName, "GET", "/unknown").
					Returns().Status(http.StatusNotFound).
					Assert("Server should return 404 for non-existent endpoints")
			},
			shouldPass: true,
		},
		{
			name: "Status Code Mismatch",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			testFunc: func(do *attest.Do, serviceName string) {
				do.HTTP(serviceName, "GET", "/").
					Returns().Status(http.StatusOK).
					Assert("Should fail when expecting 200 OK but server returns 404")
			},
			shouldPass: false,
		},
		{
			name: "Body Mismatch",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("Mombasa"))
			},
			testFunc: func(do *attest.Do, serviceName string) {
				do.HTTP(serviceName, "GET", "/").
					Returns().Status(http.StatusOK).Body("Nairobi").
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
			config: &attest.Config{ExecuteTimeout: 50 * time.Millisecond},
			testFunc: func(do *attest.Do, serviceName string) {
				do.HTTP(serviceName, "GET", "/").
					Returns().Status(http.StatusOK).Body("Done").
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
			testFunc: func(do *attest.Do, serviceName string) {
				do.HTTP(serviceName, "GET", "/").
					Eventually().
					Returns().Status(http.StatusOK).Body("Ready").
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
			testFunc: func(do *attest.Do, serviceName string) {
				do.HTTP(serviceName, "GET", "/").
					Eventually().Within(500 * time.Millisecond).
					Returns().Status(http.StatusOK).Body("Ready").
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
			testFunc: func(do *attest.Do, serviceName string) {
				do.HTTP(serviceName, "GET", "/").
					Eventually().Within(time.Second).
					Returns().Status(http.StatusOK).Body("Ready").
					Assert("Should fail when operation is cancelled before completion")
			},
			cancel: func(do *attest.Do) {
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
			testFunc: func(do *attest.Do, serviceName string) {
				do.HTTP(serviceName, "GET", "/").
					Consistently().For(500 * time.Millisecond).
					Returns().Status(http.StatusOK).Body("Stable").
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
			testFunc: func(do *attest.Do, serviceName string) {
				do.HTTP(serviceName, "GET", "/").
					Consistently().
					Returns().Status(http.StatusOK).Body("Stable").
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
			testFunc: func(do *attest.Do, serviceName string) {
				do.HTTP(serviceName, "GET", "/").
					Consistently().For(3 * time.Second).
					Returns().Status(http.StatusOK).Body("Stable").
					Assert("Should pass when cancelled during consistency check")
			},
			cancel: func(do *attest.Do) {
				go func() {
					time.Sleep(500 * time.Millisecond)
					do.Cancel()
				}()
			},
			shouldPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			serviceName := "test-service"
			port := strings.Split(server.URL, ":")[2]

			suite := attest.New()
			if tt.config != nil {
				suite = suite.WithConfig(tt.config)
			}

			success := suite.
				Setup(func(do *attest.Do) {
					do.MockProcess(serviceName, port)
					if tt.cancel != nil {
						tt.cancel(do)
					}
				}).
				Test(tt.name, func(do *attest.Do) {
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

func TestCLI(t *testing.T) {
	tests := []struct {
		name       string
		config     *attest.Config
		testFunc   func(*attest.Do)
		cancel     func(*attest.Do)
		shouldPass bool
	}{
		{
			name:   "Basic OK",
			config: &attest.Config{Command: "echo"},
			testFunc: func(do *attest.Do) {
				do.Exec("Hello World").
					Returns().Exit(0).Output("Hello World\n").
					Assert("Echo command should return expected output")
			},
			shouldPass: true,
		},
		{
			name:   "Exit Code Mismatch",
			config: &attest.Config{Command: "sh"},
			testFunc: func(do *attest.Do) {
				do.Exec("-c", "false").
					Returns().Exit(0).
					Assert("Should fail when expecting exit code 0 but command returns 1")
			},
			shouldPass: false,
		},
		{
			name:   "Output Mismatch",
			config: &attest.Config{Command: "sh"},
			testFunc: func(do *attest.Do) {
				do.Exec("-c", "echo Wrong Output").
					Returns().Output("Expected Output").
					Assert("Should fail when command output doesn't match expected text")
			},
			shouldPass: false,
		},
		{
			name:   "Timeout",
			config: &attest.Config{Command: "sleep", ExecuteTimeout: 50 * time.Millisecond},
			testFunc: func(do *attest.Do) {
				do.Exec("20").
					Returns().Exit(0).Output("Expected Output").
					Assert("Should fail when command execution exceeds timeout")
			},
			shouldPass: false,
		},
		{
			name:   "Eventually OK",
			config: &attest.Config{Command: "sh"},
			testFunc: func(do *attest.Do) {
				testFile := "/tmp/attest_ready_" + fmt.Sprintf("%d", time.Now().UnixNano())

				go func() {
					time.Sleep(500 * time.Millisecond)
					exec.Command("touch", testFile).Run()
				}()
				defer exec.Command("rm", testFile).Run()

				do.Exec("-c", fmt.Sprintf("test -f '%s' && echo 'Ready' || (echo 'Not Ready'; exit 1)", testFile)).
					Eventually().
					Returns().Exit(0).Output("Ready\n").
					Assert("Command should eventually succeed when file exists")
			},
			shouldPass: true,
		},
		{
			name:   "Eventually Timeout",
			config: &attest.Config{Command: "sh"},
			testFunc: func(do *attest.Do) {
				do.Exec("-c", "echo 'Never Ready'; exit 1").
					Eventually().Within(time.Second).
					Returns().Exit(0).Output("Ready\n").
					Assert("Should fail when command never succeeds within timeout")
			},
			shouldPass: false,
		},
		{
			name:   "Eventually Cancellation",
			config: &attest.Config{Command: "sh"},
			testFunc: func(do *attest.Do) {
				do.Exec("-c", "echo 'Never Ready'; exit 1").
					Eventually().
					Returns().Exit(0).Output("Ready\n").
					Assert("Should fail when operation is cancelled before completion")
			},
			cancel: func(do *attest.Do) {
				go func() {
					time.Sleep(500 * time.Millisecond)
					do.Cancel()
				}()
			},
			shouldPass: false,
		},
		{
			name:   "Consistently OK",
			config: &attest.Config{Command: "echo"},
			testFunc: func(do *attest.Do) {
				do.Exec("Stable").
					Consistently().For(500 * time.Millisecond).
					Returns().Exit(0).Output("Stable\n").
					Assert("Command should consistently produce stable output")
			},
			shouldPass: true,
		},
		{
			name:   "Consistently Failure",
			config: &attest.Config{Command: "sh"},
			testFunc: func(do *attest.Do) {
				do.Exec("-c", "date +%N").
					Consistently().For(500 * time.Millisecond).
					Returns().Output("12345\n").
					Assert("Should fail when command output changes between executions")
			},
			shouldPass: false,
		},
		{
			name:   "Consistently Cancellation",
			config: &attest.Config{Command: "echo"},
			testFunc: func(do *attest.Do) {
				do.Exec("Stable").
					Consistently().For(3 * time.Second).
					Returns().Exit(0).Output("Stable\n").
					Assert("Should pass when cancelled during consistency check")
			},
			cancel: func(do *attest.Do) {
				go func() {
					time.Sleep(500 * time.Millisecond)
					do.Cancel()
				}()
			},
			shouldPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suite := attest.New()
			if tt.config != nil {
				suite = suite.WithConfig(tt.config)
			}

			success := suite.
				Setup(func(do *attest.Do) {
					if tt.cancel != nil {
						tt.cancel(do)
					}
				}).
				Test(tt.name, func(do *attest.Do) {
					tt.testFunc(do)
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
