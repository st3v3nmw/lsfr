package suite

import "context"

// timing defines when assertions should be executed
type timing int

const (
	TimingImmediate timing = iota
	TimingEventually
	TimingConsistently
)

// Promise represents a deferred operation
type Promise[T any, A any] interface {
	Eventually() T   // returns Promise
	Consistently() T // returns Promise

	Returns() A // returns Assert
}

// Compile-time type checks
var _ Promise[*HTTPPromise, *HTTPAssert] = (*HTTPPromise)(nil)
var _ Promise[*CLIPromise, *CLIAssert] = (*CLIPromise)(nil)

// HTTPPromise represents a deferred HTTP request
type HTTPPromise struct {
	method  string
	url     string
	headers map[string]string
	body    []byte
	timing  timing
	ctx     context.Context
}

func (p *HTTPPromise) Eventually() *HTTPPromise {
	p.timing = TimingEventually
	return p
}

func (p *HTTPPromise) Consistently() *HTTPPromise {
	p.timing = TimingConsistently
	return p
}

func (p *HTTPPromise) Returns() *HTTPAssert {
	return &HTTPAssert{
		promise: p,
	}
}

// CLIPromise represents a deferred CLI command execution
type CLIPromise struct {
	command string
	args    []string
	timing  timing
	ctx     context.Context
}

func (p *CLIPromise) Eventually() *CLIPromise {
	p.timing = TimingEventually
	return p
}

func (p *CLIPromise) Consistently() *CLIPromise {
	p.timing = TimingConsistently
	return p
}

func (p *CLIPromise) Returns() *CLIAssert {
	return &CLIAssert{
		promise: p,
	}
}
