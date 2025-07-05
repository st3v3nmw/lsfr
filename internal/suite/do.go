package suite

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/st3v3nmw/lsfr/pkg/threadsafe"
)

const scriptPath = "./run.sh"

// Do provides test operations for running services and making assertions
type Do struct {
	services *threadsafe.Map[string, *Service]
}

// Service represents a running service process
type Service struct {
	port int
	cmd  *exec.Cmd
}

// NewDo creates a new Do instance
func NewDo() *Do {
	return &Do{
		services: threadsafe.NewMap[string, *Service](),
	}
}

// getService retrieves a service by name
func (do *Do) getService(service string) *Service {
	if svc, exists := do.services.Get(service); exists {
		return svc
	}

	panic(fmt.Sprintf("service %q not found", service))
}

// Run starts a service process using the run.sh script
func (do *Do) Run(service string, port int, args ...string) *Do {
	cmd := exec.Command(scriptPath, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	err := cmd.Start()
	if err != nil {
		panic(err.Error())
	}

	do.services.Set(service, &Service{port: port, cmd: cmd})

	return do
}

// WaitForPort waits for a service to accept connections on its port
func (do *Do) WaitForPort(service string) {
	svc := do.getService(service)
	host := fmt.Sprintf("127.0.0.1:%d", svc.port)

	deadline := time.Now().Add(30 * time.Second)
	interval := 5 * time.Millisecond
	for time.Now().Before(deadline) {
		if interval > time.Second {
			fmt.Printf("Attempting connection to %s in %s...\n", host, interval.Round(time.Second))
		}
		time.Sleep(interval)

		conn, err := net.Dial("tcp", host)
		if err == nil {
			conn.Close()

			if interval > time.Second {
				fmt.Println()
			}
			return
		}

		interval *= 2
	}

	log.Fatalf(
		"\nCould not connect to http://%s.\n\n"+
			"Possible issues:\n"+
			"- run.sh script not executable (run: chmod +x run.sh)\n"+
			"- Server not starting on port %d\n"+
			"- Server crashing during startup\n\n"+
			"Debug with: ./run.sh and check for error messages", host, svc.port,
	)
}

// Concurrently runs multiple functions in parallel and waits for completion
func (do *Do) Concurrently(fns ...func()) {
	var wg sync.WaitGroup

	for _, fn := range fns {
		wg.Add(1)
		go func(f func()) {
			defer wg.Done()

			f()
		}(fn)
	}

	wg.Wait()
}

// Eventually waits for a condition to become true within a timeout
func (do *Do) Eventually(condition func() bool) {
	deadline := time.Now().Add(30 * time.Second)
	interval := 5 * time.Millisecond

	for time.Now().Before(deadline) {
		time.Sleep(interval)

		if condition() {
			return
		}

		interval *= 2
	}

	panic("Eventually condition failed after timeout")
}

// Done cleans up all running services
func (do *Do) Done() {
	do.services.Range(func(_ string, svc *Service) bool {
		pgid := svc.cmd.Process.Pid
		err := syscall.Kill(-pgid, syscall.SIGTERM)
		if err != nil {
			fmt.Println(red("Error stopping service running @"), red(svc.port))
			return true
		}

		done := make(chan error, 1)
		go func() {
			done <- svc.cmd.Wait()
		}()

		select {
		case <-done:
		case <-time.After(30 * time.Second):
			syscall.Kill(-pgid, syscall.SIGKILL)
			<-done
		}

		return true
	})
}

// HTTP makes an HTTP request to a service
func (do *Do) HTTP(service, method, path string, args ...any) *HTTPAssert {
	svc := do.getService(service)
	client := &http.Client{Timeout: 30 * time.Second}

	var body []byte
	if len(args) >= 1 {
		body = []byte(args[0].(string))
	}

	url := fmt.Sprintf("http://127.0.0.1:%d%s", svc.port, path)
	req, err := http.NewRequest(method, url, bytes.NewReader([]byte(body)))

	if err != nil {
		return &HTTPAssert{ErrAssert: ErrAssert{err}}
	}

	if len(args) >= 2 {
		headers := args[1].(map[string]string)
		for k, v := range headers {
			req.Header.Add(k, v)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return &HTTPAssert{ErrAssert: ErrAssert{err}}
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &HTTPAssert{ErrAssert: ErrAssert{err}}
	}

	return &HTTPAssert{
		requestMethod: method,
		requestURL:    path,
		requestBody:   string(body),

		responseBody:   string(responseBody),
		responseStatus: resp.StatusCode,
	}
}

// Exec runs a command using the run.sh script
func (do *Do) Exec(args ...string) *CLIAssert {
	cmd := exec.Command(scriptPath, args...)

	stdout, err := cmd.Output()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			return &CLIAssert{
				output:   string(exitError.Stderr),
				exitCode: exitError.ExitCode(),
			}
		}

		return &CLIAssert{ErrAssert: ErrAssert{err}}
	}

	return &CLIAssert{output: string(stdout)}
}
