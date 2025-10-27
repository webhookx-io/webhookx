package deliverer

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/config"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func Test(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}

		headers := make(map[string]string)
		for k, v := range r.Header {
			if len(v) > 0 {
				headers[k] = v[0]
			}
		}

		resp := map[string]interface{}{
			"data":    string(body),
			"headers": headers,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	t.Run("sanity", func(t *testing.T) {
		cfg := config.WorkerDeliverer{
			Timeout: 10 * 1000,
		}
		deliverer := NewHTTPDeliverer(&cfg)

		req := &Request{
			URL:     server.URL,
			Method:  "POST",
			Payload: []byte(`{"foo": "bar"}`),
			Headers: map[string]string{
				"X-Key": "value",
			},
		}

		res := deliverer.Deliver(context.Background(), req)
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
			URL:     server.URL,
			Method:  "GET",
			Timeout: time.Microsecond * 1,
		}

		res := deliverer.Deliver(context.Background(), req)
		assert.NotNil(t, res.Error)
		assert.True(t, errors.Is(res.Error, context.DeadlineExceeded))
	})

}
