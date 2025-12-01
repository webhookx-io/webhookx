package reports

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/pkg/license"
	"github.com/webhookx-io/webhookx/utils"
	"go.uber.org/zap"
)

var (
	// URL is report url
	URL = "https://report.webhookx.io/report"

	uid = utils.UUIDShort()
)

type data struct {
	UID         string `json:"uid"`
	Version     string `json:"version"`
	LicenseID   string `json:"license_id"`
	LicensePlan string `json:"license_plan"`
}

func send(url string) error {
	lic := license.GetLicenser().License()
	data := data{
		UID:         uid,
		Version:     config.VERSION,
		LicenseID:   lic.ID,
		LicensePlan: lic.Plan,
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

func Report() {
	err := send(URL)
	if err != nil {
		zap.S().Debugf("failed to report anonymous data: %v", err)
	}
}
