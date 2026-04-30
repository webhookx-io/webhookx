package metrics

type Event int

const (
	Success Event = iota
	Error
)

type Snapshot interface {
	Start() int64
	Until() int64
	SuccessCount() int64
	ErrorCount() int64
}

type Aggregator interface {
	// Record records an event occurrence at the given timestamp.
	Record(timestamp int64, event Event)

	// Snapshot returns current aggregated metrics.
	Snapshot() Snapshot
}
