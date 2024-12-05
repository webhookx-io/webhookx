package api

import (
	"encoding/json"
	"errors"
	"github.com/webhookx-io/webhookx/pkg/declarative"
	"github.com/webhookx-io/webhookx/pkg/ucontext"
	"gopkg.in/yaml.v3"
	"io"
	"net/http"
	"strings"
)

func toJSON(yamlstr []byte) ([]byte, error) {
	data := make(map[string]interface{})
	err := yaml.Unmarshal(yamlstr, &data)
	if err != nil {
		return nil, err
	}
	return json.Marshal(data)
}

func (api *API) Sync(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	api.assert(err)
	defer r.Body.Close()

	ct := r.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		body, err = toJSON(body)
		if err != nil {
			api.error(400, w, errors.New("malformed yaml content: "+err.Error()))
			return
		}
	}

	var cfg declarative.Configuration
	err = json.Unmarshal(body, &cfg)
	if err != nil {
		api.error(400, w, errors.New("invalid yaml content: "+err.Error()))
		return
	}

	if err := cfg.Validate(); err != nil {
		api.error(400, w, errors.New("invalid configuration: "+err.Error()))
		return
	}

	wid := ucontext.GetWorkspaceID(r.Context())
	err = api.declarative.Sync(wid, cfg)
	api.assert(err)

	api.json(200, w, nil)
}
