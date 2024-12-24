package api

import (
	"bytes"
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

func (api *API) Dump(w http.ResponseWriter, r *http.Request) {
	wid := ucontext.GetWorkspaceID(r.Context())
	cfg, err := api.declarative.Dump(r.Context(), wid)
	if err != nil {
		api.error(400, w, err)
		return
	}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	defer encoder.Close()

	err = encoder.Encode(cfg)
	if err != nil {
		api.error(400, w, err)
		return
	}

	api.text(200, w, buf.String())
}
