package suite

import (
	"context"
	"time"
)

// timing defines when deferred operations should be executed
type timing int

const (
	TimingImmediate timing = iota
	TimingEventually
	TimingConsistently
)

// Default timeout for Eventually and Consistently operations
const defaultTimeout = 5 * time.Second

// Promise represents a deferred operation
type Promise[P any, A any] interface {
	// Eventually configures the promise to retry the operation until success or timeout
	Eventually() P
	// Within sets a custom timeout for Eventually operations
	Within(time.Duration) P
	// Consistently configures the promise to verify the operation succeeds for the entire duration
	Consistently() P
	// For sets a custom timeout for Consistently operations
	For(time.Duration) P
	// Returns creates an assertion to validate the operation's result
	Returns() A
}

// Compile-time type checks
var _ Promise[*HTTPPromise, *HTTPAssert] = (*HTTPPromise)(nil)
var _ Promise[*CLIPromise, *CLIAssert] = (*CLIPromise)(nil)

// PromiseBase provides common promise functionality
type PromiseBase struct {
	timing  timing
	timeout time.Duration
	ctx     context.Context
}

func (b *PromiseBase) setEventually() {
	b.timing = TimingEventually
	b.timeout = defaultTimeout
}

func (b *PromiseBase) setWithin(timeout time.Duration) {
	if b.timing != TimingEventually {
		panic("Within() can only be called after Eventually()")
	}

	b.timeout = timeout
}

func (b *PromiseBase) setConsistently() {
	b.timing = TimingConsistently
	b.timeout = defaultTimeout
}

func (b *PromiseBase) setFor(timeout time.Duration) {
	if b.timing != TimingConsistently {
		panic("For() can only be called after Consistently()")
	}

	b.timeout = timeout
}

// HTTPPromise represents a deferred HTTP request
type HTTPPromise struct {
	PromiseBase

	method  string
	url     string
	headers map[string]string
	body    []byte
}

func (p *HTTPPromise) Eventually() *HTTPPromise {
	p.setEventually()
	return p
}

func (p *HTTPPromise) Within(timeout time.Duration) *HTTPPromise {
	p.setWithin(timeout)
	return p
}

func (p *HTTPPromise) Consistently() *HTTPPromise {
	p.setConsistently()
	return p
}

func (p *HTTPPromise) For(timeout time.Duration) *HTTPPromise {
	p.setFor(timeout)
	return p
}

func (p *HTTPPromise) Returns() *HTTPAssert {
	return &HTTPAssert{promise: p}
}

// CLIPromise represents a deferred CLI command execution
type CLIPromise struct {
	PromiseBase

	command string
	args    []string
}

func (p *CLIPromise) Eventually() *CLIPromise {
	p.setEventually()
	return p
}

func (p *CLIPromise) Within(timeout time.Duration) *CLIPromise {
	p.setWithin(timeout)
	return p
}

func (p *CLIPromise) Consistently() *CLIPromise {
	p.setConsistently()
	return p
}

func (p *CLIPromise) For(timeout time.Duration) *CLIPromise {
	p.setFor(timeout)
	return p
}

func (p *CLIPromise) Returns() *CLIAssert {
	return &CLIAssert{promise: p}
}
