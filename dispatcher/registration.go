package dispatcher

import "github.com/webhookx-io/webhookx/db/entities"

type Registration struct {
	static map[string][]*entities.Endpoint
}

func NewRegistration(endpoints []*entities.Endpoint) *Registration {
	r := &Registration{
		static: make(map[string][]*entities.Endpoint),
	}

	for _, endpoint := range endpoints {
		for _, event := range endpoint.Events {
			r.static[event] = append(r.static[event], endpoint)
		}
	}
	return r
}

func (r *Registration) LookUp(event *entities.Event) []*entities.Endpoint {
	matched := r.static[event.EventType]
	return matched
}
