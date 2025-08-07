package reports

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSend(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	err := send(server.URL)
	assert.NotNil(t, err)
	assert.Equal(t, "HTTP status 404", err.Error())

	err = send("http://localhost:80")
	assert.NotNil(t, err)
	assert.Equal(t, "Post \"http://localhost:80\": dial tcp [::1]:80: connect: connection refused", err.Error())
}

func Test(t *testing.T) {
	var n = 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/report" {
			n = n + 1
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	url := server.URL + "/report"
	start(url, time.Second, 0)
	time.Sleep(time.Second * 3)
	assert.True(t, n >= 2)
}
