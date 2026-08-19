package dashboard

import (
	"bytes"
	"io/fs"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"

	"github.com/gorilla/mux"
	"github.com/webhookx-io/webhookx"
	"github.com/webhookx-io/webhookx/pkg/http/response"
)

type API struct {
	adminURL   *url.URL
	adminProxy *httputil.ReverseProxy
	fs         fs.FS
	fileServer http.Handler
}

func NewAPI(adminURL *url.URL, fs fs.FS) *API {
	return &API{
		adminURL:   adminURL,
		adminProxy: httputil.NewSingleHostReverseProxy(adminURL),
		fs:         fs,
		fileServer: http.FileServer(http.FS(fs)),
	}

}

func (api *API) Handler() http.Handler {
	r := mux.NewRouter()

	r.HandleFunc("/config", api.GetConfig).Methods("GET")
	r.PathPrefix("/api").Handler(http.StripPrefix("/api", api.adminProxy))
	r.PathPrefix("/").HandlerFunc(api.ServeStaticFile).Methods("GET")

	return r
}

func (api *API) GetConfig(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"admin_address": api.adminURL.String(),
		"version":       webhookx.VERSION,
		"commit_hash":   webhookx.COMMIT,
	}
	response.JSON(w, 200, data)
}

func (api *API) ServeStaticFile(w http.ResponseWriter, r *http.Request) {
	requestPath := path.Clean("/" + r.URL.Path)
	fileName := strings.TrimPrefix(requestPath, "/")
	if fileName == "" {
		fileName = "."
	}

	info, err := fs.Stat(api.fs, fileName)
	if err == nil && !info.IsDir() {
		api.fileServer.ServeHTTP(w, r)
		return
	}

	if path.Ext(requestPath) != "" {
		api.fileServer.ServeHTTP(w, r)
		return
	}

	serveIndexHTML(w, r, api.fs)
}

func serveIndexHTML(w http.ResponseWriter, r *http.Request, distFS fs.FS) {
	content, err := fs.ReadFile(distFS, "index.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	info, err := fs.Stat(distFS, "index.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	http.ServeContent(w, r, "index.html", info.ModTime(), bytes.NewReader(content))
}
