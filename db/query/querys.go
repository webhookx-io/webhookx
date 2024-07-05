package query

type EndpointQuery struct {
	Query
}

func (q *EndpointQuery) WhereMap() map[string]interface{} {
	return map[string]interface{}{}
}

type EventQuery struct {
	Query

	EndpointId *string
}

func (q EventQuery) WhereMap() map[string]interface{} {
	return map[string]interface{}{
		"endpoint_id": q.EndpointId,
	}
}
