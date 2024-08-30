package test

import (
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/suite"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/db/migrator"
	"os"
)

type BasicSuite struct {
	suite.Suite

	Client *resty.Client
}

func (s *BasicSuite) SetupSuite() {
	c := resty.New()
	c.SetBaseURL("http://localhost:8080")
	s.Client = c
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

func Start(envs map[string]string) error {
	for name, value := range envs {
		err := os.Setenv(name, value)
		if err != nil {
			return err
		}
	}

	cfg, err := config.Init()
	if err != nil {
		return err
	}

	app, err := app.NewApplication(cfg)
	if err != nil {
		return err
	}
	go app.Start()

	return nil
}
