package license

import (
	"slices"
)

type Plan struct {
	Name          string
	Features      []string
	ForbiddenAPIs map[string]*Condition
	Limits        map[string]int
}

func (p Plan) HasFeature(feature string) bool {
	return slices.Contains(p.Features, feature)
}

type Condition struct {
	Methods                 []string
	ExcludeDefaultWorkspace bool
}

var plans = map[string]Plan{
	"free": {
		Name:     "free",
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
		Features:      []string{"secret"},
		ForbiddenAPIs: map[string]*Condition{},
		Limits:        map[string]int{},
	},
}
