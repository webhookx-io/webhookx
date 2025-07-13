package utils

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestConvertJSONPaths(t *testing.T) {
	type args struct {
		input string
	}
	tests := []struct {
		name string
		args args
		want map[string]interface{}
	}{
		{
			name: "simple path",
			args: args{
				input: `{
					"@body.a": ["aaa"],
					"@body.b.c": ["bbb-ccc"],
					"@body.d.0": ["d0"],
					"@body.d.1": ["d1"]
				
				}`,
			},
			want: map[string]interface{}{
				"@body": map[string]interface{}{
					"a": []interface{}{"aaa"},
					"b": map[string]interface{}{
						"c": []interface{}{"bbb-ccc"},
					},
					"d": []interface{}{
						[]interface{}{"d0"},
						[]interface{}{"d1"},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var input map[string][]interface{}
			if err := json.Unmarshal([]byte(tt.args.input), &input); err != nil {
				t.Fatalf("failed to unmarshal input: %v", err)
			}
			if got := ConvertJSONPaths(input); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConvertJSONPaths() = %v, want %v", got, tt.want)
			} else {
				t.Logf("got: %s", func() string {
					b, _ := json.MarshalIndent(got, "", "  ")
					return string(b)
				}())
			}
		})
	}
}
