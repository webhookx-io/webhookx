package helper

import (
	"github.com/go-resty/resty/v2"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/migrator"
)

func AdminClient() *resty.Client {
	c := resty.New()
	c.SetBaseURL("http://localhost:8080")
	return c
}

func ProxyClient() *resty.Client {
	c := resty.New()
	c.SetBaseURL("http://localhost:8081")
	return c
}

func DB() *db.DB {
	cfg, err := config.Init()
	if err != nil {
		return nil
	}
	db, err := db.NewDB(&cfg.DatabaseConfig)
	if err != nil {
		return nil
	}
	return db
}

func ResetDB() error {
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
