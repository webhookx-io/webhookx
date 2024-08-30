package admin

import (
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/webhookx-io/webhookx/test"
	"github.com/webhookx-io/webhookx/test/helper"
	"testing"
)

type AdminListenSuite struct {
	test.BasicSuite

	adminClient *resty.Client
}

func (s *AdminListenSuite) SetupSuite() {
	s.BasicSuite.SetupSuite()
	assert.Nil(s.T(), s.BasicSuite.ResetDatabase())
	test.Start(map[string]string{
		"WEBHOOKX_ADMIN_LISTEN": "0.0.0.0:8080",
	})
	s.adminClient = helper.AdminClient()
}

func (s *AdminListenSuite) TearDownSuite() {
}

func (s *AdminListenSuite) Test() {
	resp, err := s.adminClient.R().Get("/")
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 200, resp.StatusCode())
}

func TestAdminListenSuite(t *testing.T) {
	suite.Run(t, new(AdminListenSuite))
}
