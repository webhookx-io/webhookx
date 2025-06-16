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
	UpTime string       `json:"uptime"`
	Memory MemoryStatus `json:"memory"`
}

type MemoryStatus struct {
	Alloc string `json:"alloc"`
	Sys   string `json:"sys"`
	GC    int64  `json:"gc"`
}
