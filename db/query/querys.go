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
