package deliverer

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test(t *testing.T) {
	deliverer := NewHTTPDeliverer(nil)
	req := &Request{
		URL:     "https://httpbin.org/anything",
		Method:  "POST",
		Payload: []byte("{\"hello\": \"world\"}"),
		Headers: map[string]string{
			"X-Key": "value",
		},
	}

	res, err := deliverer.Deliver(req)
	assert.NoError(t, err)
	assert.Equal(t, res.StatusCode, 200)
	data := make(map[string]interface{})
	json.Unmarshal(res.ResponseBody, &data)
	assert.Equal(t, data["data"], `{"hello": "world"}`)
	headers := data["headers"].(map[string]interface{})
	assert.Equal(t, headers["X-Key"], "value")
}
