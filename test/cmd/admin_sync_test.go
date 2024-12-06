package cmd

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/cmd"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/utils"
	"net/http"
	"time"
)

func fullURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s%s", scheme, r.Host, r.URL.Path)
}

func startHTTP(handler http.HandlerFunc, addr string) *http.Server {
	s := &http.Server{
		Handler: handler,
		Addr:    addr,
	}
	go func() {
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("failed to start HTTP server: %s\n", err.Error())
		}
	}()
	return s
}

var _ = Describe("admin", Ordered, func() {
	Context("sync", func() {
		Context("sanity", func() {
			var app *app.Application

			BeforeAll(func() {
				helper.InitDB(true, nil)
				app = utils.Must(helper.Start(map[string]string{
					"WEBHOOKX_ADMIN_LISTEN":   "0.0.0.0:8080",
					"WEBHOOKX_PROXY_LISTEN":   "0.0.0.0:8081",
					"WEBHOOKX_WORKER_ENABLED": "true",
				}))
			})

			AfterAll(func() {
				app.Stop()
			})

			It("sanity", func() {
				output, err := executeCommand(cmd.NewRootCmd(), "admin", "sync", "../fixtures/webhookx.yaml")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), "", output)
			})

			Context("errors", func() {
				It("missing filename", func() {
					output, err := executeCommand(cmd.NewRootCmd(), "admin", "sync")
					assert.NotNil(GinkgoT(), err)
					assert.Equal(GinkgoT(), "Error: accepts 1 arg(s), received 0\n", output)
				})
				It("invalid filename", func() {
					output, err := executeCommand(cmd.NewRootCmd(), "admin", "sync", "unknown.yaml")
					assert.NotNil(GinkgoT(), err)
					assert.Equal(GinkgoT(), "Error: open unknown.yaml: no such file or directory\n", output)
				})
				It("invalid yaml", func() {
					output, err := executeCommand(cmd.NewRootCmd(), "admin", "sync", "../fixtures/invalid_webhookx.yaml")
					assert.NotNil(GinkgoT(), err)
					assert.Equal(GinkgoT(), "Error: invalid status code: 400 {\"message\":\"malformed yaml content: yaml: unmarshal errors:\\n  line 1: cannot unmarshal !!str `webhook...` into map[string]interface {}\"}\n", output)
				})
			})
		})

		Context("flags", Ordered, func() {
			It("--timeout", func() {
				server := startHTTP(func(writer http.ResponseWriter, r *http.Request) {
					time.Sleep(time.Second * 2)
				}, ":8080")
				output, err := executeCommand(cmd.NewRootCmd(), "admin", "sync", "../fixtures/webhookx.yaml", "--timeout", "1")
				assert.NotNil(GinkgoT(), err)
				assert.Equal(GinkgoT(), "Error: timeout\n", output)
				assert.Nil(GinkgoT(), server.Shutdown(context.TODO()))
			})

			It("--workspace", func() {
				var url string
				server := startHTTP(func(writer http.ResponseWriter, r *http.Request) {
					url = fullURL(r)
				}, "127.0.0.1:8080")
				output, err := executeCommand(cmd.NewRootCmd(), "admin", "sync", "../fixtures/webhookx.yaml", "--workspace", "foo")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), "", output)
				assert.Equal(GinkgoT(), "http://localhost:8080/workspaces/foo/sync", url)
				assert.Nil(GinkgoT(), server.Shutdown(context.TODO()))
			})

			It("--addr", func() {
				time.Sleep(time.Second * 5)
				var url string
				server := startHTTP(func(writer http.ResponseWriter, r *http.Request) {
					url = fullURL(r)
				}, "127.0.0.1:8888")
				output, err := executeCommand(cmd.NewRootCmd(), "admin", "sync", "../fixtures/webhookx.yaml", "--addr", "http://localhost:8888")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), "", output)
				assert.Equal(GinkgoT(), "http://localhost:8888/workspaces/default/sync", url)
				assert.Nil(GinkgoT(), server.Shutdown(context.TODO()))
			})
		})
	})
})
