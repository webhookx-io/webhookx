package reports

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/pkg/schedule"
	"github.com/webhookx-io/webhookx/utils"
	"go.uber.org/zap"
	"io"
	"net/http"
	"time"
)

const url = "https://report.webhookx.io/report"
const internal = time.Hour * 24
const initialDelay = time.Hour

var uid = utils.UUIDShort()

type data struct {
	UID     string `json:"uid"`
	Version string `json:"version"`
}

func send() error {
	data := data{
		UID:     uid,
		Version: config.VERSION,
	}

	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(data)
	if err != nil {
		return err
	}

	client := &http.Client{
		Timeout: time.Second * 15,
		Transport: &http.Transport{
			DisableKeepAlives:     true,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
	resp, err := client.Post(url, "application/json", buf)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	return nil
}

func Start() {
	schedule.Schedule(context.TODO(), func() {
		err := send()
		if err != nil {
			zap.S().Errorf("failed to report anonymous data: %v", err)
		}
	}, internal, initialDelay)
}
