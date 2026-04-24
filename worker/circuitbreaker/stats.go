package circuitbreaker

// Stats represents statistics over the time interval [start, end)
type Stats struct {
	StartTime int64 `json:"start_time"`
	EndTime   int64 `json:"end_time"`
	Success   int64 `json:"success"`
	Failure   int64 `json:"failure"`
}

func (s *Stats) TotalSuccess() int64 {
	return s.Success
}

func (s *Stats) TotalFailures() int64 {
	return s.Failure
}

func (s *Stats) TotalRequest() int64 {
	return s.Success + s.Failure
}

func (s *Stats) FailureRate() float64 {
	n := s.TotalRequest()
	if n == 0 {
		return 0
	}
	return float64(s.Failure) / float64(n)
}
