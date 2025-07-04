package api

import (
	"github.com/gorilla/mux"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/dispatcher"
	"github.com/webhookx-io/webhookx/eventbus"
	"github.com/webhookx-io/webhookx/pkg/declarative"
	"github.com/webhookx-io/webhookx/pkg/errs"
	"github.com/webhookx-io/webhookx/pkg/http/middlewares"
	"github.com/webhookx-io/webhookx/pkg/http/response"
	"github.com/webhookx-io/webhookx/pkg/types"
	"net/http"
	"net/http/pprof"
	"strconv"
)

type API struct {
	cfg         *config.Config
	db          *db.DB
	dispatcher  *dispatcher.Dispatcher
	declarative *declarative.Declarative
	bus         eventbus.Bus
	middlewares []mux.MiddlewareFunc
}

type Options struct {
	Config      *config.Config
	DB          *db.DB
	Dispatcher  *dispatcher.Dispatcher
	Middlewares []mux.MiddlewareFunc
	EventBus    eventbus.Bus
}

func NewAPI(opts Options) *API {
	return &API{
		cfg:         opts.Config,
		db:          opts.DB,
		dispatcher:  opts.Dispatcher,
		declarative: declarative.NewDeclarative(opts.DB),
		bus:         opts.EventBus,
		middlewares: opts.Middlewares,
	}
}

// param returns the value of an url variable
func (api *API) param(r *http.Request, variable string) string {
	return mux.Vars(r)[variable]
}

// query returns the url query value if it exists.
func (api *API) query(r *http.Request, name string) string {
	return r.URL.Query().Get(name)
}

func (api *API) json(code int, w http.ResponseWriter, data interface{}) {
	response.JSON(w, code, data)
}

func (api *API) bindQuery(r *http.Request, q *query.Query) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page_no"))
	if page <= 0 {
		page = 1
	}

	pagesize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pagesize <= 0 {
		pagesize = 20
	}

	q.Page(uint64(page), uint64(pagesize))
}

func (api *API) error(code int, w http.ResponseWriter, err error) {
	if e, ok := err.(*errs.ValidateError); ok {
		api.json(code, w, types.ErrorResponse{
			Message: "Request Validation",
			Error:   e,
		})
		return
	}
	api.json(code, w, types.ErrorResponse{Message: err.Error()})
}

func (api *API) assert(err error) {
	if err != nil {
		panic(err)
	}
}

// Handler returns a http.Handler
func (api *API) Handler() http.Handler {
	r := mux.NewRouter()

	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, 404, types.ErrorResponse{Message: "not found"})
	})

	for _, m := range api.middlewares {
		r.Use(m)
	}
	r.Use(middlewares.PanicRecovery)
	r.Use(api.contextMiddleware)

	r.HandleFunc("/", api.Index).Methods("GET")

	r.HandleFunc("/workspaces/{workspace}/config/sync", api.Sync).Methods("POST")
	r.HandleFunc("/workspaces/{workspace}/config/dump", api.Dump).Methods("POST")

	r.HandleFunc("/workspaces", api.PageWorkspace).Methods("GET")
	r.HandleFunc("/workspaces", api.CreateWorkspace).Methods("POST")
	r.HandleFunc("/workspaces/{id}", api.GetWorkspace).Methods("GET")
	r.HandleFunc("/workspaces/{id}", api.UpdateWorkspace).Methods("PUT")
	r.HandleFunc("/workspaces/{id}", api.DeleteWorkspace).Methods("DELETE")

	if api.cfg.Admin.DebugEndpoints {
		r.HandleFunc("/debug/pprof/profile", pprof.Profile).Methods("GET")
		r.HandleFunc("/debug/pprof/symbol", pprof.Symbol).Methods("GET")
		r.HandleFunc("/debug/pprof/trace", pprof.Trace).Methods("GET")
		r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline).Methods("GET")
		r.PathPrefix("/debug/pprof/").HandlerFunc(pprof.Index).Methods("GET")
	}

	for _, prefix := range []string{"", "/workspaces/{workspace}"} {
		r.HandleFunc(prefix+"/endpoints", api.PageEndpoint).Methods("GET")
		r.HandleFunc(prefix+"/endpoints", api.CreateEndpoint).Methods("POST")
		r.HandleFunc(prefix+"/endpoints/{id}", api.GetEndpoint).Methods("GET")
		r.HandleFunc(prefix+"/endpoints/{id}", api.UpdateEndpoint).Methods("PUT")
		r.HandleFunc(prefix+"/endpoints/{id}", api.DeleteEndpoint).Methods("DELETE")
	}

	for _, prefix := range []string{"", "/workspaces/{workspace}"} {
		r.HandleFunc(prefix+"/sources", api.PageSource).Methods("GET")
		r.HandleFunc(prefix+"/sources", api.CreateSource).Methods("POST")
		r.HandleFunc(prefix+"/sources/{id}", api.GetSource).Methods("GET")
		r.HandleFunc(prefix+"/sources/{id}", api.UpdateSource).Methods("PUT")
		r.HandleFunc(prefix+"/sources/{id}", api.DeleteSource).Methods("DELETE")
	}

	for _, prefix := range []string{"", "/workspaces/{workspace}"} {
		r.HandleFunc(prefix+"/events", api.PageEvent).Methods("GET")
		r.HandleFunc(prefix+"/events", api.CreateEvent).Methods("POST")
		r.HandleFunc(prefix+"/events/{id}", api.GetEvent).Methods("GET")
		r.HandleFunc(prefix+"/events/{id}/retry", api.RetryEvent).Methods("POST")
	}

	for _, prefix := range []string{"", "/workspaces/{workspace}"} {
		r.HandleFunc(prefix+"/attempts", api.PageAttempt).Methods("GET")
		r.HandleFunc(prefix+"/attempts/{id}", api.GetAttempt).Methods("GET")
	}

	for _, prefix := range []string{"", "/workspaces/{workspace}"} {
		r.HandleFunc(prefix+"/plugins", api.PagePlugin).Methods("GET")
		r.HandleFunc(prefix+"/plugins", api.CreatePlugin).Methods("POST")
		r.HandleFunc(prefix+"/plugins/{id}", api.GetPlugin).Methods("GET")
		r.HandleFunc(prefix+"/plugins/{id}", api.UpdatePlugin).Methods("PUT")
		r.HandleFunc(prefix+"/plugins/{id}", api.DeletePlugin).Methods("DELETE")
	}

	return r
}
