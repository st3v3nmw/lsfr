package suite

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type Do struct {
	binary  string
	running []*exec.Cmd
	mu      sync.Mutex
}

func NewDo(binary string) *Do {
	return &Do{
		binary:  binary,
		running: make([]*exec.Cmd, 0),
	}
}

func (do *Do) Run(args ...string) error {
	cmd := exec.Command(do.binary, args...)

	err := cmd.Start()
	if err != nil {
		return err
	}

	do.mu.Lock()
	do.running = append(do.running, cmd)
	do.mu.Unlock()

	return nil
}

func (do *Do) WaitForPort(port int) error {
	deadline := time.Now().Add(30 * time.Second)
	interval := 5 * time.Millisecond

	for time.Now().Before(deadline) {
		time.Sleep(interval)

		conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
		if err == nil {
			conn.Close()
			return nil
		}

		interval *= 2
	}

	return fmt.Errorf("timeout waiting for port %d", port)
}

func (do *Do) Concurrent(fns ...func()) {
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

func (do *Do) Done() {
	do.mu.Lock()
	defer do.mu.Unlock()

	for _, cmd := range do.running {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}
}

func (do *Do) CLI(args ...string) *CLIResult {
	cmd := exec.Command(do.binary, args...)

	stdout, err := cmd.Output()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			return &CLIResult{
				DoErr:    DoErr{string(exitError.Stderr)},
				exitCode: exitError.ExitCode(),
			}
		}

		return &CLIResult{DoErr: DoErr{err.Error()}}
	}

	return &CLIResult{output: string(stdout)}
}

func (do *Do) HTTP(method, url string, body ...string) *HTTPResult {
	client := &http.Client{Timeout: 30 * time.Second}

	bodyStr := strings.Join(body, "")
	req, err := http.NewRequest(method, url, bytes.NewReader([]byte(bodyStr)))
	if err != nil {
		return &HTTPResult{DoErr: DoErr{err.Error()}}
	}

	resp, err := client.Do(req)
	if err != nil {
		return &HTTPResult{DoErr: DoErr{err.Error()}}
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &HTTPResult{
			DoErr:      DoErr{err.Error()},
			statusCode: resp.StatusCode,
		}
	}

	return &HTTPResult{
		body:       string(responseBody),
		statusCode: resp.StatusCode,
	}
}

type DoErr struct {
	err string
}

func (e *DoErr) Error(expected string) *DoErr {
	if e.err != expected {
		panic(fmt.Sprintf("expected err %q, got %q", expected, e.err))
	}

	return e
}

type CLIResult struct {
	DoErr
	output   string
	exitCode int
}

func (r *CLIResult) Got() *CLIResult {
	return r
}

func (r *CLIResult) Output(expected string) *CLIResult {
	if r.output != expected {
		panic(fmt.Sprintf("expected output %q, got %q", expected, r.output))
	}

	return r
}

func (r *CLIResult) Exit(code int) *CLIResult {
	if r.exitCode != code {
		panic(fmt.Sprintf("expected exit code %d, got %d", code, r.exitCode))
	}

	return r
}

type HTTPResult struct {
	DoErr
	body       string
	statusCode int
}

func (r *HTTPResult) Got() *HTTPResult {
	return r
}

func (r *HTTPResult) Body(expected string) *HTTPResult {
	if r.body != expected {
		panic(fmt.Sprintf("expected body %q, got %q", expected, r.body))
	}

	return r
}

func (r *HTTPResult) Status(code int) *HTTPResult {
	fmt.Printf("%#v\n", *r)
	if r.statusCode != code {
		panic(fmt.Sprintf("expected status code %d, got %d", code, r.statusCode))
	}

	return r
}
