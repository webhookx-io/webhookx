package accesslog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/pkg/accesslog"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/test/helper/factory"
	"github.com/webhookx-io/webhookx/utils"
	"net/url"
	"regexp"
	"testing"
	"time"
)

func parse(line string) (map[string]string, error) {
	var buffer bytes.Buffer
	buffer.WriteString(`(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}\.\d{3})\s`) // 1 - TimeStamp
	buffer.WriteString(`\[([^\]]+)\]\s`)                                 // 2 - Name
	buffer.WriteString(`([\w:\.]+)`)                                     // 3 - IP
	buffer.WriteString(`\s-\s`)                                          // - - Spaces
	buffer.WriteString(`(\S+)\s`)                                        // 4 - Username
	buffer.WriteString(`"(\S*)\s?`)                                      // 5 - RequestMethod
	buffer.WriteString(`((?:[^"]*(?:\\")?)*)\s`)                         // 6 - RequestPath
	buffer.WriteString(`([^"]*)"\s`)                                     // 7 - RequestProto
	buffer.WriteString(`(\S+)\s`)                                        // 8 - ResponseStatus
	buffer.WriteString(`(\S+)\s`)                                        // 9 - ResponseContentSize
	buffer.WriteString(`(\S+)\s`)                                        // 10 - Duration
	buffer.WriteString(`"([^"]+)"\s`)                                    // 11 - Referer
	buffer.WriteString(`"([^"]+)"`)                                      // 12 - User-Agent

	regex, err := regexp.Compile(buffer.String())
	if err != nil {
		return nil, err
	}

	submatch := regex.FindStringSubmatch(line)

	result := make(map[string]string)
	result["ts"] = submatch[1]
	result["name"] = submatch[2]
	result["ip"] = submatch[3]
	result["username"] = submatch[4]
	result["method"] = submatch[5]
	result["path"] = submatch[6]
	result["proto"] = submatch[7]
	result["status"] = submatch[8]
	result["response_size"] = submatch[9]
	result["latency"] = submatch[10]
	result["referer"] = submatch[11]
	result["user_agent"] = submatch[12]

	return result, nil
}

type JsonEntry struct {
	Name      string `json:"name"`
	Timestamp string `json:"ts"`
	accesslog.Entry
}

var _ = Describe("access_log", Ordered, func() {

	Context("text", func() {
		var app *app.Application
		var adminClient *resty.Client
		var proxyClient *resty.Client

		BeforeAll(func() {
			entitiesConfig := helper.EntitiesConfig{
				Endpoints: []*entities.Endpoint{factory.EndpointP()},
				Sources:   []*entities.Source{factory.SourceP()},
			}
			helper.InitDB(true, &entitiesConfig)
			app = utils.Must(helper.Start(map[string]string{
				"NO_COLOR":                 "true",
				"WEBHOOKX_ADMIN_LISTEN":    "0.0.0.0:8080",
				"WEBHOOKX_PROXY_LISTEN":    "0.0.0.0:8081",
				"WEBHOOKX_ACCESS_LOG_FILE": "webhookx-access.log",
			}))
			adminClient = helper.AdminClient()
			proxyClient = helper.ProxyClient()
		})

		AfterAll(func() {
			app.Stop()
		})

		BeforeEach(func() {
			helper.TruncateFile("webhookx-access.log")
		})

		It("admin accesslog", func() {
			resp, err := adminClient.R().
				SetHeader("User-Agent", "WebhookX/dev").
				SetBasicAuth("username", "password").
				Get("/")
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())
			line, err := helper.FileLine("webhookx-access.log", 1)
			assert.Nil(GinkgoT(), err)
			entryMap, err := parse(line)
			assert.Nil(GinkgoT(), err)
			assert.Regexp(GinkgoT(), "\\d{4}/\\d{2}/\\d{2} \\d{2}:\\d{2}:\\d{2}\\.\\d{3}", entryMap["ts"])
			assert.Equal(GinkgoT(), "username", entryMap["username"])
			assert.Equal(GinkgoT(), "admin", entryMap["name"])
			assert.Regexp(GinkgoT(), "\\d+", entryMap["response_size"])
			assert.Equal(GinkgoT(), "/", entryMap["path"])
			assert.Equal(GinkgoT(), "GET", entryMap["method"])
			assert.Equal(GinkgoT(), "HTTP/1.1", entryMap["proto"])
			assert.Equal(GinkgoT(), "200", entryMap["status"])
			assert.Regexp(GinkgoT(), "\\d+ms", entryMap["latency"])
			assert.Equal(GinkgoT(), "-", entryMap["referer"])
			assert.Equal(GinkgoT(), "WebhookX/dev", entryMap["user_agent"])
		})

		It("proxy accesslog", func() {
			u, err := url.Parse(proxyClient.BaseURL)
			newURL := fmt.Sprintf("%s://username:password@%s/%s", u.Scheme, u.Host, u.Path)
			assert.Eventually(GinkgoT(), func() bool {
				resp, err := proxyClient.R().
					SetHeader("User-Agent", "WebhookX/dev").
					SetBody(`{
					    "event_type": "foo.bar",
					    "data": {
							"key": "value"
						}
					}`).
					Post(newURL)
				return err == nil && resp.StatusCode() == 200
			}, time.Second*5, time.Second)
			line, err := helper.FileLine("webhookx-access.log", 1)
			assert.Nil(GinkgoT(), err)
			entryMap, err := parse(line)
			assert.Nil(GinkgoT(), err)
			assert.Regexp(GinkgoT(), "\\d{4}/\\d{2}/\\d{2} \\d{2}:\\d{2}:\\d{2}\\.\\d{3}", entryMap["ts"])
			assert.Equal(GinkgoT(), "username", entryMap["username"])
			assert.Equal(GinkgoT(), "proxy", entryMap["name"])
			assert.Regexp(GinkgoT(), "\\d+", entryMap["response_size"])
			assert.Equal(GinkgoT(), "/", entryMap["path"])
			assert.Equal(GinkgoT(), "POST", entryMap["method"])
			assert.Equal(GinkgoT(), "HTTP/1.1", entryMap["proto"])
			assert.Equal(GinkgoT(), "200", entryMap["status"])
			assert.Regexp(GinkgoT(), "\\d+ms", entryMap["latency"])
			assert.Equal(GinkgoT(), "-", entryMap["referer"])
			assert.Equal(GinkgoT(), "WebhookX/dev", entryMap["user_agent"])
		})
	})

	Context("json", func() {
		var app *app.Application
		var adminClient *resty.Client
		var proxyClient *resty.Client

		BeforeAll(func() {
			entitiesConfig := helper.EntitiesConfig{
				Endpoints: []*entities.Endpoint{factory.EndpointP()},
				Sources:   []*entities.Source{factory.SourceP()},
			}
			helper.InitDB(true, &entitiesConfig)
			app = utils.Must(helper.Start(map[string]string{
				"NO_COLOR":                   "true",
				"WEBHOOKX_ADMIN_LISTEN":      "0.0.0.0:8080",
				"WEBHOOKX_PROXY_LISTEN":      "0.0.0.0:8081",
				"WEBHOOKX_ACCESS_LOG_FORMAT": "json",
				"WEBHOOKX_ACCESS_LOG_FILE":   "webhookx-access.log",
			}))
			adminClient = helper.AdminClient()
			proxyClient = helper.ProxyClient()
		})

		AfterAll(func() {
			app.Stop()
		})

		BeforeEach(func() {
			helper.TruncateFile("webhookx-access.log")
		})

		It("admin accesslog", func() {
			resp, err := adminClient.R().
				SetHeader("User-Agent", "WebhookX/dev").
				SetBasicAuth("username", "password").
				Get("/")
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())
			line, err := helper.FileLine("webhookx-access.log", 1)
			assert.Nil(GinkgoT(), err)
			var entry JsonEntry
			json.Unmarshal([]byte(line), &entry)
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), "admin", entry.Name)
			assert.Regexp(GinkgoT(), "\\d{4}/\\d{2}/\\d{2} \\d{2}:\\d{2}:\\d{2}\\.\\d{3}", entry.Timestamp)
			assert.Equal(GinkgoT(), "username", entry.Username)
			assert.Equal(GinkgoT(), "/", entry.Request.Path)
			assert.Equal(GinkgoT(), "GET", entry.Request.Method)
			assert.Equal(GinkgoT(), "HTTP/1.1", entry.Request.Proto)
			assert.Equal(GinkgoT(), "", entry.Request.Headers["referer"])
			assert.Equal(GinkgoT(), "WebhookX/dev", entry.Request.Headers["user-agent"])

			assert.Regexp(GinkgoT(), "\\d+", entry.Response.Size)
			assert.EqualValues(GinkgoT(), 200, entry.Response.Status)
			assert.LessOrEqual(GinkgoT(), time.Duration(0), entry.Latency)
		})

		It("proxy accesslog", func() {
			assert.Eventually(GinkgoT(), func() bool {
				resp, err := proxyClient.R().
					SetHeader("User-Agent", "WebhookX/dev").
					SetBasicAuth("username", "password").
					SetBody(`{
					    "event_type": "foo.bar",
					    "data": {
							"key": "value"
						}
					}`).
					Post("/")
				return err == nil && resp.StatusCode() == 200
			}, time.Second*5, time.Second)
			line, err := helper.FileLine("webhookx-access.log", 1)
			assert.Nil(GinkgoT(), err)

			var entry JsonEntry
			json.Unmarshal([]byte(line), &entry)
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), "proxy", entry.Name)
			assert.Regexp(GinkgoT(), "\\d{4}/\\d{2}/\\d{2} \\d{2}:\\d{2}:\\d{2}\\.\\d{3}", entry.Timestamp)
			assert.Equal(GinkgoT(), "username", entry.Username)
			assert.Equal(GinkgoT(), "/", entry.Request.Path)
			assert.Equal(GinkgoT(), "POST", entry.Request.Method)
			assert.Equal(GinkgoT(), "HTTP/1.1", entry.Request.Proto)
			assert.Equal(GinkgoT(), "", entry.Request.Headers["referer"])
			assert.Equal(GinkgoT(), "WebhookX/dev", entry.Request.Headers["user-agent"])

			assert.Regexp(GinkgoT(), "\\d+", entry.Response.Size)
			assert.EqualValues(GinkgoT(), 200, entry.Response.Status)
			assert.LessOrEqual(GinkgoT(), time.Duration(0), entry.Latency)
		})
	})

})

func TestAccessLog(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AccessLog Suite")
}
