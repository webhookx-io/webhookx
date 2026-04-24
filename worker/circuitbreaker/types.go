package circuitbreaker

type Outcome string

const (
	Success Outcome = "success"
	Error   Outcome = "error"
)
