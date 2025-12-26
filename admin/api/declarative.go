package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/pkg/contextx"
	"github.com/webhookx-io/webhookx/pkg/declarative"
	"github.com/webhookx-io/webhookx/pkg/http/response"
	"gopkg.in/yaml.v3"
)

func yamlToJSON(yamlstr []byte) ([]byte, error) {
	data := make(map[string]interface{})
	err := yaml.Unmarshal(yamlstr, &data)
	if err != nil {
		return nil, err
	}
	return json.Marshal(data)
}

func jsonToYAML(jsonstr []byte) ([]byte, error) {
	var obj any
	if err := json.Unmarshal(jsonstr, &obj); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	defer func() { _ = enc.Close() }()

	if err := enc.Encode(obj); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (api *API) Sync(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	api.assert(err)
	defer func() { _ = r.Body.Close() }()

	ct := r.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		body, err = yamlToJSON(body)
		if err != nil {
			api.error(400, w, errors.New("malformed yaml content: "+err.Error()))
			return
		}
		// ensures request body can be read again
		r.ContentLength = int64(len(body))
		r.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(body)), nil
		}
		r.Body, _ = r.GetBody()
	}

	var cfg declarative.Configuration
	if err := ValidateRequest(r, nil, &cfg); err != nil {
		api.error(400, w, err)
		return
	}

	cfg.Init()

	if err := cfg.Validate(); err != nil {
		api.error(400, w, err)
		return
	}

	wid := contextx.GetWorkspaceID(r.Context())
	err = api.declarative.Sync(wid, &cfg)
	api.assert(err)

	api.json(200, w, nil)
}

func (api *API) Dump(w http.ResponseWriter, r *http.Request) {
	wid := contextx.GetWorkspaceID(r.Context())
	cfg, err := api.declarative.Dump(r.Context(), wid)
	if err != nil {
		api.error(400, w, err)
		return
	}

	for _, m := range cfg.Sources {
		m.BaseModel = entities.BaseModel{}
		for _, p := range m.Plugins {
			p.BaseModel = entities.BaseModel{}
		}
	}
	for _, m := range cfg.Endpoints {
		m.BaseModel = entities.BaseModel{}
		for _, p := range m.Plugins {
			p.BaseModel = entities.BaseModel{}
		}
	}

	b, err := json.Marshal(cfg)
	api.assert(err)

	b, err = jsonToYAML(b)
	api.assert(err)

	response.Text(w, 200, string(b))
}
