package proxy

import (
	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/utils"
	"testing"
)

var _ = Describe("proxy", Ordered, func() {
	var app *app.Application
	var proxyClient *resty.Client
	BeforeAll(func() {
		helper.InitDB(true, nil)
		app = utils.Must(helper.Start(map[string]string{
			"WEBHOOKX_PROXY_LISTEN": "0.0.0.0:8081",
		}))
		proxyClient = helper.ProxyClient()
	})
	AfterAll(func() {
		app.Stop()
	})

	It("proxy listen", func() {
		resp, err := proxyClient.R().Get("/")
		assert.Nil(GinkgoT(), err)
		assert.Equal(GinkgoT(), 404, resp.StatusCode())
	})

})

func TestProxyListen(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Proxy Listen Suite")
}
