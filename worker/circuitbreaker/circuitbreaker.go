package circuitbreaker

type State string

const (
	StateClosed   State = "CLOSED"
	StateHalfOpen State = "HALF_OPEN"
	StateOpen     State = "OPEN"
)

type CircuitBreaker interface {
	Name() string
	State() State
	Stats() Stats
}

type CircuitBreakerImpl struct {
	name  string
	state State
	stats Stats
}

func (c *CircuitBreakerImpl) Name() string {
	return c.name
}

func (c *CircuitBreakerImpl) State() State {
	return c.state
}

func (c *CircuitBreakerImpl) Stats() Stats {
	return c.stats
}
