package circuitbreaker

type State string

const (
	StateClosed   State = "CLOSED"
	StateHalfOpen State = "HALF_OPEN"
	StateOpen     State = "OPEN"
)

// TimeBucketMetric represents aggregated metrics over the time interval [Start, Until).
type TimeBucketMetric struct {
	Start   int64 `json:"start"`
	Until   int64 `json:"until"`
	Success int64 `json:"success"`
	Error   int64 `json:"error"`
}

func (m TimeBucketMetric) TotalRequest() int64 {
	return m.Success + m.Error
}

func (m TimeBucketMetric) FailureRate() float64 {
	n := m.TotalRequest()
	if n == 0 {
		return 0
	}
	return float64(m.Error) / float64(n)
}

type CircuitBreaker interface {
	// Name returns CircuitBreaker name
	Name() string

	// State returns CircuitBreaker State
	State() State

	// Metric returns CircuitBreaker metric
	Metric() TimeBucketMetric
}

type circuitBreaker struct {
	name   string
	state  State
	metric TimeBucketMetric
}

func (c *circuitBreaker) Name() string {
	return c.name
}

func (c *circuitBreaker) State() State {
	return c.state
}

func (c *circuitBreaker) Metric() TimeBucketMetric {
	return c.metric
}
