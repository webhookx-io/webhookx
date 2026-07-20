package metrics

import "go.opentelemetry.io/otel/sdk/instrumentation"

type ResourceMetrics struct {
	Resource     Resource       `json:"resource,omitempty"`
	ScopeMetrics []ScopeMetrics `json:"scopeMetrics"`
}

type Resource struct {
	Attributes []KeyValue `json:"attributes"`
}

type KeyValue struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

type ScopeMetrics struct {
	Scope   instrumentation.Scope `json:"scope"`
	Metrics []Metrics             `json:"metrics"`
}

type Metrics struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Unit        string `json:"unit"`
	Sum         *Sum   `json:"sum"`
}

type Sum struct {
	DataPoints []NumberDataPoint `json:"dataPoints"`
}

type NumberDataPoint struct {
	AsDouble float64 `json:"asDouble"`
}

type ExportRequest struct {
	ResourceMetrics []ResourceMetrics `json:"resourceMetrics"`
}
