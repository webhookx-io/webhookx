package test

import (
	"github.com/stretchr/testify/suite"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/db/migrator"
	"os"
	"time"
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
	for name, value := range envs {
		err := os.Setenv(name, value)
		if err != nil {
			return nil, err
		}
	}

	cfg, err := config.Init()
	if err != nil {
		return nil, err
	}

	app, err := app.NewApplication(cfg)
	if err != nil {
		return nil, err
	}
	go app.Start()

	time.Sleep(time.Second)
	return app, nil
}
