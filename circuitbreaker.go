/*
Timeout's are not yet implemented (TODO /w context.Context?)
*/
package circuitbreaker

import (
	"errors"
	"log"
	"time"
)

// circuit state
type State int

const (
	// normal operation
	SClosed State = iota
	// dead, fallback is used
	SOpen
	// trial call will happen on next Execute
	SHalfOpen
)

type Config struct {
	OpenAfterNFails int
	// Execute timeout
	Timeout time.Duration
	// trial timeout in state open
	TryCloseAgainAfter time.Duration
	// notification of state change, e.g. for logging
	NotifyStateChange func(State)
	// called on Execute while the circuite is open
	FallbackFunc func() error
}

var (
	DefaultOpenAfterNFails    = 10
	DefaultTimeout            = time.Millisecond * 500
	DefaultTryCloseAgainAfter = DefaultTimeout * 10
	DefaultNotifyStateChange  = func(s State) {
		log.Printf("StateChange to %v", s)
	}
)

type CircuitBreaker struct {
	*Config
	Failures   int
	lastState  State
	retryTimer *time.Timer
}

func NewCircuitBreaker(c *Config) *CircuitBreaker {
	cb := CircuitBreaker{Config: c}
	if cb.Timeout == 0 {
		cb.Timeout = DefaultTimeout
	}
	if cb.OpenAfterNFails == 0 {
		cb.OpenAfterNFails = DefaultOpenAfterNFails
	}
	if cb.TryCloseAgainAfter == 0 {
		cb.TryCloseAgainAfter = DefaultTryCloseAgainAfter
	}
	if cb.NotifyStateChange == nil {
		cb.NotifyStateChange = DefaultNotifyStateChange
	}
	return &cb
}

var (
	ErrOpen  = errors.New("circuit is open")
	ErrPanic = errors.New("panic() recover()d")
)

func (c *CircuitBreaker) shouldRetry() bool {
	if c.retryTimer == nil {
		return false
	}
	select {
	case <-c.retryTimer.C:
		c.retryTimer.Stop()
		c.retryTimer = nil
		return true
	default:
		return false
	}
}

func (c *CircuitBreaker) state() State {
	if c.Failures > c.OpenAfterNFails {
		if c.shouldRetry() {
			return SHalfOpen
		}
		return SOpen
	} else {
		return SClosed
	}
}

func (c *CircuitBreaker) startRetryTimer() {
	dprintf("startRetryTimer /w %v", c.TryCloseAgainAfter)
	c.retryTimer = time.NewTimer(c.TryCloseAgainAfter)
}

func (c *CircuitBreaker) Execute(fn func() error) (err error) {
	s := c.state()
	dprintf("state=%#v", s)
	if s != c.lastState {
		c.NotifyStateChange(s)
		if s == SOpen {
			// last call opened the circuit
			c.startRetryTimer()
		}
	}
	c.lastState = s
	if s == SOpen {
		err = ErrOpen
		return c.FallbackFunc()
	}
	defer func() {
		if e := recover(); e != nil {
			// Todo: configurably let this immediately open the circuit
			log.Printf("recovered panic: %v", e)
			err = ErrPanic
		}
		if err != nil {
			c.Failures += 1
		}
	}()
	err = fn()
	if s == SHalfOpen {
		// todo: c.NotifyStateChange(SClosed)
		if err == nil {
			c.reset()
		} else {
			c.startRetryTimer()
		}
	}
	return
}

func (c *CircuitBreaker) reset() {
	c.Failures = 0
}

func dprintf(fmt string, args ...interface{}) {
	log.Printf(fmt, args...)
}
