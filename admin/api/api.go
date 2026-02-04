package api

import (
	"encoding/json"
	"net/http"
	"net/http/pprof"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/entities"
	dberrs "github.com/webhookx-io/webhookx/db/errs"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/dispatcher"
	"github.com/webhookx-io/webhookx/pkg/declarative"
	"github.com/webhookx-io/webhookx/pkg/errs"
	"github.com/webhookx-io/webhookx/pkg/http/middlewares"
	"github.com/webhookx-io/webhookx/pkg/http/response"
	"github.com/webhookx-io/webhookx/pkg/openapi"
	"github.com/webhookx-io/webhookx/pkg/tracing"
	"github.com/webhookx-io/webhookx/pkg/tracing/instrumentations"
	"github.com/webhookx-io/webhookx/pkg/types"
	"github.com/webhookx-io/webhookx/services"
	"github.com/webhookx-io/webhookx/utils"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type API struct {
	cfg         *config.Config
	db          *db.DB
	dispatcher  *dispatcher.Dispatcher
	declarative *declarative.Declarative
	middlewares []mux.MiddlewareFunc
	services    *services.Services
}

type Options struct {
	Config      *config.Config
	DB          *db.DB
	Dispatcher  *dispatcher.Dispatcher
	Middlewares []mux.MiddlewareFunc
}

func NewAPI(opts Options, services *services.Services) *API {
	return &API{
		cfg:         opts.Config,
		db:          opts.DB,
		dispatcher:  opts.Dispatcher,
		declarative: declarative.NewDeclarative(opts.DB),
		middlewares: opts.Middlewares,
		services:    services,
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

func ValidateRequest(r *http.Request, defaults map[string]interface{}, target entities.Schema) error {
	data := make(map[string]interface{})
	if defaults != nil {
		utils.MergeMap(data, defaults)
	}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		return err
	}

	schema := entities.LookupSchema(target.SchemaName())
	err = openapi.Validate(schema, data)
	if err != nil {
		return err
	}

	err = utils.MapToStruct(data, target)
	if err != nil {
		panic(err)
	}

	return nil
}

func customizeErrorResponse(err error, w http.ResponseWriter) bool {
	if e, ok := err.(*dberrs.DBError); ok {
		response.JSON(w, 400, types.ErrorResponse{Message: e.Error()})
		return true
	}
	return false
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

	r.Use(instrumentations.NewInstrumentedMux().Handle)
	r.Use(middlewares.NewRecovery(customizeErrorResponse).Handle)
	r.Use(api.contextMiddleware)
	r.Use(api.licenseMiddleware)

	r.HandleFunc("/", api.Index).Methods("GET").Name("admin.root")
	r.HandleFunc("/license", api.GetLicense).Methods("GET").Name("admin.license.get")

	r.HandleFunc("/workspaces/{workspace}/config/sync", api.Sync).Methods("POST").Name("admin.config.sync")
	r.HandleFunc("/workspaces/{workspace}/config/dump", api.Dump).Methods("POST").Name("admin.config.dump")

	r.HandleFunc("/workspaces", api.PageWorkspace).Methods("GET").Name("admin.workspaces.page")
	r.HandleFunc("/workspaces", api.CreateWorkspace).Methods("POST").Name("admin.workspaces.create")
	r.HandleFunc("/workspaces/{id}", api.GetWorkspace).Methods("GET").Name("admin.workspaces.get")
	r.HandleFunc("/workspaces/{id}", api.UpdateWorkspace).Methods("PUT").Name("admin.workspaces.update")
	r.HandleFunc("/workspaces/{id}", api.DeleteWorkspace).Methods("DELETE").Name("admin.workspaces.delete")

	if api.cfg.Admin.DebugEndpoints {
		r.HandleFunc("/debug/pprof/profile", pprof.Profile).Methods("GET")
		r.HandleFunc("/debug/pprof/symbol", pprof.Symbol).Methods("GET")
		r.HandleFunc("/debug/pprof/trace", pprof.Trace).Methods("GET")
		r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline).Methods("GET")
		r.PathPrefix("/debug/pprof/").HandlerFunc(pprof.Index).Methods("GET")
	}

	for _, prefix := range []string{"", "/workspaces/{workspace}"} {
		r.HandleFunc(prefix+"/endpoints", api.PageEndpoint).Methods("GET").Name("admin.endpoints.page")
		r.HandleFunc(prefix+"/endpoints", api.CreateEndpoint).Methods("POST").Name("admin.endpoints.create")
		r.HandleFunc(prefix+"/endpoints/{id}", api.GetEndpoint).Methods("GET").Name("admin.endpoints.get")
		r.HandleFunc(prefix+"/endpoints/{id}", api.UpdateEndpoint).Methods("PUT").Name("admin.endpoints.update")
		r.HandleFunc(prefix+"/endpoints/{id}", api.DeleteEndpoint).Methods("DELETE").Name("admin.endpoints.delete")
	}

	for _, prefix := range []string{"", "/workspaces/{workspace}"} {
		r.HandleFunc(prefix+"/sources", api.PageSource).Methods("GET").Name("admin.sources.page")
		r.HandleFunc(prefix+"/sources", api.CreateSource).Methods("POST").Name("admin.sources.create")
		r.HandleFunc(prefix+"/sources/{id}", api.GetSource).Methods("GET").Name("admin.sources.get")
		r.HandleFunc(prefix+"/sources/{id}", api.UpdateSource).Methods("PUT").Name("admin.sources.update")
		r.HandleFunc(prefix+"/sources/{id}", api.DeleteSource).Methods("DELETE").Name("admin.sources.delete")
	}

	for _, prefix := range []string{"", "/workspaces/{workspace}"} {
		r.HandleFunc(prefix+"/events", api.PageEvent).Methods("GET").Name("admin.events.page")
		r.HandleFunc(prefix+"/events", api.CreateEvent).Methods("POST").Name("admin.events.create")
		r.HandleFunc(prefix+"/events/{id}", api.GetEvent).Methods("GET").Name("admin.events.get")
		r.HandleFunc(prefix+"/events/{id}/retry", api.RetryEvent).Methods("POST").Name("admin.events.retry")
	}

	for _, prefix := range []string{"", "/workspaces/{workspace}"} {
		r.HandleFunc(prefix+"/attempts", api.PageAttempt).Methods("GET").Name("admin.attempts.page")
		r.HandleFunc(prefix+"/attempts/{id}", api.GetAttempt).Methods("GET").Name("admin.attempts.get")
	}

	for _, prefix := range []string{"", "/workspaces/{workspace}"} {
		r.HandleFunc(prefix+"/plugins", api.PagePlugin).Methods("GET").Name("admin.plugins.page")
		r.HandleFunc(prefix+"/plugins", api.CreatePlugin).Methods("POST").Name("admin.plugins.create")
		r.HandleFunc(prefix+"/plugins/{id}", api.GetPlugin).Methods("GET").Name("admin.plugins.get")
		r.HandleFunc(prefix+"/plugins/{id}", api.UpdatePlugin).Methods("PUT").Name("admin.plugins.update")
		r.HandleFunc(prefix+"/plugins/{id}", api.DeletePlugin).Methods("DELETE").Name("admin.plugins.delete")
	}

	if tracing.Enabled("request") {
		return otelhttp.NewHandler(r, "admin.request")
	}
	return r
}
