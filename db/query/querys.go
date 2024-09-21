package query

type EndpointQuery struct {
	Query
}

func (q *EndpointQuery) WhereMap() map[string]interface{} {
	return map[string]interface{}{}
}

type EventQuery struct {
	Query
}

func (q EventQuery) WhereMap() map[string]interface{} {
	return map[string]interface{}{}
}

type WorkspaceQuery struct {
	Query
}

func (q *WorkspaceQuery) WhereMap() map[string]interface{} {
	return map[string]interface{}{}
}

type AttemptQuery struct {
	Query

	EventId    *string
	EndpointId *string
}

func (q *AttemptQuery) WhereMap() map[string]interface{} {
	maps := make(map[string]interface{})
	if q.EventId != nil {
		maps["event_id"] = *q.EventId
	}
	if q.EndpointId != nil {
		maps["endpoint_id"] = *q.EndpointId
	}
	return maps
}

type SourceQuery struct {
	Query
}

func (q *SourceQuery) WhereMap() map[string]interface{} {
	return map[string]interface{}{}
}
