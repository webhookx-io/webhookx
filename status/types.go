package status

type HealthResponse struct {
	Status     string                  `json:"status"`
	Components map[string]HealthResult `json:"components"`
}

type HealthResult struct {
	Status string  `json:"status"`
	Error  *string `json:"error,omitempty"`
}

type StatusResponse struct {
	UpTime                 string        `json:"uptime"`
	Runtime                RuntimeStats  `json:"runtime"`
	Memory                 MemoryStats   `json:"memory"`
	Database               DatabaseStats `json:"database"`
	InboundRequests        int64         `json:"inbound_requests"`
	InboundFailedRequests  int64         `json:"inbound_failed_requests"`
	OutboundRequests       int64         `json:"outbound_requests"`
	OutboundFailedRequests int64         `json:"outbound_failed_requests"`
	Queue                  QueueStats    `json:"queue"`
	Event                  EventStats    `json:"event"`
}

type MemoryStats struct {
	Alloc       string `json:"alloc"`
	Sys         string `json:"sys"`
	HeapAlloc   string `json:"heap_alloc"`
	HeapObjects int64  `json:"heap_objects"`
	GC          int64  `json:"gc"`
}

type DatabaseStats struct {
	TotalConnections  int `json:"total_connections"`
	ActiveConnections int `json:"active_connections"`
}

type RuntimeStats struct {
	Go         string `json:"go"`
	Goroutines int    `json:"goroutines"`
}

type QueueStats struct {
	Size           int64 `json:"size"`
	BacklogLatency int64 `json:"backlog_latency_secs"`
}

type EventStats struct {
	Pending int64 `json:"pending"`
}
