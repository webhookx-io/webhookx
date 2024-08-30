package deliverer

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/config"
	"testing"
	"time"
)

func Test(t *testing.T) {
	t.Run("sanity", func(t *testing.T) {
		cfg := config.WorkerDeliverer{
			Timeout: 10 * 1000,
		}
		deliverer := NewHTTPDeliverer(&cfg)

		req := &Request{
			URL:     "https://httpbin.org/anything",
			Method:  "POST",
			Payload: []byte(`{"foo": "bar"}`),
			Headers: map[string]string{
				"X-Key": "value",
			},
		}

		res := deliverer.Deliver(req)
		assert.NoError(t, res.Error)
		assert.Equal(t, res.StatusCode, 200)
		data := make(map[string]interface{})
		err := json.Unmarshal(res.ResponseBody, &data)
		assert.NoError(t, err)
		assert.Equal(t, data["data"], `{"foo": "bar"}`)
		headers := data["headers"].(map[string]interface{})
		assert.Equal(t, headers["X-Key"], "value")
	})

	t.Run("should fail with DeadlineExceeded error", func(t *testing.T) {
		cfg := config.WorkerDeliverer{
			Timeout: 10 * 1000,
		}
		deliverer := NewHTTPDeliverer(&cfg)

		req := &Request{
			URL:     "https://httpbin.org/anything",
			Method:  "GET",
			Timeout: time.Millisecond * 1,
		}

		res := deliverer.Deliver(req)
		assert.NotNil(t, res.Error)
		assert.True(t, errors.Is(res.Error, context.DeadlineExceeded))
	})

}
