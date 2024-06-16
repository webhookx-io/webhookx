package query

type EndpointQuery struct {
	Query
}

func (q *EndpointQuery) WhereMap() map[string]interface{} {
	return map[string]interface{}{}
}
