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

const runScriptPath = "./run.sh"

// Do provides the test harness and acts as the test runner
type Do struct {
	services *threadsafe.Map[string, *Service]

	ctx    context.Context
	cancel context.CancelFunc
}

// Service represents a running service's process
type Service struct {
	cmd  *exec.Cmd
	args []string

	realPort int
	fauxPort int
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

// Start starts a service process using the run.sh script with an OS-assigned port
func (do *Do) Start(service string, args ...string) {
	do.startWithPort(service, 0, args...)
}

// startWithPort starts a service on the specified port
func (do *Do) startWithPort(service string, port int, args ...string) {
	select {
	case <-do.ctx.Done():
		return
	default:
	}

	// Get OS-assigned port
	if port == 0 {
		listener, err := net.Listen("tcp", ":0")
		if err != nil {
			panic(fmt.Sprintf("Failed to get OS-assigned port: %v", err))
		}
		port = listener.Addr().(*net.TCPAddr).Port
		listener.Close()
	}

	// Start the service
	portArg := fmt.Sprintf("--port=%d", port)
	newArgs := append([]string{portArg}, args...)

	cmd := exec.CommandContext(do.ctx, runScriptPath, newArgs...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	err := cmd.Start()
	if err != nil {
		panic(err.Error())
	}

	svc := &Service{realPort: port, cmd: cmd, args: args}
	do.waitForPort(svc)

	do.services.Set(service, svc)
}

// waitForPort waits for a service to accept connections on its port
func (do *Do) waitForPort(svc *Service) {
	host := fmt.Sprintf("127.0.0.1:%d", svc.realPort)

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
					"Debug with: ./run.sh and check for error messages", host, svc.realPort,
			)
		}
	}
}

// Stop sends SIGTERM to a specific service, then SIGKILL after timeout
func (do *Do) Stop(service string) {
	svc := do.getService(service)
	if svc.cmd == nil || svc.cmd.Process == nil {
		return
	}

	pgid := svc.cmd.Process.Pid
	err := syscall.Kill(-pgid, syscall.SIGTERM)
	if err != nil {
		fmt.Println(red("Error stopping service running @"), red(svc.realPort))
		return
	}

	// Wait for graceful exit, force kill if timeout
	done := make(chan bool, 1)
	go func() {
		svc.cmd.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Process exited gracefully
	case <-time.After(5 * time.Second):
		do.Kill(service)
		<-done
	}
}

// Kill sends SIGKILL to a specific service immediately
func (do *Do) Kill(service string) {
	svc := do.getService(service)
	if svc.cmd == nil || svc.cmd.Process == nil {
		return
	}

	pgid := svc.cmd.Process.Pid
	err := syscall.Kill(-pgid, syscall.SIGKILL)
	if err != nil {
		fmt.Println(red("Error killing service running @"), red(svc.realPort))
	}
}

// Restart stops a service and starts it again
func (do *Do) Restart(service string, sig ...syscall.Signal) {
	svc := do.getService(service)
	if svc.cmd == nil {
		return
	}

	signal := syscall.SIGTERM
	if len(sig) > 0 {
		signal = sig[0]
	}

	switch signal {
	case syscall.SIGTERM:
		do.Stop(service)
	case syscall.SIGKILL:
		do.Kill(service)
	default:
		do.Stop(service)
	}

	time.Sleep(2_500 * time.Millisecond)

	do.startWithPort(service, svc.realPort, svc.args...)
}

// Done cleans up all running services
func (do *Do) Done() {
	do.cancel()

	var serviceNames []string
	do.services.Range(func(name string, _ *Service) bool {
		serviceNames = append(serviceNames, name)
		return true
	})

	for _, name := range serviceNames {
		do.Stop(name)
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

// HTTP creates a deferred HTTP request
func (do *Do) HTTP(service, method, path string, args ...any) *HTTPPromise {
	svc := do.getService(service)
	url := fmt.Sprintf("http://127.0.0.1:%d%s", svc.realPort, path)

	var body []byte
	if len(args) >= 1 {
		body = []byte(args[0].(string))
	}

	var headers map[string]string
	if len(args) >= 2 {
		headers = args[1].(map[string]string)
	}

	return &HTTPPromise{
		PromiseBase: PromiseBase{
			timing: TimingImmediate,
			ctx:    do.ctx,
		},

		method:  method,
		url:     url,
		headers: headers,
		body:    body,
	}
}

// Exec creates a deferred CLI command execution
func (do *Do) Exec(args ...string) *CLIPromise {
	return &CLIPromise{
		PromiseBase: PromiseBase{
			timing: TimingImmediate,
			ctx:    do.ctx,
		},

		command: runScriptPath,
		args:    args,
	}
}
