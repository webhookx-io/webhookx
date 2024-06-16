package api

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/pkg/errs"
	"net/http"
	"strconv"
)

const (
	ApplicationJsonType = "application/json"
)

type API struct {
	DB *db.DB
}

func NewAPI(cfg *config.Config) (*API, error) {
	db, err := db.NewDB(cfg)
	if err != nil {
		return nil, err
	}
	return &API{
		DB: db,
	}, nil
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
	w.Header().Set("Content-Type", ApplicationJsonType)
	w.WriteHeader(code)

	bytes, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	_, _ = w.Write(bytes)
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

	// TODO: order

	q.Page(uint64(page), uint64(pagesize))
}

func (api *API) error(code int, w http.ResponseWriter, err error) {
	if e, ok := err.(*errs.ValidateError); ok {
		api.json(code, w, ErrorResponse{
			Message: "Reqeust Validation",
			Error:   e,
		})
		return
	}
	api.json(code, w, ErrorResponse{Message: err.Error()})
}

func (api *API) notfound(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
}

func (api *API) assert(err error) {
	if err != nil {
		panic(err)
	}
}

// Handler returns a http.Handler
func (api *API) Handler() http.Handler {
	r := mux.NewRouter()

	r.Use(panicRecovery)

	r.HandleFunc("/", api.Index).Methods("GET")

	r.HandleFunc("/endpoints", api.PageEndpoint).Methods("GET")
	r.HandleFunc("/endpoints", api.CreateEndpoint).Methods("POST")
	r.HandleFunc("/endpoints/{id}", api.GetEndpoint).Methods("GET")
	r.HandleFunc("/endpoints/{id}", api.UpdateEndpoint).Methods("PUT")
	r.HandleFunc("/endpoints/{id}", api.DeleteEndpoint).Methods("DELETE")

	return r
}
