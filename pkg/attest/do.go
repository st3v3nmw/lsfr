package attest

import (
	"context"
	"fmt"
	"log"
	"net"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// Do provides the test harness and acts as the test runner
type Do struct {
	processes *syncMap[string, *Process]
	config    *Config

	ctx    context.Context
	cancel context.CancelFunc
}

// newDo creates a new Do instance with custom configuration
func newDo(ctx context.Context, config *Config) *Do {
	doCtx, cancel := context.WithCancel(ctx)
	return &Do{
		processes: newMap[string, *Process](),
		config:    config,
		ctx:       doCtx,
		cancel:    cancel,
	}
}

// Process represents a running process
type Process struct {
	cmd  *exec.Cmd
	args []string

	realPort int
	fauxPort int
}

// getProcess retrieves a process by name or panics if not found
func (do *Do) getProcess(name string) *Process {
	if proc, exists := do.processes.Get(name); exists {
		return proc
	}

	panic(fmt.Sprintf("process %q not found", name))
}

// Start starts the process with an OS-assigned port
func (do *Do) Start(name string, args ...string) {
	do.startWithPort(name, 0, args...)
}

// startWithPort starts the process on the specified port
func (do *Do) startWithPort(name string, port int, args ...string) {
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

	// Start the process
	portArg := fmt.Sprintf("--port=%d", port)
	newArgs := append([]string{portArg}, args...)

	cmd := exec.CommandContext(do.ctx, do.config.Command, newArgs...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	err := cmd.Start()
	if err != nil {
		panic(err.Error())
	}

	proc := &Process{realPort: port, cmd: cmd, args: args}
	do.waitForPort(proc)

	do.processes.Set(name, proc)
}

// waitForPort waits for a process to accept connections on its port
func (do *Do) waitForPort(proc *Process) {
	host := fmt.Sprintf("127.0.0.1:%d", proc.realPort)

	succeeded := eventually(do.ctx, func() bool {
		conn, err := net.DialTimeout("tcp", host, 100*time.Millisecond)
		if err != nil {
			return false
		}

		conn.Close()
		return true
	}, do.config.ProcessStartTimeout, do.config.RetryPollInterval)

	if !succeeded {
		select {
		case <-do.ctx.Done():
			return
		default:
			log.Fatalf(
				"\nCould not connect to http://%s.\n\n"+
					"Possible issues:\n"+
					"- run.sh script not executable (run: chmod +x run.sh)\n"+
					"- Process not starting on port %d\n"+
					"- Process crashing during startup\n\n"+
					"Debug with: ./run.sh and check for error messages", host, proc.realPort,
			)
		}
	}
}

// Stop sends SIGTERM to the process, then SIGKILL after timeout
func (do *Do) Stop(name string) {
	proc := do.getProcess(name)
	if proc.cmd == nil || proc.cmd.Process == nil {
		return
	}

	pgid := proc.cmd.Process.Pid
	err := syscall.Kill(-pgid, syscall.SIGTERM)
	if err != nil {
		fmt.Println(red("Error stopping process running @"), red(proc.realPort))
		return
	}

	// Wait for graceful exit, force kill if timeout
	done := make(chan bool, 1)
	go func() {
		proc.cmd.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Process exited gracefully
	case <-time.After(do.config.ProcessShutdownTimeout):
		do.Kill(name)
		<-done
	}
}

// Kill sends SIGKILL to kill the process immediately
func (do *Do) Kill(name string) {
	proc := do.getProcess(name)
	if proc.cmd == nil || proc.cmd.Process == nil {
		return
	}

	pgid := proc.cmd.Process.Pid
	err := syscall.Kill(-pgid, syscall.SIGKILL)
	if err != nil {
		fmt.Println(red("Error killing process running @"), red(proc.realPort))
	}
}

// Restart stops the process and starts it again
func (do *Do) Restart(name string, sig ...syscall.Signal) {
	proc := do.getProcess(name)
	if proc.cmd == nil {
		return
	}

	signal := syscall.SIGTERM
	if len(sig) > 0 {
		signal = sig[0]
	}

	switch signal {
	case syscall.SIGTERM:
		do.Stop(name)
	case syscall.SIGKILL:
		do.Kill(name)
	default:
		do.Stop(name)
	}

	time.Sleep(do.config.ProcessRestartDelay)

	do.startWithPort(name, proc.realPort, proc.args...)
}

// Done cleans up all running processes
func (do *Do) Done() {
	do.cancel()

	var processNames []string
	do.processes.Range(func(name string, _ *Process) bool {
		processNames = append(processNames, name)
		return true
	})

	for _, name := range processNames {
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
func (do *Do) HTTP(name, method, path string, args ...any) *HTTPPromise {
	proc := do.getProcess(name)
	url := fmt.Sprintf("http://127.0.0.1:%d%s", proc.realPort, path)

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
			config: do.config,
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
			config: do.config,
		},

		command: do.config.Command,
		args:    args,
	}
}
