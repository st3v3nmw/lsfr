package suite

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"sync"
	"time"

	"github.com/st3v3nmw/lsfr/pkg/safe"
)

const scriptPath = "./run.sh"

type Do struct {
	services *safe.Map[string, *Service]
	verbose  bool
}

type Service struct {
	port int
	cmd  *exec.Cmd
}

func NewDo(verbose bool) *Do {
	return &Do{
		services: safe.NewMap[string, *Service](),
		verbose:  verbose,
	}
}

func (do *Do) getService(service string) *Service {
	if svc, exists := do.services.Get(service); exists {
		return svc
	}

	panic(fmt.Sprintf("service %q not found", service))
}

func (do *Do) Run(service string, port int, args ...string) *Do {
	cmd := exec.Command(scriptPath, args...)

	err := cmd.Start()
	if err != nil {
		panic(err.Error())
	}

	do.services.Set(service, &Service{port: port, cmd: cmd})

	return do
}

func (do *Do) WaitForPort(service string) {
	svc := do.getService(service)

	deadline := time.Now().Add(30 * time.Second)
	interval := 5 * time.Millisecond

	for time.Now().Before(deadline) {
		time.Sleep(interval)

		conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", svc.port))
		if err == nil {
			conn.Close()
			return
		}

		interval *= 2
	}

	panic(fmt.Sprintf("timeout waiting for port %d", svc.port))
}

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

func (do *Do) Eventually(condition func() bool) {
	deadline := time.Now().Add(30 * time.Second)
	interval := 5 * time.Millisecond

	for time.Now().Before(deadline) {
		if condition() {
			return
		}

		time.Sleep(interval)
		interval *= 2
	}

	panic("Eventually condition failed after timeout")
}

func (do *Do) Done() {
	do.services.Range(func(_ string, svc *Service) bool {
		svc.cmd.Process.Kill()
		return true
	})
}

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
		body:       string(responseBody),
		statusCode: resp.StatusCode,
	}
}

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
