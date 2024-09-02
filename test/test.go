package test

import (
	"github.com/stretchr/testify/suite"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/db/migrator"
	"github.com/webhookx-io/webhookx/test/helper"
)

type BasicSuite struct {
	suite.Suite
}

func (s *BasicSuite) SetupSuite() {

}

func (s *BasicSuite) ResetDatabase() error {
	cfg, err := config.Init()
	if err != nil {
		return err
	}

	migrator := migrator.New(&cfg.DatabaseConfig)
	err = migrator.Reset()
	if err != nil {
		return err
	}
	return migrator.Up()
}

func Start(envs map[string]string) (*app.Application, error) {
	return helper.Start(envs)
}
