package proxy

import (
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/test"
	"github.com/webhookx-io/webhookx/test/helper"
	"testing"
)

type ProxyListenSuite struct {
	test.BasicSuite

	app         *app.Application
	proxyClient *resty.Client
}

func (s *ProxyListenSuite) SetupSuite() {
	s.BasicSuite.SetupSuite()
	assert.Nil(s.T(), s.BasicSuite.ResetDatabase())
	app, err := test.Start(map[string]string{
		"WEBHOOKX_PROXY_LISTEN": "0.0.0.0:8081",
	})
	assert.Nil(s.T(), err)
	s.app = app
	s.proxyClient = helper.ProxyClient()
}

func (s *ProxyListenSuite) TearDownSuite() {
	s.app.Stop()
}

func (s *ProxyListenSuite) Test() {
	resp, err := s.proxyClient.R().Get("/")
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 404, resp.StatusCode())
}

func TestProxyListenSuite(t *testing.T) {
	suite.Run(t, new(ProxyListenSuite))
}
