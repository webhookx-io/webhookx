package query

type EndpointQuery struct {
	Query

	Enabled     *bool
	WorkspaceId *string
}

func (q *EndpointQuery) WhereMap() map[string]interface{} {
	maps := make(map[string]interface{})
	if q.Enabled != nil {
		maps["enabled"] = *q.Enabled
	}
	if q.WorkspaceId != nil {
		maps["ws_id"] = *q.WorkspaceId
	}
	return maps
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
	Status     *string
}

func (q *AttemptQuery) WhereMap() map[string]interface{} {
	maps := make(map[string]interface{})
	if q.EventId != nil {
		maps["event_id"] = *q.EventId
	}
	if q.EndpointId != nil {
		maps["endpoint_id"] = *q.EndpointId
	}
	if q.Status != nil {
		maps["status"] = *q.Status
	}
	return maps
}

type SourceQuery struct {
	Query

	WorkspaceId *string
}

func (q *SourceQuery) WhereMap() map[string]interface{} {
	maps := make(map[string]interface{})
	if q.WorkspaceId != nil {
		maps["ws_id"] = *q.WorkspaceId
	}
	return maps
}

type PluginQuery struct {
	Query

	WorkspaceId *string
	EndpointId  *string
	Enabled     *bool
}

func (q *PluginQuery) WhereMap() map[string]interface{} {
	maps := make(map[string]interface{})
	if q.WorkspaceId != nil {
		maps["ws_id"] = *q.WorkspaceId
	}
	if q.EndpointId != nil {
		maps["endpoint_id"] = *q.EndpointId
	}
	if q.Enabled != nil {
		maps["enabled"] = *q.Enabled
	}
	return maps
}
