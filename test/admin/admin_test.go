package admin

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/webhookx-io/webhookx/test"
	"testing"
)

type AdminSuite struct {
	test.BasicSuite
}

func (s *AdminSuite) SetupSuite() {
	s.BasicSuite.SetupSuite()
	assert.Nil(s.T(), s.BasicSuite.ResetDatabase())
	test.Start(map[string]string{
		"WEBHOOKX_ADMIN_LISTEN": "0.0.0.0:8080",
	})
}

func (s *AdminSuite) TearDownSuite() {
}

func (s *AdminSuite) Test() {
	resp, err := s.Client.R().Get("/")
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 200, resp.StatusCode())
}

func TestAdminAPISuite(t *testing.T) {
	suite.Run(t, new(AdminSuite))
}
