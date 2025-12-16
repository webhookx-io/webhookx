package license

import (
	"slices"
)

type Plan struct {
	Name          string
	Features      []string
	Plugins       []string
	ForbiddenAPIs map[string]*Condition
	Limits        map[string]int
}

func (p Plan) HasFeature(feature string) bool {
	return slices.Contains(p.Features, feature)
}

func (p Plan) HasPlugin(name string) bool {
	return slices.Contains(p.Plugins, name)
}

type Condition struct {
	Methods                 []string
	ExcludeDefaultWorkspace bool
}

var (
	FreePlugins = []string{
		"webhookx-signature",
		"wasm",
		"function",
		"basic-auth",
		"key-auth",
		"hmac-auth",
		"jsonschema-validator",
	}

	EnterprisePlugins = []string{
		"integration-auth",
	}
)

var plans = map[string]Plan{
	"free": {
		Name:     "free",
		Plugins:  FreePlugins,
		Features: []string{},
		ForbiddenAPIs: map[string]*Condition{
			"/workspaces":                               {Methods: []string{"POST"}},
			"/workspaces/{id}":                          {Methods: []string{"DELETE"}},
			"/workspaces/{workspace}/config/sync":       {Methods: []string{"POST"}, ExcludeDefaultWorkspace: true},
			"/workspaces/{workspace}/config/dump":       {Methods: []string{"POST"}, ExcludeDefaultWorkspace: true},
			"/workspaces/{workspace}/endpoints":         {Methods: []string{"POST"}, ExcludeDefaultWorkspace: true},
			"/workspaces/{workspace}/endpoints/{id}":    {Methods: []string{"PUT", "DELETE"}, ExcludeDefaultWorkspace: true},
			"/workspaces/{workspace}/sources":           {Methods: []string{"POST"}, ExcludeDefaultWorkspace: true},
			"/workspaces/{workspace}/sources/{id}":      {Methods: []string{"PUT", "DELETE"}, ExcludeDefaultWorkspace: true},
			"/workspaces/{workspace}/events":            {Methods: []string{"POST"}, ExcludeDefaultWorkspace: true},
			"/workspaces/{workspace}/events/{id}/retry": {Methods: []string{"POST"}, ExcludeDefaultWorkspace: true},
			"/workspaces/{workspace}/plugins":           {Methods: []string{"POST"}, ExcludeDefaultWorkspace: true},
			"/workspaces/{workspace}/plugins/{id}":      {Methods: []string{"PUT", "DELETE"}, ExcludeDefaultWorkspace: true},
		},
		Limits: map[string]int{},
	},
	"enterprise": {
		Name:          "enterprise",
		Plugins:       append(FreePlugins, EnterprisePlugins...),
		Features:      []string{"secret"},
		ForbiddenAPIs: map[string]*Condition{},
		Limits:        map[string]int{},
	},
}
