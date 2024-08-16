package router

import (
	"net/http"
	"slices"
)

type Router struct {
	routes []*Route
}

func NewRouter(routes []*Route) *Router {
	router := &Router{
		routes: routes,
	}
	return router
}

func (r *Router) Execute(req *http.Request) interface{} {
	path := req.URL.Path
	method := req.Method
	for _, route := range r.routes {
		if route.Paths[0] == path && slices.Contains(route.Methods, method) {
			return route.Handler
		}
	}
	return nil
}
