package proxy

import (
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/webhookx-io/webhookx/admin/api"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/test"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/utils"
	"testing"
	"time"
)

type ProxySuite struct {
	test.BasicSuite

	app *app.Application

	adminClient *resty.Client
	proxyClient *resty.Client

	endpoint entities.Endpoint
	source   entities.Source
}

func (s *ProxySuite) SetupSuite() {
	s.BasicSuite.SetupSuite()
	assert.Nil(s.T(), s.BasicSuite.ResetDatabase())
	app, err := test.Start(map[string]string{
		"WEBHOOKX_ADMIN_LISTEN":   "0.0.0.0:8080",
		"WEBHOOKX_PROXY_LISTEN":   "0.0.0.0:8081",
		"WEBHOOKX_WORKER_ENABLED": "true",
	})
	assert.Nil(s.T(), err)
	s.app = app

	s.adminClient = helper.AdminClient()
	s.proxyClient = helper.ProxyClient()

	resp, err := s.adminClient.R().
		SetBody(`
		{
	        "request": {
	            "url": "https://httpbin.org/anything",
	            "method": "POST"
	        },
	        "events": [
	          "charge.succeeded"
	        ]
	    }`).
		SetResult(&s.endpoint).
		Post("/workspaces/default/endpoints")
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 201, resp.StatusCode())

	resp, err = s.adminClient.R().
		SetBody(`
		{
		    "path": "/",
			"methods": ["POST"]
		}`).
		SetResult(&s.source).
		Post("/workspaces/default/sources")
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 201, resp.StatusCode())

}

func (s *ProxySuite) TearDownSuite() {
	s.app.Stop()
}

func (s *ProxySuite) Test_SentEvent() {
	time.Sleep(time.Second * 5)

	resp, err := s.proxyClient.R().
		SetBody(`
		{
		    "event_type": "charge.succeeded",
		    "data": {
		        "key": "value"
		    }
		}`).
		Post("/")
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 200, resp.StatusCode())

	time.Sleep(time.Second * 10)

	resp, err = s.adminClient.R().
		SetResult(&api.Pagination[*entities.Attempt]{}).
		Get("/workspaces/default/attempts")
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 200, resp.StatusCode())
	result := resp.Result().(*api.Pagination[*entities.Attempt])
	assert.Equal(s.T(), int64(1), result.Total)
	attempt := result.Data[0]
	assert.Equal(s.T(), entities.AttemptStatusSuccess, attempt.Status)
	assert.Equal(s.T(), s.endpoint.ID, attempt.EndpointId)

	assert.Equal(s.T(), &entities.AttemptRequest{
		Method: "POST",
		URL:    "https://httpbin.org/anything",
		Headers: map[string]string{
			"Content-Type": "application/json; charset=utf-8",
			"User-Agent":   "WebhookX/" + config.VERSION,
		},
		Body: utils.Pointer(`{"key": "value"}`),
	}, attempt.Request)
	assert.Equal(s.T(), 200, attempt.Response.Status)
}

func TestProxySuite(t *testing.T) {
	suite.Run(t, new(ProxySuite))
}
