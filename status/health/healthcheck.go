package health

type Status string

const (
	StatusUp   = "UP"
	StatusDown = "DOWN"
)

type Indicator struct {
	Name  string
	Check func() error
}
