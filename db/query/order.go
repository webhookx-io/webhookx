package query

type Sort = string

const (
	ASC  Sort = "ASC"
	DESC Sort = "DESC"
)

type Order struct {
	Column string
	Sort   Sort
}

func (o Order) String() string {
	return o.Column + " " + o.Sort
}
