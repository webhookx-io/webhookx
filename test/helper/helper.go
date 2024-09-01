package helper

import (
	"bufio"
	"github.com/go-resty/resty/v2"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/migrator"
	"os"
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

func TruncateFile(filename string) {
	err := os.Truncate(filename, 0)
	if err != nil {
		panic("failed to truncate file: " + err.Error())
	}
}

func FileLine(filename string, n int) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for i := 1; scanner.Scan(); i++ {
		s := scanner.Text()
		if i == n {
			return s, nil
		}
	}

	return "", nil
}
