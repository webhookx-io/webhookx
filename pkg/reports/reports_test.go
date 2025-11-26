package reports

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
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
