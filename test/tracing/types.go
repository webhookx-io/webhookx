package tracing

import "fmt"

type ExportedTrace struct {
	ResourceSpans []struct {
		Resource struct {
			Attributes []struct {
				Key   string `json:"key"`
				Value struct {
					ArrayValue *struct {
						Values []struct {
							StringValue string `json:"stringValue"`
						} `json:"values"`
					} `json:"arrayValue,omitempty"`
					IntValue    *string `json:"intValue,omitempty"`
					StringValue *string `json:"stringValue,omitempty"`
				} `json:"value"`
			} `json:"attributes"`
		} `json:"resource"`
		SchemaURL  string `json:"schemaUrl"`
		ScopeSpans []struct {
			Scope struct {
				Name    string  `json:"name"`
				Version *string `json:"version,omitempty"`
			} `json:"scope"`
			Spans []struct {
				Attributes []struct {
					Key   string `json:"key"`
					Value struct {
						BoolValue   *bool   `json:"boolValue,omitempty"`
						IntValue    *string `json:"intValue,omitempty"`
						StringValue *string `json:"stringValue,omitempty"`
						ArrayValue  *struct {
							Values []struct {
								StringValue string `json:"stringValue"`
							} `json:"values"`
						} `json:"arrayValue,omitempty"`
					} `json:"value"`
				} `json:"attributes,omitempty"`
				EndTimeUnixNano   string `json:"endTimeUnixNano"`
				Flags             int    `json:"flags"`
				Kind              int    `json:"kind"`
				Name              string `json:"name"`
				ParentSpanID      string `json:"parentSpanId"`
				SpanID            string `json:"spanId"`
				StartTimeUnixNano string `json:"startTimeUnixNano"`
				Status            struct {
				} `json:"status"`
				TraceID string `json:"traceId"`
			} `json:"spans"`
		} `json:"scopeSpans"`
	} `json:"resourceSpans"`
}

func (t *ExportedTrace) filterSpansByTraceID(traceID string) (scopeNames map[string]bool, spanAttrs map[string]map[string]string) {
	scopeNames = make(map[string]bool)
	spanAttrs = make(map[string]map[string]string)
	for _, resourceSpan := range t.ResourceSpans {
		scopeSpans := resourceSpan.ScopeSpans
		for _, scopeSpan := range scopeSpans {
			scopeNames[scopeSpan.Scope.Name] = true
			for _, span := range scopeSpan.Spans {
				if span.TraceID != traceID {
					continue
				}
				attributes := make(map[string]string)
				for _, attr := range span.Attributes {
					if attr.Value.StringValue != nil {
						attributes[attr.Key] = *attr.Value.StringValue
					} else if attr.Value.IntValue != nil {
						attributes[attr.Key] = *attr.Value.IntValue
					} else if attr.Value.BoolValue != nil {
						attributes[attr.Key] = fmt.Sprint(*attr.Value.BoolValue)
					} else if attr.Value.ArrayValue != nil {
						if len(attr.Value.ArrayValue.Values) == 1 {
							attributes[attr.Key] = attr.Value.ArrayValue.Values[0].StringValue
						} else {
							var values []string
							for _, v := range attr.Value.ArrayValue.Values {
								values = append(values, v.StringValue)
							}
							attributes[attr.Key] = fmt.Sprintf("[%s]", values)
						}
					}
				}
				spanAttrs[span.Name] = attributes
			}
		}
	}
	return
}

func (t *ExportedTrace) getTraceIDBySpanName(spanName string) string {
	for _, resourceSpan := range t.ResourceSpans {
		scopeSpans := resourceSpan.ScopeSpans
		for _, scopeSpan := range scopeSpans {
			for _, span := range scopeSpan.Spans {
				if span.Name == spanName {
					return span.TraceID
				}
			}
		}
	}
	return ""
}
