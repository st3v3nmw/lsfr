package suite

import (
	"context"
	"fmt"
	"log"
	"net"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/st3v3nmw/lsfr/pkg/threadsafe"
)

const scriptPath = "./run.sh"

// Do provides the test harness and acts as the test runner
type Do struct {
	services *threadsafe.Map[string, *Service]
	ctx      context.Context
	cancel   context.CancelFunc
}

// Service represents a running service process
type Service struct {
	port int
	cmd  *exec.Cmd
}

// NewDo creates a new Do instance with context-aware cleanup
func NewDo(ctx context.Context) *Do {
	doCtx, cancel := context.WithCancel(ctx)
	return &Do{
		services: threadsafe.NewMap[string, *Service](),
		ctx:      doCtx,
		cancel:   cancel,
	}
}

// getService retrieves a service by name or panics if not found
func (do *Do) getService(service string) *Service {
	if svc, exists := do.services.Get(service); exists {
		return svc
	}

	panic(fmt.Sprintf("service %q not found", service))
}

// Start starts a service process using the run.sh script
func (do *Do) Start(service string, port int, args ...string) *Do {
	select {
	case <-do.ctx.Done():
		return do
	default:
	}

	cmd := exec.CommandContext(do.ctx, scriptPath, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	err := cmd.Start()
	if err != nil {
		panic(err.Error())
	}

	do.services.Set(service, &Service{port: port, cmd: cmd})
	return do
}

// WaitForPort waits for a service to accept connections on its port with timeout
func (do *Do) WaitForPort(service string) {
	svc := do.getService(service)
	host := fmt.Sprintf("127.0.0.1:%d", svc.port)

	succeeded := Eventually(do.ctx, func() bool {
		conn, err := net.DialTimeout("tcp", host, 100*time.Millisecond)
		if err != nil {
			return false
		}

		conn.Close()
		return true
	}, 15*time.Second)

	if !succeeded {
		select {
		case <-do.ctx.Done():
			return
		default:
			log.Fatalf(
				"\nCould not connect to http://%s.\n\n"+
					"Possible issues:\n"+
					"- run.sh script not executable (run: chmod +x run.sh)\n"+
					"- Server not starting on port %d\n"+
					"- Server crashing during startup\n\n"+
					"Debug with: ./run.sh and check for error messages", host, svc.port,
			)
		}
	}
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

// Done cleans up all running services
func (do *Do) Done() {
	do.cancel()

	do.services.Range(func(_ string, svc *Service) bool {
		do.stopService(svc)
		return true
	})
}

// stopService stops a single service with proper cleanup
func (do *Do) stopService(svc *Service) {
	if svc.cmd == nil || svc.cmd.Process == nil {
		return
	}

	pgid := svc.cmd.Process.Pid

	// Send SIGTERM to process group for graceful shutdown
	err := syscall.Kill(-pgid, syscall.SIGTERM)
	if err != nil {
		fmt.Println(red("Error stopping service running @"), red(svc.port))
		return
	}

	// Wait for graceful exit, force kill if timeout
	done := make(chan error, 1)
	go func() {
		done <- svc.cmd.Wait()
	}()

	select {
	case <-done:
		// Process exited gracefully
	case <-time.After(5 * time.Second):
		fmt.Printf("Service on port %d not responding to SIGTERM, force killing...\n", svc.port)
		syscall.Kill(-pgid, syscall.SIGKILL)
		<-done
	}
}

// HTTP creates a deferred HTTP request
func (do *Do) HTTP(service, method, path string, args ...any) *HTTPPromise {
	svc := do.getService(service)
	url := fmt.Sprintf("http://127.0.0.1:%d%s", svc.port, path)

	// Extract optional request body and headers
	var body []byte
	if len(args) >= 1 {
		body = []byte(args[0].(string))
	}

	var headers map[string]string
	if len(args) >= 2 {
		headers = args[1].(map[string]string)
	}

	return &HTTPPromise{
		method:  method,
		url:     url,
		headers: headers,
		body:    body,
		timing:  TimingImmediate,
		ctx:     do.ctx,
	}
}

// Exec creates a deferred CLI command execution
func (do *Do) Exec(args ...string) *CLIPromise {
	return &CLIPromise{
		command: scriptPath,
		args:    args,
		timing:  TimingImmediate,
		ctx:     do.ctx,
	}
}
